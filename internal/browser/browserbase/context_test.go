package browserbase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
	"github.com/sunholo-data/ailang/internal/browser/auth"
)

// testContextID is the value that must never appear in any presentation. It is
// deliberately distinctive so a substring search cannot pass by accident.
const testContextID = "ctx-canary-do-not-leak"

// recordedRequest captures what the stub actually received. Tests assert on the
// decoded body rather than on raw JSON so field ordering cannot make them flaky.
type recordedRequest struct {
	Method string
	Path   string
	Body   map[string]any
}

type requestLog struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (l *requestLog) record(r *http.Request) {
	entry := recordedRequest{Method: r.Method, Path: r.URL.Path}
	if r.Body != nil {
		var decoded map[string]any
		if err := json.NewDecoder(r.Body).Decode(&decoded); err == nil {
			entry.Body = decoded
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.requests = append(l.requests, entry)
}

func (l *requestLog) all() []recordedRequest {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]recordedRequest(nil), l.requests...)
}

func (l *requestLog) count() int { return len(l.all()) }

// contextStub serves the Browserbase Contexts endpoints plus session creation.
func contextStub(t *testing.T, log *requestLog) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.record(r)
		if r.Header.Get("X-BB-API-Key") != "test-key" {
			t.Errorf("context request missing API key header: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/contexts":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"id":%q,"projectId":"project-1","createdAt":"2026-08-25T10:00:00Z"}`, testContextID)
		case r.Method == http.MethodGet && r.URL.Path == "/v1/contexts/"+testContextID:
			_, _ = fmt.Fprintf(w, `{"id":%q,"projectId":"project-1","createdAt":"2026-08-25T10:00:00Z"}`, testContextID)
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/contexts/"+testContextID:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/sessions":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"bb-ctx-1","projectId":"project-1","status":"RUNNING","connectUrl":"wss://connect.example?token=secret","createdAt":"2026-08-25T10:00:01Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newContextProvider(t *testing.T, server *httptest.Server) *Provider {
	t.Helper()
	provider, err := New(Config{
		APIKey: "test-key", ProjectID: "project-1", BaseURL: server.URL,
		NpxPath: "/test/npx", HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return provider
}

func providerContextMaterial(t *testing.T) auth.SensitiveProfileMaterial {
	t.Helper()
	return auth.NewProviderContextMaterial(testContextID)
}

// TestContextLifecycleAgainstStub covers create/get/delete over the injectable
// bounded HTTP client (acceptance criterion 1).
func TestContextLifecycleAgainstStub(t *testing.T) {
	log := &requestLog{}
	server := contextStub(t, log)
	provider := newContextProvider(t, server)

	material, err := provider.CreateContext(context.Background())
	if err != nil {
		t.Fatalf("CreateContext: %v", err)
	}
	if material.Kind() != auth.MaterialProviderContext {
		t.Fatalf("material kind = %q, want %q", material.Kind(), auth.MaterialProviderContext)
	}
	if _, _, id := material.Materialize(); id != testContextID {
		t.Fatalf("materialized context id = %q, want %q", id, testContextID)
	}

	status, err := provider.GetContext(context.Background(), material)
	if err != nil {
		t.Fatalf("GetContext: %v", err)
	}
	if !status.Present {
		t.Fatalf("status = %#v, want Present", status)
	}

	if err := provider.DeleteContext(context.Background(), material); err != nil {
		t.Fatalf("DeleteContext: %v", err)
	}

	var got []string
	for _, request := range log.all() {
		got = append(got, request.Method+" "+request.Path)
	}
	want := []string{
		"POST /v1/contexts",
		"GET /v1/contexts/" + testContextID,
		"DELETE /v1/contexts/" + testContextID,
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("requests = %v, want %v", got, want)
	}

	// The bounded, injectable client must be the one that carried the traffic:
	// no context operation may open its own transport.
	if provider.client != server.Client() {
		t.Fatalf("context operations did not use the injected HTTP client")
	}
}

// TestDefaultHTTPClientIsBounded guards criterion 1 from the other direction: a
// provider built without an injected client still gets a deadline.
func TestDefaultHTTPClientIsBounded(t *testing.T) {
	provider, err := New(Config{APIKey: "key", BaseURL: "https://api.example", NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	if provider.client.Timeout <= 0 {
		t.Fatalf("default HTTP client timeout = %v, want a bounded deadline", provider.client.Timeout)
	}
}

// TestReadLeasePersistRequestRefusedBeforeAnyHTTPRequest is acceptance criterion
// 2's fail-closed half: the refusal must happen before the wire is touched, so
// the assertion is on the stub's request count, not only on the error.
func TestReadLeasePersistRequestRefusedBeforeAnyHTTPRequest(t *testing.T) {
	log := &requestLog{}
	server := contextStub(t, log)
	provider := newContextProvider(t, server)

	_, err := provider.CreateWithContext(context.Background(),
		browser.SessionSpec{RunID: "run-1", AuthMode: string(auth.LeaseRead)},
		ContextAttachment{
			Material: providerContextMaterial(t),
			Mode:     auth.LeaseRead,
			Persist:  true,
		})
	if !auth.IsFailure(err, auth.FailureWritebackDenied) {
		t.Fatalf("error = %v, want %s", err, auth.FailureWritebackDenied)
	}
	if count := log.count(); count != 0 {
		t.Fatalf("stub received %d request(s); write-back denial must precede all HTTP: %#v", count, log.all())
	}
	if strings.Contains(err.Error(), testContextID) {
		t.Fatalf("write-back denial leaked the context id: %v", err)
	}
}

// TestPersistModeMatchesLeaseMode is acceptance criterion 2's positive half.
func TestPersistModeMatchesLeaseMode(t *testing.T) {
	for _, tc := range []struct {
		name        string
		mode        auth.LeaseMode
		persist     bool
		wantPersist bool
	}{
		{"ordinary read session discards state", auth.LeaseRead, false, false},
		{"refresh session may write back", auth.LeaseRefresh, true, true},
		{"refresh session that does not ask still discards", auth.LeaseRefresh, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := &requestLog{}
			server := contextStub(t, log)
			provider := newContextProvider(t, server)

			if _, err := provider.CreateWithContext(context.Background(),
				browser.SessionSpec{RunID: "run-1", MaximumDuration: time.Minute},
				ContextAttachment{Material: providerContextMaterial(t), Mode: tc.mode, Persist: tc.persist},
			); err != nil {
				t.Fatalf("CreateWithContext: %v", err)
			}

			requests := log.all()
			if len(requests) != 1 {
				t.Fatalf("requests = %#v, want exactly one session create", requests)
			}
			settings, ok := requests[0].Body["browserSettings"].(map[string]any)
			if !ok {
				t.Fatalf("session body has no browserSettings: %#v", requests[0].Body)
			}
			contextBlock, ok := settings["context"].(map[string]any)
			if !ok {
				t.Fatalf("browserSettings has no context block: %#v", settings)
			}
			if contextBlock["id"] != testContextID {
				t.Fatalf("context id = %v, want %q", contextBlock["id"], testContextID)
			}
			if contextBlock["persist"] != tc.wantPersist {
				t.Fatalf("persist = %v, want %v", contextBlock["persist"], tc.wantPersist)
			}
			// The pre-existing timeout setting must survive the context merge.
			if settings["timeout"] != float64(60) {
				t.Fatalf("browserSettings lost the session timeout: %#v", settings)
			}
		})
	}
}

// TestUnauthenticatedCreateSendsNoContextBlock proves the context wiring is
// additive: an ordinary non-authenticated session is unchanged.
func TestUnauthenticatedCreateSendsNoContextBlock(t *testing.T) {
	log := &requestLog{}
	server := contextStub(t, log)
	provider := newContextProvider(t, server)

	if _, err := provider.Create(context.Background(), browser.SessionSpec{RunID: "run-1"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	requests := log.all()
	if len(requests) != 1 {
		t.Fatalf("requests = %#v", requests)
	}
	if settings, ok := requests[0].Body["browserSettings"].(map[string]any); ok {
		if _, hasContext := settings["context"]; hasContext {
			t.Fatalf("unauthenticated session sent a context block: %#v", settings)
		}
	}
}

// TestContextIDAbsentFromContainingStructs is acceptance criterion 3. Every
// assertion is made on the struct that CONTAINS the id, because a leak happens
// through a containing value's formatting, not through the opaque type itself.
func TestContextIDAbsentFromContainingStructs(t *testing.T) {
	log := &requestLog{}
	server := contextStub(t, log)
	provider := newContextProvider(t, server)

	spec := browser.SessionSpec{
		RunID:       "run-1",
		ProfileRef:  "shop-admin@v3",
		ProfileHash: "sha256:abc",
		AuthLeaseID: "lease-1",
		AuthMode:    string(auth.LeaseRead),
	}
	session, err := provider.CreateWithContext(context.Background(), spec,
		ContextAttachment{Material: providerContextMaterial(t), Mode: auth.LeaseRead})
	if err != nil {
		t.Fatalf("CreateWithContext: %v", err)
	}

	manifest := browser.BrowserRunManifest{
		RunID:              spec.RunID,
		Provider:           provider.Name(),
		ProviderSessionID:  session.ID,
		ProfileHash:        spec.ProfileHash,
		AuthProfileAlias:   "shop-admin",
		AuthProfileVersion: "v3",
		AuthLeaseID:        spec.AuthLeaseID,
		AuthMode:           spec.AuthMode,
	}

	// The stored provider-private binding is the struct most likely to leak.
	provider.mu.Lock()
	binding, bound := provider.contexts[session.ID]
	provider.mu.Unlock()
	if !bound {
		t.Fatalf("provider did not retain a context binding for the session")
	}

	specJSON, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	providerJSON, err := json.Marshal(provider)
	if err != nil {
		t.Fatal(err)
	}
	bindingJSON, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}

	for _, presentation := range []struct{ name, value string }{
		{"SessionSpec JSON", string(specJSON)},
		{"SessionSpec %v", fmt.Sprintf("%v", spec)},
		{"SessionSpec %+v", fmt.Sprintf("%+v", spec)},
		{"SessionSpec %#v", fmt.Sprintf("%#v", spec)},
		{"BrowserRunManifest JSON", string(manifestJSON)},
		{"BrowserRunManifest %+v", fmt.Sprintf("%+v", manifest)},
		{"BrowserRunManifest %#v", fmt.Sprintf("%#v", manifest)},
		{"Session JSON", string(sessionJSON)},
		{"Session %+v", fmt.Sprintf("%+v", session)},
		{"Provider JSON", string(providerJSON)},
		{"Provider %v", fmt.Sprintf("%v", provider)},
		{"Provider %+v", fmt.Sprintf("%+v", provider)},
		{"Provider %#v", fmt.Sprintf("%#v", provider)},
		{"binding JSON", string(bindingJSON)},
		{"binding %v", fmt.Sprintf("%v", binding)},
		{"binding %+v", fmt.Sprintf("%+v", binding)},
		{"binding %#v", fmt.Sprintf("%#v", binding)},
		{"contexts map %+v", fmt.Sprintf("%+v", provider.contexts)},
		{"contexts map %#v", fmt.Sprintf("%#v", provider.contexts)},
	} {
		if strings.Contains(presentation.value, testContextID) {
			t.Errorf("%s leaked the context id: %s", presentation.name, presentation.value)
		}
	}

	// SanitizeDiagnostics must also redact a context id carried under a
	// conventional key inside a nested diagnostic map.
	sanitized := browser.SanitizeDiagnostics(map[string]any{
		"session": map[string]any{"context_id": testContextID, "run_id": "run-1"},
	})
	if strings.Contains(fmt.Sprintf("%v", sanitized), testContextID) {
		t.Errorf("SanitizeDiagnostics leaked the context id: %v", sanitized)
	}

	// The assertions above all run through String/GoString/MarshalJSON, so they
	// prove those methods redact but would NOT notice a raw exported field. The
	// two structural checks below close that gap: they are what actually stops a
	// context id from reaching a manifest through a plain string field.
	assertNoExportedFields(t, binding)
	assertNoExportedStringContains(t, "SessionSpec", reflect.ValueOf(spec), testContextID)
	assertNoExportedStringContains(t, "BrowserRunManifest", reflect.ValueOf(manifest), testContextID)
	assertNoExportedStringContains(t, "Session", reflect.ValueOf(session), testContextID)
}

// assertNoExportedFields enforces the SensitiveProfileMaterial discipline on a
// provider-private record. With no exported fields there is nothing for a
// generic marshaller, a structured logger, or a manifest builder to pick up —
// which holds even if the type's own redacting methods were later removed.
func assertNoExportedFields(t *testing.T, value any) {
	t.Helper()
	typ := reflect.TypeOf(value)
	if typ.Kind() != reflect.Struct {
		t.Fatalf("assertNoExportedFields wants a struct, got %s", typ.Kind())
	}
	for i := range typ.NumField() {
		if field := typ.Field(i); field.IsExported() {
			t.Errorf("%s.%s is exported; a credential-grade record must keep every field unexported",
				typ.Name(), field.Name)
		}
	}
}

// assertNoExportedStringContains walks the exported fields of a value and fails
// if any reachable string carries the needle. It bypasses custom formatting on
// purpose: this is the path a manifest writer or JSON encoder would take.
func assertNoExportedStringContains(t *testing.T, label string, value reflect.Value, needle string) {
	t.Helper()
	switch value.Kind() {
	case reflect.String:
		if strings.Contains(value.String(), needle) {
			t.Errorf("%s reached a credential-grade value through an exported field", label)
		}
	case reflect.Struct:
		typ := value.Type()
		for i := range value.NumField() {
			if !typ.Field(i).IsExported() {
				continue
			}
			assertNoExportedStringContains(t, label+"."+typ.Field(i).Name, value.Field(i), needle)
		}
	case reflect.Pointer, reflect.Interface:
		if !value.IsNil() {
			assertNoExportedStringContains(t, label, value.Elem(), needle)
		}
	case reflect.Slice, reflect.Array:
		for i := range value.Len() {
			assertNoExportedStringContains(t, fmt.Sprintf("%s[%d]", label, i), value.Index(i), needle)
		}
	case reflect.Map:
		for _, key := range value.MapKeys() {
			assertNoExportedStringContains(t, label+"[key]", key, needle)
			assertNoExportedStringContains(t, label+"[value]", value.MapIndex(key), needle)
		}
	default:
	}
}

// TestContextErrorsNeverCarryTheContextID checks the error surface named by
// acceptance criterion 3.
func TestContextErrorsNeverCarryTheContextID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, `{"message":"context %s failed"}`, testContextID)
	}))
	defer server.Close()
	provider := newContextProvider(t, server)
	material := providerContextMaterial(t)

	_, createErr := provider.CreateContext(context.Background())
	_, getErr := provider.GetContext(context.Background(), material)
	deleteErr := provider.DeleteContext(context.Background(), material)
	_, sessionErr := provider.CreateWithContext(context.Background(),
		browser.SessionSpec{RunID: "run-1"},
		ContextAttachment{Material: material, Mode: auth.LeaseRead})

	for name, err := range map[string]error{
		"CreateContext": createErr, "GetContext": getErr,
		"DeleteContext": deleteErr, "CreateWithContext": sessionErr,
	} {
		if err == nil {
			t.Fatalf("%s returned no error against a failing provider", name)
		}
		if strings.Contains(err.Error(), testContextID) {
			t.Errorf("%s error leaked the context id: %v", name, err)
		}
	}
}

// TestContextOperationFailureMapping is acceptance criterion 4's provider-error
// half: auth failure, quota exhaustion, malformed response, and timeout.
func TestContextOperationFailureMapping(t *testing.T) {
	t.Run("auth failure", func(t *testing.T) {
		provider := newContextProvider(t, statusStub(t, http.StatusUnauthorized))
		if _, err := provider.CreateContext(context.Background()); !browser.IsFailure(err, browser.FailureProvision) {
			t.Fatalf("error = %v, want %s", err, browser.FailureProvision)
		}
	})

	t.Run("quota exhaustion", func(t *testing.T) {
		provider := newContextProvider(t, statusStub(t, http.StatusTooManyRequests))
		if _, err := provider.CreateContext(context.Background()); !browser.IsFailure(err, browser.FailureCapacityExhausted) {
			t.Fatalf("error = %v, want %s", err, browser.FailureCapacityExhausted)
		}
	})

	t.Run("gateway timeout status", func(t *testing.T) {
		provider := newContextProvider(t, statusStub(t, http.StatusGatewayTimeout))
		if _, err := provider.CreateContext(context.Background()); !browser.IsFailure(err, browser.FailureSessionTimeout) {
			t.Fatalf("error = %v, want %s", err, browser.FailureSessionTimeout)
		}
	})

	t.Run("client timeout", func(t *testing.T) {
		// The handler blocks until the test releases it. A client-side
		// Client.Timeout does NOT cancel the server's request context, so
		// waiting on r.Context().Done() here would wedge Close() forever.
		// Deferred calls run last-in-first-out, so release happens first.
		release := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			<-release
		}))
		defer server.Close()
		defer close(release)
		client := server.Client()
		client.Timeout = 50 * time.Millisecond
		provider, err := New(Config{
			APIKey: "test-key", ProjectID: "project-1", BaseURL: server.URL,
			NpxPath: "/test/npx", HTTPClient: client,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.CreateContext(context.Background()); !browser.IsFailure(err, browser.FailureSessionTimeout) {
			t.Fatalf("error = %v, want %s", err, browser.FailureSessionTimeout)
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"id":`))
		}))
		defer server.Close()
		provider := newContextProvider(t, server)
		if _, err := provider.CreateContext(context.Background()); !auth.IsFailure(err, auth.FailureMaterializeFailed) {
			t.Fatalf("error = %v, want %s", err, auth.FailureMaterializeFailed)
		}
	})

	t.Run("well formed but empty response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"projectId":"project-1"}`))
		}))
		defer server.Close()
		provider := newContextProvider(t, server)
		if _, err := provider.CreateContext(context.Background()); !auth.IsFailure(err, auth.FailureMaterializeFailed) {
			t.Fatalf("error = %v, want %s", err, auth.FailureMaterializeFailed)
		}
	})
}

