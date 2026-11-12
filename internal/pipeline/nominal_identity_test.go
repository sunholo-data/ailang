package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// M-V1-NOMINAL-TYPE-IDENTITY M2: a nominal type is its defining module plus
// its name. The rows below are §1 of the design doc (a, a′, a″, a‴) and the
// spike's hard cases; each is checked cold (no cache) and warm (every
// dependency served from the compile cache), because a serialiser that drops
// TCon.Module strips identity on warm runs only.

const (
	nomQA = "module q/a\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\n"
	nomQB = "module q/b\nexport type T = A(string)\n"
)

type nominalCase struct {
	name   string
	files  map[string]string
	entry  string
	prime  string   // a passing module importing the same dependencies (warm run)
	reject []string // substrings the rejection must contain; nil = must check
}

func nominalCases() []nominalCase {
	return []nominalCase{
		{
			name: "a: same-named ADTs from two modules",
			files: map[string]string{
				"q/a.ail": nomQA, "q/b.ail": nomQB,
				"q/main.ail": "module q/main\nimport q/a (mk)\nimport q/b (T, A)\nimport std/string (toUpper)\n" +
					"pure func use(t: T) -> string = match t { A(s) => toUpper(s) }\n" +
					"export func main() -> () ! {IO} = println(use(mk()))\n",
				"q/prime.ail": "module q/prime\nimport q/a (mk)\nimport q/b (T, A)\nimport std/string (toUpper)\nexport pure func main() -> int = 1\n",
			},
			entry: "q/main.ail", prime: "q/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "T (defined in q/a)", "T (defined in q/b)"},
		},
		{
			name: "a-prime: disjoint constructors",
			files: map[string]string{
				"v2/a.ail": "module v2/a\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\n",
				"v2/b.ail": "module v2/b\nexport type T = B(string)\n",
				"v2/main.ail": "module v2/main\nimport v2/a (mk)\nimport v2/b (T, B)\n" +
					"pure func use(t: T) -> string = match t { B(s) => s }\n" +
					"export func main() -> () ! {IO} = println(use(mk()))\n",
				"v2/prime.ail": "module v2/prime\nimport v2/a (mk)\nimport v2/b (T, B)\nexport pure func main() -> int = 1\n",
			},
			entry: "v2/main.ail", prime: "v2/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "v2/a", "v2/b"},
		},
		{
			name: "a-double-prime: a local ADT does not capture an imported one",
			files: map[string]string{
				"v6/a.ail": "module v6/a\nexport type T = A(int)\nexport pure func mk() -> T = A(42)\n",
				"v6/main.ail": "module v6/main\nimport v6/a (mk)\nimport std/string (toUpper)\ntype T = A(string)\n" +
					"pure func use(t: T) -> string = match t { A(s) => toUpper(s) }\n" +
					"export func main() -> () ! {IO} = println(use(mk()))\n",
				"v6/prime.ail": "module v6/prime\nimport v6/a (mk)\nimport std/string (toUpper)\nexport pure func main() -> int = 1\n",
			},
			entry: "v6/main.ail", prime: "v6/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "v6/a", "v6/main"},
		},
		{
			name: "a-triple-prime: structural twins do not cross",
			files: map[string]string{
				"v14/a.ail":     "module v14/a\nexport type Color = Red | Green deriving (Eq)\nexport pure func pick() -> Color = Green\n",
				"v14/b.ail":     "module v14/b\nexport type Color = Red | Green deriving (Eq)\nexport pure func name(c: Color) -> string = match c { Red => \"red\", Green => \"green\" }\n",
				"v14/main.ail":  "module v14/main\nimport v14/a (pick)\nimport v14/b (name)\nexport func main() -> () ! {IO} = println(name(pick()))\n",
				"v14/prime.ail": "module v14/prime\nimport v14/a (pick)\nimport v14/b (name)\nexport pure func main() -> int = 1\n",
			},
			entry: "v14/main.ail", prime: "v14/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "Color (defined in v14/a)", "Color (defined in v14/b)"},
		},
		{
			name: "h4c: a newtype-record ADT is not the same-named sum type",
			files: map[string]string{
				"h4c/a.ail":     "module h4c/a\nexport type W = W({ x: int })\nexport pure func mk() -> W = W({ x: 1 })\n",
				"h4c/b.ail":     "module h4c/b\nexport type W = W({ x: int }) | Z\nexport pure func getb(w: W) -> int = match w { W(r) => r.x, Z => 0 }\n",
				"h4c/main.ail":  "module h4c/main\nimport h4c/a (mk)\nimport h4c/b (getb)\nexport func main() -> () ! {IO} = println(show(getb(mk())))\n",
				"h4c/prime.ail": "module h4c/prime\nimport h4c/a (mk)\nimport h4c/b (getb)\nexport pure func main() -> int = 1\n",
			},
			entry: "h4c/main.ail", prime: "h4c/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "W (defined in h4c/a)", "W (defined in h4c/b)"},
		},
		{
			name: "h4d: two same-shaped newtype-record ADTs do not cross",
			files: map[string]string{
				"h4d/a.ail":     "module h4d/a\nexport type W = W({ x: int })\nexport pure func mk() -> W = W({ x: 1 })\n",
				"h4d/b.ail":     "module h4d/b\nexport type W = W({ x: int })\nexport pure func getb(w: W) -> int = w.x\n",
				"h4d/main.ail":  "module h4d/main\nimport h4d/a (mk)\nimport h4d/b (getb)\nexport func main() -> () ! {IO} = println(show(getb(mk())))\n",
				"h4d/prime.ail": "module h4d/prime\nimport h4d/a (mk)\nimport h4d/b (getb)\nexport pure func main() -> int = 1\n",
			},
			entry: "h4d/main.ail", prime: "h4d/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "h4d/a", "h4d/b"},
		},
		{
			name: "v15: same-named ADTs used disjointly",
			files: map[string]string{
				"v15/a.ail": "module v15/a\nexport type T = A(int) | B\nexport pure func mk() -> T = B\n" +
					"export pure func desc(t: T) -> string = match t { A(n) => \"a.A ${show(n)}\", B => \"a.B\" }\n",
				"v15/b.ail": "module v15/b\nexport type T = B | C(string) | A(string)\nexport pure func mk() -> T = A(\"hi\")\n" +
					"export pure func desc(t: T) -> string = match t { B => \"b.B\", C(s) => \"b.C\", A(s) => \"b.A ${s}\" }\n",
				"v15/main.ail": "module v15/main\nimport v15/a as Ma (mk, desc)\nimport v15/b as Mb (mk, desc)\n" +
					"export func main() -> () ! {IO} { println(Ma.desc(Ma.mk())); println(Mb.desc(Mb.mk())) }\n",
			},
			entry: "v15/main.ail", prime: "v15/main.ail",
		},
		{
			name: "h2: a qualified constructor import next to an auto-imported same-named one",
			files: map[string]string{
				"h2/a.ail":    "module h2/a\nexport type T = A(int)\nexport pure func get(t: T) -> int = match t { A(n) => n }\n",
				"h2/b.ail":    "module h2/b\nexport type T = A(string)\nexport pure func getb(t: T) -> string = match t { A(s) => s }\n",
				"h2/main.ail": "module h2/main\nimport h2/b (getb)\nimport h2/a (A, get)\nexport func main() -> () ! {IO} = println(show(get(A(7))))\n",
			},
			entry: "h2/main.ail", prime: "h2/main.ail",
		},
		{
			name: "parameterized same-named ADTs do not cross (type-application heads)",
			files: map[string]string{
				"pb/a.ail":     "module pb/a\nexport type Box[t] = Box(t)\nexport pure func mk() -> Box[int] = Box(1)\n",
				"pb/b.ail":     "module pb/b\nexport type Box[t] = Box(t)\nexport pure func get(b: Box[int]) -> int = match b { Box(n) => n }\n",
				"pb/main.ail":  "module pb/main\nimport pb/a (mk)\nimport pb/b (get)\nexport pure func main() -> int = get(mk())\n",
				"pb/prime.ail": "module pb/prime\nimport pb/a (mk)\nimport pb/b (get)\nexport pure func main() -> int = 1\n",
			},
			entry: "pb/main.ail", prime: "pb/prime.ail",
			reject: []string{types.TC_NOMINAL_001, "Box (defined in pb/a)", "Box (defined in pb/b)"},
		},
		{
			name: "a constructor field naming an imported type keeps that type's identity",
			files: map[string]string{
				"cf/a.ail":    "module cf/a\nexport type Inner = I(int)\n",
				"cf/b.ail":    "module cf/b\nimport cf/a (Inner)\nexport type Outer = O(Inner)\n",
				"cf/main.ail": "module cf/main\nimport cf/a (I)\nimport cf/b (O)\nexport pure func main() -> int = match O(I(3)) { O(I(n)) => n }\n",
			},
			entry: "cf/main.ail", prime: "cf/main.ail",
		},
		{
			name: "same-named record aliases each expand to their own body",
			files: map[string]string{
				"ra/a.ail":    "module ra/a\nexport type R = {x: int}\nexport pure func mkA() -> R = {x: 1}\n",
				"ra/b.ail":    "module ra/b\nexport type R = {y: string}\nexport pure func mkB() -> R = {y: \"s\"}\n",
				"ra/main.ail": "module ra/main\nimport ra/a (R, mkA)\nimport ra/b (mkB)\npure func f(r: R) -> int = r.x\nexport pure func main() -> string = \"${show(f(mkA()))}${mkB().y}\"\n",
			},
			entry: "ra/main.ail", prime: "ra/main.ail",
		},
		{
			name: "newtype-record ADT field access across modules",
			files: map[string]string{
				"nt/a.ail":    "module nt/a\nexport type W = W({ x: int })\nexport pure func mk() -> W = W({ x: 5 })\n",
				"nt/main.ail": "module nt/main\nimport nt/a (W, mk)\npure func gx(w: W) -> int = w.x\nexport pure func main() -> int = gx(mk()) + mk().x\n",
			},
			entry: "nt/main.ail", prime: "nt/main.ail",
		},
		{
			name: "derived Eq of one module's Row is not another module's",
			files: map[string]string{
				"de/a.ail":     "module de/a\nexport type Row = R(int) deriving (Eq)\n",
				"de/b.ail":     "module de/b\nexport type Row = R(int)\nexport pure func mk() -> Row = R(1)\n",
				"de/main.ail":  "module de/main\nimport de/a (Row)\nimport de/b (mk)\nexport pure func main() -> bool = mk() == mk()\n",
				"de/prime.ail": "module de/prime\nimport de/a (Row)\nimport de/b (mk)\nexport pure func main() -> int = 1\n",
			},
			entry: "de/main.ail", prime: "de/prime.ail",
			reject: []string{"Eq"},
		},
		{
			// #2247: constraint types are alias-expanded before instance lookup,
			// but a newtype is nominal: expanding b's Row to its record would let
			// it borrow a's `deriving (Eq)` through the bare-name record path.
			name: "derived Eq of one module's newtype is not another module's",
			files: map[string]string{
				"dn/a.ail":     "module dn/a\nexport type Row = R({n: int}) deriving (Eq)\n",
				"dn/b.ail":     "module dn/b\nexport type Row = R({n: int})\nexport pure func mk() -> Row = R({n: 1})\n",
				"dn/main.ail":  "module dn/main\nimport dn/a (Row)\nimport dn/b (mk)\nexport pure func main() -> bool = mk() == mk()\n",
				"dn/prime.ail": "module dn/prime\nimport dn/a (Row)\nimport dn/b (mk)\nexport pure func main() -> int = 1\n",
			},
			entry: "dn/main.ail", prime: "dn/prime.ail",
			reject: []string{"Eq"},
		},
		{
			name: "renamed import of an ADT is still that module's ADT",
			files: map[string]string{
				"rn/a.ail":    "module rn/a\nexport type T = A(int)\nexport pure func mk() -> T = A(1)\n",
				"rn/main.ail": "module rn/main\nimport rn/a (T as AT, mk)\npure func use(t: AT) -> int = match t { A(n) => n }\nexport pure func main() -> int = use(mk())\n",
			},
			entry: "rn/main.ail", prime: "rn/main.ail",
		},
	}
}

func writeNominalFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func assertNominalVerdict(t *testing.T, c nominalCase, err error) {
	t.Helper()
	if c.reject == nil {
		if err != nil {
			t.Fatalf("must type-check: %v", err)
		}
		return
	}
	if err == nil {
		t.Fatal("accepted a program that mixes two different types sharing a name")
	}
	for _, want := range c.reject {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("rejection must contain %q:\n%v", want, err)
		}
	}
}

func TestNominalIdentity_RowsCold(t *testing.T) {
	for _, c := range nominalCases() {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			writeNominalFiles(t, root, c.files)
			_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: c.entry})
			assertNominalVerdict(t, c, err)
		})
	}
}

// Warm: the prime module compiles every dependency into the cache; the entry
// is then checked with those dependencies served from it (asserted through
// the cache's own hit counter, so a run that silently recompiled would fail).
func TestNominalIdentity_RowsWarm(t *testing.T) {
	for _, c := range nominalCases() {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
			writeNominalFiles(t, root, c.files)
			if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: c.prime}); err != nil {
				t.Fatalf("prime compile: %v", err)
			}
			var err error
			out := capturePipelineStderr(func() {
				_, err = Run(Config{Mode: ModeCheck, DebugCompile: true}, Source{Filename: c.entry})
			})
			assertNominalVerdict(t, c, err)
			for rel := range c.files {
				if rel == c.entry || rel == c.prime {
					continue
				}
				dep := strings.TrimSuffix(rel, ".ail")
				if !strings.Contains(out, "[CACHE] "+dep+": SKIP") {
					t.Errorf("warm run did not serve dependency %s from the cache; the verdict was not measured warm", dep)
				}
			}
		})
	}
}

