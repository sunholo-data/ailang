package pipeline

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildCanonicalJSON_ReturnsErrorNotExit(t *testing.T) {
	if os.Getenv("AILANG_CANONICAL_JSON_EXIT_PROBE") == "1" {
		_, err := BuildCanonicalJSON(context.Background(), t.TempDir(), "missing")
		if err == nil {
			os.Exit(43)
		}
		os.Exit(42)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestBuildCanonicalJSON_ReturnsErrorNotExit$")
	cmd.Env = append(os.Environ(), "AILANG_CANONICAL_JSON_EXIT_PROBE=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 42 {
		t.Fatalf("BuildCanonicalJSON terminated its caller or failed to return the missing-file error: exit=%v, want 42", err)
	}
}

func TestBuildCanonicalJSON_ModulePathResolution(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "declared", "module.ail")
	_, err := BuildCanonicalJSON(context.Background(), dir, "declared/module")
	if err == nil {
		t.Fatal("BuildCanonicalJSON succeeded without the resolved .ail source file")
	}
	if !strings.Contains(filepath.ToSlash(err.Error()), filepath.ToSlash(missing)) {
		t.Fatalf("error %q does not name resolved missing file %q", err, missing)
	}
}

func TestBuildCanonicalJSON_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	writeCanonicalJSONFixture(t, filepath.Join(dir, "main.ail"), "module main\n\nexport pure func answer() -> int = 42\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := BuildCanonicalJSON(ctx, dir, "main")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("BuildCanonicalJSON error = %v, want context.Canceled", err)
	}
}

func TestBuildCanonicalJSON_ResolvesIntraPackageImport(t *testing.T) {
	dir := t.TempDir()
	writeCanonicalJSONFixture(t, filepath.Join(dir, "helper.ail"), "module helper\n\nexport pure func value() -> int = 42\n")
	writeCanonicalJSONFixture(t, filepath.Join(dir, "main.ail"), "module main\n\nimport helper (value)\n\nexport pure func answer() -> int = value()\n")
	jsonBytes, err := BuildCanonicalJSON(context.Background(), dir, "main")
	if err != nil {
		t.Fatalf("BuildCanonicalJSON failed to resolve sibling import: %v", err)
	}
	if !strings.Contains(string(jsonBytes), `"name": "answer"`) {
		t.Fatalf("normalized interface does not contain imported module consumer: %s", jsonBytes)
	}
}

func writeCanonicalJSONFixture(t *testing.T, path, source string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create fixture directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
