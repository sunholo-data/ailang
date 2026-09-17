package pkg

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// M-PKG-QUALITY-LADDER M1: the signature-sensitive interface hash must build
// from the layouts packages are actually published in — flat tarballs whose
// modules import siblings through the package (`import ./types` →
// `pkg/<self>/types`), and module_prefix packages — not only from a canonical
// <pkgdir>/<vendor>/<name>/<module>.ail checkout. Measured 2026-09-17: 0/53
// registry packages built under the canonical-only resolution.

func qualityFixture(t *testing.T, name string) string {
	t.Helper()
	dir, err := filepath.Abs(filepath.Join("testdata", "quality", name))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestInterfaceHashV2_FlatLayoutWithSelfImport(t *testing.T) {
	dir := qualityFixture(t, "flat_self_import")
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	hash, sigs, err := InterfaceHashV2(context.Background(), dir, m, DefaultPublishLimits())
	if err != nil {
		t.Fatalf("InterfaceHashV2 on flat layout: %v", err)
	}
	if !strings.HasPrefix(hash, interfaceHashV2Prefix) {
		t.Fatalf("hash = %q, want %s prefix", hash, interfaceHashV2Prefix)
	}
	joined := strings.Join(sigs, "\n")
	for _, want := range []string{"capAt", "floorPeriods", "asMoney", "mkMoney"} {
		if !strings.Contains(joined, want) {
			t.Errorf("signature set missing %s:\n%s", want, joined)
		}
	}
	// Deterministic across runs (M5 diffs these sets).
	hash2, _, err := InterfaceHashV2(context.Background(), dir, m, DefaultPublishLimits())
	if err != nil || hash2 != hash {
		t.Fatalf("second build hash = %q (err %v), want %q", hash2, err, hash)
	}
}

func TestInterfaceHashV2_ModulePrefixLayout(t *testing.T) {
	dir := qualityFixture(t, "prefixed")
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, sigs, err := InterfaceHashV2(context.Background(), dir, m, DefaultPublishLimits())
	if err != nil {
		t.Fatalf("InterfaceHashV2 on module_prefix layout: %v", err)
	}
	if !strings.Contains(strings.Join(sigs, "\n"), "double") {
		t.Fatalf("signature set missing double: %v", sigs)
	}
}

func TestResolveModuleToFile_Layouts(t *testing.T) {
	flat := qualityFixture(t, "flat_self_import")
	if got := ResolveModuleToFile(flat, "test/flatself", "test/flatself/settle"); got != filepath.Join(flat, "settle.ail") {
		t.Errorf("flat: got %q", got)
	}
	prefixed := qualityFixture(t, "prefixed")
	if got := ResolveModuleToFile(prefixed, "test/prefixed_pkg", "pfx/core"); got != filepath.Join(prefixed, "pfx", "core.ail") {
		t.Errorf("prefixed: got %q", got)
	}
	// Canonical checkout layout (<pkgdir>/<vendor>/<name>/<module>.ail) — the
	// layout the pre-M1 V2 builder assumed — must keep resolving.
	canon := t.TempDir()
	writeFile(t, filepath.Join(canon, "test", "canon", "main.ail"), "module test/canon/main\n")
	if got := ResolveModuleToFile(canon, "test/canon", "test/canon/main"); got != filepath.Join(canon, "test", "canon", "main.ail") {
		t.Errorf("canonical: got %q", got)
	}
}
