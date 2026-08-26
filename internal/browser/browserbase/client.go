// Package browserbase implements the Browserbase Sessions API as an AILANG
// browser.SessionProvider.
package browserbase

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
)

const (
	ProviderName      = "browserbase"
	DefaultBaseURL    = "https://api.browserbase.com"
	DefaultMCPVersion = "0.0.79"
)

type Config struct {
	APIKey     string
	ProjectID  string
	BaseURL    string
	NpxPath    string
	MCPVersion string
	HTTPClient *http.Client

	// ContextSyncDelay is how long the refresh workflow waits for Browserbase to
	// publish an updated Context before the new profile version is considered
	// readable. Zero selects DefaultContextSyncDelay; a negative value disables
	// waiting. It is a parameter rather than a literal sleep so tests can drive
	// it deterministically.
	ContextSyncDelay time.Duration
	// Sleep is the injectable waiter used for ContextSyncDelay. Tests supply a
	// recorder; production leaves it nil and gets a context-aware timer.
	Sleep func(ctx context.Context, d time.Duration) error
	// Now is the injectable clock used for Context expiry decisions.
	Now func() time.Time
}

type Provider struct {
	apiKey           string
	projectID        string
	baseURL          string
	npxPath          string
	mcpVersion       string
	client           *http.Client
	contextSyncDelay time.Duration
	sleep            func(ctx context.Context, d time.Duration) error
	now              func() time.Time
	mu               sync.Mutex
	sessions         map[string]sessionResponse
	specs            map[string]browser.SessionSpec
	artifacts        map[string]string
	stopped          map[string]browser.Usage
	// contexts holds the provider-private hosted-Context binding for each
	// authenticated session. The Context ID lives only inside the opaque
	// material; it is never mirrored into SessionSpec or any manifest.
	contexts map[string]contextBinding
}

type sessionResponse struct {
	ID         string `json:"id"`
	ProjectID  string `json:"projectId"`
	Status     string `json:"status"`
	Region     string `json:"region"`
	ConnectURL string `json:"connectUrl"`
	ProxyBytes int64  `json:"proxyBytes"`
	CreatedAt  string `json:"createdAt"`
	StartedAt  string `json:"startedAt"`
	EndedAt    string `json:"endedAt"`
}

func New(config Config) (*Provider, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, browser.NewFailure(browser.FailureProvision, "configure Browserbase API key", fmt.Errorf("missing key"))
	}
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	parsed, err := url.Parse(config.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, browser.NewFailure(browser.FailureProvision, "configure Browserbase URL", err)
	}
	if config.NpxPath == "" {
		path, pathErr := exec.LookPath("npx")
		if pathErr != nil {
			return nil, browser.NewFailure(browser.FailureProvision, "find npx", pathErr)
		}
		config.NpxPath = path
	}
	if config.MCPVersion == "" {
		config.MCPVersion = DefaultMCPVersion
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if config.ContextSyncDelay == 0 {
		config.ContextSyncDelay = DefaultContextSyncDelay
	}
	if config.Sleep == nil {
		config.Sleep = defaultSleep
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Provider{
		apiKey: config.APIKey, projectID: config.ProjectID,
		baseURL: strings.TrimRight(config.BaseURL, "/"), npxPath: config.NpxPath,
		mcpVersion: config.MCPVersion, client: config.HTTPClient,
		contextSyncDelay: config.ContextSyncDelay, sleep: config.Sleep, now: config.Now,
		sessions: make(map[string]sessionResponse), specs: make(map[string]browser.SessionSpec), artifacts: make(map[string]string),
		stopped:  make(map[string]browser.Usage),
		contexts: make(map[string]contextBinding),
	}, nil
}

// clock reads the injectable clock. A nil clock means a zero-value Provider was
// built by a test helper rather than by New; fall back rather than panic.
func (p *Provider) clock() time.Time {
	if p.now == nil {
		return time.Now()
	}
	return p.now()
}

func (p *Provider) Name() string { return ProviderName }

func (p *Provider) String() string {
	return fmt.Sprintf("%s provider (credentials=%s)", ProviderName, browser.Redacted)
}

// GoString stops %#v from falling back to Go-syntax printing of the unexported
// fields, which would otherwise render the API key and every retained hosted
// Context binding verbatim.
func (p *Provider) GoString() string { return p.String() }

func (p *Provider) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"provider": p.Name(), "base_url": p.baseURL,
		"project_configured": p.projectID != "", "credentials": browser.Redacted,
	})
}

// Create provisions an unauthenticated session. No hosted Context is attached,
// so nothing this session does can reach a stored profile.
func (p *Provider) Create(ctx context.Context, spec browser.SessionSpec) (browser.Session, error) {
	return p.createSession(ctx, spec, p.sessionBody(spec), nil)
}

