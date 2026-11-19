package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The dashboard's route CONTRACT (design_docs/planned/m-dashboard-headless-api.md,
// "Stability contract"). One row per kept route: the method its client uses,
// what an anonymous caller gets, what a caller holding the dashboard credential
// gets, and — for the reads a CLI or script parses — the JSON keys that client
// reads. It runs the REAL handler chain (httpHandler: origin policy → read gate
// → mux → handler) against temp-dir stores, bound to a non-loopback address so
// the gate is enforced.
//
// read_auth_test.go proves every registered pattern is CLASSIFIED; this file
// proves each kept route still DOES its job for its caller. A route added to
// the table in the design doc gets a row here.

const approvalRealm = `Bearer realm="ailang-approvals"`

// contractServer is a non-loopback server with every route family, the
// dashboard credential, a SQLite coordinator store (exec sessions, task
// events, approvals) and a seeded task.
func contractServer(t *testing.T) http.Handler {
	t.Helper()
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	cs, err := coordinator.NewSQLiteStore(filepath.Join(t.TempDir(), "coordinator.db"))
	if err != nil {
		t.Fatalf("coordinator store: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	s.SetTaskEventStore(cs)
	s.SetApprovalStore(cs)
	s.SetCoordinatorStoreRaw(cs)

	ctx := context.Background()
	if err := cs.CreateTask(ctx, &coordinator.TaskRecord{
		ID: "task-contract", Title: "contract", Status: coordinator.TaskStatusCompleted, CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	if err := cs.StoreTaskEvent(ctx, &coordinator.TaskEventRecord{
		TaskID: "task-contract", StreamType: "text", Text: "hello", CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("seed event: %v", err)
	}

	// The secret-approval intake checks its own shared token.
	t.Setenv("AILANG_APPROVAL_INTAKE_AUTH", "")
	t.Setenv("AILANG_APPROVAL_TOKEN", testSharedApprovalToken)

	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatalf("httpHandler: %v", err)
	}
	return h
}

type routeContract struct {
	name           string
	method, target string
	body           string // JSON; sent with Content-Type: application/json
	anon           int    // status with no credential
	authed         int    // status with the dashboard credential
	// anonRealm is the WWW-Authenticate a 401 must carry: the dashboard gate's
	// realm for gated routes, the approvals realm for routes with own auth.
	anonRealm string
	keys      []string // JSON object keys the authed response must carry
	array     bool     // the authed response is a JSON array
}

var keptRouteContracts = []routeContract{
	// Public reads.
	{name: "health", method: "GET", target: "/health", anon: 200, authed: 200, keys: []string{"status", "version", "connections"}},
	{name: "version", method: "GET", target: "/api/version", anon: 200, authed: 200, keys: []string{"version"}},
	// /benchmarks/ reads GCS; the contract here is that the route is public and GET-only.
	{name: "benchmarks method", method: "POST", target: "/benchmarks/latest.json", anon: 405, authed: 405},
	{name: "benchmarks path guard", method: "GET", target: "/benchmarks/not-json.txt", anon: 404, authed: 404},

	// Public ingest.
	{name: "otlp traces", method: "POST", target: "/v1/traces", body: `{"resourceSpans":[]}`, anon: 200, authed: 200},
	{name: "otlp metrics", method: "POST", target: "/v1/metrics", body: `{"resourceMetrics":[]}`, anon: 200, authed: 200},
	{name: "otlp logs", method: "POST", target: "/v1/logs", body: `{"resourceLogs":[]}`, anon: 200, authed: 200},
	{name: "claude http hook", method: "POST", target: "/api/hooks/claude", body: `{"hook_event_name":"Notification","session_id":"contract-s"}`, anon: 200, authed: 200, keys: []string{"status"}},
	{name: "observatory hook", method: "POST", target: "/api/observatory/hooks", body: `{"event":"SessionStart","session_id":"contract-s","workspace":"/tmp/contract"}`, anon: 200, authed: 200, keys: []string{"status"}},
	{name: "exec session", method: "POST", target: "/api/exec/sessions", body: `{"session_id":"contract-exec","workspace":"/tmp/contract","provider":"claude"}`, anon: 200, authed: 200, keys: []string{"task_id", "session_id", "status"}},
	{name: "exec event", method: "POST", target: "/api/exec/events", body: `{"session_id":"contract-exec","stream_type":"text","text":"hi"}`, anon: 200, authed: 200},
	{name: "coordinator event", method: "POST", target: "/api/coordinator/events", body: `{"task_id":"task-contract","stream_type":"text","text":"x"}`, anon: 200, authed: 200, keys: []string{"success"}},
	{name: "eval telemetry", method: "POST", target: "/api/telemetry", body: `{"pid":4242,"status":"running"}`, anon: 200, authed: 200, keys: []string{"status"}},

	// Secret approvals: public at the gate, refused by their OWN check.
	{name: "approval intake", method: "POST", target: "/api/approvals", body: `{"ref":"op://x/y","agent":"contract"}`, anon: 401, authed: 401, anonRealm: approvalRealm},
	{name: "approval poll", method: "GET", target: "/api/approvals/secret-nope", anon: 401, authed: 401, anonRealm: approvalRealm},
	// A decision without the signed ntfy token needs the dashboard credential.
	{name: "approval decision", method: "POST", target: "/api/approvals/secret-nope/approve", anon: 401, authed: 404, anonRealm: dashboardAuthRealm},

	// Gated reads, with the keys their clients parse.
	{name: "inbox", method: "GET", target: "/api/inbox?limit=5", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"messages"}},
	{name: "exec hierarchy", method: "GET", target: "/api/controlplane/exec-hierarchy?limit=5", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"hierarchy"}},
	{name: "stats breakdown", method: "GET", target: "/api/controlplane/stats/breakdown", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"by_provider", "by_model", "by_source_type"}},
	{name: "task events json", method: "GET", target: "/api/coordinator/tasks/task-contract/events", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"task_id", "total_events", "events"}},
	{name: "task events text", method: "GET", target: "/api/coordinator/tasks/task-contract/events?format=text", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"content", "total_events", "total_turns"}},
	{name: "task events summary", method: "GET", target: "/api/coordinator/tasks/task-contract/events?format=summary", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"content"}},
	{name: "task events unknown", method: "GET", target: "/api/coordinator/tasks/no-such-task/events", anon: 401, authed: 404, anonRealm: dashboardAuthRealm},
	{name: "observatory spans", method: "GET", target: "/api/observatory/spans?limit=1", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, array: true},
	{name: "observatory traces", method: "GET", target: "/api/observatory/traces?limit=1", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, array: true},
	{name: "observatory sessions", method: "GET", target: "/api/observatory/sessions?limit=1", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, keys: []string{"sessions"}},
	{name: "chains", method: "GET", target: "/api/chains?limit=1", anon: 401, authed: 200, anonRealm: dashboardAuthRealm, array: true},
	{name: "chains stats", method: "GET", target: "/api/chains/stats", anon: 401, authed: 200, anonRealm: dashboardAuthRealm},
	// The WebSocket's external-client contract is ?token=<key> (the website
	// builder's proxy; browsers cannot set headers on a socket). Without it the
	// gate refuses an anonymous caller, and the hub itself refuses a
	// cross-origin client that only sent the header. With it the upgrade is
	// attempted — a recorder cannot be hijacked, so the hub answers 400: not
	// 401, not 404.
	{name: "websocket without ?token", method: "GET", target: "/ws", anon: 401, authed: 401, anonRealm: dashboardAuthRealm},
	{name: "websocket ?token", method: "GET", target: "/ws?token=" + testAPIKey, anon: 400, authed: 400},
	{name: "websocket bad ?token", method: "GET", target: "/ws?token=wrong", anon: 401, authed: 401, anonRealm: dashboardAuthRealm},
}

