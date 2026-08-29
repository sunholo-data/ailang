package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-DX-PI-HARNESS Distribution v2: the managed-file install contract.
// decidePiInstall must never plan a clobber of user-owned content.
func TestDecidePiInstall(t *testing.T) {
	const embeddedContent = "extension content v2"
	const v2 = "v0.35.0"

	tests := []struct {
		name      string
		diskHash  string // "" = absent
		managed   *piManagedFile
		want      string
		suggested bool
	}{
		{name: "absent → install", diskHash: "", managed: nil, want: "install"},
		{name: "identical unmanaged → adopt", diskHash: sha256Hex([]byte(embeddedContent)), managed: nil, want: "adopt"},
		{name: "managed identical, same binary → current",
			diskHash: sha256Hex([]byte(embeddedContent)),
			managed:  &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: v2},
			want:     "current"},
		{name: "managed identical but older binary → update (stamp refresh)",
			diskHash: sha256Hex([]byte(embeddedContent)),
			managed:  &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: "v0.34.0"},
			want:     "update"},
		{name: "managed old asset unchanged on disk → safe content update",
			diskHash: sha256Hex([]byte("extension content v1")),
			managed:  &piManagedFile{SHA256: sha256Hex([]byte("extension content v1")), Version: "v0.34.0"},
			want:     "update"},
		{name: "managed but user-modified → conflict, preserve", //nolint:dupl // table rows are intentionally parallel
			diskHash:  "deadbeef",
			managed:   &piManagedFile{SHA256: sha256Hex([]byte(embeddedContent)), Version: v2},
			want:      "conflict-user-modified",
			suggested: true},
		{name: "unmanaged different content → conflict, preserve", //nolint:dupl // sibling row
			diskHash:  "deadbeef",
			managed:   nil,
			want:      "conflict-unmanaged",
			suggested: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action, suggested := decidePiInstall("x.ts", []byte(embeddedContent), tc.diskHash, tc.managed, v2)
			if action != tc.want {
				t.Errorf("decidePiInstall(%s) action = %q, want %q", tc.name, action, tc.want)
			}
			if suggested != tc.suggested {
				t.Errorf("decidePiInstall(%s) suggested = %v, want %v", tc.name, suggested, tc.suggested)
			}
		})
	}
}

func TestPiFilesystemLifecycle(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := installPiExtensions(home, &stdout, &stderr); err != nil {
		t.Fatalf("first install: %v", err)
	}

	embedded, names, err := piEmbeddedFiles()
	if err != nil {
		t.Fatalf("embedded files: %v", err)
	}
	if got, want := len(names), 9; got != want {
		t.Fatalf("embedded asset count = %d, want %d (8 extensions + README)", got, want)
	}
	var extensionCount int
	for _, name := range names {
		if strings.HasSuffix(name, ".ts") {
			extensionCount++
		}
		got, err := os.ReadFile(filepath.Join(piExtensionsDir(home), name))
		if err != nil {
			t.Fatalf("read installed %s: %v", name, err)
		}
		if !bytes.Equal(got, embedded[name]) {
			t.Errorf("installed %s differs from embedded asset", name)
		}
	}
	if extensionCount != 8 {
		t.Fatalf("installed extension count = %d, want 8", extensionCount)
	}

	manifestBefore := readPiManifestForTest(t, home)
	if got, want := len(manifestBefore.Files), len(names); got != want {
		t.Fatalf("manifest file count = %d, want %d", got, want)
	}
	stdout.Reset()
	stderr.Reset()
	if err := installPiExtensions(home, &stdout, &stderr); err != nil {
		t.Fatalf("idempotent install: %v", err)
	}
	if !strings.Contains(stdout.String(), "current: 9") {
		t.Fatalf("second install did not report all assets current:\n%s", stdout.String())
	}
	manifestAfter := readPiManifestForTest(t, home)
	if !piManifestsEqual(manifestBefore, manifestAfter) {
		t.Fatal("idempotent install changed managed manifest")
	}

	modifiedName := "binary-freshness.ts"
	modifiedPath := filepath.Join(piExtensionsDir(home), modifiedName)
	modified := append(append([]byte{}, embedded[modifiedName]...), []byte("\n// user modification\n")...)
	if err := os.WriteFile(modifiedPath, modified, 0o644); err != nil {
		t.Fatalf("modify managed extension: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := installPiExtensions(home, &stdout, &stderr); err != nil {
		t.Fatalf("conflict install: %v", err)
	}
	gotModified, err := os.ReadFile(modifiedPath)
	if err != nil {
		t.Fatalf("read preserved extension: %v", err)
	}
	if !bytes.Equal(gotModified, modified) {
		t.Fatal("install clobbered a user-modified managed extension")
	}
	gotSuggested, err := os.ReadFile(piSuggestedPath(home, modifiedName))
	if err != nil {
		t.Fatalf("read suggested extension: %v", err)
	}
	if !bytes.Equal(gotSuggested, embedded[modifiedName]) {
		t.Fatal("suggested extension differs from the embedded asset")
	}
	if !strings.Contains(stderr.String(), "preserved") {
		t.Fatalf("conflict warning missing:\n%s", stderr.String())
	}

	foreignPath := filepath.Join(piExtensionsDir(home), "user-owned.ts")
	if err := os.WriteFile(foreignPath, []byte("// user owned\n"), 0o644); err != nil {
		t.Fatalf("write foreign extension: %v", err)
	}
	stdout.Reset()
	if err := uninstallPiExtensions(home, &stdout); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	for _, path := range []string{modifiedPath, foreignPath} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("uninstall removed preserved file %s: %v", filepath.Base(path), err)
		}
	}
	if _, err := os.Stat(piManifestPath(home)); !os.IsNotExist(err) {
		t.Errorf("manifest still exists after uninstall: %v", err)
	}
	for _, name := range names {
		if name == modifiedName {
			continue
		}
		if _, err := os.Stat(filepath.Join(piExtensionsDir(home), name)); !os.IsNotExist(err) {
			t.Errorf("clean managed file %s still exists after uninstall: %v", name, err)
		}
	}
}