// TestCacheKey_NominalOriginEditInvalidates: m's export `get` changes which T
// it returns (a's -> b's) without changing a single rendered type string. The
// importer's source is unchanged; only the qualified half of m's interface
// digest moves, and the importer must NOT be served its old verdict.
func TestCacheKey_NominalOriginEditInvalidates(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	files := map[string]string{
		"a.ail": "module a\nexport type T = A(int)\n",
		"b.ail": "module b\nexport type T = A(int)\n",
		// m imports BOTH modules in both versions, so main's import closure and
		// every rendered type string stay the same: only the module half of
		// m's export `get` moves (a.T -> b.T), which only the qualified part
		// of the interface digest sees.
		"m.ail":    "module m\nimport a (T as AT, A as AA)\nimport b (T as BT, A as BA)\nexport pure func get() -> AT = AA(1)\n",
		"main.ail": "module main\nimport a (T, A)\nimport m (get)\npure func use(t: T) -> int = match t { A(n) => n }\nexport pure func main() -> int = use(get())\n",
	}
	writeNominalFiles(t, root, files)
	if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "main.ail"}); err != nil {
		t.Fatalf("seed compile should pass: %v", err)
	}
	writeNominalFiles(t, root, map[string]string{"m.ail": "module m\nimport a (T as AT, A as AA)\nimport b (T as BT, A as BA)\nexport pure func get() -> BT = BA(1)\n"})
	_, err := Run(Config{Mode: ModeCheck}, Source{Filename: "main.ail"})
	if err == nil {
		t.Fatal("stale cache: main was served its old verdict after m switched get's T from a to b")
	}
	if !strings.Contains(err.Error(), types.TC_NOMINAL_001) {
		t.Fatalf("expected TC_NOMINAL_001, got: %v", err)
	}
}