func statusStub(t *testing.T, status int) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"api-key=must-not-leak"}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestContextSynchronizationDelayIsInjectable covers the documented provider
// publish delay. The delay must be a parameter a test can drive deterministically,
// never a hard-coded sleep — so the test asserts on the waited duration without
// spending it.
func TestContextSynchronizationDelayIsInjectable(t *testing.T) {
	t.Run("configured delay is observed once", func(t *testing.T) {
		var mu sync.Mutex
		var waited []time.Duration
		server := contextStub(t, &requestLog{})
		provider, err := New(Config{
			APIKey: "test-key", ProjectID: "project-1", BaseURL: server.URL,
			NpxPath: "/test/npx", HTTPClient: server.Client(),
			ContextSyncDelay: 7 * time.Second,
			Sleep: func(_ context.Context, d time.Duration) error {
				mu.Lock()
				waited = append(waited, d)
				mu.Unlock()
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		if err := provider.AwaitContextSync(context.Background()); err != nil {
			t.Fatalf("AwaitContextSync: %v", err)
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("injected waiter still spent real time: %v", elapsed)
		}
		mu.Lock()
		defer mu.Unlock()
		if len(waited) != 1 || waited[0] != 7*time.Second {
			t.Fatalf("waited = %v, want exactly [7s]", waited)
		}
	})

	t.Run("default delay is non-zero", func(t *testing.T) {
		provider, err := New(Config{APIKey: "key", BaseURL: "https://api.example", NpxPath: "/test/npx"})
		if err != nil {
			t.Fatal(err)
		}
		if provider.contextSyncDelay != DefaultContextSyncDelay || DefaultContextSyncDelay <= 0 {
			t.Fatalf("default sync delay = %v, want %v > 0", provider.contextSyncDelay, DefaultContextSyncDelay)
		}
	})

	t.Run("zero delay skips waiting entirely", func(t *testing.T) {
		called := false
		provider, err := New(Config{
			APIKey: "key", BaseURL: "https://api.example", NpxPath: "/test/npx",
			ContextSyncDelay: -1,
			Sleep: func(_ context.Context, _ time.Duration) error {
				called = true
				return nil
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := provider.AwaitContextSync(context.Background()); err != nil {
			t.Fatalf("AwaitContextSync: %v", err)
		}
		if called {
			t.Fatalf("a non-positive delay must not wait at all")
		}
	})

	t.Run("cancellation during the delay is a timeout failure", func(t *testing.T) {
		provider, err := New(Config{
			APIKey: "key", BaseURL: "https://api.example", NpxPath: "/test/npx",
			ContextSyncDelay: time.Hour,
		})
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := provider.AwaitContextSync(ctx); !browser.IsFailure(err, browser.FailureSessionTimeout) {
			t.Fatalf("error = %v, want %s", err, browser.FailureSessionTimeout)
		}
	})
}

// TestContextExpiryAndAbsence covers acceptance criterion 4's expiry case.
func TestContextExpiryAndAbsence(t *testing.T) {
	t.Run("expired context is a terminal profile-expired failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, `{"id":%q,"expiresAt":"2026-08-01T00:00:00Z"}`, testContextID)
		}))
		defer server.Close()
		provider, err := New(Config{
			APIKey: "test-key", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client(),
			Now: func() time.Time { return time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC) },
		})
		if err != nil {
			t.Fatal(err)
		}
		_, err = provider.GetContext(context.Background(), providerContextMaterial(t))
		if !auth.IsFailure(err, auth.FailureProfileExpired) {
			t.Fatalf("error = %v, want %s", err, auth.FailureProfileExpired)
		}
	})

	t.Run("not yet expired context resolves", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprintf(w, `{"id":%q,"expiresAt":"2026-09-01T00:00:00Z"}`, testContextID)
		}))
		defer server.Close()
		provider, err := New(Config{
			APIKey: "test-key", BaseURL: server.URL, NpxPath: "/test/npx", HTTPClient: server.Client(),
			Now: func() time.Time { return time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC) },
		})
		if err != nil {
			t.Fatal(err)
		}
		status, err := provider.GetContext(context.Background(), providerContextMaterial(t))
		if err != nil || !status.Present {
			t.Fatalf("status = %#v, err = %v", status, err)
		}
	})

	t.Run("missing context is profile-not-found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"message":"no such context"}`, http.StatusNotFound)
		}))
		defer server.Close()
		provider := newContextProvider(t, server)
		if _, err := provider.GetContext(context.Background(), providerContextMaterial(t)); !auth.IsFailure(err, auth.FailureProfileNotFound) {
			t.Fatalf("error = %v, want %s", err, auth.FailureProfileNotFound)
		}
	})
}

