// Package mcpcheck verifies a live MCP endpoint against the requirements of
// the Anthropic and OpenAI directories (M-SERVEAPI-DIRECTORY-READY L3). It
// speaks raw JSON-RPC over streamable HTTP so it judges exactly what is on the
// wire, not what an SDK fills in. Every finding cites its requirement from
// design_docs/planned/v0_51_0/m-serveapi-directory-ready-sources.md.
//
// Probing: to find gated tools it sends tools/call with {} and no token, but
// only to tools that are read-only or have required parameters (an MCP server
// rejects those before running anything). A write tool with no required
// parameters is never called.
package mcpcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Status of one finding. Only FAIL makes the run fail.
const (
	Pass = "PASS"
	Fail = "FAIL"
	Warn = "WARN"
	Skip = "SKIP"
)

// Finding is one check result.
type Finding struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Tool    string `json:"tool,omitempty"`
	Message string `json:"message"`
	Source  string `json:"source,omitempty"` // requirement ID in the sources file
}

// Options configures a run. Target is "anthropic", "openai" or "both".
type Options struct {
	URL    string
	Target string
	Client *http.Client
}

// Failed reports whether any finding is a FAIL.
func Failed(fs []Finding) bool {
	for _, f := range fs {
		if f.Status == Fail {
			return true
		}
	}
	return false
}

type tool struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
	Meta        map[string]any `json:"_meta"`
	// SecuritySchemes is OpenAI's top-level mixed-auth declaration (O10).
	SecuritySchemes any `json:"securitySchemes"`
}

type session struct {
	opts Options
	sid  string
	next int
}

// Run checks the endpoint. An error means it could not be checked at all
// (unreachable, or not an MCP endpoint), which is not the same as a FAIL.
func Run(ctx context.Context, opts Options) ([]Finding, error) {
	if opts.Client == nil {
		opts.Client = &http.Client{Timeout: 30 * time.Second}
	}
	if opts.Target == "" {
		opts.Target = "anthropic"
	}
	s := &session{opts: opts}
	if st, _, _, err := s.rpc(ctx, "initialize", map[string]any{
		"protocolVersion": "2025-06-18", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "ailang-mcp-check", "version": "1"},
	}); err != nil {
		return nil, err
	} else if st == http.StatusUnauthorized {
		return []Finding{{Check: "lazy-auth", Status: Warn, Source: "A3",
			Message: "initialize itself requires a token, so tools cannot be listed without signing in; Claude expects discovery to work before sign-in"}}, nil
	}
	_, _, _, _ = s.rpc(ctx, "notifications/initialized", nil)
	st, _, res, err := s.rpc(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if st != http.StatusOK {
		return nil, fmt.Errorf("tools/list: HTTP %d", st)
	}
	var list struct {
		Tools []tool `json:"tools"`
	}
	if err := json.Unmarshal(res, &list); err != nil {
		return nil, fmt.Errorf("tools/list: %v", err)
	}

	var fs []Finding
	fs = append(fs, checkAnnotations(list.Tools, opts.Target)...)
	creds := checkCredentials(list.Tools)
	fs = append(fs, creds...)
	fs = append(fs, checkZeroArg(list.Tools)...)
	authFs, issuer, gated := s.checkLazyAuth(ctx, list.Tools, hasFail(creds))
	fs = append(fs, authFs...)
	fs = append(fs, s.checkAuthorizationServer(ctx, issuer)...)
	if opts.Target == "openai" || opts.Target == "both" {
		fs = append(fs, checkFileParams(list.Tools)...)
		fs = append(fs, checkSecuritySchemes(list.Tools, gated)...)
	}
	return fs, nil
}

func hasFail(fs []Finding) bool { return Failed(fs) }

// ---- check 1: titles and hints (A7; O8 for openai) ----

func checkAnnotations(tools []tool, target string) []Finding {
	var fs []Finding
	openai := target == "openai" || target == "both"
	for _, t := range tools {
		title := t.Title
		if title == "" {
			if s, ok := t.Annotations["title"].(string); ok {
				title = s
			}
		}
		if title == "" {
			fs = append(fs, Finding{"annotations", Fail, t.Name, "no title", "A7"})
		}
		_, ro := t.Annotations["readOnlyHint"].(bool)
		_, de := t.Annotations["destructiveHint"].(bool)
		if !ro && !de {
			fs = append(fs, Finding{"annotations", Fail, t.Name, "neither readOnlyHint nor destructiveHint is declared", "A7"})
		}
		if openai {
			for _, k := range []string{"readOnlyHint", "destructiveHint", "openWorldHint"} {
				if _, ok := t.Annotations[k].(bool); !ok {
					fs = append(fs, Finding{"annotations", Fail, t.Name, k + " is not an explicit boolean", "O8"})
				}
			}
		}
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"annotations", Pass, "", fmt.Sprintf("all %d tools have a title and hints", len(tools)), "A7"})
	}
	return fs
}

