package apiserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// M-MCP-OAUTH-CLIENT-MODELS M2 (D5, D6): the verified identity reaches the
// tool, and only through the Bearer gate.
//
//   - @mcp_token_verifier may return Option[{subject, client_id, scopes}]
//     instead of bool. None, or Some with an empty subject, is a rejected
//     token (401).
//   - A declared _auth param of that record type makes the export
//     identity-required. _auth binds only from the gate's verified result,
//     so the export is served only as a gated tool on /mcp/connect/: it is
//     omitted from /mcp/, stdio, REST, OpenAPI, A2A and _meta. There is no
//     zero or empty identity on any path.
//   - The gate hands the identity to the tool handler on identityHeader,
//     which is deleted from every client request first and never shows in a
//     _headers binding (the publicBaseHeader precedent).
//   - --mcp-subject-rpm limits gated calls per verified subject: 429 +
//     Retry-After before the tool runs.

const (
	// authParam is the reserved parameter bound from the verified identity.
	authParam = "_auth"
	// identityHeader carries the verified identity from the gate to the tool
	// handler on one request. serve-api's own: stripped on entry.
	identityHeader = "X-Ailang-Mcp-Identity"
	// identityShape names the one accepted record, in error messages.
	identityShape = "{subject: string, client_id: string, scopes: [string]}"
)

// verifierSignature reports whether a @mcp_token_verifier's declared
// signature is accepted, and whether it is the identity form:
// (token: string) -> bool, or (token: string) -> Option[R] with R the identity
// record, inline or a same-module record type.
func verifierSignature(fn *ast.FuncDecl, aliases map[string]*ast.RecordType) (ok, identity bool) {
	if len(fn.Params) != 1 || paramTypeToString(fn.Params[0].Type) != "string" || fn.ReturnType == nil {
		return false, false
	}
	if paramTypeToString(fn.ReturnType) == "bool" {
		return true, false
	}
	app, isApp := fn.ReturnType.(*ast.TypeApp)
	if !isApp || app.Constructor != "Option" || len(app.Args) != 1 {
		return false, false
	}
	if identityRecordIssue(app.Args[0], aliases) != "" {
		return false, false
	}
	return true, true
}

// identityRecordIssue is why t is not the identity record ("" = it is). The
// record is closed and has exactly the three fields, so the value serve-api
// builds always type-checks against it.
func identityRecordIssue(t ast.Type, aliases map[string]*ast.RecordType) string {
	if lt, ok := t.(*ast.LabelledType); ok {
		t = lt.Base
	}
	rec, ok := t.(*ast.RecordType)
	if !ok {
		if st, isSimple := t.(*ast.SimpleType); isSimple {
			rec = aliases[st.Name]
		}
	}
	if rec == nil {
		return "the type is not a record (declare it inline or as a record type in this module)"
	}
	if rec.Row != nil {
		return "the record is open (| r); it must be closed"
	}
	want := map[string]bool{"subject": false, "client_id": false, "scopes": false}
	for _, f := range rec.Fields {
		seen, known := want[f.Name]
		if !known || seen {
			return fmt.Sprintf("unexpected field %s", f.Name)
		}
		want[f.Name] = true
		if f.Name == "scopes" {
			if !isStringList(f.Type) {
				return "field scopes must be [string]"
			}
		} else if paramTypeToString(f.Type) != "string" {
			return fmt.Sprintf("field %s must be string", f.Name)
		}
	}
	for _, name := range []string{"subject", "client_id", "scopes"} {
		if !want[name] {
			return "the record lacks " + name
		}
	}
	return ""
}

func isStringList(t ast.Type) bool {
	switch v := t.(type) {
	case *ast.ListType:
		return paramTypeToString(v.Element) == "string"
	case *ast.TypeApp:
		return v.Constructor == "list" && len(v.Args) == 1 && paramTypeToString(v.Args[0]) == "string"
	}
	return false
}