// TestContextDeletion covers acceptance criterion 4's deletion case, including
// the idempotency the cleanup path depends on.
func TestContextDeletion(t *testing.T) {
	t.Run("deletion is idempotent when the context is already gone", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, `{"message":"already deleted"}`, http.StatusNotFound)
		}))
		defer server.Close()
		provider := newContextProvider(t, server)
		if err := provider.DeleteContext(context.Background(), providerContextMaterial(t)); err != nil {
			t.Fatalf("second delete must succeed, got %v", err)
		}
	})

	t.Run("deletion failure is a structured cleanup failure", func(t *testing.T) {
		provider := newContextProvider(t, statusStub(t, http.StatusInternalServerError))
		err := provider.DeleteContext(context.Background(), providerContextMaterial(t))
		if !auth.IsFailure(err, auth.FailureCleanupFailed) {
			t.Fatalf("error = %v, want %s", err, auth.FailureCleanupFailed)
		}
	})

	t.Run("stopping a session drops its context binding", func(t *testing.T) {
		log := &requestLog{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.record(r)
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPost && r.URL.Path == "/v1/sessions" {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"bb-ctx-1","status":"RUNNING","connectUrl":"wss://c.example","createdAt":"2026-08-25T10:00:01Z"}`))
				return
			}
			_, _ = w.Write([]byte(`{"id":"bb-ctx-1","status":"COMPLETED","proxyBytes":1}`))
		}))
		defer server.Close()
		provider := newContextProvider(t, server)
		session, err := provider.CreateWithContext(context.Background(),
			browser.SessionSpec{RunID: "run-1"},
			ContextAttachment{Material: providerContextMaterial(t), Mode: auth.LeaseRead})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.Stop(context.Background(), session); err != nil {
			t.Fatal(err)
		}
		provider.mu.Lock()
		_, retained := provider.contexts[session.ID]
		provider.mu.Unlock()
		if retained {
			t.Fatalf("stopped session retained its provider context binding")
		}
	})
}

// TestContextAttachmentValidation covers the pre-HTTP argument checks.
func TestContextAttachmentValidation(t *testing.T) {
	for _, tc := range []struct {
		name     string
		attach   ContextAttachment
		wantAuth auth.FailureCategory
	}{
		{
			name:     "unknown lease mode",
			attach:   ContextAttachment{Material: auth.NewProviderContextMaterial(testContextID), Mode: auth.LeaseMode("bogus")},
			wantAuth: auth.FailureScopeDenied,
		},
		{
			name:     "empty material",
			attach:   ContextAttachment{Material: auth.SensitiveProfileMaterial{}, Mode: auth.LeaseRead},
			wantAuth: auth.FailureMaterializeFailed,
		},
		{
			name:     "wrong material kind",
			attach:   ContextAttachment{Material: auth.NewStorageStateMaterial([]byte("cookies")), Mode: auth.LeaseRead},
			wantAuth: auth.FailureMaterializeFailed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := &requestLog{}
			server := contextStub(t, log)
			provider := newContextProvider(t, server)
			_, err := provider.CreateWithContext(context.Background(),
				browser.SessionSpec{RunID: "run-1"}, tc.attach)
			if !auth.IsFailure(err, tc.wantAuth) {
				t.Fatalf("error = %v, want %s", err, tc.wantAuth)
			}
			if count := log.count(); count != 0 {
				t.Fatalf("attachment validation must precede HTTP; stub got %d request(s)", count)
			}
		})
	}
}
