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
}

type Provider struct {
	apiKey     string
	projectID  string
	baseURL    string
	npxPath    string
	mcpVersion string
	client     *http.Client
	mu         sync.Mutex
	sessions   map[string]sessionResponse
	specs      map[string]browser.SessionSpec
	artifacts  map[string]string
	stopped    map[string]browser.Usage
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
	return &Provider{
		apiKey: config.APIKey, projectID: config.ProjectID,
		baseURL: strings.TrimRight(config.BaseURL, "/"), npxPath: config.NpxPath,
		mcpVersion: config.MCPVersion, client: config.HTTPClient,
		sessions: make(map[string]sessionResponse), specs: make(map[string]browser.SessionSpec), artifacts: make(map[string]string),
		stopped: make(map[string]browser.Usage),
	}, nil
}

func (p *Provider) Name() string { return ProviderName }

func (p *Provider) String() string {
	return fmt.Sprintf("%s provider (credentials=%s)", ProviderName, browser.Redacted)
}

func (p *Provider) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"provider": p.Name(), "base_url": p.baseURL,
		"project_configured": p.projectID != "", "credentials": browser.Redacted,
	})
}

func (p *Provider) Create(ctx context.Context, spec browser.SessionSpec) (browser.Session, error) {
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
	var response sessionResponse
	if err := p.doJSON(ctx, http.MethodPost, "/v1/sessions", body, &response, browser.FailureProvision); err != nil {
		return browser.Session{}, err
	}
	if response.ID == "" || response.ConnectURL == "" {
		return browser.Session{}, browser.NewFailure(browser.FailureProvision, "decode Browserbase session", fmt.Errorf("missing fields"))
	}
	createdAt := parseTime(response.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	p.mu.Lock()
	p.sessions[response.ID] = response
	p.specs[response.ID] = spec
	p.artifacts[response.ID] = spec.ArtifactDir
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
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return browser.NewFailure(fallback, "encode Browserbase request", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return browser.NewFailure(fallback, "create Browserbase request", err)
	}
	request.Header.Set("X-BB-API-Key", p.apiKey)
	request.Header.Set("Content-Type", "application/json")
	response, err := p.client.Do(request)
	if err != nil {
		return browser.NewFailure(httpFailure(err, fallback), "call Browserbase", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return browser.NewFailure(statusFailure(response.StatusCode, fallback), "call Browserbase", fmt.Errorf("HTTP %d", response.StatusCode))
	}
	if output == nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
		return nil
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, 1<<20))
	if err := decoder.Decode(output); err != nil {
		return browser.NewFailure(fallback, "decode Browserbase response", err)
	}
	return nil
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