// extractAuthParams marks each export that declares _auth and refuses, at
// load, every use that could bind it without a verified identity. Rules that
// depend on server flags or on the verifier (another module may declare it)
// are checked by authParamIssue at registration and at Start.
func extractAuthParams(modInfo *ModuleInfo, file *ast.File) error {
	aliases := recordAliases(file)
	for _, fn := range file.Funcs {
		pi := -1
		for i, p := range fn.Params {
			if p.Name == authParam {
				pi = i
				break
			}
		}
		if pi < 0 {
			continue
		}
		idx := -1
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				idx = i
				break
			}
		}
		if idx < 0 {
			continue // not exported: never served, nothing to bind
		}
		e := &modInfo.Exports[idx]
		if err := authParamLoadIssue(fn, e, pi, aliases); err != nil {
			return fmt.Errorf("%s.%s: %w", modInfo.Path, fn.Name, err)
		}
		e.NeedsIdentity = true
		if pi < len(e.ParamTypes) {
			e.ParamTypes[pi] = "record" // a same-module alias binds as the record it is
		}
	}
	return nil
}

func authParamLoadIssue(fn *ast.FuncDecl, e *ExportInfo, pi int, aliases map[string]*ast.RecordType) error {
	if issue := identityRecordIssue(fn.Params[pi].Type, aliases); issue != "" {
		return fmt.Errorf("_auth: %s; declare it as %s", issue, identityShape)
	}
	served := "_auth is bound only from a verified identity, so the export is served only as a gated tool on /mcp/connect/"
	switch {
	case e.IsTokenVerifier:
		return errors.New("_auth: a @mcp_token_verifier cannot take _auth")
	case e.MCPAuth == mcpAuthNoAuth:
		return fmt.Errorf("_auth with @mcp_auth(%q): %s; use @mcp_auth(%q)", mcpAuthNoAuth, served, mcpAuthOAuth2)
	case e.RoutePath != "":
		return fmt.Errorf("_auth with @route: %s, never as an HTTP route", served)
	case e.IsNoMCP:
		return fmt.Errorf("_auth with @nomcp: %s; @nomcp would leave it served nowhere", served)
	case e.IsAgentOnly:
		return fmt.Errorf("_auth with @mcp_agent_only: %s, which @mcp_agent_only removes it from", served)
	case e.UIResource != "":
		return errors.New("_auth with @mcp_ui_resource: a widget resource read has no verified identity")
	}
	for _, n := range e.Optional {
		if n == authParam {
			return errors.New("@optional(\"_auth\"): _auth is bound from the verified identity, never absent")
		}
	}
	if isSecretParam(*e, authParam) {
		return errors.New("@mcp_secret(\"_auth\"): _auth is bound from the verified identity, not supplied by the client")
	}
	if isFileParam(*e, authParam) {
		return errors.New("@mcp_file(\"_auth\"): _auth is bound from the verified identity, not a file")
	}
	return nil
}

// authParamIssue is the flag- and verifier-dependent rule for an _auth
// export: it must be gated (@mcp_auth("oauth2"), or any tool under
// --mcp-connect-only), and the server's verifier must return an identity.
func authParamIssue(e ExportInfo, connectOnly bool, verifier *tokenVerifier, verifierErr error) error {
	if !e.NeedsIdentity {
		return nil
	}
	gated := e.MCPAuth == mcpAuthOAuth2 || (connectOnly && e.MCPAuth == "")
	switch {
	case !gated:
		return fmt.Errorf("_auth needs a gated tool: add @mcp_auth(%q) or serve with --mcp-connect-only", mcpAuthOAuth2)
	case verifierErr != nil:
		return fmt.Errorf("_auth: %v", verifierErr)
	case verifier == nil:
		return fmt.Errorf("_auth needs one @mcp_token_verifier returning Option[%s]", identityShape)
	case !verifier.identity:
		return fmt.Errorf("_auth needs an identity verifier: @mcp_token_verifier %s returns bool, which names no subject; return Option[%s]", verifier.name, identityShape)
	}
	return nil
}

