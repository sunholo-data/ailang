package pipeline

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"unicode"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-NOMINAL-TYPE-IDENTITY, review round 1: a name's meaning must not depend
// on import order, file order or module path, and must never be a silent
// pick among several declarations.

const (
	ordA = "module q/a\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n"
	ordB = "module q/b\nexport type T = C(int) | E\nexport pure func mkb() -> T = C(41)\n"
)

type orderCase struct {
	name    string
	deps    map[string]string
	imports []string // permuted
	body    string   // after the imports
	want    string   // "" = must check; else the code the error must carry
}

func orderCases() []orderCase {
	ab := map[string]string{"q/a.ail": ordA, "q/b.ail": ordB}
	return []orderCase{
		{"pattern constructor resolved by the scrutinee", ab,
			[]string{"import q/a (mka)", "import q/b (mkb)"},
			"export pure func run() -> int = match mkb() { C(n) => n + 1, _ => 0 }\n", ""},
		{"pattern constructor with an unknown scrutinee", ab,
			[]string{"import q/a (mka)", "import q/b (mkb)"},
			"export pure func run() -> int {\n  let f = \\t. match t { C(n) => n, _ => 0 };\n  f(mkb())\n}\n", types.TC_AMBIGUOUS_CTOR_001},
		{"constructor expression", ab,
			[]string{"import q/a (mka)", "import q/b (mkb)"},
			"export pure func run() -> int = match C(3) { C(n) => n, _ => 0 }\n", types.TC_AMBIGUOUS_CTOR_001},
		{"explicit constructor import decides", ab,
			[]string{"import q/a (mka)", "import q/b (mkb, C)"},
			"export pure func run() -> int = match C(3) { C(n) => n, _ => 0 }\n", ""},
		{"two explicit imports of one type", ab,
			[]string{"import q/a (T, mka)", "import q/b (T)"},
			"export pure func run() -> int = 0\n", "MOD015"},
		{"two explicit imports of one constructor", ab,
			[]string{"import q/a (C)", "import q/b (C)"},
			"export pure func run() -> int = 0\n", "MOD015"},
		{"type exported by two direct dependencies", ab,
			[]string{"import q/a (mka)", "import q/b (mkb)"},
			"pure func id(t: T) -> T = t\nexport pure func run() -> int = 0\n", types.TC_AMBIGUOUS_TYPE_001},
		{"explicit type import beats the direct dependencies", ab,
			[]string{"import q/a (mka)", "import q/b (T, mkb)"},
			"pure func id(t: T) -> T = t\nexport pure func run() -> int = match id(mkb()) { _ => 0 }\n", ""},
		{"type reached only through the import closure (cst6)", map[string]string{
			"q/b0.ail": "module q/b0\nexport type T = Z(int)\nexport pure func zero() -> int = 0\n",
			"q/m.ail":  "module q/m\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\n",
			"q/c.ail":  "module q/c\nimport q/b0 (zero)\nimport q/m (mk)\nexport pure func ga() = mk()\nexport pure func z() -> int = zero()\n",
		},
			[]string{"import q/c (ga, z)"},
			"pure func idT(t: T) -> T = t\nexport pure func run() -> int = match idT(ga()) { _ => 0 }\n", types.TC_AMBIGUOUS_TYPE_001},
	}
}

func permutations(xs []string) [][]string {
	if len(xs) <= 1 {
		return [][]string{append([]string(nil), xs...)}
	}
	var out [][]string
	for i := range xs {
		rest := append(append([]string(nil), xs[:i]...), xs[i+1:]...)
		for _, p := range permutations(rest) {
			out = append(out, append([]string{xs[i]}, p...))
		}
	}
	return out
}

var positionRe = regexp.MustCompile(`\d+:\d+`)

