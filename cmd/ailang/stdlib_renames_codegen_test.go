package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/builtins"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// The Go emitter resolves a std call through builtins.StdlibIndex. A rename
// alias must reach its replacement's builtin there, and list.member — whose
// replacement contains takes the arguments the other way round — must keep its
// own builtin: aliasing it to _list_contains would swap its arguments in
// generated Go. This compiles one program through the Go emitter, runs it with
// `go test`, and compares every result with the evaluator's. (std/list
// contains and nthOr are not emittable by the Go backend today — "undefined:
// Contains" — on dev too, so the swap is pinned by TestStdlibRenameAliasIndex.)
func TestStdlibRenamesGoCodegenMatchesEvaluator(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "renameprobe.ail")
	code := `module renameprobe
import std/list (member)
import std/bytes (fromString, toString, slice, sliceLen)
import std/math (abs_Float, absFloat)
import std/option (Option, Some, None)
pure func showOpt(o: Option[bytes]) -> string = match o { Some(b) => toString(b), None => "none" }
export pure func summary() -> string = "${show(member(2, [1, 2, 3]))},${show(member(5, [1, 2, 3]))},${showOpt(slice(fromString("hello"), 1, 3))},${showOpt(sliceLen(fromString("hello"), 1, 3))},${show(abs_Float(0.0 - 2.5))},${show(absFloat(0.0 - 2.5))}"
`
	if err := os.WriteFile(probe, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	want, stderr, rc := runAilangBin(t, bin, "run", "--stdlib-path", filepath.Join(root, "std"), "--entry", "summary", probe)
	if rc != 0 {
		t.Fatalf("evaluator: exit %d\n%s", rc, stderr)
	}
	want = strings.TrimSpace(want)
	if want != "true,false,ell,ell,2.5,2.5" {
		t.Fatalf("evaluator transcript changed: %q", want)
	}

	outdir := filepath.Join(dir, "generated")
	out, stderr, rc := runAilangBin(t, bin, "compile", "--emit-go", "--package-name", "renameprobe", "--out", outdir, probe)
	if rc != 0 {
		t.Fatalf("compile --emit-go: exit %d\n%s\n%s", rc, out, stderr)
	}
	generated := filepath.Join(outdir, "renameprobe")
	if err := os.WriteFile(filepath.Join(generated, "go.mod"), []byte("module renameprobe\n\ngo 1.21\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	test := "package renameprobe\nimport \"testing\"\nfunc TestSummary(t *testing.T) {\n if got := renameprobe__Summary(); got != " +
		quoteGo(want) + " { t.Fatalf(\"generated Go: %v\", got) }\n}\n"
	if err := os.WriteFile(filepath.Join(generated, "summary_test.go"), []byte(test), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr2, rc := testutil.RunBounded(t, generated, 90*time.Second, "go", "test", "-count=1", ".")
	if rc != 0 {
		t.Fatalf("generated Go disagrees with the evaluator or does not build: exit %d\n%s\n%s", rc, stdout, stderr2)
	}
}

func quoteGo(s string) string { return "\"" + strings.ReplaceAll(s, "\"", "\\\"") + "\"" }

// Every drop-in rename resolves to the same builtin as its replacement; the
// one rename that swaps arguments (member) keeps its own builtin, even when no
// spec claims its old name (so the "taken" check alone cannot protect it).
func TestStdlibRenameAliasIndex(t *testing.T) {
	for _, r := range importhint.StdlibRenames {
		got := builtins.StdlibIndex[builtins.StdlibKey{Module: r.Module, Name: r.Old}]
		repl := builtins.StdlibIndex[builtins.StdlibKey{Module: r.Module, Name: r.New}]
		if r.Swapped && got != "" && got == repl {
			t.Errorf("%s.%s resolves to its swapped replacement's builtin %s", r.Module, r.Old, got)
		}
	}
	if got := builtins.StdlibIndex[builtins.StdlibKey{Module: "std/list", Name: "member"}]; got != "_list_member" {
		t.Errorf("std/list.member resolves to %q, want _list_member", got)
	}
	// The alias rule on its own, with member unclaimed.
	idx := map[builtins.StdlibKey]string{
		{Module: "std/list", Name: "contains"}:  "_list_contains",
		{Module: "std/bytes", Name: "sliceLen"}: "_bytes_slice",
	}
	builtins.AddRenameAliases(idx)
	if _, ok := idx[builtins.StdlibKey{Module: "std/list", Name: "member"}]; ok {
		t.Error("an unclaimed member was aliased to contains: generated Go would swap its arguments")
	}
	if idx[builtins.StdlibKey{Module: "std/bytes", Name: "slice"}] != "_bytes_slice" {
		t.Error("the drop-in alias slice -> sliceLen was not added")
	}
}
