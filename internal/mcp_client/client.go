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
//   - Stateless: opens + closes one MCP session per call. Caching is the
//     caller's responsibility (see internal/prompt for the on-disk cache).
//
// Usage:
//
//	c := mcp_client.New(mcp_client.Options{
//	    BaseURL:        os.Getenv("AILANG_MCP_URL"),  // empty -> default prod
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

// ProtocolVersion is the MCP wire version we negotiate during initialize.
const ProtocolVersion = "2024-11-05"

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

	sessionID, err := c.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize: %w", err)
	}

	if err := c.sendInitialized(ctx, sessionID); err != nil {
		return nil, fmt.Errorf("notifications/initialized: %w", err)
	}

	body, err := c.callTool(ctx, sessionID, toolName, args)
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
		if got, ok := body["served_for"].(string); ok && got != "" && got != want {
			return body, ErrVersionMismatch
		}
	}

	return body, nil
}

// ─── internals ──────────────────────────────────────────────────────────

func (c *Client) initialize(ctx context.Context) (string, error) {
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
	resp, err := c.do(ctx, "POST", c.baseURL+"/", payload, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	sessionID := resp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		return "", errors.New("server did not return Mcp-Session-Id")
	}
	// Drain the SSE response — we don't need the initialize body content,
	// just a successful 200 + session header.
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("initialize: HTTP %d", resp.StatusCode)
	}
	return sessionID, nil
}

func (c *Client) sendInitialized(ctx context.Context, sessionID string) error {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	resp, err := c.do(ctx, "POST", c.baseURL+"/", payload, sessionID)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	// Notifications get 202 Accepted (no response body); anything <300 is fine.
	if resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) callTool(ctx context.Context, sessionID, toolName string, args map[string]any) (map[string]any, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params":  map[string]any{"name": toolName, "arguments": args},
	}
	resp, err := c.do(ctx, "POST", c.baseURL+"/", payload, sessionID)
	if err != nil {
		return nil, fmt.Errorf("tools/call: %w", err)
	}
	defer resp.Body.Close()

	// Streamable HTTP responses for non-notification calls come back as SSE.
	// Each line of the form `data: {...}` is a JSON-RPC message.
	body, err := readSingleSSEFrame(resp.Body)
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

func (c *Client) do(ctx context.Context, method, url string, payload any, sessionID string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	return c.http.Do(req)
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
