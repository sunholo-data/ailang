package pipeline

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-NOMINAL-TYPE-IDENTITY, review rounds 1–2: whether a program is
// accepted, and which constructor each pattern denotes, must not depend on
// import order, on the order of independent lets, call arguments, match arms
// or declarations, nor on module paths. Every case below is compiled under
// every combination of: each permutation of its imports, each of its
// equivalent body spellings, and two module namings that flip the modules'
// sort order. All runs must give the same verdict and error code, and an
// accepted program must choose the same constructors (mapped back through the
// renaming).

type permCase struct {
	name    string
	deps    map[string]string // "A"/"B" -> source; {A}/{B} expand to the module paths
	imports []string
	bodies  []string // equivalent spellings of main's body
	want    string   // "" = accepted; else the code every run must report
	chosen  string   // accepted: "C@B" — constructor C must resolve into module B
}

const (
	permA = "module {A}\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\nexport pure func apply(f: T -> int, v: T) -> int = f(v)\n"
	permB = "module {B}\nexport type U = C(int) | E\nexport pure func mkb() -> U = C(41)\nexport pure func useU(u: U) -> int = 0\nexport pure func both(x: int, y: int) -> int = x + y\n"
)

func permCases() []permCase {
	ab := map[string]string{"A": permA, "B": permB}
	return []permCase{
		{name: "r1-r4: scrutinee decided by a later or earlier use", deps: ab,
			imports: []string{"import {A} (mka, T)", "import {B} (mkb, U, useU, both)"},
			bodies: []string{
				"export pure func run() -> int {\n  let f = \\y. { let a = useU(y); let r = match y { C(n) => n + 1, _ => 0 }; a + r };\n  f(mkb())\n}\n",
				"export pure func run() -> int {\n  let f = \\y. { let r = match y { C(n) => n + 1, _ => 0 }; let a = useU(y); a + r };\n  f(mkb())\n}\n",
				"export pure func run() -> int {\n  let f = \\y. both(useU(y), match y { C(n) => n + 1, _ => 0 });\n  f(mkb())\n}\n",
				"export pure func run() -> int {\n  let f = \\y. both(match y { C(n) => n + 1, _ => 0 }, useU(y));\n  f(mkb())\n}\n",
			},
			chosen: "C@B"},
		{name: "q2-q6: lambda decided by its use or an annotation", deps: ab,
			imports: []string{"import {A} (mka, T, apply)", "import {B} (mkb, U)"},
			bodies: []string{
				"export pure func run() -> int {\n  let f = \\x. match x { C(n) => n + 1, _ => 0 };\n  f(mka())\n}\n",
				"export pure func run() -> int {\n  let f = func(x: T) -> int { match x { C(n) => n + 1, _ => 0 } };\n  f(mka())\n}\n",
				"export pure func run() -> int = apply(\\x. match x { C(n) => n + 1, _ => 0 }, mka())\n",
				"pure func h(x: T) -> int { let g = \\y. match y { C(n) => n, _ => 0 }; g(x) }\nexport pure func run() -> int = h(mka())\n",
				"export pure func run() -> int = h(mka())\npure func h(x: T) -> int { let g = \\y. match y { C(n) => n, _ => 0 }; g(x) }\n",
			},
			chosen: "C@A"},
		{name: "match-arm order", deps: ab,
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies: []string{
				"export pure func run() -> int = match mkb() { C(n) => n, E => 1 }\n",
				"export pure func run() -> int = match mkb() { E => 1, C(n) => n }\n",
			},
			chosen: "C@B"},
		{name: "ar1/ar2: arity narrows the candidates", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n",
			"B": "module {B}\nexport type U = C | E(int)\nexport pure func mkb() -> U = C\n"},
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int = match mkb() { C => 1, E(n) => n }\n", "export pure func run() -> int = match mkb() { E(n) => n, C => 1 }\n"},
			chosen:  "C@B"},
		{name: "the pattern's arity alone decides", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n",
			"B": "module {B}\nexport type U = C | E(int)\nexport pure func mkb() -> U = C\n"},
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int {\n  let f = \\t. match t { C => 1, _ => 0 };\n  0\n}\n"},
			chosen:  "C@B"},
		{name: "same-named ADTs both offering the constructor", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int) | D\nexport pure func mka() -> T = C(1)\n",
			"B": "module {B}\nexport type T = C(int) | E\nexport pure func mkb() -> T = C(41)\n"},
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int = match mkb() { C(n) => n, E => 1 }\n", "export pure func run() -> int = match mkb() { E => 1, C(n) => n }\n"},
			chosen:  "C@B"},
		{name: "v1: the scrutinee's type is an alias of a candidate", deps: map[string]string{
			"A": permA, "B": "module {B}\nexport type U = C(int) | E\nexport type UA = U\nexport pure func mkb() -> UA = C(41)\n"},
			imports: []string{"import {A} (mka, T)", "import {B} (mkb, U, UA)"},
			bodies: []string{
				"type V = U\npure func g(x: V) -> int = match x { C(n) => n + 1, _ => 0 }\nexport pure func run() -> int = g(mkb())\n",
				"export pure func run() -> int = g(mkb())\npure func g(x: UA) -> int = match x { C(n) => n + 1, _ => 0 }\n",
			},
			chosen: "C@B"},
		{name: "al1/al2: an alias two reachable modules declare", deps: map[string]string{
			"A": "module {A}\nexport type Num = int\nexport pure func base() -> int = 1\n",
			"B": "module {B}\nexport type Num = float\nexport pure func unrelated() -> int = 0\n",
			"Z": "module {Z}\nimport {A} (base)\nexport pure func cb() -> int = base()\n"},
			imports: []string{"import {Z} (cb)", "import {B} (unrelated)"},
			bodies:  []string{"export pure func run() -> int { let v: Num = 7; 0 }\n"},
			want:    types.TC_AMBIGUOUS_TYPE_001},
		{name: "scrutinee type never decided", deps: ab,
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int {\n  let f = \\t. match t { C(n) => n, _ => 0 };\n  0\n}\n"},
			want:    types.TC_AMBIGUOUS_CTOR_001},
		{name: "scrutinee decided as a type with no such constructor", deps: map[string]string{
			"A": permA, "B": permB, "Z": "module {Z}\nexport type Z = Zed | Zod\nexport pure func mkz() -> Z = Zed\n"},
			imports: []string{"import {A} (mka)", "import {B} (mkb)", "import {Z} (mkz)"},
			bodies:  []string{"export pure func run() -> int = match mkz() { C(n) => n, _ => 0 }\n"},
			want:    types.TC_AMBIGUOUS_CTOR_001},
		{name: "constructor expression", deps: ab,
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int = match C(3) { _ => 0 }\n"},
			want:    types.TC_AMBIGUOUS_CTOR_001},
		{name: "two explicit imports of one type", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int)\n", "B": "module {B}\nexport type T = C(string)\n"},
			imports: []string{"import {A} (T)", "import {B} (T)"},
			bodies:  []string{"export pure func run() -> int = 0\n"},
			want:    "MOD015"},
		{name: "type two direct dependencies export", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int)\nexport pure func a() -> int = 0\n", "B": "module {B}\nexport type T = C(string)\nexport pure func b() -> int = 0\n"},
			imports: []string{"import {A} (a)", "import {B} (b)"},
			bodies:  []string{"pure func id(t: T) -> T = t\nexport pure func run() -> int = 0\n", "export pure func run() -> int = 0\npure func id(t: T) -> T = t\n"},
			want:    types.TC_AMBIGUOUS_TYPE_001},
	}
}

