package pkg

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestInterfaceHashV2_SensitiveToAddedExport(t *testing.T) {
	a, b := v2HashesForBodies(t, "export func first() -> int = 1\n", "export func first() -> int = 1\nexport func second() -> int = 2\n")
	assertV2HashesDiffer(t, a, b)
}

func TestInterfaceHashV2_SensitiveToRemovedExport(t *testing.T) {
	a, b := v2HashesForBodies(t, "export func first() -> int = 1\nexport func second() -> int = 2\n", "export func first() -> int = 1\n")
	assertV2HashesDiffer(t, a, b)
}

func TestInterfaceHashV2_SensitiveToRetype(t *testing.T) {
	a, b := v2HashesForBodies(t, "export func value(x: int) -> int = x\n", "export func value(x: int, y: int) -> int = x + y\n")
	assertV2HashesDiffer(t, a, b)
}

func TestInterfaceHashV2_Deterministic(t *testing.T) {
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", "export func value() -> int = 1\n"}}, nil)
	want, wantSigs := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	for i := 0; i < 10; i++ {
		got, gotSigs := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
		if got != want || !reflect.DeepEqual(gotSigs, wantSigs) {
			t.Fatalf("iteration %d: got (%q, %#v), want (%q, %#v)", i, got, gotSigs, want, wantSigs)
		}
	}
}

func TestInterfaceHashV2_SignatureSetSortedAndDeduplicated(t *testing.T) {
	// SIX exports, not two, and an exact-sequence assertion rather than a bare
	// sortedness check: this pins de-duplication, per-signature content AND order in
	// one arm. It does NOT pin the sort itself — see the declared residual in
	// hasher_v2.go. Widening the fixture from two to six was measured at iteration 314
	// as taking the sort-removal kill from ~1-in-15 to 4-in-8, i.e. still a coin flip,
	// which refuted the "1/6! per run" reasoning that motivated the widening; the real
	// fix was to make collection deterministic instead of enlarging the sample.
	modules := []v2Module{{"test/pkg/main", "export func zed() -> int = 1\nexport func alpha() -> int = 2\n" +
		"export func mid() -> int = 3\nexport func beta() -> int = 4\n" +
		"export func yak() -> int = 5\nexport func delta() -> int = 6\n"}}
	dir, manifest := writeV2Package(t, modules, nil)
	manifest.Exports.Modules = append(manifest.Exports.Modules, manifest.Exports.Modules[0])
	_, signatures := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	if len(signatures) != 6 {
		t.Fatalf("signature set = %#v, want six de-duplicated signatures", signatures)
	}
	if !sort.StringsAreSorted(signatures) {
		t.Fatalf("signature set is not sorted: %#v", signatures)
	}
	for i, want := range []string{"alpha", "beta", "delta", "mid", "yak", "zed"} {
		if !strings.Contains(signatures[i], ":func:"+want+":") {
			t.Fatalf("signature set[%d] = %q, want the %q export", i, signatures[i], want)
		}
	}
}

func TestInterfaceHashV2_OrderIndependent(t *testing.T) {
	mods := []v2Module{
		{"test/pkg/a", "export func a() -> int = 1\n"},
		{"test/pkg/b", "export func b() -> int = 2\n"},
	}
	dir, first := writeV2Package(t, mods, []string{"IO", "FS"})
	second := *first
	second.Exports.Modules = []string{"test/pkg/b", "test/pkg/a"}
	second.Effects.Max = []string{"FS", "IO"}
	a, _ := mustV2Hash(t, dir, first, DefaultPublishLimits())
	b, _ := mustV2Hash(t, dir, &second, DefaultPublishLimits())
	if a != b {
		t.Fatalf("reordering exports/effects changed hash: %q != %q", a, b)
	}
}

