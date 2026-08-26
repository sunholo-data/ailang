package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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

// M2 criterion 3: --storage-state travels alongside --isolated. The path may
// appear in child argv; the storage-state CONTENTS and the canonical object
// reference (profile hash) may not appear anywhere in the generated MCP config.
func TestConnectionPassesStorageStateAlongsideIsolated(t *testing.T) {
	const cookieValue = "CANONICAL-SESSION-VALUE"
	const profileHash = "sha256:deadbeefdeadbeef"

	statePath := filepath.Join(t.TempDir(), "storage-state.json")
	if err := os.WriteFile(statePath, []byte(`{"cookies":[{"name":"sid","value":"`+cookieValue+`"}]}`), 0600); err != nil {
		t.Fatal(err)
	}

	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.CreateWithStorageState(context.Background(),
		browser.SessionSpec{
			RunID:       "run",
			Headless:    true,
			ProfileRef:  "shop@v3",
			ProfileHash: profileHash,
			AuthLeaseID: "lease-1",
			AuthMode:    "read",
		},
		StorageStateAttachment{Path: statePath})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := provider.Connection(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	mcp, env := conn.Materialize()

	want := []string{
		"-y", "@playwright/mcp@0.0.79", "--headless",
		"--isolated", "--storage-state", statePath,
		"--output-dir", session.ArtifactDir, "--save-session",
		"--viewport-size", "1280x720",
	}
	if !reflect.DeepEqual(mcp.Args, want) {
		t.Fatalf("args = %#v, want %#v", mcp.Args, want)
	}
	isolatedAt, storageAt := indexOf(mcp.Args, "--isolated"), indexOf(mcp.Args, "--storage-state")
	if isolatedAt < 0 || storageAt != isolatedAt+1 {
		t.Fatalf("--storage-state is not adjacent to --isolated: %#v", mcp.Args)
	}

	// The whole serialized surface: args, env var names, env values, JSON.
	encoded, err := json.Marshal(conn)
	if err != nil {
		t.Fatal(err)
	}
	surfaces := []string{
		strings.Join(mcp.Args, " "),
		strings.Join(mcp.EnvVars, " "),
		fmt.Sprint(env),
		string(encoded),
		conn.String(),
	}
	for i, surface := range surfaces {
		if strings.Contains(surface, cookieValue) {
			t.Fatalf("surface %d leaked storage-state contents: %s", i, surface)
		}
		if strings.Contains(surface, profileHash) || strings.Contains(surface, "deadbeef") {
			t.Fatalf("surface %d leaked the canonical object reference: %s", i, surface)
		}
	}

	// The path itself is permitted in argv, and only there.
	if !slices.Contains(mcp.Args, statePath) {
		t.Fatalf("storage-state path missing from argv: %#v", mcp.Args)
	}
	if strings.Contains(string(encoded), cookieValue) {
		t.Fatalf("connection JSON leaked state contents: %s", encoded)
	}
}

func TestConnectionOmitsStorageStateWhenUnauthenticated(t *testing.T) {
	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.Create(context.Background(), browser.SessionSpec{RunID: "run"})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := provider.Connection(context.Background(), session)
	if err != nil {
		t.Fatal(err)
	}
	mcp, _ := conn.Materialize()
	if slices.Contains(mcp.Args, "--storage-state") {
		t.Fatalf("unauthenticated session got --storage-state: %#v", mcp.Args)
	}
	if !slices.Contains(mcp.Args, "--isolated") {
		t.Fatalf("--isolated missing: %#v", mcp.Args)
	}
}

func indexOf(values []string, want string) int {
	return slices.Index(values, want)
}

// A destroyed or never-created materialization must fail loudly at attach time.
// Silently launching without --storage-state would turn a cleanup bug into an
// unauthenticated run that merely looks like a failed login.
func TestCreateWithStorageStateRejectsUnusablePaths(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"empty":       "",
		"missing":     filepath.Join(dir, "gone.json"),
		"a directory": dir,
	}
	for name, path := range cases {
		t.Run(name, func(t *testing.T) {
			provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
			if err != nil {
				t.Fatal(err)
			}
			_, err = provider.CreateWithStorageState(context.Background(),
				browser.SessionSpec{RunID: "run"}, StorageStateAttachment{Path: path})
			if !browser.IsFailure(err, browser.FailureProvision) {
				t.Fatalf("error = %v, want provision failure", err)
			}
		})
	}
}

func TestStopForgetsStorageStateWithoutDeletingIt(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "storage-state.json")
	if err := os.WriteFile(statePath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	provider, err := New(Config{BaseDir: t.TempDir(), NpxPath: "/test/npx"})
	if err != nil {
		t.Fatal(err)
	}
	session, err := provider.CreateWithStorageState(context.Background(),
		browser.SessionSpec{RunID: "run"}, StorageStateAttachment{Path: statePath})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := provider.Stop(context.Background(), session); err != nil {
		t.Fatal(err)
	}
	if provider.storageStatePath(session) != "" {
		t.Fatal("provider still holds the storage-state path after Stop")
	}
	// The file belongs to the auth broker's materialization handle, not to the
	// provider; Stop must not delete it out from under its owner.
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("Stop deleted a file it does not own: %v", err)
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
