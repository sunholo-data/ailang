// Package mcp_client is a minimal MCP Streamable HTTP client used by the
// ailang CLI to fetch fresh content (prompts, docs search, stdlib) from
// mcp.ailang.sunholo.com when --source=mcp or --source=auto is in effect.
//
// Goals:
//   - Tiny surface: just enough to call one tool per CLI command
//   - Version-locked: every call passes for_version=<CLI's compile-time version>
//     and the response's served_for is checked. Mismatch -> ErrVersionMismatch
//     so the caller falls back silently to embedded.
//   - Bounded latency: 1.5s default timeout, configurable via env
//   - Stateless: one initialize handshake per call. The server may or may not
//     issue an Mcp-Session-Id (prod runs the SDK with Stateless: true and does
//     not); the client echoes one only if it was issued, and sends the
//     negotiated MCP-Protocol-Version on every request after initialize.
//     Caching is the caller's responsibility (internal/prompt's on-disk cache).
//
// Usage:
//
//	c := mcp_client.New(mcp_client.Options{
//	    BaseURL:        config.MCPURL(),  // empty -> default prod
//	    AILangVersion:  version.Version,
//	    Timeout:        1500 * time.Millisecond,
//	})
//	out, err := c.CallTool(ctx, "prompt_get", map[string]any{
//	    "forVersion": c.AILangVersion,
//	    "kind":       "agent",
//	})
package mcp_client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultURL is the canonical MCP endpoint for ailang.sunholo.com.
const DefaultURL = "https://mcp.ailang.sunholo.com/mcp/"

// DefaultTimeout caps a single MCP round-trip. Network failures should not
// stall an interactive `ailang prompt` command; the embedded fallback wins
// after this elapses.
const DefaultTimeout = 1500 * time.Millisecond

// ProtocolVersion is the MCP wire version we ask for during initialize. It is
// the first version with the MCP-Protocol-Version header, which is what lets a
// stateless server (no Mcp-Session-Id) know the version of later requests.
const ProtocolVersion = "2025-06-18"

// WireVersion maps a CLI version onto the key the MCP snapshot is published
// under: release snapshots are keyed "0.47.1", while the CLI's compile-time
// version is "v0.47.1". Only the leading "v" is dropped; a git-describe dev
// build ("v0.47.1-21-gabc") stays distinct, so the server correctly answers
// unknown_version for content no release has shipped.
func WireVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// ErrVersionMismatch means the server returned content tagged for a different
// AILANG version (typically because the snapshot doesn't have content for the
// caller's version). Callers should silently fall back to embedded.
var ErrVersionMismatch = errors.New("mcp_client: server has no content for this AILANG version")

// ErrToolError means the tool call returned an MCP error envelope (isError=true)
// or a structured {error: ...} JSON body. The body is preserved in the error.
type ErrToolError struct {
	Code   string
	Detail string
	Body   string
}

func (e *ErrToolError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("mcp_client: tool error %s: %s", e.Code, e.Detail)
	}
	return fmt.Sprintf("mcp_client: tool error: %s", e.Body)
}

// Options bundles the CLI-side config surface.
type Options struct {
	// BaseURL is the MCP endpoint. Empty resolves to DefaultURL.
	BaseURL string
	// AILangVersion is the CLI's compile-time version, passed as for_version
	// in every tool call.
	AILangVersion string
	// Timeout is per-request. Zero resolves to DefaultTimeout.
	Timeout time.Duration
	// HTTPClient lets tests inject a fake transport.
	HTTPClient *http.Client
}

// Client is a minimal MCP Streamable HTTP client.
type Client struct {
	baseURL       string
	ailangVersion string
	timeout       time.Duration
	http          *http.Client
}

// New constructs a Client. It does NOT open a session — sessions are per-call.
func New(opts Options) *Client {
	c := &Client{
		baseURL:       strings.TrimRight(opts.BaseURL, "/"),
		ailangVersion: opts.AILangVersion,
		timeout:       opts.Timeout,
		http:          opts.HTTPClient,
	}
	if c.baseURL == "" {
		c.baseURL = strings.TrimRight(DefaultURL, "/")
	}
	if c.timeout == 0 {
		c.timeout = DefaultTimeout
	}
	if c.http == nil {
		c.http = &http.Client{Timeout: c.timeout}
	}
	return c
}

// AILangVersion returns the CLI version this client was constructed with.
func (c *Client) AILangVersion() string {
	return c.ailangVersion
}

// BaseURL returns the configured MCP endpoint (without trailing slash).
func (c *Client) BaseURL() string {
	return c.baseURL
}

