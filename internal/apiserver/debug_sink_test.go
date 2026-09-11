package apiserver

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

const debugFixture = "internal/embed/testdata/debug_structured"

// captureStderr swaps os.Stderr (the sink's raw writer) AND the standard
// logger's output (the sink's decorated writer) for the duration of fn.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	log.SetOutput(w)
	done := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	os.Stderr = oldStderr
	log.SetOutput(oldStderr)
	_ = w.Close()
	return <-done
}

func newDebugTestServer(t *testing.T, effCtx *effects.EffContext) *Server {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolving repo root: %v", err)
	}
	t.Setenv("AILANG_STDLIB_PATH", root)
	srv := New(root, Config{EffCtx: effCtx})
	t.Cleanup(func() { _ = srv.Close() })
	if err := srv.LoadModules([]string{filepath.Join(root, debugFixture+".ail")}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	return srv
}

// TestServeAPI_StructuredDebugLinesReachStderrVerbatim pins the production
// bug: a JSON Debug.log line and a failed Debug.check must each be ONE bare
// JSON line with severity ERROR — no "2026/09/09 13:32:14 [Debug] " prefix —
// while a plain line keeps the timestamped "[Debug] " decoration.
func TestServeAPI_StructuredDebugLinesReachStderrVerbatim(t *testing.T) {
	srv := newDebugTestServer(t, effects.NewEffContext(nil))

	out := captureStderr(t, func() {
		code, body := callRouteFixture(t, srv, debugFixture)
		if code != 200 || body["result"] != "ok" {
			t.Fatalf("route call failed: %d %v", code, body)
		}
	})

	var structured []map[string]any
	var plain []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "{") {
			var m map[string]any
			if err := json.Unmarshal([]byte(line), &m); err != nil {
				t.Errorf("line starts with { but is not valid JSON (decorated?): %q", line)
				continue
			}
			structured = append(structured, m)
		} else if strings.Contains(line, "[Debug]") {
			plain = append(plain, line)
		}
	}

	if len(structured) != 2 {
		t.Fatalf("want 2 bare JSON lines (log + failed check), got %d; stderr:\n%s", len(structured), out)
	}
	for _, m := range structured {
		if m["severity"] != "ERROR" {
			t.Errorf("structured line severity = %v, want ERROR: %v", m["severity"], m)
		}
	}
	if structured[0]["message"] != "structured-line" {
		t.Errorf("first structured line = %v", structured[0])
	}
	if structured[1]["source"] != "Debug.check" || structured[1]["message"] != "assertion failed: served-check" {
		t.Errorf("failed check was not emitted as structured ERROR: %v", structured[1])
	}
	// Call-site location, injected by the compiler: the fixture's Debug.check
	// is on line 13; the file is under the repo root (cwd of this test is the
	// package dir, so the path stays absolute), slash-normalised.
	if loc, _ := structured[1]["location"].(string); !strings.HasSuffix(loc, "/"+debugFixture+".ail:13") {
		t.Errorf("location = %q, want suffix %q", loc, "/"+debugFixture+".ail:13")
	}
	if len(plain) != 1 || !strings.HasSuffix(plain[0], "[Debug] plain-line") {
		t.Errorf("plain line should keep its [Debug] decoration exactly once; got %v", plain)
	}
	if strings.Contains(out, "[Debug] {") || strings.Contains(out, "ASSERT FAIL") {
		t.Errorf("old decorated forms still present:\n%s", out)
	}
}

// TestServeAPI_LogLevelFiltersStructuredLines pins that --log-level still
// applies through the shared sink: at "none" (4) the structured log line is
// suppressed but the failed check — always ERROR-structured — still surfaces.
func TestServeAPI_LogLevelFiltersStructuredLines(t *testing.T) {
	root, _ := filepath.Abs(filepath.Join("..", ".."))
	t.Setenv("AILANG_STDLIB_PATH", root)
	srv := New(root, Config{EffCtx: effects.NewEffContext(nil), LogLevel: 4})
	t.Cleanup(func() { _ = srv.Close() })
	if err := srv.LoadModules([]string{filepath.Join(root, debugFixture+".ail")}); err != nil {
		t.Fatalf("LoadModules: %v", err)
	}
	out := captureStderr(t, func() { callRouteFixture(t, srv, debugFixture) })
	if strings.Contains(out, "structured-line") {
		t.Errorf("--log-level none did not suppress the ERROR log line:\n%s", out)
	}
	if !strings.Contains(out, `"source":"Debug.check"`) {
		t.Errorf("failed check must surface regardless of log level:\n%s", out)
	}
}