// sessionBody builds the provider-neutral part of a session-create request. It
// is shared with the authenticated path so the two cannot drift.
func (p *Provider) sessionBody(spec browser.SessionSpec) map[string]any {
	body := map[string]any{"keepAlive": true}
	if p.projectID != "" {
		body["projectId"] = p.projectID
	}
	if spec.Region != "" {
		body["region"] = spec.Region
	}
	if spec.MaximumDuration > 0 {
		body["browserSettings"] = map[string]any{"timeout": int(spec.MaximumDuration.Seconds())}
	}
	if spec.RunID != "" {
		body["userMetadata"] = map[string]string{"ailangRunId": spec.RunID}
	}
	return body
}

// browserSettings returns the request's browserSettings map, creating it if the
// spec did not already need one. Merging rather than overwriting keeps the
// session timeout intact when a Context is attached.
func browserSettings(body map[string]any) map[string]any {
	if existing, ok := body["browserSettings"].(map[string]any); ok {
		return existing
	}
	settings := make(map[string]any, 1)
	body["browserSettings"] = settings
	return settings
}

// createSession issues the create call and records session state. binding is
// non-nil only for authenticated sessions.
func (p *Provider) createSession(ctx context.Context, spec browser.SessionSpec, body map[string]any, binding *contextBinding) (browser.Session, error) {
	var response sessionResponse
	if err := p.doJSON(ctx, http.MethodPost, "/v1/sessions", body, &response, browser.FailureProvision); err != nil {
		return browser.Session{}, err
	}
	if response.ID == "" || response.ConnectURL == "" {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "decode Browserbase session", fmt.Errorf("missing fields"))
	}
	createdAt := parseTime(response.CreatedAt)
	if createdAt.IsZero() {
		createdAt = p.clock()
	}
	p.mu.Lock()
	p.sessions[response.ID] = response
	p.specs[response.ID] = spec
	p.artifacts[response.ID] = spec.ArtifactDir
	if binding != nil {
		p.contexts[response.ID] = *binding
	}
	p.mu.Unlock()
	return browser.Session{ID: response.ID, Provider: p.Name(), CreatedAt: createdAt, ArtifactDir: spec.ArtifactDir}, nil
}

func (p *Provider) Connection(ctx context.Context, session browser.Session) (browser.SensitiveConnection, error) {
	if err := ctx.Err(); err != nil {
		return browser.SensitiveConnection{}, browser.NewFailure(browser.FailureConnect, "build Browserbase connection", err)
	}
	p.mu.Lock()
	record, ok := p.sessions[session.ID]
	spec := p.specs[session.ID]
	p.mu.Unlock()
	if !ok || record.ConnectURL == "" {
		return browser.SensitiveConnection{}, browser.NewFailure(browser.FailureConnect, "find Browserbase connection", fmt.Errorf("unknown session"))
	}
	args := []string{"-y", "@playwright/mcp@" + p.mcpVersion, "--headless"}
	if spec.ViewportWidth > 0 && spec.ViewportHeight > 0 {
		args = append(args, "--viewport-size", fmt.Sprintf("%dx%d", spec.ViewportWidth, spec.ViewportHeight))
	}
	if spec.ActionTimeout > 0 {
		args = append(args, "--timeout-action", strconv.FormatInt(spec.ActionTimeout.Milliseconds(), 10))
	}
	if session.ArtifactDir != "" {
		args = append(args, "--output-dir", session.ArtifactDir, "--save-session")
	}
	const endpointEnv = "PLAYWRIGHT_MCP_CDP_ENDPOINT"
	return browser.NewSensitiveConnection(browser.MCPServerSpec{
		Name: "playwright", Command: p.npxPath, Args: args,
		EnvVars: []string{endpointEnv}, Required: true,
	}, map[string]string{endpointEnv: record.ConnectURL}), nil
}

func (p *Provider) Inspect(ctx context.Context, session browser.Session) (browser.InspectionRef, error) {
	var response struct {
		DebuggerURL string `json:"debuggerUrl"`
	}
	path := "/v1/sessions/" + url.PathEscape(session.ID) + "/debug"
	if err := p.doJSON(ctx, http.MethodGet, path, nil, &response, browser.FailureConnect); err != nil {
		return browser.InspectionRef{}, err
	}
	return browser.InspectionRef{Available: response.DebuggerURL != "", Ref: "browserbase-session:" + session.ID}, nil
}

func (p *Provider) Export(ctx context.Context, session browser.Session, dst string) (browser.ArtifactManifest, error) {
	if dst == "" {
		p.mu.Lock()
		dst = p.artifacts[session.ID]
		p.mu.Unlock()
	}
	if dst == "" {
		return browser.ArtifactManifest{Complete: true}, nil
	}
	if err := os.MkdirAll(dst, 0700); err != nil {
		return browser.ArtifactManifest{}, browser.NewFailure(browser.FailureArtifactExport, "create Browserbase artifact directory", err)
	}
	refs := make([]browser.ArtifactRef, 0, 2)
	for _, artifact := range []struct{ endpoint, name, kind string }{
		{"logs", "browserbase-logs.json", "browserbase-logs"},
		{"recording", "browserbase-recording.json", "browserbase-recording"},
	} {
		path := "/v1/sessions/" + url.PathEscape(session.ID) + "/" + artifact.endpoint
		raw, err := p.getRawJSON(ctx, path)
		if err != nil {
			return browser.ArtifactManifest{Refs: refs}, err
		}
		outputPath := filepath.Join(dst, artifact.name)
		if err := os.WriteFile(outputPath, raw, 0600); err != nil {
			return browser.ArtifactManifest{Refs: refs}, browser.NewFailure(browser.FailureArtifactExport, "write Browserbase artifact", err)
		}
		digest := sha256.Sum256(raw)
		refs = append(refs, browser.ArtifactRef{Kind: artifact.kind, Path: artifact.name, SHA256: hex.EncodeToString(digest[:])})
	}
	return browser.ArtifactManifest{Complete: true, Refs: refs}, nil
}

