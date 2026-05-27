package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
