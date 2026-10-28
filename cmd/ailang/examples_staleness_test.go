package main

import (
	"encoding/json"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"

	examplescorpus "github.com/sunholo-data/ailang/examples"
)

// embeddedManifestVariant returns the binary's embedded manifest, edited by
// edit, as JSON — so a test can plant a download that is older, the same, or
// newer than the binary without hard-coding its 200-odd entries.
func embeddedManifestVariant(t *testing.T, edit func(*ExampleManifest)) string {
	t.Helper()
	data, err := fs.ReadFile(examplescorpus.Corpus, "manifest.json")
	if err != nil {
		t.Skipf("binary carries no embedded corpus: %v", err)
	}
	var m ExampleManifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if len(m.Examples) < 2 {
		t.Fatalf("embedded manifest has %d entries; the test needs at least 2", len(m.Examples))
	}
	edit(&m)
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// plantDownload writes manifest into ~/.ailang/examples under the test HOME.
func plantDownload(t *testing.T, manifest string) string {
	t.Helper()
	dl, err := defaultExamplesDownloadDir()
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dl, "manifest.json"), manifest)
	writeTestFile(t, filepath.Join(dl, "runnable", "planted.ail"), "-- planted\n")
	return dl
}

// The reported bug: an old `ailang examples download` (one example fewer than
// the binary carries) shadowed the embedded corpus, so the CLI showed 238 of
// 239. A stale download must lose to the embedded corpus, with one stderr note.
func TestExamplesCorpus_StaleDownloadDoesNotShadowEmbedded(t *testing.T) {
	isolateExamplesResolution(t)
	var dropped string
	dl := plantDownload(t, embeddedManifestVariant(t, func(m *ExampleManifest) {
		dropped = m.Examples[len(m.Examples)-1].Path
		m.Examples = m.Examples[:len(m.Examples)-1]
	}))
	staleDownloadNote = sync.Once{}

	var c *examplesCorpus
	stderr := captureStderr(t, func() {
		var err error
		if c, err = openExamplesCorpus(); err != nil {
			t.Fatal(err)
		}
	})
	if c.dir != "" {
		t.Fatalf("stale download %s shadowed the embedded corpus (it lacks %s)", c.dir, dropped)
	}
	if !strings.Contains(stderr, dl) || !strings.Contains(stderr, "lacks 1 example") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("want one stderr line naming %s and the 1 missing example, got %q", dl, stderr)
	}
	// Resolving again (docs and examples both resolve) prints nothing more.
	if again := captureStderr(t, func() { _, _ = openExamplesCorpus() }); again != "" {
		t.Errorf("note repeated: %q", again)
	}
}

// A download as new as the binary — or newer, with examples the binary lacks —
// is still used. Path spelling ("runnable/x.ail" vs "x.ail") is not a difference.
func TestExamplesCorpus_CurrentOrNewerDownloadStillWins(t *testing.T) {
	cases := map[string]func(*ExampleManifest){
		"same as the binary": func(*ExampleManifest) {},
		"newer than the binary": func(m *ExampleManifest) {
			m.Examples = append(m.Examples, ExampleEntry{Path: "runnable/planted.ail", Status: "working"})
		},
		"other path spelling": func(m *ExampleManifest) {
			for i := range m.Examples {
				if p := m.Examples[i].Path; strings.HasPrefix(p, "runnable/") {
					m.Examples[i].Path = strings.TrimPrefix(p, "runnable/")
				} else {
					m.Examples[i].Path = "runnable/" + p
				}
			}
		},
	}
	for name, edit := range cases {
		t.Run(name, func(t *testing.T) {
			isolateExamplesResolution(t)
			dl := plantDownload(t, embeddedManifestVariant(t, edit))
			c, err := openExamplesCorpus()
			if err != nil {
				t.Fatal(err)
			}
			if c.dir != dl {
				t.Fatalf("resolved %q, want the download %s", c.dir, dl)
			}
		})
	}
}

// AILANG_EXAMPLES is set by the host, so it wins even when it is older than
// the embedded corpus.
func TestExamplesCorpus_EnvWinsEvenWhenStale(t *testing.T) {
	isolateExamplesResolution(t)
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "manifest.json"), plantedManifest)
	t.Setenv(config.EnvExamples, dir)
	c, err := openExamplesCorpus()
	if err != nil {
		t.Fatal(err)
	}
	if c.dir != dir {
		t.Fatalf("resolved %q, want AILANG_EXAMPLES %s", c.dir, dir)
	}
}

// A download whose manifest cannot be read is treated as stale, not trusted.
func TestExamplesLaggingEmbedded_UnreadableManifest(t *testing.T) {
	embedded, err := readManifestPaths(examplescorpus.Corpus)
	if err != nil {
		t.Skipf("no embedded corpus: %v", err)
	}
	dir := t.TempDir()
	if got := examplesLaggingEmbedded(dir); got != len(embedded) {
		t.Errorf("no manifest: lag = %d, want %d", got, len(embedded))
	}
	writeTestFile(t, filepath.Join(dir, "manifest.json"), "{not json")
	if got := examplesLaggingEmbedded(dir); got != len(embedded) {
		t.Errorf("bad manifest: lag = %d, want %d", got, len(embedded))
	}
}