func TestInterfaceHashVersion(t *testing.T) {
	valid := "sha256:ifacev2:" + strings.Repeat("a", 64)
	for input, want := range map[string]int{
		valid:                               2,
		"sha256:" + strings.Repeat("a", 64): 0,
		"sha256:ifacev2:" + strings.Repeat("g", 64): 0,
		"garbage": 0,
	} {
		if got := InterfaceHashVersion(input); got != want {
			t.Errorf("InterfaceHashVersion(%q) = %d, want %d", input, got, want)
		}
	}
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", "export func value() -> int = 1\n"}}, nil)
	hash, _ := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	if got := InterfaceHashVersion(hash); got != 2 {
		t.Fatalf("InterfaceHashVersion(InterfaceHashV2(...)) = %d, want 2 (hash %q)", got, hash)
	}
}

func TestInterfaceHashV2_IgnoresTypeAliases(t *testing.T) {
	first := "export type Value = int\nexport func value() -> int = 1\n"
	second := "export type Value = string\nexport func value() -> int = 1\n"
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", first}}, nil)
	j, err := BuildModuleIface(context.Background(), dir, "test/pkg/main", DefaultPublishLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(j.Types) == 0 || j.Types[0].Alias == "" {
		t.Fatalf("instrument failure: fixture did not produce a type alias: %#v", j.Types)
	}
	a, _ := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	writeFile(t, filepath.Join(dir, "test", "pkg", "main.ail"), "module test/pkg/main\n"+second)
	b, _ := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	if a != b {
		t.Fatalf("changing only an alias target changed hash: %q != %q", a, b)
	}
}

func TestInterfaceHashV2_RefusesOnBrokenModule(t *testing.T) {
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", "export func broken() -> int = \"wrong\"\n"}}, nil)
	if hash, _, err := InterfaceHashV2(context.Background(), dir, manifest, DefaultPublishLimits()); err == nil {
		t.Fatalf("InterfaceHashV2 returned hash %q for broken exported module", hash)
	}
}

func TestInterfaceHashV2_SensitiveToEffectCeiling(t *testing.T) {
	dir, first := writeV2Package(t, []v2Module{{"test/pkg/main", "export func value() -> int = 1\n"}}, []string{"IO"})
	second := *first
	second.Effects.Max = []string{"IO", "FS"}
	a, _ := mustV2Hash(t, dir, first, DefaultPublishLimits())
	b, _ := mustV2Hash(t, dir, &second, DefaultPublishLimits())
	assertV2HashesDiffer(t, a, b)
}

func TestInterfaceHashV2_EnforcesExportLimit(t *testing.T) {
	mods := []v2Module{
		{"test/pkg/a", "export func a() -> int = 1\n"},
		{"test/pkg/b", "export func b() -> int = 2\n"},
	}
	dir, manifest := writeV2Package(t, mods, nil)
	lim := DefaultPublishLimits()
	lim.MaxExportedModules = 2
	if _, _, err := InterfaceHashV2(context.Background(), dir, manifest, lim); err != nil {
		t.Fatalf("exactly-at-limit package refused: %v", err)
	}
	lim.MaxExportedModules = 1
	if _, _, err := InterfaceHashV2(context.Background(), dir, manifest, lim); err == nil {
		t.Fatal("over-limit package succeeded")
	}

	// The arms above are satisfied by BuildModuleIface's own identical check
	// (iface_subprocess.go), which re-loads the manifest from DISK — so neutering
	// InterfaceHashV2's check leaves them green (measured, iteration 314). This arm
	// discriminates the two: InterfaceHashV2 must enforce the limit against the
	// manifest it was HANDED, which is the caller's view and the one that gets
	// hashed. Here the in-memory manifest lists two exports while the on-disk one
	// lists one, so only InterfaceHashV2's own check can refuse.
	single, inMemory := writeV2Package(t, []v2Module{{"test/pkg/main", "export func a() -> int = 1\n"}}, nil)
	inMemory.Exports.Modules = append(inMemory.Exports.Modules, inMemory.Exports.Modules[0])
	onDisk, err := LoadManifest(single)
	if err != nil {
		t.Fatal(err)
	}
	if len(onDisk.Exports.Modules) != 1 {
		t.Fatalf("instrument failure: on-disk manifest lists %d exports, want 1", len(onDisk.Exports.Modules))
	}
	lim.MaxExportedModules = 1
	_, _, err = InterfaceHashV2(context.Background(), single, inMemory, lim)
	if err == nil {
		t.Fatal("in-memory over-limit manifest was accepted; the caller-supplied export list is not checked")
	}
	if !strings.Contains(err.Error(), "exceeding limit of 1") {
		t.Fatalf("error does not come from the export-limit check, got: %v", err)
	}
}

