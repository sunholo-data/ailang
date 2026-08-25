package local

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/browser"
)

func TestProviderCreatesUniqueIsolatedSessions(t *testing.T) {
	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx", Now: func() time.Time { return time.Unix(10, 0) }})
	if err != nil {
		t.Fatal(err)
	}
	var sessions [2]browser.Session
	var errs [2]error
	var wg sync.WaitGroup
	for i := range sessions {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sessions[index], errs[index] = provider.Create(context.Background(), browser.SessionSpec{RunID: "same-run"})
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if sessions[0].ID == sessions[1].ID || sessions[0].StateDir == sessions[1].StateDir || sessions[0].ArtifactDir == sessions[1].ArtifactDir {
		t.Fatalf("sessions share identity or storage: %#v %#v", sessions[0], sessions[1])
	}
	for _, session := range sessions {
		if info, err := os.Stat(session.StateDir); err != nil || !info.IsDir() {
			t.Fatalf("state directory unavailable: %v", err)
		}
		if info, err := os.Stat(session.ArtifactDir); err != nil || !info.IsDir() {
			t.Fatalf("artifact directory unavailable: %v", err)
		}
	}
	_, _ = provider.Stop(context.Background(), sessions[0])
	_, _ = provider.Stop(context.Background(), sessions[1])
}

func TestConnectionUsesPinnedExactPlaywrightMCPArgs(t *testing.T) {
	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.Create(context.Background(), browser.SessionSpec{RunID: "run", Headless: true, ViewportWidth: 1440, ViewportHeight: 900, ActionTimeout: 7 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := provider.Connection(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	mcp, env := conn.Materialize()
	want := []string{
		"-y", "@playwright/mcp@0.0.79", "--headless", "--isolated",
		"--output-dir", session.ArtifactDir, "--save-session",
		"--viewport-size", "1440x900", "--timeout-action", "7000",
	}
	if mcp.Command != "/test/npx" || !reflect.DeepEqual(mcp.Args, want) || !mcp.Required {
		t.Fatalf("MCP spec = %#v, want args %#v", mcp, want)
	}
	if len(env) != 0 || len(mcp.EnvVars) != 0 {
		t.Fatalf("local connection unexpectedly has secrets: env=%v names=%v", env, mcp.EnvVars)
	}
	b, _ := json.Marshal(conn)
	if strings.Contains(string(b), session.StateDir) {
		t.Fatalf("connection JSON leaked state directory: %s", b)
	}
}

func TestNewFailsLoudlyWhenNpxUnavailable(t *testing.T) {
	_, err := New(Config{BaseDir: t.TempDir(), LookPath: func(string) (string, error) { return "", os.ErrNotExist }})
	if !browser.IsFailure(err, browser.FailureProvision) {
		t.Fatalf("error = %v, want provision failure", err)
	}
}

func TestExportHashesArtifactsAndStopIsIdempotent(t *testing.T) {
	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.Create(context.Background(), browser.SessionSpec{RunID: "run"})
	if err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(session.ArtifactDir, "session.md")
	if err := os.WriteFile(artifact, []byte("safe transcript"), 0600); err != nil {
		t.Fatal(err)
	}
	manifest, err := provider.Export(context.Background(), session, session.ArtifactDir)
	if err != nil {
		t.Fatal(err)
	}
	if !manifest.Complete || len(manifest.Refs) != 1 || manifest.Refs[0].SHA256 == "" {
		t.Fatalf("unexpected artifact manifest: %#v", manifest)
	}
	if _, err := provider.Stop(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Stop(context.Background(), session); err != nil {
		t.Fatalf("idempotent stop failed: %v", err)
	}
	if _, err := os.Stat(session.StateDir); !os.IsNotExist(err) {
		t.Fatalf("state directory survives stop: %v", err)
	}
}
