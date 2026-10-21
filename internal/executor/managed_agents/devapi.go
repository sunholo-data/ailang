package managed_agents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/executor"
)

// gemini_agents — the same Managed Agents interactions contract served by the
// Gemini DEVELOPER API (generativelanguage.googleapis.com) instead of Vertex.
//
// Why a second executor rather than a flag on managed_agents: the two surfaces
// differ in exactly the way that matters to a fleet. Live-probed 2026-10-13:
//
//   - Vertex accepts only gcs/skill_registry sources, wildcard egress, the
//     05-2026 agent, and has no files API — so edits never leave the sandbox
//     (CapRemoteSandbox; evaluator-only).
//   - The Developer API accepts repository/inline sources, lists the 09-2026
//     agent, and serves GET environments/{id}/files/{path}?alt=media — so this
//     executor mounts the task workspace, lets the agent edit it, downloads a
//     patch and applies it to the LOCAL workspace. It is a real editing
//     executor: it does NOT advertise CapRemoteSandbox.
//
// Auth: the Developer API refuses service-account OAuth ("Access to Gemini API
// is restricted with service accounts. Use authorization keys instead"), so
// this executor needs an API key. The sanctioned one is SA-bound, restricted to
// generativelanguage, lives in the locked ailang-gemini-agents project behind an
// AI Studio spend cap (scripts/security/setup_gemini_agents_project.sh), and is
// read from DevAPIKeyEnv. The key value never enters a log or error string.

const (
	// DevAPIExecutorName is the agent_cli value that selects this executor.
	DevAPIExecutorName = "gemini_agents"

	// DevAPIKeyEnv names the env var holding the Developer API key.
	DevAPIKeyEnv = config.EnvGeminiAgentsAPIKey

	devAPIHost       = "https://generativelanguage.googleapis.com"
	devAPIVersion    = "v1beta"
	devAPIKeyHeader  = "x-goog-api-key"
	devAPIAgentModel = "antigravity" // agent_config.type for model pinning

	// devAPIDefaultAgent is the newest agent the Developer API lists
	// (2026-10-13); Vertex does not serve it.
	devAPIDefaultAgent = "antigravity-preview-09-2026"
)

// NewDevAPI constructs the gemini_agents executor.
func NewDevAPI(cfg *executor.Config) (*Executor, error) {
	e, err := New(cfg)
	if err != nil {
		return nil, err
	}
	e.name = DevAPIExecutorName
	e.devapi = true
	e.agent = devAPIDefaultAgent
	e.baseURL = devAPIHost
	e.apiKey = config.GeminiAgentsAPIKey
	return e, nil
}

// splitAgentModel splits a registry agent_model_name of the form
// "<agent>/<model>" into the agent name and an optional pinned model.
//
// The pin travels as agent_config {type: antigravity, model: <model>}. Since
// 2026-10-21 the Developer API rejects an unknown model (404; on 10-13 it was
// accepted), so an accepted pin names a real model — but only the billing SKU
// proves it served the run. ProviderData records it as a request. Vertex
// ignores the field entirely, so Execute refuses a pin there.
func splitAgentModel(s string) (agent, model string) {
	if i := strings.Index(s, "/"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// devAPIKey returns the API key or a named error. Never a silent empty key.
func (e *Executor) devAPIKey() (string, error) {
	if e.apiKey == nil {
		return "", fmt.Errorf("%s: API key source not configured", DevAPIExecutorName)
	}
	k := e.apiKey()
	if k == "" {
		return "", fmt.Errorf("%s: %s is not set (see scripts/security/setup_gemini_agents_project.sh)", DevAPIExecutorName, DevAPIKeyEnv)
	}
	return k, nil
}

// devAPIRequest issues one authenticated Developer API request.
func (e *Executor) devAPIRequest(ctx context.Context, method, path string, body []byte, accept string) (*http.Response, error) {
	key, err := e.devAPIKey()
	if err != nil {
		return nil, err
	}
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, e.baseURL+"/"+devAPIVersion+"/"+path, rd)
	if err != nil {
		return nil, fmt.Errorf("%s: build request: %w", DevAPIExecutorName, err)
	}
	req.Header.Set(devAPIKeyHeader, key)
	req.Header.Set(apiRevisionHeader, apiRevision)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: HTTP do: %w", DevAPIExecutorName, err)
	}
	return resp, nil
}

// sendDevAPIInteraction POSTs an interaction and returns the live SSE body.
func (e *Executor) sendDevAPIInteraction(ctx context.Context, body any) (io.ReadCloser, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", DevAPIExecutorName, err)
	}
	resp, err := e.devAPIRequest(ctx, http.MethodPost, "interactions", buf, "text/event-stream")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpError(DevAPIExecutorName, resp)
	}
	return resp.Body, nil
}

// errFileNotFound marks a 404 from the files API — distinct from transport
// failures, because "the agent never wrote the patch" has its own recovery.
var errFileNotFound = fmt.Errorf("file not found in environment")

// getEnvFile downloads one file from a sandbox environment.
func (e *Executor) getEnvFile(ctx context.Context, envID, path string) ([]byte, error) {
	p := "environments/" + url.PathEscape(envID) + "/files/" + strings.TrimPrefix(path, "/") + "?alt=media"
	resp, err := e.devAPIRequest(ctx, http.MethodGet, p, nil, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%s: %s: %w", DevAPIExecutorName, path, errFileNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, httpError(DevAPIExecutorName, resp)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxPatchBytes+1))
}

// deleteEnv removes a sandbox environment. Best effort: the caller records a
// failure in ProviderData rather than failing a task whose edits already landed.
func (e *Executor) deleteEnv(ctx context.Context, envID string) error {
	resp, err := e.devAPIRequest(ctx, http.MethodDelete, "environments/"+url.PathEscape(envID), nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return httpError(DevAPIExecutorName, resp)
	}
	return nil
}

// httpError drains a non-2xx response into an error carrying the API message.
func httpError(prefix string, resp *http.Response) error {
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var ep errorPayload
	if json.Unmarshal(b, &ep) == nil && ep.Error.Message != "" {
		return fmt.Errorf("%s: HTTP %d: %s (code=%s)", prefix, resp.StatusCode, ep.Error.Message, ep.Error.Code)
	}
	preview := string(b)
	if len(preview) > 500 {
		preview = preview[:500] + "..."
	}
	return fmt.Errorf("%s: HTTP %d: %s", prefix, resp.StatusCode, preview)
}
