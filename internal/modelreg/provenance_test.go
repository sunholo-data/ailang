package modelreg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-MODEL-REGISTRY-SINGLE-SOURCE M4 (decision D1(a), ratified 2026-08-27).
//
// Before this, InitModelsConfig tried the EMBEDDED registry first and only fell
// back to disk for development. That is why a published registry was ignored by
// every installed binary — the central mechanism problem D1 exists to fix.
// Precedence is now explicit path -> published -> embedded, and the process says
// which one won.

func writeRegistry(t *testing.T, dir, marker string) string {
	t.Helper()
	p := filepath.Join(dir, "models.yml")
	// A minimal but VALID registry: one row that GetExecutorForModel can resolve,
	// so a test asserting "this source won" is asserting on real parsed content.
	body := "models:\n  " + marker + ":\n    api_name: \"" + marker + "\"\n" +
		"    provider: \"openrouter\"\n    agent_cli: \"opencode\"\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestPrecedence_ExplicitPathWins(t *testing.T) {
	dir := t.TempDir()
	p := writeRegistry(t, dir, "sentinel-explicit")
	t.Setenv(ModelsPathEnv, p)

	src, err := initModelsConfigFrom()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if src.Kind != SourceExplicitPath {
		t.Errorf("Kind = %q, want %q", src.Kind, SourceExplicitPath)
	}
	if _, err := GlobalModelsConfig.GetModel("sentinel-explicit"); err != nil {
		t.Errorf("explicit path did not win: sentinel row absent (%v)", err)
	}
}

func TestPrecedence_PublishedBeatsEmbedded(t *testing.T) {
	dir := t.TempDir()
	writeRegistry(t, dir, "sentinel-published")
	t.Setenv(ModelsPathEnv, "")
	t.Setenv(PublishedDirEnv, dir)

	src, err := initModelsConfigFrom()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if src.Kind != SourcePublished {
		t.Errorf("Kind = %q, want %q — embed still wins, which is the D1 defect", src.Kind, SourcePublished)
	}
	if _, err := GlobalModelsConfig.GetModel("sentinel-published"); err != nil {
		t.Errorf("published registry did not win over embed (%v)", err)
	}
}

func TestPrecedence_EmbeddedIsTheFloor(t *testing.T) {
	t.Setenv(ModelsPathEnv, "")
	t.Setenv(PublishedDirEnv, filepath.Join(t.TempDir(), "does-not-exist"))

	src, err := initModelsConfigFrom()
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if src.Kind != SourceEmbedded {
		t.Errorf("Kind = %q, want %q", src.Kind, SourceEmbedded)
	}
	// Control: the embedded floor must be the REAL registry, not an empty shell.
	if _, err := GlobalModelsConfig.GetModel("gpt5-6-sol"); err != nil {
		t.Errorf("embedded floor is not the real registry (%v)", err)
	}
}

// An unparseable published registry must fall back to embed AND SAY SO. A silent
// fallback here is a silent model-assignment change on a billing path.
func TestPrecedence_UnparseablePublishedFallsBackLoudly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "models.yml"), []byte("models: [this is not a map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(ModelsPathEnv, "")
	t.Setenv(PublishedDirEnv, dir)

	src, err := initModelsConfigFrom()
	if err != nil {
		t.Fatalf("init should fall back, not fail: %v", err)
	}
	if src.Kind != SourceEmbedded {
		t.Fatalf("Kind = %q, want %q (embed is the floor)", src.Kind, SourceEmbedded)
	}
	if src.Degraded == "" {
		t.Error("fell back silently: Degraded is empty, so nothing tells an operator " +
			"the published registry was rejected")
	}
	if !strings.Contains(src.Degraded, dir) {
		t.Errorf("Degraded should name the rejected path so it can be fixed; got %q", src.Degraded)
	}
}

// The startup line is the answer to "which registry is this process using?".
func TestSourceProvenanceLineIsGreppable(t *testing.T) {
	t.Setenv(ModelsPathEnv, "")
	t.Setenv(PublishedDirEnv, filepath.Join(t.TempDir(), "nope"))

	src, err := initModelsConfigFrom()
	if err != nil {
		t.Fatal(err)
	}
	line := src.String()
	for _, want := range []string{"models registry:", string(SourceEmbedded)} {
		if !strings.Contains(line, want) {
			t.Errorf("provenance line %q missing %q", line, want)
		}
	}
	if src.Version == "" {
		t.Error("provenance line carries no version; 'which registry' is then unanswerable")
	}
}