func (p *Provider) Stop(ctx context.Context, session browser.Session) (browser.Usage, error) {
	p.mu.Lock()
	if usage, ok := p.stopped[session.ID]; ok {
		p.mu.Unlock()
		return usage, nil
	}
	p.mu.Unlock()
	body := map[string]any{"status": "REQUEST_RELEASE"}
	if p.projectID != "" {
		body["projectId"] = p.projectID
	}
	var response sessionResponse
	path := "/v1/sessions/" + url.PathEscape(session.ID)
	if err := p.doJSON(ctx, http.MethodPost, path, body, &response, browser.FailureCleanup); err != nil {
		return browser.Usage{}, err
	}
	usage := browser.Usage{ProxyBytes: response.ProxyBytes}
	startedAt, endedAt := parseTime(response.StartedAt), parseTime(response.EndedAt)
	if !startedAt.IsZero() && !endedAt.IsZero() {
		usage.DurationMS = endedAt.Sub(startedAt).Milliseconds()
	}
	p.mu.Lock()
	p.stopped[session.ID] = usage
	// Stop retaining the sensitive connection endpoint once the remote browser
	// has been released. The usage record is sufficient for idempotent Stop.
	delete(p.sessions, session.ID)
	delete(p.specs, session.ID)
	delete(p.artifacts, session.ID)
	// The hosted-Context binding is credential-grade; drop it with the rest of
	// the session's sensitive state rather than letting it outlive the browser.
	delete(p.contexts, session.ID)
	p.mu.Unlock()
	return usage, nil
}

// Audit returns known sessions that the provider still reports as active.
func (p *Provider) Audit(ctx context.Context, sessions []browser.Session) ([]string, error) {
	leaked := make([]string, 0)
	for _, session := range sessions {
		var response sessionResponse
		path := "/v1/sessions/" + url.PathEscape(session.ID)
		if err := p.doJSON(ctx, http.MethodGet, path, nil, &response, browser.FailureCleanup); err != nil {
			return leaked, err
		}
		if response.Status == "PENDING" || response.Status == "RUNNING" {
			leaked = append(leaked, session.ID)
		}
	}
	return leaked, nil
}

func (p *Provider) doJSON(ctx context.Context, method, path string, body any, output any, fallback browser.FailureCategory) error {
	_, err := p.doJSONWithStatus(ctx, method, path, body, output, fallback)
	return err
}

// doJSONWithStatus is doJSON plus the HTTP status code, which the Context
// operations need in order to tell provider-level errors apart from
// profile-level ones (a 404 means the Context is gone, not that the call
// failed). The returned status is 0 when no response was received, so callers
// can distinguish transport failures from a malformed 2xx body.
func (p *Provider) doJSONWithStatus(ctx context.Context, method, path string, body any, output any, fallback browser.FailureCategory) (int, error) {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return 0, browser.NewFailure(fallback, "encode Browserbase request", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return 0, browser.NewFailure(fallback, "create Browserbase request", err)
	}
	request.Header.Set("X-BB-API-Key", p.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return 0, browser.NewFailure(httpFailure(err, fallback), "call Browserbase", err)
	}
	defer response.Body.Close()
	status := response.StatusCode
	if status < 200 || status >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return status, browser.NewFailure(statusFailure(status, fallback), "call Browserbase", fmt.Errorf("HTTP %d", status))
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return status, nil
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(output); err != nil {
		return status, browser.NewFailure(fallback, "decode Browserbase response", err)
	}
	return status, nil
}

func (p *Provider) getRawJSON(ctx context.Context, path string) ([]byte, error) {
	var raw json.RawMessage
	if err := p.doJSON(ctx, http.MethodGet, path, nil, &raw, browser.FailureArtifactExport); err != nil {
		return nil, err
	}
	return []byte(raw), nil
}

func statusFailure(status int, fallback browser.FailureCategory) browser.FailureCategory {
	switch status {
	case http.StatusTooManyRequests:
		return browser.FailureCapacityExhausted
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return browser.FailureSessionTimeout
	case http.StatusUnauthorized, http.StatusForbidden:
		return browser.FailureProvision
	default:
		return fallback
	}
}

func httpFailure(err error, fallback browser.FailureCategory) browser.FailureCategory {
	if os.IsTimeout(err) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
		return browser.FailureSessionTimeout
	}
	return fallback
}

func parseTime(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}
