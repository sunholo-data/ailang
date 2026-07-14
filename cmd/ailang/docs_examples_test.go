package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// TestDocsExamples_InstalledBinaryLayout proves `ailang docs --examples <module>`
// prints its "Try it" section from a download-style examples layout resolved via
// AILANG_EXAMPLES, with the process CWD OUTSIDE any source checkout — i.e. the
// installed-binary path, no network, no repo-relative resolution.
func TestDocsExamples_InstalledBinaryLayout(t *testing.T) {
	bin := testutil.FindAilangBinary(t)

	repoRoot := findRepoRootForTest(t)

	// Build a temp dir laid out exactly like `ailang examples download` produces:
	//   <examplesDir>/manifest.json
	//   <examplesDir>/runnable/stdlib_gzip.ail
	examplesDir := t.TempDir()
	runnable := filepath.Join(examplesDir, "runnable")
	if err := os.MkdirAll(runnable, 0o755); err != nil {
		t.Fatalf("mkdir runnable: %v", err)
	}

	// Copy the real example so its imports match the manifest `modules` entry.
	srcExample := filepath.Join(repoRoot, "examples", "runnable", "stdlib_gzip.ail")
	copyFile(t, srcExample, filepath.Join(runnable, "stdlib_gzip.ail"))

	// Minimal manifest carrying the committed `modules` field for the example.
	manifest := `{
  "schema": "ailang.manifest/v1",
  "schema_version": "1.0.0",
  "examples": [
    {
      "path": "runnable/stdlib_gzip.ail",
      "status": "working",
      "tags": ["stdlib", "std/gzip"],
      "description": "gzip round-trip demo",
      "modules": ["std/bytes", "std/gzip", "std/io", "std/result"]
    }
  ],
  "statistics": {"total": 1, "working": 1, "broken": 0, "experimental": 0, "coverage": 1.0}
}
`
	if err := os.WriteFile(filepath.Join(examplesDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// CWD outside any source checkout.
	outsideCWD := t.TempDir()

	cmd := exec.Command(bin, "docs", "--examples", "gzip")
	cmd.Dir = outsideCWD
	cmd.Env = append(os.Environ(),
		"AILANG_EXAMPLES="+examplesDir,
		// Installed binaries ship stdlib too; point at the repo's std for the
		// module-doc header render. The Try-it section under test is driven by
		// AILANG_EXAMPLES, not this.
		"AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("docs --examples gzip failed: %v\n%s", err, out)
	}

	got := string(out)
	if !strings.Contains(got, "## Try it") {
		t.Fatalf("expected a '## Try it' section, got:\n%s", got)
	}
	if !strings.Contains(got, "examples/runnable/stdlib_gzip.ail") {
		t.Fatalf("expected the gzip example path in Try-it, got:\n%s", got)
	}
	if !strings.Contains(got, "ailang run examples/runnable/stdlib_gzip.ail") {
		t.Fatalf("expected a run command in Try-it, got:\n%s", got)
	}
}

// TestDocsExamples_NoMatchesIsExplicit proves a module with no registered
// examples says so explicitly (never silently identical to flagless output).
func TestDocsExamples_NoMatchesIsExplicit(t *testing.T) {
	bin := testutil.FindAilangBinary(t)
	repoRoot := findRepoRootForTest(t)

	examplesDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(examplesDir, "runnable"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Manifest with one example that imports std/io only — nothing imports std/gzip.
	manifest := `{
  "schema": "ailang.manifest/v1",
  "schema_version": "1.0.0",
  "examples": [
    {"path": "runnable/x.ail", "status": "working", "modules": ["std/io"]}
  ],
  "statistics": {"total": 1, "working": 1, "broken": 0, "experimental": 0, "coverage": 1.0}
}
`
	if err := os.WriteFile(filepath.Join(examplesDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cmd := exec.Command(bin, "docs", "--examples", "gzip")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(),
		"AILANG_EXAMPLES="+examplesDir,
		"AILANG_STDLIB_PATH="+filepath.Join(repoRoot, "std"),
	)
	out, _ := cmd.CombinedOutput()
	got := string(out)
	if !strings.Contains(got, "No registered examples import std/gzip yet") {
		t.Fatalf("expected explicit no-examples message, got:\n%s", got)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// findRepoRootForTest walks up from the test's working directory to the repo
// root (the dir containing go.mod). Used only to locate source example/std files
// to seed the temp download layout — the binary under test never sees this path.
func findRepoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root (go.mod) from test cwd")
		}
		dir = parent
	}
}
