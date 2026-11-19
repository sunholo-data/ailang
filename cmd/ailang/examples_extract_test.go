package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestZip builds a zip at dir/name holding the given entries. An entry
// whose name ends in "/" is a directory; a mode is applied when non-zero.
func writeTestZip(t *testing.T, dir string, entries []zipTestEntry) string {
	t.Helper()
	path := filepath.Join(dir, "examples.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range entries {
		hdr := &zip.FileHeader{Name: e.name, Method: zip.Deflate}
		if e.mode != 0 {
			hdr.SetMode(e.mode)
		}
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

type zipTestEntry struct {
	name string
	body string
	mode os.FileMode
}

func TestExtractZipStripsPrefix(t *testing.T) {
	tmp := t.TempDir()
	zipPath := writeTestZip(t, tmp, []zipTestEntry{
		{name: "examples/manifest.json", body: "{}"},
		{name: "examples/a/b.ail", body: "module a/b"},
	})
	dest := filepath.Join(tmp, "out")
	if err := extractZip(zipPath, dest); err != nil {
		t.Fatalf("extractZip: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "a", "b.ail"))
	if err != nil || string(got) != "module a/b" {
		t.Fatalf("a/b.ail = %q, %v", got, err)
	}
}

// A traversal entry must fail the extraction, not be skipped: skipping left a
// quietly partial examples tree and hid a hostile archive.
func TestExtractZipRejectsTraversal(t *testing.T) {
	for _, name := range []string{"../evil.ail", "examples/../../evil.ail", "/abs/evil.ail"} {
		t.Run(name, func(t *testing.T) {
			tmp := t.TempDir()
			zipPath := writeTestZip(t, tmp, []zipTestEntry{
				{name: "manifest.json", body: "{}"},
				{name: name, body: "pwned"},
			})
			dest := filepath.Join(tmp, "out")
			err := extractZip(zipPath, dest)
			if err == nil || !strings.Contains(err.Error(), "escapes the destination") {
				t.Fatalf("extractZip(%q) err = %v, want an escape error", name, err)
			}
			if _, statErr := os.Stat(filepath.Join(tmp, "evil.ail")); statErr == nil {
				t.Fatal("traversal entry was written outside the destination")
			}
		})
	}
}

func TestExtractZipRejectsSymlinkEntry(t *testing.T) {
	tmp := t.TempDir()
	zipPath := writeTestZip(t, tmp, []zipTestEntry{
		{name: "manifest.json", body: "{}"},
		{name: "link", body: "/etc/passwd", mode: os.ModeSymlink | 0o777},
	})
	err := extractZip(zipPath, filepath.Join(tmp, "out"))
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("extractZip err = %v, want a not-a-regular-file error", err)
	}
}
