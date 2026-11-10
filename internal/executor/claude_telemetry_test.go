package executor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// claude_telemetry.sh (embedded, written beside the Claude settings) posts to
// three hub-class ingest routes. It must present AILANG_HUB_TOKEN on each when
// set and send no ingest header when unset. Run with /bin/bash where present:
// on macOS that is bash 3.2, whose `set -u` treats an empty array as unbound,
// so the unset case also guards the expansion idiom.
func TestClaudeTelemetryHook_HubToken(t *testing.T) {
	shell := "/bin/bash"
	if _, err := os.Stat(shell); err != nil {
		shell = "bash"
	}
	for _, bin := range []string{shell, "jq", "curl"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not available: %v", bin, err)
		}
	}
	script := filepath.Join(t.TempDir(), "claude_telemetry.sh")
	if err := os.WriteFile(script, embeddedHookScript, 0o755); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct{ name, token string }{
		{"set", "hub-test-token"},
		{"unset", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			seen := map[string][]string{} // path -> token header values (nil = absent)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				seen[r.URL.Path] = r.Header.Values(config.IngestTokenHeader)
				mu.Unlock()
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			env := []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=" + t.TempDir(),
				"AILANG_OBSERVATORY_URL=" + srv.URL,
			}
			if tc.token != "" {
				env = append(env, config.EnvHubToken+"="+tc.token)
			}
			for _, payload := range []string{
				`{"hook_event_name":"SessionStart","session_id":"s1","cwd":"/w"}`,
				`{"hook_event_name":"PreToolUse","session_id":"s1","tool_name":"Bash","tool_use_id":"c1","tool_input":{}}`,
			} {
				cmd := exec.Command(shell, script)
				cmd.Env = env
				cmd.Stdin = strings.NewReader(payload)
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("hook exited %v: %s", err, out)
				}
			}

			// The exec posts are backgrounded (&): wait for all three routes.
			want := []string{"/api/observatory/hooks", "/api/exec/sessions", "/api/exec/events"}
			deadline := time.Now().Add(10 * time.Second)
			for {
				mu.Lock()
				n := 0
				for _, p := range want {
					if _, ok := seen[p]; ok {
						n++
					}
				}
				mu.Unlock()
				if n == len(want) {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("not every hub route was posted; saw %d of %d", n, len(want))
				}
				time.Sleep(20 * time.Millisecond)
			}

			mu.Lock()
			defer mu.Unlock()
			for _, p := range want {
				got := seen[p]
				switch {
				case tc.token == "" && len(got) != 0:
					t.Errorf("%s: token unset but %s was sent", p, config.IngestTokenHeader)
				case tc.token != "" && (len(got) != 1 || got[0] != tc.token):
					t.Errorf("%s: %s did not carry the token (%d values)", p, config.IngestTokenHeader, len(got))
				}
			}
		})
	}
}
