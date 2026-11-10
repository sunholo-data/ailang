package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A corrupt cached snapshot fails one run, is removed, and the next run heals.
func TestCorruptSnapshotCacheHeals(t *testing.T) {
	f := newFixture(t, oneBlocking, "")
	f.remote.repos["owner/app"] = map[string]string{"a/ok.ail": "module a/ok\n"}
	m := &Materialiser{StateDir: filepath.Join(f.dir, "state")}
	tp := m.tarPath("owner/app", strings.Repeat("a", 40))
	if err := os.MkdirAll(filepath.Dir(tp), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tp, []byte("not a tarball"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _ := f.run(f.options(f.base, f.base)); code != exitInstrument {
		t.Fatalf("corrupt cache: exit %d, want 3", code)
	}
	if _, err := os.Stat(tp); !os.IsNotExist(err) {
		t.Fatalf("corrupt cached tarball still in the cache: %v", err)
	}
	o := f.options(f.base, f.base)
	o.Out = filepath.Join(f.dir, "out2")
	if code, rep := f.run(o); code != exitPass {
		t.Fatalf("second run exit %d, want 0 after refetch; failures %v", code, rep.Failures)
	}
}

type garbageRemote struct{ stubRemote }

func (g *garbageRemote) Tarball(repo, sha string, w io.Writer) error {
	_, err := w.Write([]byte("<html>rate limited</html>"))
	return err
}

// A download that is not a readable tar.gz is a loud error and is never cached.
func TestUnreadableDownloadIsNotCached(t *testing.T) {
	dir := t.TempDir()
	m := &Materialiser{StateDir: dir, Remote: &garbageRemote{}}
	if _, _, err := m.snapshot("owner/app", ""); err == nil || !strings.Contains(err.Error(), "not a readable tar.gz") {
		t.Fatalf("snapshot of a garbage download: %v, want a loud error", err)
	}
	if _, err := os.Stat(m.tarPath("owner/app", strings.Repeat("a", 40))); !os.IsNotExist(err) {
		t.Fatalf("garbage download was cached: %v", err)
	}
}

func TestExtractionSizeCaps(t *testing.T) {
	saveEntry, saveTotal := maxEntryBytes, maxTotalBytes
	defer func() { maxEntryBytes, maxTotalBytes = saveEntry, saveTotal }()
	tarOf := func(sizes ...int) *bytes.Buffer {
		var buf bytes.Buffer
		tw := tar.NewWriter(&buf)
		for i, n := range sizes {
			_ = tw.WriteHeader(&tar.Header{Name: fmt.Sprintf("top/f%d.ail", i), Mode: 0o644, Size: int64(n), Typeflag: tar.TypeReg})
			_, _ = tw.Write(bytes.Repeat([]byte("x"), n))
		}
		_ = tw.Close()
		return &buf
	}
	maxEntryBytes, maxTotalBytes = 10, 15
	if _, err := extractTar(tarOf(11), t.TempDir(), 1); err == nil || !strings.Contains(err.Error(), "per-file cap") {
		t.Errorf("11-byte file under a 10-byte cap: %v", err)
	}
	if _, err := extractTar(tarOf(8, 8), t.TempDir(), 1); err == nil || !strings.Contains(err.Error(), "total cap") {
		t.Errorf("16 bytes under a 15-byte total cap: %v", err)
	}
	if _, err := extractTar(tarOf(7, 8), t.TempDir(), 1); err != nil {
		t.Errorf("15 bytes within the caps: %v", err)
	}
}

// Skipped links are visible in report.md, not only in report.json.
func TestSkippedLinksInMarkdown(t *testing.T) {
	var md strings.Builder
	(&Report{Sources: []Source{{Corpus: "app", Skipped: 7}}}).renderMarkdown(&md)
	if !strings.Contains(md.String(), "links skipped") || !strings.Contains(md.String(), "| 7 |") {
		t.Fatalf("report.md lacks the skipped-link count:\n%s", md.String())
	}
}

func TestAllowanceNeedsAReason(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(p, []byte(oneBlocking+"  allow_not_measured: {max: 3}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCorpora(p); err == nil {
		t.Fatal("allow_not_measured without a reason was accepted")
	}
}
