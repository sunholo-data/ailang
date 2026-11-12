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
// `go test`, and compares every result with the evaluator's — for every pure
// pair whose old name compiles to Go.
func TestStdlibRenamesGoCodegenMatchesEvaluator(t *testing.T) {
	bin := buildAilang(t)
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	probe := filepath.Join(dir, "renameprobe.ail")
	code := `module renameprobe
import std/list (member, contains)
import std/bytes (fromString, toString, slice, sliceLen)
import std/math (abs_Float, absFloat, abs_Int, absInt)
import std/array (fromList, get, unsafeGet)
import std/option (Option, Some, None)
pure func showOpt(o: Option[bytes]) -> string = match o { Some(b) => toString(b), None => "none" }
export pure func summary() -> string = "${show(member(2, [1, 2, 3]))},${show(member(5, [1, 2, 3]))},${showOpt(slice(fromString("hello"), 1, 3))},${showOpt(sliceLen(fromString("hello"), 1, 3))},${show(abs_Float(0.0 - 2.5))},${show(absFloat(0.0 - 2.5))},${show(contains([1, 2, 3], 2))},${show(contains([1, 2, 3], 5))},${show(abs_Int(0 - 4))},${show(absInt(0 - 4))},${show(unsafeGet(fromList([7, 8, 9]), 2))},${show(get(fromList([7, 8, 9]), 2))}"
`
	if err := os.WriteFile(probe, []byte(code), 0o600); err != nil {
		t.Fatal(err)
	}
	want, stderr, rc := runAilangBin(t, bin, "run", "--stdlib-path", filepath.Join(root, "std"), "--entry", "summary", probe)
	if rc != 0 {
		t.Fatalf("evaluator: exit %d\n%s", rc, stderr)
	}
	want = strings.TrimSpace(want)
	if want != "true,false,ell,ell,2.5,2.5,true,false,4,4,9,9" {
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

// goEmitPair is one rename pair as the Go emitter sees it: a one-function
// module per spelling.
type goEmitPair struct {
	module, old, new, ret, body, imports string
}

var goEmitPairs = []goEmitPair{
	{"std/list", "head_or", "headOr", "string", "show({F}([4, 5], 0))", ""},
	{"std/list", "last_or", "lastOr", "string", "show({F}([4, 5], 0))", ""},
	{"std/list", "nth_or", "nthOr", "string", "show({F}([4, 5], 1, 0))", ""},
	{"std/string", "charAt_or", "charAtOr", "string", `{F}("hey", 1, "?")`, ""},
	{"std/math", "abs_Int", "absInt", "string", "show({F}(0 - 3))", ""},
	{"std/math", "abs_Float", "absFloat", "string", "show({F}(0.0 - 2.5))", ""},
	{"std/rand", "rand_int", "randInt", "int ! {Rand}", "{F}(1, 6)", ""},
	{"std/rand", "rand_float", "randFloat", "float ! {Rand}", "{F}(0.0, 1.0)", ""},
	{"std/rand", "rand_bool", "randBool", "bool ! {Rand}", "{F}()", ""},
	{"std/rand", "rand_seed", "randSeed", "() ! {Rand}", "{F}(7)", ""},
	{"std/fs", "readFileBytes", "readFileBase64", "Result[string, string] ! {FS}", `{F}("x")`, "import std/result (Result)\n"},
	{"std/fs", "glob", "walkWithSuffix", "[string] ! {FS}", `{F}(".", ".ail")`, ""},
	{"std/io", "printErr", "eprint", "() ! {IO}", `{F}("x")`, ""},
	{"std/bytes", "slice", "sliceLen", "string", `match {F}(fromString("hello"), 1, 3) { Some(b) => toString(b), None => "none" }`, "import std/bytes (fromString, toString)\nimport std/option (Option, Some, None)\n"},
	{"std/list", "member", "contains", "string", "show({F}({A}))", ""},
	{"std/array", "unsafeGet", "get", "string", "show({F}(fromList([1, 2, 3]), 2))", "import std/array (fromList)\n"},
}

// goEmitBothFail are the pairs whose OLD name never compiled to Go either
// (pre-existing on dev: the emitter does not compile pure-AILANG std bodies
// such as list.nthOr, nor Rand/IO/FS ops without a spec). Not regressions;
// listed in docs/LIMITATIONS.md. A pair leaving this list must be removed here.
var goEmitBothFail = map[string]bool{
	"head_or": true, "last_or": true, "nth_or": true, "charAt_or": true,
	"rand_bool": true, "rand_seed": true, "glob": true, "printErr": true,
}

// Every new name compiles through the Go emitter wherever its old name does
// (a warning must never steer a Go-codegen user from a working name to a broken
// one), and for the drop-in pairs the emitted module is the same code. Runtime
// equality with the evaluator for the pure pairs is
// TestStdlibRenamesGoCodegenMatchesEvaluator.
func TestStdlibRenamePairsGoEmitter(t *testing.T) {
	bin := buildAilang(t)
	dir := t.TempDir()
	emit := func(name string, p goEmitPair) (string, bool) {
		mod := "p_" + strings.ToLower(name)
		body := strings.ReplaceAll(p.body, "{F}", name)
		switch name {
		case "member":
			body = strings.ReplaceAll(body, "{A}", "2, [1, 2, 3]")
		case "contains":
			body = strings.ReplaceAll(body, "{A}", "[1, 2, 3], 2")
		}
		pure := "pure "
		if strings.Contains(p.ret, "!") {
			pure = ""
		}
		src := "module " + mod + "\nimport " + p.module + " (" + name + ")\n" + p.imports +
			"export " + pure + "func probe() -> " + p.ret + " = " + body + "\n"
		file := filepath.Join(dir, mod+".ail")
		if err := os.WriteFile(file, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
		out := filepath.Join(dir, "out_"+mod)
		_, _, rc := runAilangBin(t, bin, "compile", "--emit-go", "--package-name", mod, "--out", out, file)
		if rc != 0 {
			return "", false
		}
		gen, err := os.ReadFile(filepath.Join(out, mod, mod+".go"))
		if err != nil {
			t.Fatalf("%s: compiled but no module file: %v", name, err)
		}
		return strings.ReplaceAll(string(gen), mod, "MOD"), true
	}
	for _, p := range goEmitPairs {
		oldGo, oldOK := emit(p.old, p)
		newGo, newOK := emit(p.new, p)
		switch {
		case oldOK && !newOK:
			t.Errorf("%s: %s compiles to Go but its replacement %s does not", p.module, p.old, p.new)
		case !oldOK && goEmitBothFail[p.old]:
			// pre-existing; documented
		case !oldOK:
			t.Errorf("%s.%s no longer compiles to Go (it is not in goEmitBothFail)", p.module, p.old)
		case goEmitBothFail[p.old]:
			t.Errorf("%s.%s compiles to Go now: remove it from goEmitBothFail and docs/LIMITATIONS.md", p.module, p.old)
		case p.old != "member" && p.old != "unsafeGet" && oldGo != newGo:
			t.Errorf("%s: %s and %s emit different Go:\n%s\n---\n%s", p.module, p.old, p.new, oldGo, newGo)
		}
	}
}
