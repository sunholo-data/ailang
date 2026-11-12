package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tarEntry is one entry for buildTarGz. A nil body with size > 0 writes size
// zero bytes, which gzip shrinks ~1000:1 — the shape of a decompression bomb.
type tarEntry struct {
	name     string
	size     int64
	typeflag byte
}

func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	zeros := make([]byte, 64<<10)
	for _, e := range entries {
		tf := e.typeflag
		if tf == 0 {
			tf = tar.TypeReg
		}
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Size: e.size, Mode: 0o644, Typeflag: tf}); err != nil {
			t.Fatalf("WriteHeader %s: %v", e.name, err)
		}
		for left := e.size; left > 0; {
			n := int64(len(zeros))
			if left < n {
				n = left
			}
			if _, err := tw.Write(zeros[:n]); err != nil {
				t.Fatalf("Write %s: %v", e.name, err)
			}
			left -= n
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func wantLimitError(t *testing.T, err error, constName string) {
	t.Helper()
	var le *TarballLimitError
	if !errors.As(err, &le) {
		t.Fatalf("want *TarballLimitError naming %s, got %v", constName, err)
	}
	if le.Const != constName {
		t.Fatalf("want limit %s, got %s (%v)", constName, le.Const, err)
	}
	if !strings.Contains(err.Error(), constName) {
		t.Fatalf("error must name the limit %s: %v", constName, err)
	}
}

func TestExtractTarball_OversizedEntryRefused(t *testing.T) {
	data := buildTarGz(t, []tarEntry{
		{name: "ailang.toml", size: 10},
		{name: "assets/big.bin", size: MaxTarballEntryBytes + 1},
	})
	dest := t.TempDir()
	err := ExtractTarball(data, dest)
	wantLimitError(t, err, "pkg.MaxTarballEntryBytes")
	if _, statErr := os.Stat(filepath.Join(dest, "assets", "big.bin")); statErr == nil {
		t.Fatal("oversized entry must not be written, not even truncated")
	}
}

func TestExtractTarball_OversizedTotalRefused(t *testing.T) {
	// Each entry is within the per-entry limit; together they pass the total.
	per := int64(MaxTarballEntryBytes)
	n := int(MaxTarballTotalBytes/per) + 1
	entries := make([]tarEntry, n)
	for i := range entries {
		entries[i] = tarEntry{name: fmt.Sprintf("assets/part%02d.bin", i), size: per}
	}
	err := ExtractTarball(buildTarGz(t, entries), t.TempDir())
	wantLimitError(t, err, "pkg.MaxTarballTotalBytes")
}

func TestExtractTarball_TooManyEntriesRefused(t *testing.T) {
	entries := make([]tarEntry, MaxTarballEntries+1)
	for i := range entries {
		entries[i] = tarEntry{name: fmt.Sprintf("m%04d.ail", i), size: 1}
	}
	err := ExtractTarball(buildTarGz(t, entries), t.TempDir())
	wantLimitError(t, err, "pkg.MaxTarballEntries")
}

// A skipped entry type still has its body decompressed by archive/tar, so it
// counts against the limits too.
func TestExtractTarball_SkippedTypeStillBounded(t *testing.T) {
	data := buildTarGz(t, []tarEntry{
		{name: "weird", size: MaxTarballEntryBytes + 1, typeflag: 'Z'},
	})
	err := ExtractTarball(data, t.TempDir())
	wantLimitError(t, err, "pkg.MaxTarballEntryBytes")
}

func TestExtractTarball_CompressedSizeRefused(t *testing.T) {
	err := ExtractTarball(make([]byte, MaxTarballBytes+1), t.TempDir())
	wantLimitError(t, err, "pkg.MaxTarballBytes")
}

func TestCappedReader_RefusesPastCap(t *testing.T) {
	cr := &cappedReader{r: bytes.NewReader(make([]byte, 100)), max: 50}
	buf := make([]byte, 30)
	var got int
	var err error
	for err == nil {
		var n int
		n, err = cr.Read(buf)
		got += n
	}
	var le *TarballLimitError
	if !errors.As(err, &le) {
		t.Fatalf("want *TarballLimitError past the cap, got %v after %d bytes", err, got)
	}
	if got > 51 {
		t.Fatalf("read %d bytes, want at most cap+1 = 51", got)
	}
}

// A package at the limits — the largest one each limit allows — still extracts.
func TestExtractTarball_AtLimitsAccepted(t *testing.T) {
	per := int64(MaxTarballEntryBytes)
	n := int(MaxTarballTotalBytes / per)
	entries := []tarEntry{}
	for i := 0; i < n; i++ {
		entries = append(entries, tarEntry{name: fmt.Sprintf("assets/part%02d.bin", i), size: per})
	}
	for len(entries) < MaxTarballEntries {
		entries = append(entries, tarEntry{name: fmt.Sprintf("d%04d/", len(entries)), typeflag: tar.TypeDir})
	}
	if err := ExtractTarball(buildTarGz(t, entries), t.TempDir()); err != nil {
		t.Fatalf("a package exactly at every limit must extract: %v", err)
	}
}

// A normal package, built by CreateTarball, round-trips unchanged.
func TestExtractTarball_NormalPackageUnaffected(t *testing.T) {
	src := t.TempDir()
	files := map[string]string{
		ManifestFile:       "[package]\nname = \"test/pkg\"\nversion = \"0.1.0\"\n",
		"core.ail":         "module test/pkg/core\n",
		"AGENT.md":         "# test/pkg\n",
		"assets/data.json": strings.Repeat("{}", 200<<10), // 400 KB, near the largest real entry
	}
	for rel, body := range files {
		p := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := CreateTarball(src)
	if err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := ExtractTarball(data, dest); err != nil {
		t.Fatalf("normal package refused: %v", err)
	}
	for rel, body := range files {
		got, err := os.ReadFile(filepath.Join(dest, rel))
		if err != nil || string(got) != body {
			t.Fatalf("%s did not round-trip (err %v, %d bytes)", rel, err, len(got))
		}
	}
}