func TestPiStatusReportsManagedStates(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := installPiExtensions(home, &stdout, &stderr); err != nil {
		t.Fatalf("install: %v", err)
	}

	driftName := "binary-freshness.ts"
	if err := os.WriteFile(filepath.Join(piExtensionsDir(home), driftName), []byte("modified\n"), 0o644); err != nil {
		t.Fatalf("seed drift: %v", err)
	}
	missingName := "provider-quota.ts"
	if err := os.Remove(filepath.Join(piExtensionsDir(home), missingName)); err != nil {
		t.Fatalf("seed missing: %v", err)
	}
	unmanagedName := "ailang-lsp-lite.ts"
	manifest := readPiManifestForTest(t, home)
	delete(manifest.Files, unmanagedName)
	if err := writePiManaged(home, manifest.Files); err != nil {
		t.Fatalf("seed unmanaged: %v", err)
	}

	stdout.Reset()
	if err := statusPiExtensions(home, &stdout); err != nil {
		t.Fatalf("status: %v", err)
	}
	output := stdout.String()
	for _, want := range []string{
		"FRESH      README.md",
		"DRIFT      " + driftName,
		"MISSING    " + missingName,
		"UNMANAGED  " + unmanagedName,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("status missing %q:\n%s", want, output)
		}
	}
}

func TestPiInstallPreservesUnmanagedConflict(t *testing.T) {
	home := t.TempDir()
	extDir := piExtensionsDir(home)
	if err := os.MkdirAll(extDir, 0o755); err != nil {
		t.Fatalf("mkdir extension dir: %v", err)
	}
	name := "provider-quota.ts"
	userContent := []byte("// independently installed\n")
	if err := os.WriteFile(filepath.Join(extDir, name), userContent, 0o644); err != nil {
		t.Fatalf("write unmanaged conflict: %v", err)
	}

	var stdout, stderr bytes.Buffer
	if err := installPiExtensions(home, &stdout, &stderr); err != nil {
		t.Fatalf("install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(extDir, name))
	if err != nil {
		t.Fatalf("read unmanaged conflict: %v", err)
	}
	if !bytes.Equal(got, userContent) {
		t.Fatal("install clobbered an unmanaged conflicting extension")
	}
	if _, managed := readPiManifestForTest(t, home).Files[name]; managed {
		t.Fatal("unmanaged conflicting extension was recorded as managed")
	}
	embedded, _, err := piEmbeddedFiles()
	if err != nil {
		t.Fatalf("embedded files: %v", err)
	}
	suggested, err := os.ReadFile(piSuggestedPath(home, name))
	if err != nil {
		t.Fatalf("read suggestion: %v", err)
	}
	if !bytes.Equal(suggested, embedded[name]) {
		t.Fatal("unmanaged conflict suggestion differs from embedded asset")
	}
}

func readPiManifestForTest(t *testing.T, home string) piManagedManifest {
	t.Helper()
	data, err := os.ReadFile(piManifestPath(home))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest piManagedManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	return manifest
}

func piManifestsEqual(a, b piManagedManifest) bool {
	left, _ := json.Marshal(a)
	right, _ := json.Marshal(b)
	return bytes.Equal(left, right)
}
