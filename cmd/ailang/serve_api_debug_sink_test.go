package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestServeAPI_NoCapsStillFlushesDebugOutput is the D5 regression test for
// M-DEBUG-SINK-STRUCTURED-LINES: before the fix, serve-api constructed its
// effect context only when --caps/--ai-*/--verify-contracts was given, so a
// bare `ailang serve-api <dir>` silently DROPPED every Debug.log line. The
// same run also pins the wire format: the structured line and the failed
// check must reach stderr as bare JSON, never behind a timestamp prefix.
func TestServeAPI_NoCapsStillFlushesDebugOutput(t *testing.T) {
	binary := buildAilang(t)

	moduleRoot := t.TempDir()
	modulePath := filepath.Join(moduleRoot, "api", "logs.ail")
	if err := os.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
		t.Fatal(err)
	}
	content := `module api/logs
import std/debug as Debug

export func ping() -> string =
  let _ = Debug.log("{\"severity\":\"ERROR\",\"message\":\"nocaps-structured\"}") in
  let _ = Debug.check(false, "nocaps-check") in
  "pong"
`
	if err := os.WriteFile(modulePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	port := freePort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	// Deliberately NO --caps: the bug only reproduces without it.
	cmd := exec.CommandContext(ctx, binary, "serve-api", "--port", port, moduleRoot)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	base := "http://127.0.0.1:" + port
	waitForServer(t, base+"/api/_health", &stderr)

	resp, err := http.Post(base+"/api/api/logs/ping", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("POST: %v; stderr:\n%s", err, stderr.String())
	}
	defer resp.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil || body["result"] != "pong" {
		t.Fatalf("route call failed (%v): %v; stderr:\n%s", err, body, stderr.String())
	}

	// The flush is deferred to after the response is written; give it a beat.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !strings.Contains(stderr.String(), "nocaps-check") {
		time.Sleep(50 * time.Millisecond)
	}
	out := stderr.String()

	var want = []string{
		`{"severity":"ERROR","message":"nocaps-structured"}`,
		`{"severity":"ERROR","message":"assertion failed: nocaps-check","location":"unknown","source":"Debug.check"}`,
	}
	for _, line := range want {
		if !hasExactLine(out, line) {
			t.Errorf("expected bare JSON line %q on stderr (silently dropped, or decorated?); stderr:\n%s", line, out)
		}
	}
	if strings.Contains(out, "[Debug] {") || strings.Contains(out, "ASSERT FAIL") {
		t.Errorf("old decorated forms present:\n%s", out)
	}
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

func waitForServer(t *testing.T, healthURL string, stderr *bytes.Buffer) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("serve-api never became healthy; stderr:\n%s", stderr.String())
}