// subjectLimit is --mcp-subject-rpm/--mcp-subject-burst.
type subjectLimit struct{ rpm, burst int }

// limiter is the per-subject token bucket (nil when off). An unset burst
// allows one minute's calls at once.
func (l subjectLimit) limiter() *IPRateLimiter {
	burst := l.burst
	if burst <= 0 {
		burst = l.rpm
	}
	return NewIPRateLimiter(l.rpm, burst)
}

// ValidateMCPIdentity refuses to start a server whose _auth exports or
// per-subject limit could not work as declared: an _auth tool that is not
// gated or has no identity verifier, or a subject limit with no subject to
// key on. Each would otherwise be a tool silently missing, or a limit
// silently off.
func (s *Server) ValidateMCPIdentity() error {
	modules := s.GetModules()
	verifier, verifierErr := resolveTokenVerifier(modules)
	var issues []string
	for _, info := range modules {
		for _, e := range info.Exports {
			if err := authParamIssue(e, s.mcpConnectOnly, verifier, verifierErr); err != nil {
				issues = append(issues, fmt.Sprintf("%s/%s: %v", info.Path, e.Name, err))
			}
		}
	}
	sort.Strings(issues)
	if len(issues) > 0 {
		return errors.New(strings.Join(issues, "; "))
	}
	l := s.subjectLimit
	switch {
	case l.rpm < 0 || l.burst < 0:
		return errors.New("--mcp-subject-rpm and --mcp-subject-burst must be >= 0")
	case l.burst > 0 && l.rpm == 0:
		return errors.New("--mcp-subject-burst needs --mcp-subject-rpm")
	case l.rpm == 0:
		return nil
	case !s.mcpEnabled:
		return errors.New("--mcp-subject-rpm needs --mcp-http: it limits gated /mcp/connect/ calls")
	case verifierErr != nil:
		return fmt.Errorf("--mcp-subject-rpm: %w", verifierErr)
	case verifier == nil || !verifier.identity:
		return fmt.Errorf("--mcp-subject-rpm needs an @mcp_token_verifier returning Option[%s]: a bool verifier names no subject to limit", identityShape)
	}
	return nil
}

// verifyIdentity calls an identity verifier. None is a rejected token (nil,
// nil); the gate also rejects an empty subject.
func (ms *MCPServer) verifyIdentity(ctx context.Context, token string) (*protocol.Identity, error) {
	v := ms.verifier
	if v == nil || !v.identity {
		return nil, errors.New("no identity @mcp_token_verifier")
	}
	res, err := ms.server.engine.CallPrepared(func(eff interface{}) {
		if ec, ok := eff.(*effects.EffContext); ok {
			ec.GoCtx = ctx
		}
	}, v.modPath, v.name, token)
	if err != nil {
		return nil, err
	}
	val, err := embed.ToGo(res)
	if err != nil {
		return nil, err
	}
	id, err := decodeIdentityOption(val)
	if err != nil {
		return nil, fmt.Errorf("@mcp_token_verifier %s: %w", v.name, err)
	}
	return id, nil
}