// ---- check 2: no credential-shaped parameters (A1/A2; O1) ----

var credentialName = regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|passwd|credential|^auth$|authorization)`)

// A widget-only tool (MCP Apps _meta.ui.visibility without "model", e.g.
// serve-api's @mcp_app_only) is called by the host's widget and never listed
// to the model, so a credential-shaped param there is not the model handling
// a secret: it is reported as SKIP, not FAIL.
func checkCredentials(tools []tool) []Finding {
	var fs []Finding
	skipped := false
	for _, t := range tools {
		props, _ := t.InputSchema["properties"].(map[string]any)
		widgetOnly := hiddenFromModel(t)
		for name := range props {
			if !credentialName.MatchString(name) {
				continue
			}
			if widgetOnly {
				skipped = true
				fs = append(fs, Finding{"credentials", Skip, t.Name,
					fmt.Sprintf("parameter %q looks like a credential — skipped: widget-only tool (_meta.ui.visibility excludes \"model\", so the model never sees it)", name), "A2,O1"})
				continue
			}
			fs = append(fs, Finding{"credentials", Fail, t.Name,
				fmt.Sprintf("parameter %q looks like a credential: the model would handle a secret (use OAuth; on serve-api mark it @mcp_secret)", name), "A2,O1"})
		}
	}
	switch {
	case Failed(fs):
	case skipped:
		fs = append(fs, Finding{"credentials", Pass, "", "no credential-shaped parameters on model-visible tools", "A2,O1"})
	default:
		fs = append(fs, Finding{"credentials", Pass, "", "no credential-shaped parameters", "A2,O1"})
	}
	return fs
}

// hiddenFromModel: the tool declares an MCP Apps visibility that excludes
// "model" (absent = the default ["model", "app"]).
func hiddenFromModel(t tool) bool {
	ui, _ := t.Meta["ui"].(map[string]any)
	vis, declared := ui["visibility"].([]any)
	if !declared {
		return false
	}
	for _, v := range vis {
		if v == "model" {
			return false
		}
	}
	return true
}

// ---- check 3: zero-argument tools are callable with {} ----

func checkZeroArg(tools []tool) []Finding {
	var fs []Finding
	for _, t := range tools {
		for _, r := range required(t) {
			if r == "_" {
				fs = append(fs, Finding{"zero-arg", Fail, t.Name,
					`requires a parameter named "_" (a desugared unit argument), so a call with {} is rejected`, ""})
			}
		}
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"zero-arg", Pass, "", "no tool requires a placeholder argument", ""})
	}
	return fs
}

func required(t tool) []string {
	var out []string
	if rs, ok := t.InputSchema["required"].([]any); ok {
		for _, r := range rs {
			if s, ok := r.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// ---- check 4: lazy auth (A3) ----

var resourceMetadataRe = regexp.MustCompile(`resource_metadata="([^"]+)"`)

func (s *session) checkLazyAuth(ctx context.Context, tools []tool, credentialParams bool) ([]Finding, string, map[string]bool) {
	var metaURL string
	gated := 0
	gatedNames := map[string]bool{}
	for _, t := range tools {
		ro, _ := t.Annotations["readOnlyHint"].(bool)
		if !ro && len(required(t)) == 0 {
			continue // never call a write tool that could run with {}
		}
		st, hdr, _, err := s.rpc(ctx, "tools/call", map[string]any{"name": t.Name, "arguments": map[string]any{}})
		if err != nil || st != http.StatusUnauthorized {
			continue
		}
		gated++
		gatedNames[t.Name] = true
		m := resourceMetadataRe.FindStringSubmatch(hdr.Get("WWW-Authenticate"))
		if m == nil {
			return []Finding{{"lazy-auth", Fail, t.Name, "401 without a resource_metadata challenge in WWW-Authenticate", "A3"}}, "", gatedNames
		}
		metaURL = m[1]
	}
	if gated == 0 {
		if credentialParams {
			return []Finding{{"lazy-auth", Fail, "", "no tool answers 401 without a token, yet tools take credentials as arguments: authenticate with OAuth, not arguments", "A1,A3"}}, "", gatedNames
		}
		return []Finding{{"lazy-auth", Pass, "", "no gated tools found (a public-data service needs no auth)", "A2"}}, "", gatedNames
	}
	var meta struct {
		Resource string   `json:"resource"`
		Servers  []string `json:"authorization_servers"`
	}
	if err := s.getJSON(ctx, metaURL, &meta); err != nil {
		return []Finding{{"lazy-auth", Fail, "", fmt.Sprintf("resource metadata %s: %v", metaURL, err), "A3"}}, "", gatedNames
	}
	var fs []Finding
	if strings.TrimSuffix(meta.Resource, "/") != strings.TrimSuffix(s.opts.URL, "/") {
		fs = append(fs, Finding{"lazy-auth", Fail, "", fmt.Sprintf("resource metadata names resource %q, but the endpoint is %q", meta.Resource, s.opts.URL), "A3"})
	}
	if len(meta.Servers) == 0 {
		fs = append(fs, Finding{"lazy-auth", Fail, "", "resource metadata lists no authorization_servers", "A3"})
		return fs, "", gatedNames
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"lazy-auth", Pass, "", fmt.Sprintf("%d gated tool(s) answer 401 with resource metadata naming %s", gated, meta.Servers[0]), "A3"})
	}
	return fs, meta.Servers[0], gatedNames
}