// CallTool runs initialize -> notifications/initialized -> tools/call against
// the MCP endpoint and returns the parsed response payload.
//
// If the response is an envelope of the form {served_for, data, ...} (which
// every version-scoped MCP tool returns) AND served_for != AILangVersion AND
// AILangVersion was passed in args under "forVersion", the call returns
// ErrVersionMismatch so the caller can silently fall back to embedded.
//
// If the tool returned an error envelope ({error, detail}), the call returns
// *ErrToolError with the parsed code/detail.
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]any) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if v, ok := args["forVersion"].(string); ok {
		args["forVersion"] = WireVersion(v)
	}

	sess, err := c.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.sendInitialized(ctx, sess); err != nil {
		return nil, fmt.Errorf("notifications/initialized: %w", err)
	}

	body, err := c.callTool(ctx, sess, toolName, args)
	if err != nil {
		return nil, err
	}

	// Detect a structured tool-side error first (the AILANG-side errorJson
	// helper returns these). If "error" is the only top-level key, surface as
	// ErrToolError.
	if errCode, ok := body["error"].(string); ok {
		detail, _ := body["detail"].(string)
		return nil, &ErrToolError{Code: errCode, Detail: detail}
	}

	// Version-scoped tools return {served_for, data, ...}. Verify match if the
	// caller passed forVersion and the server told us what it served.
	if want, ok := args["forVersion"].(string); ok && want != "" {
		if got, ok := body["served_for"].(string); ok && got != "" && WireVersion(got) != WireVersion(want) {
			return body, ErrVersionMismatch
		}
	}

	return body, nil
}

// ─── internals ──────────────────────────────────────────────────────────

// session is what initialize learned: the negotiated protocol version, and a
// session id when (and only when) the server issued one.
type session struct {
	id      string
	version string
}

func (c *Client) initialize(ctx context.Context) (session, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": ProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "ailang-cli", "version": c.ailangVersion},
		},
	}
	resp, err := c.do(ctx, c.baseURL+"/", payload, session{})
	if err != nil {
		return session{}, err
	}
	defer resp.Body.Close()
	// Status first: a 5xx must be reported as a 5xx, not as a missing header.
	if err := checkStatus(resp); err != nil {
		return session{}, err
	}
	body, err := readRPC(resp)
	if err != nil {
		return session{}, err
	}
	var rpc struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpc); err != nil {
		return session{}, fmt.Errorf("parse initialize result: %w", err)
	}
	if rpc.Error != nil {
		return session{}, fmt.Errorf("server refused initialize: %d %s", rpc.Error.Code, rpc.Error.Message)
	}
	if rpc.Result.ProtocolVersion == "" {
		return session{}, errors.New("initialize result has no protocolVersion")
	}
	return session{id: resp.Header.Get("Mcp-Session-Id"), version: rpc.Result.ProtocolVersion}, nil
}

func (c *Client) sendInitialized(ctx context.Context, sess session) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	resp, err := c.do(ctx, c.baseURL+"/", payload, sess)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	// Notifications get 202 Accepted (no response body); anything <300 is fine.
	return checkStatus(resp)
}

func (c *Client) callTool(ctx context.Context, sess session, toolName string, args map[string]any) (map[string]any, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  map[string]any{"name": toolName, "arguments": args},
	}
	resp, err := c.do(ctx, c.baseURL+"/", payload, sess)
	if err != nil {
		return nil, fmt.Errorf("tools/call: %w", err)
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return nil, fmt.Errorf("tools/call: %w", err)
	}

	body, err := readRPC(resp)
	if err != nil {
		return nil, fmt.Errorf("tools/call: %w", err)
	}

	var rpc struct {
		Result struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &rpc); err != nil {
		return nil, fmt.Errorf("tools/call: parse RPC envelope: %w (body: %s)", err, string(body))
	}
	if rpc.Error != nil {
		return nil, &ErrToolError{Code: fmt.Sprintf("rpc_%d", rpc.Error.Code), Detail: rpc.Error.Message, Body: string(body)}
	}
	if len(rpc.Result.Content) == 0 {
		return nil, errors.New("tools/call: empty result content")
	}
	if rpc.Result.IsError {
		return nil, &ErrToolError{Code: "tool_error", Detail: rpc.Result.Content[0].Text, Body: rpc.Result.Content[0].Text}
	}

	// The text content is a JSON-encoded payload (the tool's actual return
	// value, post the embed.ToGo Json unwrap).
	var out map[string]any
	if err := json.Unmarshal([]byte(rpc.Result.Content[0].Text), &out); err != nil {
		return nil, fmt.Errorf("tools/call: parse tool payload: %w (text: %s)", err, rpc.Result.Content[0].Text)
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, url string, payload any, sess session) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sess.version != "" {
		req.Header.Set("MCP-Protocol-Version", sess.version)
	}
	if sess.id != "" {
		req.Header.Set("Mcp-Session-Id", sess.id)
	}
	return c.http.Do(req)
}

// checkStatus turns a non-2xx response into an error carrying the status and
// the start of the body, so a server-side failure is diagnosable from the CLI.
func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 300))
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
}

// readRPC returns the JSON-RPC message in a response. The Streamable HTTP
// transport lets a server answer a request with either one JSON object or an
// SSE stream, and a client MUST accept both.
func readRPC(resp *http.Response) ([]byte, error) {
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
		if err != nil {
			return nil, fmt.Errorf("read JSON response: %w", err)
		}
		return body, nil
	}
	return readSingleSSEFrame(resp.Body)
}

// readSingleSSEFrame reads the first `data: {...}` line from an SSE stream
// and returns the JSON payload bytes. Sufficient for our request/response
// usage where we expect exactly one frame per call.
func readSingleSSEFrame(r io.Reader) ([]byte, error) {
	scanner := bufio.NewScanner(r)
	// Allow long single-line MCP payloads (prompts can be ~70KB).
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			return []byte(strings.TrimPrefix(line, "data: ")), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read SSE: %w", err)
	}
	return nil, errors.New("no SSE data frame in response")
}