// TestInterfaceHashV2_SensitiveToPackageIdentity pins the legacy identity fields the
// v2 fold retains so a v2 hash stays a strict superset of what InterfaceHash folds.
// Measured at iteration 314: dropping any of the name:, edition: or ailang: writes
// reddened NOTHING in the suite as delivered. The export: write is deliberately not
// pinned here and cannot be — the module path is already inside each module's folded
// projection, so export: is redundant-by-construction parity with the legacy encoding
// rather than a discriminating input. That is a declared residual, not an oversight.
func TestInterfaceHashV2_SensitiveToPackageIdentity(t *testing.T) {
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", "export func a() -> int = 1\n"}}, nil)
	lim := DefaultPublishLimits()
	base, _ := mustV2Hash(t, dir, manifest, lim)

	for _, tc := range []struct {
		field  string
		mutate func(m *PackageManifest)
	}{
		{"name", func(m *PackageManifest) { m.Package.Name = m.Package.Name + "-renamed" }},
		{"edition", func(m *PackageManifest) { m.Package.Edition = m.Package.Edition + "-next" }},
		{"ailang", func(m *PackageManifest) { m.Package.AILANG = ">=0.35.0" }},
	} {
		altered := *manifest
		altered.Package = manifest.Package
		tc.mutate(&altered)
		got, _ := mustV2Hash(t, dir, &altered, lim)
		if got == base {
			t.Fatalf("changing package %s did not change the v2 hash (%q)", tc.field, base)
		}
	}
}

type v2Module struct{ path, body string }

func writeV2Package(t *testing.T, modules []v2Module, effects []string) (string, *PackageManifest) {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, len(modules))
	quoted := make([]string, 0, len(modules))
	for _, module := range modules {
		paths = append(paths, module.path)
		quoted = append(quoted, fmt.Sprintf("%q", module.path))
		writeFile(t, filepath.Join(dir, filepath.FromSlash(module.path)+".ail"), "module "+module.path+"\n"+module.body)
	}
	quotedEffects := make([]string, 0, len(effects))
	for _, effect := range effects {
		quotedEffects = append(quotedEffects, fmt.Sprintf("%q", effect))
	}
	manifestText := manifestFor(strings.Join(quoted, ", "))
	if len(effects) > 0 {
		manifestText += "\n[effects]\nmax = [" + strings.Join(quotedEffects, ", ") + "]\n"
	}
	writeFile(t, filepath.Join(dir, "ailang.toml"), manifestText)
	manifest, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(manifest.Exports.Modules, paths) {
		t.Fatalf("manifest exports = %#v, want %#v", manifest.Exports.Modules, paths)
	}
	return dir, manifest
}

func v2HashesForBodies(t *testing.T, first, second string) (string, string) {
	t.Helper()
	dir, manifest := writeV2Package(t, []v2Module{{"test/pkg/main", first}}, nil)
	a, _ := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	writeFile(t, filepath.Join(dir, "test", "pkg", "main.ail"), "module test/pkg/main\n"+second)
	b, _ := mustV2Hash(t, dir, manifest, DefaultPublishLimits())
	return a, b
}

func mustV2Hash(t *testing.T, dir string, manifest *PackageManifest, lim PublishLimits) (string, []string) {
	t.Helper()
	hash, signatures, err := InterfaceHashV2(context.Background(), dir, manifest, lim)
	if err != nil {
		t.Fatalf("InterfaceHashV2: %v", err)
	}
	return hash, signatures
}

func assertV2HashesDiffer(t *testing.T, a, b string) {
	t.Helper()
	if a == b {
		t.Fatalf("hashes unexpectedly equal: %q", a)
	}
}
