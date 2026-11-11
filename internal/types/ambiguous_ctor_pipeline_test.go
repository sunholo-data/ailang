package types_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/types"
)

// checkModules writes files under a temp root and checks m/main.ail.
func checkModules(t *testing.T, files map[string]string) error {
	t.Helper()
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(repo, "std"))
	root := t.TempDir()
	t.Chdir(root)
	for rel, src := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err = pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, NoCache: true}, pipeline.Source{Filename: "m/main.ail"})
	return err
}

// Two modules both offering constructor C (auto-imported, never by name).
const (
	ctorA = "module m/a\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n"
	ctorB = "module m/b\nexport type U = C(int) | E\nexport type UA = U\nexport pure func mkb() -> U = C(41)\n"
)

func withMain(body string) map[string]string {
	return map[string]string{"m/a.ail": ctorA, "m/b.ail": ctorB,
		"m/main.ail": "module m/main\nimport m/a (mka)\nimport m/b (mkb, UA)\n" + body}
}

// The deferred constructor choice (types/ambiguous_ctor.go) seen through the
// pipeline, so that the choice, its resolution and its errors are exercised in
// this package.
func TestAmbiguousCtorChoice_Pipeline(t *testing.T) {
	for _, c := range []struct {
		name, body, want string
	}{
		{"decided by a later use of a let-bound lambda",
			"export pure func run() -> int {\n  let f = \\t. match t { C(n) => n, _ => 0 };\n  f(mkb())\n}\n", ""},
		{"decided by the scrutinee's annotation through an alias",
			"pure func g(x: UA) -> int = match x { C(n) => n + 1, _ => 0 }\nexport pure func run() -> int = g(mkb())\n", ""},
		{"a use in ANOTHER top-level declaration does not decide it",
			"pure func g(x) = match x { C(n) => n, _ => 0 }\nexport pure func run() -> int = g(mka())\n", "is ambiguous"},
		{"the pattern's arity does not fit the type matched",
			"export pure func run() -> int = match mka() { C(a, b) => a, _ => 0 }\n", "has type T (defined in m/a), whose C takes 1"},
		{"the pattern's arity matches no candidate",
			"export pure func run() -> int {\n  let f = \\t. match t { C(a, b) => a, _ => 0 };\n  0\n}\n", "no imported constructor C takes 2"},
		{"the scrutinee's type is none of the candidates",
			"export pure func run() -> int = match \"s\" { C(n) => n, _ => 0 }\n", "the value matched has type"},
		{"nothing in the declaration decides it (reported at the pattern)",
			"export pure func run() -> int {\n  let f = \\t. match t { C(n) => n, _ => 0 };\n  0\n}\n", "at m/main.ail:5:"},
		{"a constructor expression has no annotation hint",
			"export pure func run() -> int { let v: UA = C(5); 0 }\n", "import m/a (C)"},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkModules(t, withMain(c.body))
			switch {
			case c.want == "" && err != nil:
				t.Fatalf("must check: %v", err)
			case c.want != "" && err == nil:
				t.Fatalf("must be %s (%s)", types.TC_AMBIGUOUS_CTOR_001, c.want)
			case c.want != "" && (!strings.Contains(err.Error(), types.TC_AMBIGUOUS_CTOR_001) || !strings.Contains(err.Error(), c.want)):
				t.Fatalf("want %s with %q, got %v", types.TC_AMBIGUOUS_CTOR_001, c.want, err)
			case strings.Contains(c.name, "expression") && strings.Contains(err.Error(), "annotate"):
				t.Fatalf("an expression is decided by its imports; an annotation does not help: %v", err)
			}
		})
	}
}

// The measurement instrument sees the pattern and its resolution.
func TestAmbiguousCtorChoice_InstrumentSeesTheResolution(t *testing.T) {
	var kinds []string
	restore := types.SetAmbiguityInstrument(func(e types.AmbiguityEvent) { kinds = append(kinds, e.Kind) }, true)
	defer restore()
	if err := checkModules(t, withMain("export pure func run() -> int = match mkb() { C(n) => n, _ => 0 }\n")); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(kinds, ",")
	for _, want := range []string{"ctor-pattern", "ctor-pattern-resolved"} {
		if !strings.Contains(got, want) {
			t.Errorf("instrument missed %s: %s", want, got)
		}
	}
}

// Review round 3 (mp1, mp2, c_poly, ca2b): a unification failure an ambiguous
// constructor is behind is reported as TC_AMBIGUOUS_CTOR_001 at the pattern,
// naming the cause, with the failure kept.
func TestAmbiguousCtorChoice_FailureNamesItsCause(t *testing.T) {
	boxA := "module m/a\nexport type Box[a] = C(a) | D\nexport pure func boxi() -> Box[int] = C(1)\nexport pure func boxs() -> Box[string] = C(\"s\")\n"
	boxB := "module m/b\nexport type Bx[a] = C(a) | E\nexport pure func bxi() -> Bx[int] = C(2)\n"
	two := "module m/b\nexport type U = C(int, int) | E\nexport pure func mkb() -> U = C(40, 2)\n"
	for _, c := range []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{"mp1: a monomorphic lambda used at Box[int] and Box[string]", map[string]string{"m/a.ail": boxA, "m/b.ail": boxB,
			"m/main.ail": "module m/main\nimport m/a (boxi, boxs)\nimport m/b (bxi)\nexport pure func run() -> int { let f = \\x. match x { C(_) => 1, _ => 0 }; f(boxi()) + f(boxs()) }\n"},
			[]string{"at m/main.ail:4:", "where `f` is defined", "cannot be generic", "Caused:", "int vs string", "import m/a (C)"}},
		{"c_poly: a lambda used at both candidates' types", withMain("export pure func run() -> int { let f = \\x. match x { C(n) => n, _ => 0 }; f(mka()) + f(mkb()) }\n"),
			[]string{"at m/main.ail:", "where `f` is defined", "Caused:"}},
		{"ca2b: a pattern read by its arity, used at the other type", map[string]string{"m/a.ail": ctorA, "m/b.ail": two,
			"m/main.ail": "module m/main\nimport m/a (mka)\nimport m/b (mkb)\nexport pure func run() -> int { let f = \\x. match x { C(n) => n, _ => 0 }; f(mkb()) }\n"},
			[]string{"at m/main.ail:4:", "has 1 argument(s)", "whose C takes 2", "Caused:", "C(_, _)"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := checkModules(t, c.files)
			if err == nil {
				t.Fatal("must be rejected")
			}
			for _, w := range append([]string{types.TC_AMBIGUOUS_CTOR_001}, c.want...) {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("missing %q in %v", w, err)
				}
			}
		})
	}
}