// chosenConstructors walks the root Core program and lists every constructor
// pattern as "Name@Module".
func chosenConstructors(prog *core.Program) []string {
	var out []string
	seen := map[uintptr]bool{}
	var walk func(v reflect.Value)
	walk = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Ptr:
			if v.IsNil() || seen[v.Pointer()] {
				return
			}
			seen[v.Pointer()] = true
			if p, ok := v.Interface().(*core.ConstructorPattern); ok {
				out = append(out, p.Name+"@"+p.TypeModule)
			}
			walk(v.Elem())
		case reflect.Interface:
			if !v.IsNil() {
				walk(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).IsExported() {
					walk(v.Field(i))
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				walk(v.Index(i))
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				walk(v.MapIndex(k))
			}
		}
	}
	walk(reflect.ValueOf(prog))
	sort.Strings(out)
	return out
}

func TestNominalIdentity_PermutationInvariant(t *testing.T) {
	namings := []map[string]string{
		{"A": "q/a", "B": "q/b", "Z": "q/z"},
		{"A": "q/zz", "B": "q/aa", "Z": "q/mm"}, // flips the sort order of A and B
	}
	posRe := regexp.MustCompile(`\d+:\d+`)
	for _, c := range permCases() {
		t.Run(c.name, func(t *testing.T) {
			var firstVerdict string
			runs := 0
			for _, naming := range namings {
				expand := func(s string) string {
					for k, v := range naming {
						s = strings.ReplaceAll(s, "{"+k+"}", v)
					}
					return s
				}
				for _, perm := range permutations(c.imports) {
					for bi, body := range c.bodies {
						root := t.TempDir()
						t.Chdir(root)
						files := map[string]string{}
						for k, src := range c.deps {
							files[naming[k]+".ail"] = expand(src)
						}
						files["q/main.ail"] = "module q/main\n" + expand(strings.Join(perm, "\n")) + "\n" + body
						writeNominalFiles(t, root, files)
						res, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "q/main.ail"})
						runs++
						verdict := "ok"
						if err != nil {
							verdict = "error"
							if c.want == "" {
								t.Fatalf("naming %v, imports %v, body %d: must be accepted: %v", naming, perm, bi, err)
							}
							if !strings.Contains(err.Error(), c.want) {
								t.Fatalf("naming %v, imports %v, body %d: want %s, got %v", naming, perm, bi, c.want, posRe.ReplaceAllString(err.Error(), "L:C"))
							}
						} else if c.want != "" {
							t.Fatalf("naming %v, imports %v, body %d: accepted, want %s", naming, perm, bi, c.want)
						} else {
							got := strings.Join(chosenConstructors(res.Artifacts.Core), ",")
							ctor := strings.SplitN(c.chosen, "@", 2)
							want := ctor[0] + "@" + naming[ctor[1]]
							for _, ch := range chosenConstructors(res.Artifacts.Core) {
								if strings.HasPrefix(ch, ctor[0]+"@") && ch != want {
									t.Fatalf("naming %v, imports %v, body %d: %s chose %s (all: %s)", naming, perm, bi, ctor[0], ch, got)
								}
							}
							if !strings.Contains(got, want) {
								t.Fatalf("naming %v, imports %v, body %d: no pattern resolved to %s (all: %s)", naming, perm, bi, want, got)
							}
						}
						if runs == 1 {
							firstVerdict = verdict
						} else if verdict != firstVerdict {
							t.Fatalf("verdict changed: %s vs %s at naming %v, imports %v, body %d", verdict, firstVerdict, naming, perm, bi)
						}
					}
				}
			}
			if runs < 2 {
				t.Fatalf("only %d variant(s): the harness is not permuting", runs)
			}
		})
	}
	_ = fmt.Sprint
}
