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
	chosen  string   // accepted: "C@B[,K@B…]" — each constructor must resolve into that module
}

const (
	mpA   = "module {A}\nexport type Box[a] = C(a) | D\nexport pure func boxi() -> Box[int] = C(1)\nexport pure func boxs() -> Box[string] = C(\"s\")\n"
	mpB   = "module {B}\nexport type Bx[a] = C(a) | E\nexport pure func bxi() -> Bx[int] = C(2)\n"
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
		// Round 3, item 4: the boundary of indirect names (type_scope.go step 6).
		{name: "s67: an indirect module's T never re-points the T an import exports", deps: map[string]string{
			"A": "module {A}\nimport {Z} (zz)\nexport type T = Ak(int)\nexport pure func mk() -> T = Ak(5)\n",
			"Z": "module {Z}\nexport type T = Zk(string)\nexport pure func zz() -> int = 0\n"},
			imports: []string{"import {A} (mk)"},
			bodies:  []string{"pure func get(t: T) -> int = match t { Ak(n) => n }\nexport pure func run() -> int = get(mk())\n"},
			chosen:  "Ak@A"},
		{name: "importing T by name always wins, over imports and indirect modules", deps: map[string]string{
			"A": "module {A}\nimport {Z} (zz)\nexport type T = Ak(int)\nexport pure func mk() -> T = Ak(5)\n",
			"B": "module {B}\nexport type T = Bk(int)\nexport pure func b() -> int = 0\n",
			"Z": "module {Z}\nexport type T = Zk(string)\nexport pure func zz() -> int = 0\n"},
			imports: []string{"import {A} (mk, T)", "import {B} (b)"},
			bodies:  []string{"pure func get(t: T) -> int = match t { Ak(n) => n }\nexport pure func run() -> int = get(mk())\n"},
			chosen:  "Ak@A"},
		{name: "a name only indirect modules declare, two of them", deps: map[string]string{
			"A": "module {A}\nimport {Y} (yy)\nexport pure func a() -> int = yy()\n",
			"B": "module {B}\nimport {Z} (zz)\nexport pure func b() -> int = zz()\n",
			"Y": "module {Y}\nexport type T = Yk(int)\nexport pure func yy() -> int = 0\n",
			"Z": "module {Z}\nexport type T = Zk(string)\nexport pure func zz() -> int = 0\n"},
			imports: []string{"import {A} (a)", "import {B} (b)"},
			bodies:  []string{"pure func id(t: T) -> T = t\nexport pure func run() -> int = 0\n"},
			want:    types.TC_AMBIGUOUS_TYPE_001},
		{name: "al2 shape: the imported module's alias beats an indirect one (as on dev)", deps: map[string]string{
			"A": "module {A}\nexport type Num = int\nexport pure func base() -> int = 1\n",
			"B": "module {B}\nexport type Num = string\nexport pure func unrelated() -> int = 0\n",
			"Z": "module {Z}\nimport {A} (base)\nexport pure func cb() -> int = base()\n"},
			imports: []string{"import {Z} (cb)", "import {B} (unrelated)"},
			bodies:  []string{"export pure func run() -> int { let v: Num = \"s\"; 0 }\n"}},
		// Round 3, item 1: a let-bound lambda matching an ambiguous constructor
		// is not generalised while the choice is open; its uses at two types are
		// reported as TC_AMBIGUOUS_CTOR_001 (the cause), and importing the
		// constructor by name makes it generic again.
		{name: "mp1/mp2: monomorphic while the constructor is undecided", deps: map[string]string{"A": mpA, "B": mpB},
			imports: []string{"import {A} (boxi, boxs)", "import {B} (bxi)"},
			bodies: []string{
				"export pure func run() -> int { let f = \\x. match x { C(_) => 1, _ => 0 }; f(boxi()) + f(boxs()) }\n",
				"export pure func run() -> int { let f = \\x. match x { C(_) => 1, _ => 0 }; f(boxs()) + f(boxi()) }\n",
				"pure func g(x: Box[int], y: Box[string]) -> int { let f = \\z. match z { C(_) => 1, _ => 0 }; f(x) + f(y) }\nexport pure func run() -> int = g(boxi(), boxs())\n",
			},
			want: types.TC_AMBIGUOUS_CTOR_001},
		{name: "mp1ctl: importing the constructor by name keeps the lambda generic", deps: map[string]string{"A": mpA, "B": mpB},
			imports: []string{"import {A} (boxi, boxs, C)", "import {B} (bxi)"},
			bodies:  []string{"export pure func run() -> int { let f = \\x. match x { C(_) => 1, _ => 0 }; f(boxi()) + f(boxs()) }\n"},
			chosen:  "C@A"},
		{name: "c_poly: one lambda used at both candidates' types", deps: ab,
			imports: []string{"import {A} (mka)", "import {B} (mkb)"},
			bodies:  []string{"export pure func run() -> int { let f = \\x. match x { C(n) => n, _ => 0 }; f(mka()) + f(mkb()) }\n"},
			want:    types.TC_AMBIGUOUS_CTOR_001},
		// Round 3, item 2 (mutant G8): nested ambiguous constructors are decided
		// one after another — the fixed point must run to completion.
		{name: "g8d: four nested ambiguous constructors", deps: map[string]string{
			"A": "module {A}\nexport type T = C(int) | D\nexport type M = N(T) | M0\nexport type W = P(M) | Q\nexport type Z = K(W) | Z0\nexport pure func mka() -> T = C(1)\n",
			"B": "module {B}\nexport type U = C(int) | E\nexport type MB = N(U) | M1\nexport type V = P(MB) | R\nexport type Y = K(V) | Y0\nexport pure func mky() -> Y = K(P(N(C(40))))\n"},
			imports: []string{"import {A} (mka)", "import {B} (mky)"},
			bodies:  []string{"export pure func run() -> int = (\\w. match w { K(P(N(C(n)))) => n + 2, _ => 0 })(mky())\n"},
			chosen:  "K@B,P@B,N@B,C@B"},
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
		{"A": "q/a", "B": "q/b", "Y": "q/y", "Z": "q/z"},
		{"A": "q/zz", "B": "q/aa", "Y": "q/nn", "Z": "q/mm"}, // flips the sort order of A and B, and of Y and Z
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
							for _, one := range strings.Split(c.chosen, ",") {
								if one == "" {
									continue
								}
								ctor := strings.SplitN(one, "@", 2)
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
