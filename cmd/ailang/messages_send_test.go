package main

import (
	"bytes"
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/messaging"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestSplitAndTrim covers the comma-separated --requires parsing.
//
// M-COORD-TAG-ROUTING-LASTMILE: the --requires flag accepts both
// "agent:motoko" (single tag) and "agent:motoko,ollama:gemma4-26b-ailang"
// (multi-tag). Whitespace around items is trimmed, empty items dropped —
// matches the worker_tags YAML parsing convention.
func TestSplitAndTrim(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{"single tag", "agent:motoko", []string{"agent:motoko"}},
		{"two tags", "agent:motoko,ollama:gemma4-26b-ailang", []string{"agent:motoko", "ollama:gemma4-26b-ailang"}},
		{"three tags with spaces", " agent:motoko , ollama:gemma4 , gpu:m4-max ", []string{"agent:motoko", "ollama:gemma4", "gpu:m4-max"}},
		{"trailing comma", "agent:motoko,", []string{"agent:motoko"}},
		{"empty between commas", "agent:motoko,,ollama:gemma4", []string{"agent:motoko", "ollama:gemma4"}},
		{"empty string", "", []string{}},
		{"only whitespace", "   ", []string{}},
		{"only commas", ",,,", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitAndTrim(tc.in, ",")
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("splitAndTrim(%q, \",\") = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

// TestSendViaHTTP_PostsCorrectShape verifies that sendViaHTTP posts the
// expected JSON body (with requires) to /api/messages and reports success
// for 2xx responses. M-COORD-TAG-ROUTING-LASTMILE acceptance criterion:
// "comma-separated form `--requires 'agent:motoko,ollama:gemma4-26b-ailang'`
// accepts and stores them as a 2-element slice".
func TestSendViaHTTP_PostsCorrectShape(t *testing.T) {
	var gotBody map[string]interface{}
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			// Mirror the real daemon's /health response so probeCoordinatorHTTP
			// returns true and sendViaHTTP proceeds with the POST.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/messages":
			if r.Method != http.MethodPost {
				http.Error(w, "wrong method", 405)
				return
			}
			gotAuth = r.Header.Get("Authorization")
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &gotBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"message_id":"test_msg_001","inbox":"eval-rig","status":"unread"}`))
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer srv.Close()

	// httptest provides the URL — extract the port and patch the
	// AILANG_COORD_HTTP_PORT env var so discoverCoordinatorHTTPPort picks
	// it up. Reset the env var on test exit.
	hostport := strings.TrimPrefix(srv.URL, "http://")
	port := hostport[strings.LastIndex(hostport, ":")+1:]
	t.Setenv("AILANG_COORD_HTTP_PORT", port)

	tags := []string{"agent:motoko", "ollama:gemma4-26b-ailang"}
	err := sendViaHTTP("eval-rig", "test title", "test content", "sprint-executor", "general", "", tags)
	if err != nil {
		t.Fatalf("sendViaHTTP returned error: %v", err)
	}

	// Validate request shape — these match the postMessageRequest fields
	// in internal/coordinator/daemon_http.go.
	want := map[string]interface{}{
		"inbox":    "eval-rig",
		"title":    "test title",
		"content":  "test content",
		"from":     "sprint-executor",
		"category": "general",
		"requires": []interface{}{"agent:motoko", "ollama:gemma4-26b-ailang"},
	}
	for k, v := range want {
		if !reflect.DeepEqual(gotBody[k], v) {
			t.Errorf("body[%q] = %v, want %v", k, gotBody[k], v)
		}
	}

	// No API key configured in test env, so Authorization should be empty
	// (the daemon's middleware accepts open requests when COORDINATOR_API_KEY
	// is unset, matching the local-mode default).
	if gotAuth != "" {
		t.Errorf("Authorization header = %q, want empty (no COORDINATOR_API_KEY in test env)", gotAuth)
	}
}

// TestSendViaHTTP_HonorsAPIKey verifies that when COORDINATOR_API_KEY is
// set in the environment, sendViaHTTP attaches a Bearer Authorization
// header. Important for production-mode where the cloud coordinator
// rejects unauthenticated requests.
func TestSendViaHTTP_HonorsAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case "/api/messages":
			gotAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"message_id":"test","inbox":"x","status":"unread"}`))
		}
	}))
	defer srv.Close()

	hostport := strings.TrimPrefix(srv.URL, "http://")
	port := hostport[strings.LastIndex(hostport, ":")+1:]
	t.Setenv("AILANG_COORD_HTTP_PORT", port)
	t.Setenv("COORDINATOR_API_KEY", "test-secret-key")

	err := sendViaHTTP("eval-rig", "t", "c", "f", "", "", []string{"agent:motoko"})
	if err != nil {
		t.Fatalf("sendViaHTTP returned error: %v", err)
	}
	wantAuth := "Bearer test-secret-key"
	if gotAuth != wantAuth {
		t.Errorf("Authorization header = %q, want %q", gotAuth, wantAuth)
	}
}

// TestSendViaHTTP_ErrorWhenUnreachable verifies the actionable error when
// the daemon HTTP listener isn't configured (no PORT in env or plist).
// M-COORD-TAG-ROUTING-LASTMILE: this is the canonical failure mode when
// --requires is used before `make coord-install` enables the listener.
func TestSendViaHTTP_ErrorWhenUnreachable(t *testing.T) {
	// Force discoverCoordinatorHTTPPort to find no port: clear env vars
	// and point HOME at a dir without a launchd plist.
	t.Setenv("AILANG_COORD_HTTP_PORT", "")
	t.Setenv("PORT", "")
	t.Setenv("HOME", t.TempDir())

	err := sendViaHTTP("eval-rig", "t", "c", "f", "", "", []string{"agent:motoko"})
	if err == nil {
		t.Fatal("expected error when no PORT configured, got nil")
	}
	if !strings.Contains(err.Error(), "no PORT") {
		t.Errorf("error message should mention 'no PORT': %v", err)
	}
	if !strings.Contains(err.Error(), "make coord-install") {
		t.Errorf("error message should suggest `make coord-install`: %v", err)
	}
}

