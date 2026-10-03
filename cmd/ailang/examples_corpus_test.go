package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// isolateExamplesResolution points every on-disk corpus location the
// resolver consults at empty temp dirs, so only what a test plants (or the
// embedded corpus) can be found.
func isolateExamplesResolution(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(config.EnvExamples, "")
	t.Setenv(config.EnvAgentPolicy, "")
	cwd := filepath.Join(t.TempDir(), "a", "b", "c")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	return cwd
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const plantedManifest = `{"schema":"ailang.manifest/v1","examples":[{"path":"planted.ail","status":"working","tags":["planted"],"description":"PLANTED corpus"}]}`

// #1553: a corpus with runnable/*.ail but no manifest.json is a clear error
// naming the directory — never a nil dereference.
func TestExamplesCorpus_MissingManifestIsNamedError(t *testing.T) {
	isolateExamplesResolution(t)
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "runnable", "x.ail"), "module x\n")
	t.Setenv(config.EnvExamples, dir)

	c, err := openExamplesCorpus()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := c.manifest(); err == nil || !strings.Contains(err.Error(), "manifest.json") || !strings.Contains(err.Error(), dir) {
		t.Fatalf("manifest() error must name manifest.json and %s, got %v", dir, err)
	}
	if _, _, err := searchExamples(c, "x", 10); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("searchExamples must return an error naming %s, got %v", dir, err)
	}
	if _, err := loadExamplesManifest(); err == nil || !strings.Contains(err.Error(), dir) {
		t.Fatalf("loadExamplesManifest must return an error naming %s, got %v", dir, err)
	}
}

// #1552: with no corpus on disk anywhere, the corpus built into the binary
// answers search, list (manifest) and show.
func TestExamplesCorpus_EmbeddedFallback(t *testing.T) {
	isolateExamplesResolution(t)

	c, err := openExamplesCorpus()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if c.dir != "" {
		t.Fatalf("expected the embedded corpus, resolved on-disk %s", c.dir)
	}
	m, err := c.manifest()
	if err != nil || len(m.Examples) == 0 {
		t.Fatalf("embedded manifest: %v (%d entries)", err, len(m.Examples))
	}
	res, _, err := searchExamples(c, "pattern matching", 5)
	if err != nil || len(res) == 0 {
		t.Fatalf("embedded search: %v, %d results", err, len(res))
	}
	if _, err := fsReadExample(c, "adt_option"); err != nil {
		t.Fatalf("embedded show adt_option: %v", err)
	}
}

// #1552: a confined tool (AILANG_AGENT_POLICY set — the ailang_only lane, and
// every child policy-tool spawns) never picks up a cwd-relative examples/ dir
// an agent could have planted. Unconfined, the cwd fallback still works.
func TestExamplesCorpus_ConfinedIgnoresCwdExamples(t *testing.T) {
	cwd := isolateExamplesResolution(t)
	// ../../examples relative to cwd.
	planted := filepath.Join(cwd, "..", "..", "examples")
	writeTestFile(t, filepath.Join(planted, "manifest.json"), plantedManifest)
	writeTestFile(t, filepath.Join(planted, "runnable", "planted.ail"), "-- planted\n")

	c, err := openExamplesCorpus()
	if err != nil || c.dir == "" {
		t.Fatalf("unconfined should resolve the cwd-relative corpus: dir=%q err=%v", c.dir, err)
	}

	t.Setenv(config.EnvAgentPolicy, filepath.Join(t.TempDir(), "policy.toml"))
	c, err = openExamplesCorpus()
	if err != nil {
		t.Fatalf("open (confined): %v", err)
	}
	if c.dir != "" {
		t.Fatalf("confined resolution picked an on-disk corpus %s", c.dir)
	}
	res, meta, err := searchExamples(c, "planted", 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res {
		if r.path == "planted.ail" || strings.Contains(meta[r.path].Description, "PLANTED") {
			t.Fatalf("confined search returned the planted example: %+v", r)
		}
	}
}

// #1552: a host can give each task its own writable HOME, so a confined tool
// must not read ~/.ailang/examples either.
func TestExamplesCorpus_ConfinedIgnoresHomeExamples(t *testing.T) {
	isolateExamplesResolution(t)
	dl, err := defaultExamplesDownloadDir()
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dl, "manifest.json"), plantedManifest)
	writeTestFile(t, filepath.Join(dl, "runnable", "planted.ail"), "-- planted\n")

	c, err := openExamplesCorpus()
	if err != nil || c.dir == "" {
		t.Fatalf("unconfined should resolve ~/.ailang/examples: dir=%q err=%v", c.dir, err)
	}
	t.Setenv(config.EnvAgentPolicy, filepath.Join(t.TempDir(), "policy.toml"))
	c, err = openExamplesCorpus()
	if err != nil {
		t.Fatalf("open (confined): %v", err)
	}
	if c.dir != "" {
		t.Fatalf("confined resolution picked the HOME corpus %s", c.dir)
	}
}

// AILANG_EXAMPLES still wins over the embedded corpus, confined or not: it is
// set by the host, not the agent.
func TestExamplesCorpus_EnvBeatsEmbedded(t *testing.T) {
	isolateExamplesResolution(t)
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.json"), plantedManifest)
	writeTestFile(t, filepath.Join(dir, "runnable", "planted.ail"), "-- planted\n")
	t.Setenv(config.EnvExamples, dir)
	t.Setenv(config.EnvAgentPolicy, filepath.Join(t.TempDir(), "policy.toml"))

	c, err := openExamplesCorpus()
	if err != nil {
		t.Fatal(err)
	}
	res, _, err := searchExamples(c, "planted", 10)
	if err != nil || len(res) != 1 || res[0].path != "planted.ail" {
		t.Fatalf("AILANG_EXAMPLES corpus not used: %+v %v", res, err)
	}
}
