package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDocsEmbeddedSourceSupportsAllReadModes(t *testing.T) {
	source := docsSource{FS: embeddedStdlibFS, Label: "embedded stdlib"}
	modules := discoverModulesFS(source)
	if len(modules) == 0 {
		t.Fatal("embedded stdlib returned no modules")
	}
	var ioFound bool
	for _, mod := range modules {
		if mod.Name == "std/io" {
			ioFound = true
			sigs, _, err := parseExportSignaturesFS(mod.SourceFS, mod.FilePath)
			if err != nil || len(sigs) == 0 {
				t.Fatalf("embedded io signatures: %v (%d signatures)", err, len(sigs))
			}
		}
	}
	if !ioFound {
		t.Fatal("embedded stdlib did not contain std/io")
	}
	embeddedLines := buildAllFunctionsLinesFS(source)
	if len(embeddedLines) == 0 {
		t.Fatal("embedded --all-functions returned no lines")
	}
	filesystemLines := buildAllFunctionsLines(stdlibDirForTest(t))
	if !reflect.DeepEqual(embeddedLines, filesystemLines) {
		t.Fatal("embedded and filesystem stdlib docs differ")
	}
}

func TestDocsFilesystemSourcePrecedesEmbedded(t *testing.T) {
	root := t.TempDir()
	content := []byte("-- override copy\nmodule std/io\n")
	if err := os.WriteFile(filepath.Join(root, "io.ail"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", root)

	source, err := findStdlibSource()
	if err != nil {
		t.Fatal(err)
	}
	if source.Label != root {
		t.Fatalf("source = %q, want filesystem override %q", source.Label, root)
	}
}

func TestDocsSourceFailsLoudlyWithoutFilesystemOrEmbed(t *testing.T) {
	original := embeddedStdlibFS
	embeddedStdlibFS = nil
	t.Cleanup(func() { embeddedStdlibFS = original })
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "missing"))
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("APPDATA", filepath.Join(root, "appdata"))

	_, err := findStdlibSource()
	if err == nil || !strings.Contains(err.Error(), "searched:") || !strings.Contains(err.Error(), "AILANG_STDLIB_PATH") {
		t.Fatalf("expected loud traced error, got %v", err)
	}
}
