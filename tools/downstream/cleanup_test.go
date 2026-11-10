package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range ents {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// A run leaves only its reports behind. A release-tier run clones every corpus
// (about 3.4 GB); twenty of them filled 80 GB of one session's scratch dir on
// 2026-11-10 because nothing removed them.
func TestWorkTreesAreRemovedByDefault(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	o := f.options(f.base, f.base)
	o.KeepWork = false
	var log bytes.Buffer
	code, err := Run(o, f.remote, &log)
	if err != nil || code != exitPass {
		t.Fatalf("exit %d err %v; log: %s", code, err, log.String())
	}
	got := strings.Join(dirNames(t, o.Out), " ")
	if got != "pins.md report.json report.md rows.jsonl" {
		t.Fatalf("run dir holds %q, want only the four reports", got)
	}
	if !strings.Contains(log.String(), "KEEP_WORK=1") {
		t.Errorf("the log does not say the working trees were removed or how to keep them: %s", log.String())
	}

	// The opt-out keeps them.
	o.Out = filepath.Join(f.dir, "out-kept")
	o.KeepWork = true
	if code, err := Run(o, f.remote, &log); err != nil || code != exitPass {
		t.Fatalf("exit %d err %v", code, err)
	}
	if _, err := os.Stat(filepath.Join(o.Out, "corpora")); err != nil {
		t.Errorf("KeepWork did not keep the corpus clones: %v", err)
	}
}

// Read-only trees (a module cache, a read-only checkout) and symbolic links must
// not defeat the cleanup, and a link's target must survive it.
func TestCleanWorkHandlesReadOnlyTreesAndLinks(t *testing.T) {
	out := t.TempDir()
	outside := filepath.Join(t.TempDir(), "keep.txt")
	if err := os.WriteFile(outside, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	ro := filepath.Join(out, "corpora", "x", "ro")
	if err := os.MkdirAll(ro, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ro, "f"), []byte("12345"), 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(out, "corpora", "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(outside), filepath.Join(out, "dirlink")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	if err := os.WriteFile(filepath.Join(out, "report.md"), []byte("r"), 0o644); err != nil {
		t.Fatal(err)
	}
	freed, err := cleanWork(out)
	if err != nil {
		t.Fatal(err)
	}
	if freed != 5 {
		t.Errorf("freed %d bytes, want 5 (the linked file is not ours to count)", freed)
	}
	if got := strings.Join(dirNames(t, out), " "); got != "report.md" {
		t.Errorf("left %q, want only report.md", got)
	}
	if data, err := os.ReadFile(outside); err != nil || string(data) != "precious" {
		t.Errorf("a symbolic link's target was touched: %q %v", data, err)
	}
}
