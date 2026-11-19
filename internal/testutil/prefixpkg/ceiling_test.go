package prefixpkg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

// In-package tests so the coverage filter keeps these helpers; their real
// consumers are the pipeline and runner M1b tests.

func TestCeilingWritesEveryCase(t *testing.T) {
	app := Ceiling(t)
	for sp := range Spellings {
		for _, c := range append([]string{"pure"}, CeilingCases...) {
			if _, err := os.Stat(filepath.Join(app, c+"_"+sp+".ail")); err != nil {
				t.Errorf("missing %s_%s.ail: %v", c, sp, err)
			}
		}
		src, err := os.ReadFile(filepath.Join(app, "call_"+sp+".ail"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(src), filepath.ToSlash(CeilingMarker(app, sp))) || strings.Contains(string(src), `\`) {
			t.Errorf("call_%s.ail must embed the marker path with forward slashes only", sp)
		}
	}
	lf, err := pkg.LoadLockFile(app)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := lf.FindPackage("evil/dep"); !ok {
		t.Fatal("ceiling fixture lock must list evil/dep")
	}
}

func TestManifestRendersEverySection(t *testing.T) {
	got := Manifest("a/b", "pf", []string{"pf/x", "pf/y"}, map[string]string{"z/z": "../z", "c/c": "../c"}, []string{"IO", "FS"})
	for _, want := range []string{
		"name = \"a/b\"", "module_prefix = \"pf\"", "[exports]\nmodules = [\"pf/x\", \"pf/y\"]",
		"[dependencies]\n\"c/c\" = { path = \"../c\" }\n\"z/z\" = { path = \"../z\" }",
		"[effects]\nmax = [\"IO\", \"FS\"]",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("manifest missing %q:\n%s", want, got)
		}
	}
	bare := Manifest("a/b", "", nil, nil, nil)
	for _, absent := range []string{"module_prefix", "[exports]", "[dependencies]", "[effects]"} {
		if strings.Contains(bare, absent) {
			t.Errorf("bare manifest should not contain %q:\n%s", absent, bare)
		}
	}
	if !strings.Contains(Manifest("a/b", "", nil, nil, []string{}), "[effects]\nmax = []") {
		t.Error("an empty, non-nil max renders as an empty ceiling")
	}
}