// decodeIdentityOption reads the engine's Option envelope ({__type, __tag,
// fields}, internal/embed/convert.go). Anything but None or Some(identity
// record) is an error, which the gate answers 503: never an admit.
func decodeIdentityOption(val any) (*protocol.Identity, error) {
	env, ok := val.(map[string]any)
	if !ok || env["__type"] != "Option" {
		return nil, fmt.Errorf("returned %T, want Option[%s]", val, identityShape)
	}
	switch env["__tag"] {
	case "None":
		return nil, nil
	case "Some":
	default:
		return nil, fmt.Errorf("returned Option tag %v", env["__tag"])
	}
	fields, _ := env["fields"].([]any)
	if len(fields) != 1 {
		return nil, errors.New("returned Some without a value")
	}
	rec, ok := fields[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("returned Some(%T), want %s", fields[0], identityShape)
	}
	id := &protocol.Identity{Scopes: []string{}}
	id.Subject, ok = rec["subject"].(string)
	if !ok {
		return nil, errors.New("subject is not a string")
	}
	if id.ClientID, ok = rec["client_id"].(string); !ok {
		return nil, errors.New("client_id is not a string")
	}
	scopes, ok := rec["scopes"].([]any)
	if !ok && rec["scopes"] != nil {
		return nil, errors.New("scopes is not a list")
	}
	for _, sc := range scopes {
		str, isStr := sc.(string)
		if !isStr {
			return nil, errors.New("scopes holds a non-string")
		}
		id.Scopes = append(id.Scopes, str)
	}
	return id, nil
}

// withoutIdentityHeader deletes any client-sent identityHeader before
// anything else sees the request: only the gate, after a verify, sets it.
func withoutIdentityHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(identityHeader)
		next.ServeHTTP(w, r)
	})
}

// admitGated runs the gate for one gated tools/call and, on an identity gate,
// stamps the identity and applies the per-subject limit. It reports whether
// the request may proceed; otherwise the refusal is written.
func admitGated(gate *protocol.BearerGate, identity bool, limit *IPRateLimiter, calls int, w http.ResponseWriter, r *http.Request, callID json.RawMessage) bool {
	if !identity {
		return gate.AdmitCall(w, r, callID)
	}
	id, ok := gate.AdmitCallIdentity(w, r, callID)
	if !ok {
		return false
	}
	// One token per gated call in the body, so a JSON-RPC batch cannot
	// multiply a subject's allowance.
	for i := 0; i < calls; i++ {
		if allowed, wait := limit.AllowAfter(id.Subject); !allowed {
			protocol.WriteRateLimited(w, callID, wait)
			return false
		}
	}
	b, _ := json.Marshal(id) // strings and a string slice: cannot fail
	r.Header.Set(identityHeader, string(b))
	return true
}

// identityArg is the _auth value for this call: the identity the gate
// stamped. A missing or malformed header fails the call; it never binds an
// empty identity.
func identityArg(req *mcp.CallToolRequest) (map[string]any, error) {
	var raw string
	if extra := req.GetExtra(); extra != nil && extra.Header != nil {
		raw = extra.Header.Get(identityHeader)
	}
	if raw == "" {
		return nil, errors.New("no verified identity for this call")
	}
	var id protocol.Identity
	if err := json.Unmarshal([]byte(raw), &id); err != nil || id.Subject == "" {
		return nil, errors.New("no verified identity for this call")
	}
	scopes := make([]any, len(id.Scopes))
	for i, sc := range id.Scopes {
		scopes[i] = sc
	}
	return map[string]any{"subject": id.Subject, "client_id": id.ClientID, "scopes": scopes}, nil
}

// bindAuthArg overwrites the _auth slot with the verified identity, whatever
// the client sent there, named or positional.
func bindAuthArg(export ExportInfo, args []any, req *mcp.CallToolRequest) ([]any, error) {
	if !export.NeedsIdentity {
		return args, nil
	}
	for i, name := range export.ParamNames {
		if name != authParam {
			continue
		}
		if i >= len(args) {
			return nil, errors.New("missing arguments before _auth")
		}
		id, err := identityArg(req)
		if err != nil {
			return nil, err
		}
		args[i] = id
	}
	return args, nil
}

// logIdentityOmitted is the startup line for an _auth export a surface does
// not serve, the way @mcp_agent_only omissions are visible in tools/list.
func logIdentityOmitted(modPath string, e ExportInfo) {
	log.Printf("  MCP: %s/%s takes _auth and is served only on the gated %s", modPath, e.Name, listedMCPPath)
}