func doContract(h http.Handler, c routeContract, header http.Header) *httptest.ResponseRecorder {
	var r *http.Request
	if c.body != "" {
		r = httptest.NewRequest(c.method, c.target, strings.NewReader(c.body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(c.method, c.target, nil)
	}
	for k, v := range header {
		r.Header[k] = v
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestContract_KeptRoutes(t *testing.T) {
	h := contractServer(t)
	for _, c := range keptRouteContracts {
		t.Run(c.name, func(t *testing.T) {
			w := doContract(h, c, nil)
			if w.Code != c.anon {
				t.Fatalf("%s %s anonymous: got %d %q, want %d", c.method, c.target, w.Code, w.Body.String(), c.anon)
			}
			if c.anon == http.StatusUnauthorized {
				if got := w.Header().Get("WWW-Authenticate"); got != c.anonRealm {
					t.Errorf("%s %s anonymous 401 came from %q, want %q", c.method, c.target, got, c.anonRealm)
				}
			}

			w = doContract(h, c, bearer(testAPIKey))
			if w.Code != c.authed {
				t.Fatalf("%s %s with the dashboard credential: got %d %q, want %d", c.method, c.target, w.Code, w.Body.String(), c.authed)
			}
			if c.array {
				var arr []json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
					t.Errorf("%s %s: want a JSON array, got %q (%v)", c.method, c.target, w.Body.String(), err)
				}
			}
			if len(c.keys) > 0 {
				var obj map[string]json.RawMessage
				if err := json.Unmarshal(w.Body.Bytes(), &obj); err != nil {
					t.Fatalf("%s %s: want a JSON object, got %q (%v)", c.method, c.target, w.Body.String(), err)
				}
				for _, k := range c.keys {
					if _, ok := obj[k]; !ok {
						t.Errorf("%s %s: response lacks %q, which its client reads (got keys %v)", c.method, c.target, k, keysOf(obj))
					}
				}
			}
		})
	}
}

// The secret-approval round trip the cloud jobs make: intake with the shared
// token → poll by id. Both routes are public at the gate and authenticate the
// caller themselves.
func TestContract_SecretApprovalRoundTrip(t *testing.T) {
	h := contractServer(t)
	tok := http.Header{"Authorization": {"Bearer " + testSharedApprovalToken}}

	w := doContract(h, routeContract{method: "POST", target: "/api/approvals", body: `{"ref":"op://Prod/x/y","purpose":"contract","agent":"contract-agent"}`}, tok)
	if w.Code != http.StatusCreated {
		t.Fatalf("intake: got %d %q, want 201", w.Code, w.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.ID == "" {
		t.Fatalf("intake response lacks id: %q", w.Body.String())
	}

	w = doContract(h, routeContract{method: "GET", target: "/api/approvals/" + created.ID}, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("poll: got %d %q, want 200", w.Code, w.Body.String())
	}
	var status struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || status.ID != created.ID || status.Status != "pending" {
		t.Fatalf("poll: got %q, want id %s status pending", w.Body.String(), created.ID)
	}
}

// Every contract row names a route that is registered and classified, so the
// table cannot outlive the route it describes.
func TestContract_RowsAreRegisteredRoutes(t *testing.T) {
	s := fullRouteServer(t, "0.0.0.0:0", WithAPIKey(testAPIKey))
	_, patterns, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	for _, p := range patterns {
		mux.HandleFunc(p, func(http.ResponseWriter, *http.Request) {})
	}
	for _, c := range keptRouteContracts {
		r := httptest.NewRequest(c.method, c.target, nil)
		if _, pattern := mux.Handler(r); pattern == "" || pattern == "/" {
			t.Errorf("contract %q (%s %s) matches no kept route (pattern %q)", c.name, c.method, c.target, pattern)
		}
	}
}

// A loopback-bound server (the rig's `ailang serve`) answers the gated reads
// without a credential — `ailang chains` and the rig's scripts rely on it.
func TestContract_LoopbackReadsNeedNoCredential(t *testing.T) {
	s := fullRouteServer(t, "127.0.0.1:0")
	h, _, err := s.httpHandler()
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/api/observatory/spans?limit=1", "/api/chains?limit=1", "/api/inbox"} {
		if w := serve(h, http.MethodGet, target, nil); w.Code != http.StatusOK {
			t.Errorf("loopback GET %s: got %d, want 200", target, w.Code)
		}
	}
}

func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
