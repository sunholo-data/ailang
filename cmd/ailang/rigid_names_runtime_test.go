package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// #2273: each declaration's signature variables are its own. On dev every
// declaration's `a` was one variable, so one declaration's top-level numeric
// defaulting (`add`'s Num a := int) reached another's compiled body: `sh`
// below printed its float argument with intToStr and failed at run time, and
// `ident`'s Go wrapper was emitted at int64.
const rigidNamesSrc = `module rigid_names
export pure func add[a](x: a, y: a) -> a = x + y
export pure func lt[a](x: a, y: a) -> bool = x < y
export pure func sh[a](x: a) -> string = show(x)
export pure func useAll[a](x: a, y: a) -> string = "${sh(add(x, y))} ${show(lt(x, y))}"
export func main() -> () ! {IO} {
  println(useAll(1, 2));
  println(useAll(1.5, 2.5))
}
`

func TestRigidNames_OneDeclarationsDefaultingStaysItsOwn(t *testing.T) {
	bin := buildAilang(t)
	file := filepath.Join(t.TempDir(), "rigid_names.ail")
	if err := os.WriteFile(file, []byte(rigidNamesSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, engine := range [][]string{nil, {"--strict-bytecode"}} {
		args := append([]string{"run", "--quiet"}, engine...)
		args = append(args, "--caps", "IO", "--entry", "main", file)
		out, stderr, code := runAilangBin(t, bin, args...)
		if code != 0 {
			t.Fatalf("%v: exit %d: %s", engine, code, stderr)
		}
		if want := "3 true\n4.0 true"; strings.TrimSpace(out) != want {
			t.Errorf("%v: output %q, want %q", engine, strings.TrimSpace(out), want)
		}
	}
}

func TestRigidNames_GoWrapperOfAGenericExportStaysGeneric(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	file := filepath.Join(dir, "gi.ail")
	src := "module gi\nexport pure func ident[a](x: a) -> a = x\nexport pure func add[a](x: a, y: a) -> a = x + y\n"
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "gen")
	if _, stderr, code := runAilangBin(t, bin, "compile", "--emit-go", "--out", out, file); code != 0 {
		t.Fatalf("compile: exit %d: %s", code, stderr)
	}
	var goSrc strings.Builder
	_ = filepath.Walk(out, func(p string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(p, ".go") {
			b, _ := os.ReadFile(p)
			goSrc.Write(b)
		}
		return nil
	})
	if !strings.Contains(goSrc.String(), "Ident(x interface{}) interface{}") {
		t.Errorf("ident's Go wrapper is not generic:\n%s", goSrc.String())
	}
}