// TestNominalIdentity_ImportOrderIndependent compiles every case under every
// permutation of its imports and requires the same verdict and the same
// diagnostic (positions aside, which move with the lines).
func TestNominalIdentity_ImportOrderIndependent(t *testing.T) {
	for _, c := range orderCases() {
		t.Run(c.name, func(t *testing.T) {
			var first string
			for i, perm := range permutations(c.imports) {
				root := t.TempDir()
				t.Chdir(root)
				files := map[string]string{}
				for k, v := range c.deps {
					files[k] = v
				}
				files["q/main.ail"] = "module q/main\n" + strings.Join(perm, "\n") + "\n" + c.body
				writeNominalFiles(t, root, files)
				_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "q/main.ail"})
				got := "ok"
				if err != nil {
					got = positionRe.ReplaceAllString(err.Error(), "L:C")
				}
				if c.want == "" && err != nil {
					t.Fatalf("order %v: must check: %v", perm, err)
				}
				if c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
					t.Fatalf("order %v: want %s, got %v", perm, c.want, err)
				}
				if i == 0 {
					first = got
				} else if got != first {
					t.Fatalf("import order changed the outcome:\n%v\n  %s\nvs first order\n  %s", perm, got, first)
				}
			}
		})
	}
}

// builtinLikeNames are every built-in, prelude and builtin-home type name that
// a user may declare (uppercase), read from the tables themselves.
func builtinLikeNames(t *testing.T) []string {
	var names []string
	seen := map[string]bool{}
	for _, list := range [][]string{types.ModulelessNames(), types.PreludeTypeNames(), types.BuiltinHomeNames()} {
		for _, n := range list {
			if n == "" || !unicode.IsUpper([]rune(n)[0]) || seen[n] {
				continue
			}
			seen[n] = true
			names = append(names, n)
		}
	}
	if len(names) < 25 {
		t.Fatalf("only %d names read from the tables", len(names))
	}
	return names
}

// TestNominalIdentity_UserTypeNamedLikeABuiltin: a module may declare, or
// import by name, its own type called Map, List, Array, Bytes, Unit, Int,
// Option, Json, DOMEvent, ... and use it (review probes bsmap, bslist,
// bsarray, bsbytes, bsunit, bmaprec — all false rejections in round 1).
func TestNominalIdentity_UserTypeNamedLikeABuiltin(t *testing.T) {
	for _, n := range builtinLikeNames(t) {
		t.Run(n, func(t *testing.T) {
			decl := fmt.Sprintf("export type %s = Mk%s(int)\n", n, n)
			use := fmt.Sprintf("pure func get(t: %s) -> int = match t { Mk%s(k) => k }\npure func mk() -> %s = Mk%s(5)\nexport pure func run() -> int = get(mk())\n", n, n, n, n)
			local := map[string]string{"x/m.ail": "module x/m\n" + decl + use}
			root := t.TempDir()
			t.Chdir(root)
			writeNominalFiles(t, root, local)
			if _, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "x/m.ail"}); err != nil {
				t.Fatalf("local %s: %v", n, err)
			}
			imported := map[string]string{
				"x/d.ail": "module x/d\n" + decl,
				"x/u.ail": fmt.Sprintf("module x/u\nimport x/d (%s, Mk%s)\n", n, n) + use,
			}
			root = t.TempDir()
			t.Chdir(root)
			writeNominalFiles(t, root, imported)
			if _, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "x/u.ail"}); err != nil {
				t.Fatalf("imported %s: %v", n, err)
			}
		})
	}
	// The record shape of probe bmaprec.
	root := t.TempDir()
	t.Chdir(root)
	writeNominalFiles(t, root, map[string]string{"x/r.ail": "module x/r\ntype Map = {w: int, h: int}\npure func area(m: Map) -> int = m.w * m.h\npure func mk() -> Map = {w: 2, h: 3}\nexport pure func run() -> int = area(mk())\n"})
	if _, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "x/r.ail"}); err != nil {
		t.Fatalf("record named Map: %v", err)
	}
}