// captureStderr runs fn with os.Stderr redirected and returns what it wrote.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stderr = orig
	return <-done
}

// TestWarnIfFiledButUndispatchable pins the guard on the seam that let three
// pkg:sunholo/ailang_parse reports sit unread with no task and no job: the write
// to Firestore succeeded, no Pub/Sub notification was published, and the CLI
// still printed a bare success line. The cloud coordinator's intake is Pub/Sub
// ONLY, so silence there means the work never starts.
func TestWarnIfFiledButUndispatchable(t *testing.T) {
	tests := []struct {
		name     string
		store    string
		project  string
		notified bool
		wantWarn bool
	}{
		{"gcp store, not notified -> WARN", "gcp", "ailang-multivac", false, true},
		{"gcp store, notified -> silent", "gcp", "ailang-multivac", true, false},
		{"local store, not notified -> silent (daemon polls the store)", "local", "", false, false},
		{"hybrid store keeps messaging in SQLite -> silent", "hybrid", "", false, false},
		{"unset store defaults local -> silent", "", "", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AILANG_MESSAGES_STORE", tc.store)
			t.Setenv("AILANG_MESSAGES_PROJECT", tc.project)
			t.Setenv("AILANG_STORAGE", "")
			t.Setenv("AILANG_CLOUD_PROJECT", "")

			out := captureStderr(t, func() {
				warnIfFiledButUndispatchable("pkg:sunholo/ailang_parse", tc.notified)
			})

			gotWarn := strings.Contains(out, "FILED, NOT DISPATCHED")
			if gotWarn != tc.wantWarn {
				t.Errorf("warn = %v, want %v\nstderr:\n%s", gotWarn, tc.wantWarn, out)
			}
			if tc.wantWarn {
				// The warning has to be ACTIONABLE, not just loud: it must name
				// the inbox the message is readable from and the config fix.
				for _, want := range []string{"pkg:sunholo/ailang_parse", "pubsub:", "enabled: true", "ailang-multivac"} {
					if !strings.Contains(out, want) {
						t.Errorf("warning missing %q\nstderr:\n%s", want, out)
					}
				}
			}
		})
	}
}

// TestTopicPrefixForProject pins the project→prefix pairing. The prefix is
// per-environment infrastructure (terraform sets AILANG_TOPIC_PREFIX = var.prefix),
// so carrying one environment's prefix into another publishes to a topic that
// does not exist there.
func TestTopicPrefixForProject(t *testing.T) {
	for _, tc := range []struct {
		project string
		want    string
		ok      bool
	}{
		{"ailang-multivac", "ailang", true},
		{"ailang-multivac-dev", "ailang-dev", true},
		{"ailang-multivac-test", "ailang-test", true},
		// An unknown project must NOT get a guessed prefix: a wrong topic is a
		// silently undelivered message, which is the whole failure class here.
		{"some-other-project", "", false},
		{"", "", false},
	} {
		got, ok := topicPrefixForProject(tc.project)
		if got != tc.want || ok != tc.ok {
			t.Errorf("topicPrefixForProject(%q) = (%q,%v), want (%q,%v)", tc.project, got, ok, tc.want, tc.ok)
		}
	}
}

// TestNotifyConfigForStoreFollowsStore pins the fix for the split-brain measured
// 2026-08-31: a probe written to ailang-multivac-dev published its notification
// to ailang-multivac because the config pinned project_id to prod. The dev
// coordinator was never told and the task never ran. Notifying a project you did
// not write to is never correct, so the store wins.
func TestNotifyConfigForStoreFollowsStore(t *testing.T) {
	t.Setenv("AILANG_MESSAGES_STORE", "gcp")
	t.Setenv("AILANG_MESSAGES_PROJECT", "ailang-multivac-dev")
	t.Setenv("AILANG_STORAGE", "")
	t.Setenv("AILANG_CLOUD_PROJECT", "")

	in := &messaging.PubSubConfig{Enabled: true, ProjectID: "ailang-multivac", TopicPrefix: "ailang"}
	out := notifyConfigForStore(in)

	if out.ProjectID != "ailang-multivac-dev" {
		t.Errorf("ProjectID = %q, want the STORE's project", out.ProjectID)
	}
	if out.TopicPrefix != "ailang-dev" {
		t.Errorf("TopicPrefix = %q, want ailang-dev — project and prefix must move together", out.TopicPrefix)
	}
	// The caller's config must not be mutated underneath them.
	if in.ProjectID != "ailang-multivac" || in.TopicPrefix != "ailang" {
		t.Errorf("input config was mutated: %+v", in)
	}
}

// TestNotifyConfigForStoreLeavesLocalAlone: a local store needs no notification
// and must not have its config rewritten.
func TestNotifyConfigForStoreLeavesLocalAlone(t *testing.T) {
	t.Setenv("AILANG_MESSAGES_STORE", "local")
	t.Setenv("AILANG_MESSAGES_PROJECT", "")
	t.Setenv("AILANG_STORAGE", "")
	t.Setenv("AILANG_CLOUD_PROJECT", "")

	in := &messaging.PubSubConfig{Enabled: true, ProjectID: "ailang-multivac", TopicPrefix: "ailang"}
	if out := notifyConfigForStore(in); out != in {
		t.Errorf("local store should pass the config through unchanged, got %+v", out)
	}
}
