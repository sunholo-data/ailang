package browserbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
)

func TestProviderLifecycleAgainstStub(t *testing.T) {
	var mu sync.Mutex
	requests := make([]string, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.Header.Get("X-BB-API-Key") != "test-key" {
			t.Errorf("missing API key header")
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"bb-1","projectId":"project-1","status":"RUNNING","region":"eu-central-1","connectUrl":"wss://connect.example?token=secret","createdAt":"2026-08-23T20:00:00Z","startedAt":"2026-08-23T20:00:01Z","proxyBytes":12}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/bb-1/debug":
			_, _ = w.Write([]byte(`{"debuggerUrl":"https://debug.example/private","wsUrl":"wss://debug.example/private","pages":[]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/bb-1/logs":
			_, _ = w.Write([]byte(`[{"method":"Page.navigate","timestamp":1}]`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/sessions/bb-1/recording":
			_, _ = w.Write([]byte(`[{"type":1,"timestamp":2,"data":{}}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions/bb-1":
			_, _ = w.Write([]byte(`{"id":"bb-1","projectId":"project-1","status":"COMPLETED","region":"eu-central-1","proxyBytes":42,"startedAt":"2026-08-23T20:00:01Z","endedAt":"2026-08-23T20:00:04Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider, err := New(Config{APIKey: "test-key", ProjectID: "project-1", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	artifacts := t.TempDir()
	session, err := provider.Create(context.Background(), browser.SessionSpec{RunID: "run-1", Region: "eu-central-1", MaximumDuration: 2 * time.Minute, ArtifactDir: artifacts})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := provider.Connection(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	mcp, env := conn.Materialize()
	if got := env["PLAYWRIGHT_MCP_CDP_ENDPOINT"]; !strings.Contains(got, "token=secret") {
		t.Fatalf("endpoint was not materialized into env: %q", got)
	}
	if strings.Contains(strings.Join(mcp.Args, " "), "secret") || len(mcp.EnvVars) != 1 {
		t.Fatalf("secret leaked into MCP argv: %#v", mcp)
	}
	inspection, err := provider.Inspect(context.Background(), session)
	if err != nil || inspection.Ref != "browserbase-session:bb-1" {
		t.Fatalf("inspection = %#v, err=%v", inspection, err)
	}
	manifest, err := provider.Export(context.Background(), session, artifacts)
	if err != nil || !manifest.Complete || len(manifest.Refs) != 2 {
		t.Fatalf("manifest = %#v, err=%v", manifest, err)
	}
	for _, name := range []string{"browserbase-logs.json", "browserbase-recording.json"} {
		if _, err := os.Stat(filepath.Join(artifacts, name)); err != nil {
			t.Fatalf("missing artifact %s: %v", name, err)
		}
	}
	usage, err := provider.Stop(context.Background(), session)
	if err != nil || usage.ProxyBytes != 42 || usage.DurationMS != 3000 {
		t.Fatalf("usage = %#v, err=%v", usage, err)
	}
	if _, err := provider.Stop(context.Background(), session); err != nil {
		t.Fatalf("idempotent stop failed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(requests, ","); strings.Count(got, "POST /v1/sessions/bb-1") != 1 {
		t.Fatalf("stop called more than once: %s", got)
	}
}

func TestProviderMapsHTTPFailuresWithoutLeakingBody(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   browser.FailureCategory
	}{
		{http.StatusUnauthorized, browser.FailureProvision},
		{http.StatusTooManyRequests, browser.FailureCapacityExhausted},
		{http.StatusGatewayTimeout, browser.FailureSessionTimeout},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"message":"api-key=must-not-leak"}`))
			}))
			defer server.Close()
			provider, err := New(Config{APIKey: "must-not-leak", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client()})
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.Create(context.Background(), browser.SessionSpec{RunID: "run"})
			if !browser.IsFailure(err, tc.want) || strings.Contains(err.Error(), "must-not-leak") {
				t.Fatalf("error = %v, want %s without secret", err, tc.want)
			}
		})
	}
}

func TestProviderConfigAndConnectionJSONNeverContainCredentials(t *testing.T) {
	provider, err := New(Config{APIKey: "secret-api-key", ProjectID: "secret-project", BaseURL: "https://api.example", NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(provider)
	if err != nil {
		t.Fatal(err)
	}
	printed := fmt.Sprint(provider)
	if strings.Contains(string(b), "secret-api-key") || strings.Contains(string(b), "secret-project") ||
		strings.Contains(printed, "secret-api-key") || strings.Contains(printed, "secret-project") {
		t.Fatalf("provider presentation leaked credentials: json=%s printed=%s", b, printed)
	}
}

func TestProviderRejectsCredentialBearingBaseURL(t *testing.T) {
	_, err := New(Config{APIKey: "key", BaseURL: "https://user:password@api.example?token=secret", NpxPath: "/test/npx"})
	if !browser.IsFailure(err, browser.FailureProvision) || strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe base URL error = %v", err)
	}
}

func TestProviderMalformedExportCleanupAndAuditFailures(t *testing.T) {
	t.Run("malformed create response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":`))
		}))
		defer server.Close()
		provider, err := New(Config{APIKey: "key", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		_, err = provider.Create(context.Background(), browser.SessionSpec{RunID: "run"})
		if !browser.IsFailure(err, browser.FailureProvision) {
			t.Fatalf("error = %v, want provision failure", err)
		}
	})

	t.Run("export and cleanup status failures", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/active") {
				_, _ = w.Write([]byte(`{"id":"active","status":"RUNNING"}`))
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		provider, err := New(Config{APIKey: "key", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client()})
		if err != nil {
			t.Fatal(err)
		}
		session := browser.Session{ID: "broken", Provider: ProviderName, ArtifactDir: t.TempDir()}
		provider.sessions[session.ID] = sessionResponse{ID: session.ID, ConnectURL: "wss://safe.invalid"}
		if _, err := provider.Export(context.Background(), session, session.ArtifactDir); !browser.IsFailure(err, browser.FailureArtifactExport) {
			t.Fatalf("export error = %v", err)
		}
		if _, err := provider.Stop(context.Background(), session); !browser.IsFailure(err, browser.FailureCleanup) {
			t.Fatalf("cleanup error = %v", err)
		}
		leaked, err := provider.Audit(context.Background(), []browser.Session{{ID: "active"}})
		if err != nil || len(leaked) != 1 || leaked[0] != "active" {
			t.Fatalf("audit leaked=%v err=%v", leaked, err)
		}
	})
}