// ---- check 5: authorization server metadata (A5/A6; O6) ----

func (s *session) checkAuthorizationServer(ctx context.Context, issuer string) []Finding {
	if issuer == "" {
		return []Finding{{"authorization-server", Skip, "", "no authorization server named (no gated tools)", "A6"}}
	}
	start := time.Now()
	var md struct {
		PKCE     []string `json:"code_challenge_methods_supported"`
		CIMD     bool     `json:"client_id_metadata_document_supported"`
		AuthMeth []string `json:"token_endpoint_auth_methods_supported"`
		Register string   `json:"registration_endpoint"`
	}
	url := wellKnown(issuer, "oauth-authorization-server")
	if err := s.getJSON(ctx, url, &md); err != nil {
		if err2 := s.getJSON(ctx, wellKnown(issuer, "openid-configuration"), &md); err2 != nil {
			return []Finding{{"authorization-server", Fail, "", fmt.Sprintf("no metadata at %s: %v", url, err), "A6"}}
		}
	}
	var fs []Finding
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		fs = append(fs, Finding{"authorization-server", Fail, "", fmt.Sprintf("metadata took %v (Claude allows 10s)", elapsed.Round(time.Millisecond)), "A5"})
	}
	if !contains(md.PKCE, "S256") {
		fs = append(fs, Finding{"authorization-server", Fail, "", "code_challenge_methods_supported lacks S256", "A6"})
	}
	if !(md.CIMD && contains(md.AuthMeth, "none")) && md.Register == "" {
		fs = append(fs, Finding{"authorization-server", Fail, "", "neither CIMD (client_id_metadata_document_supported + \"none\" auth) nor a registration_endpoint", "A6,O6"})
	}
	if len(fs) == 0 {
		fs = append(fs, Finding{"authorization-server", Pass, "", issuer + " advertises S256 and a client registration method", "A6,O6"})
	}
	return fs
}

// wellKnown builds the RFC 8414 metadata URL: the well-known segment goes
// between the host and any issuer path.
func wellKnown(issuer, name string) string {
	issuer = strings.TrimSuffix(issuer, "/")
	if i := strings.Index(issuer, "://"); i >= 0 {
		if j := strings.Index(issuer[i+3:], "/"); j >= 0 {
			host, path := issuer[:i+3+j], issuer[i+3+j:]
			return host + "/.well-known/" + name + path
		}
	}
	return issuer + "/.well-known/" + name
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// ---- wire ----

// rpc posts one JSON-RPC message and returns the HTTP status, headers and the
// result member (nil for notifications and non-2xx responses).
func (s *session) rpc(ctx context.Context, method string, params any) (int, http.Header, json.RawMessage, error) {
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	notification := strings.HasPrefix(method, "notifications/")
	if !notification {
		s.next++
		msg["id"] = s.next
	}
	if params != nil {
		msg["params"] = params
	}
	body, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.opts.URL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-06-18")
	if s.sid != "" {
		req.Header.Set("Mcp-Session-Id", s.sid)
	}
	resp, err := s.opts.Client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	if id := resp.Header.Get("Mcp-Session-Id"); id != "" {
		s.sid = id
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if notification || resp.StatusCode != http.StatusOK {
		return resp.StatusCode, resp.Header, nil, nil
	}
	payload := raw
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
		payload = nil
		for _, line := range strings.Split(string(raw), "\n") {
			if d, ok := strings.CutPrefix(line, "data: "); ok {
				payload = []byte(d)
				break
			}
		}
	}
	var env struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(payload, &env); err != nil {
		return resp.StatusCode, resp.Header, nil, fmt.Errorf("%s: not a JSON-RPC response: %v", method, err)
	}
	if env.Error != nil {
		return resp.StatusCode, resp.Header, nil, fmt.Errorf("%s: %s", method, env.Error.Message)
	}
	return resp.StatusCode, resp.Header, env.Result, nil
}

func (s *session) getJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := s.opts.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
