package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2211 / #2240: whether a name in type position is a type variable is decided
// by scope, not by the case of its first letter. A name an enclosing `[T]`
// list binds is a variable whatever its case; a lowercase name the module
// declares or imports by name is that type; another lowercase name is an
// implicitly quantified variable; another uppercase name is a type.
//
// Two readings keep a lowercase name a type variable and warn
// TC_UNBOUND_TYPE_002: an import exports a type of that name but it is not
// imported by name, or it is imported by name as a value.
//
// Every row runs cold and warm (dependencies served from the compile cache):
// a lowercase type crosses the module boundary through the cached interface,
// and the warnings must not be lost on a cache hit.

const (
	// q/shapes declares lowercase types and a function over them.
	tnShapes = "module q/shapes\n" +
		"export type point = {x: int, y: int}\n" +
		"export type label = string\n" +
		"export pure func origin() -> point = {x: 0, y: 0}\n" +
		"export type seg = {a: point, b: point}\n" +
		"export type tagged = Tag(label)\n" +
		"export type box[a] = Bx(a)\n" +
		"export pure func px(p: point) -> int = p.x\n"
	// q/prime compiles q/shapes into the cache for the warm runs.
	tnPrime = "module q/prime\nimport q/shapes (origin)\nexport pure func main() -> int = 1\n"
)

// tnCase is one row: reject lists substrings the error must contain (nil:
// must check); warn lists substrings of the one TC_UNBOUND_TYPE_002 warning
// (nil: there must be none).
type tnCase struct {
	name   string
	files  map[string]string
	reject []string
	warn   []string
}

func typeNameScopeCases() []tnCase {
	shapes := func(main string) map[string]string {
		return map[string]string{"q/shapes.ail": tnShapes, "q/prime.ail": tnPrime, "q/main.ail": main}
	}
	solo := func(main string) map[string]string {
		return map[string]string{"q/prime.ail": "module q/prime\nexport pure func main() -> int = 1\n", "q/main.ail": main}
	}
	c := func(name string, files map[string]string, reject ...string) tnCase {
		return tnCase{name: name, files: files, reject: reject}
	}
	w := func(name string, files map[string]string, warn ...string) tnCase {
		return tnCase{name: name, files: files, warn: warn}
	}
	return []tnCase{
		// ---- #2211: a lowercase declared type is enforced ----
		c("2211 repro: return value does not match the lowercase record type", solo(
			"module q/main\ntype point = {x: int, y: int}\n"+
				"export pure func origin() -> point = {x: \"zero\", label: true}\n"), "record field"),
		c("2211 repro: an int passed where the lowercase type is expected", solo(
			"module q/main\ntype point = {x: int, y: int}\n"+
				"export pure func idp(p: point) -> point = p\n"+
				"export pure func main() -> int = match idp(42) { _ => 0 }\n"), "int"),
		c("2211: a use before the lowercase declaration is the type too", solo(
			"module q/main\nexport pure func idp(p: point) -> point = p\n"+
				"export pure func main() -> int = match idp(42) { _ => 0 }\n"+
				"type point = {x: int, y: int}\n"), "int"),
		c("2211: lowercase alias of a primitive", solo(
			"module q/main\ntype name = string\n"+
				"export pure func greet(n: name) -> name = n\n"+
				"export pure func main() -> int = match greet(7) { _ => 0 }\n"), "string"),
		c("2211: lowercase type imported by name is enforced", shapes(
			"module q/main\nimport q/shapes (point, px)\n"+
				"pure func idp(p: point) -> point = p\n"+
				"export pure func main() -> int = px(idp(42))\n"), "int"),
		c("2211: lowercase type of another module inside a record field", shapes(
			"module q/main\nimport q/shapes (point, px)\n"+
				"type seg = {a: point, b: point}\n"+
				"pure func mk() -> seg = {a: 1, b: 2}\n"+
				"export pure func main() -> int = px(mk().a)\n"), "int"),
		c("2211: an imported lowercase alias whose body names another lowercase type", shapes(
			"module q/main\nimport q/shapes (seg, px)\n"+
				"pure func mk() -> seg = {a: 1, b: 2}\n"+
				"export pure func main() -> int = px(mk().a)\n"), "int"),
		c("2211: an imported constructor whose field is a lowercase type", shapes(
			"module q/main\nimport q/shapes (tagged, Tag)\n"+
				"pure func mk() -> tagged = Tag(42)\n"+
				"export pure func main() -> int = match mk() { Tag(_) => 0 }\n"), "string"),
		c("2211: a lowercase type of an import, applied, not imported by name, is that type", shapes(
			"module q/main\nimport q/shapes (px)\n"+
				"pure func f(b: box[int]) -> box[int] = b\n"+
				"export pure func main() -> int = match f(42) { _ => 0 }\n"), "int"),
		c("2211: an applied lowercase name that is no type is TC_UNBOUND_TYPE_001", shapes(
			"module q/main\nimport q/shapes (px)\n"+
				"pure func f(x: zz[int]) -> int = 0\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_001, "no type constructor 'zz'"),
		c("2211: a same-file lowercase type that reads like a variable gets a note", solo(
			"module q/main\ntype acc = {total: int}\n"+
				"pure func fold2(f: (acc, elem) -> acc, z: acc, xs: [elem]) -> acc = match xs { [] => z, h :: t => fold2(f, f(z, h), t) }\n"+
				"export pure func main() -> int = fold2(\\a x. a + x, 0, [1, 2])\n"),
			"Note: 'acc' in the signature of fold2 is the type declared at"),
		// ---- #2211: a type declaration has no implicit variables ----
		c("2211: a misspelt type in an alias is TC_UNBOUND_TYPE_003 with a did-you-mean", solo(
			"module q/main\ntype Count = integer\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "type variable 'integer' is not a parameter of Count",
			"write `type Count[integer] = …`", "did you mean the type `int`?"),
		c("2211: a free variable in an alias is TC_UNBOUND_TYPE_003", solo(
			"module q/main\ntype Any = a\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "type variable 'a' is not a parameter of Any"),
		c("2211: a free variable beside a parameter, in a record field", solo(
			"module q/main\ntype Pair[a] = {x: a, y: b}\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "'b' is not a parameter of Pair", "type Pair[a, b]"),
		c("2211: a free variable in a constructor field", solo(
			"module q/main\ntype Box = B(elem) | E\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "'elem' is not a parameter of Box"),
		c("2211: a miscased type in a declaration suggests the type", shapes(
			"module q/main\nimport q/shapes (px)\ntype Seg = {a: Point, b: pointt}\ntype Point = {x: int}\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "'pointt' is not a parameter of Seg"),
		c("2211: an import's lowercase type not imported by name, in a declaration", shapes(
			"module q/main\nimport q/shapes (px)\ntype W = {p: point}\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "import the type: import q/shapes (point)"),
		c("2211: a value imported by name, in a declaration", shapes(
			"module q/main\nimport q/shapes (origin)\ntype F = (origin) -> int\nexport pure func main() -> int = 0\n"),
			types.TC_UNBOUND_TYPE_003, "'origin' is not a parameter of F"),
		c("2211: an exported declaration with a free variable fails in its module", map[string]string{
			"q/prime.ail": "module q/prime\nexport pure func main() -> int = 1\n",
			"q/bad.ail":   "module q/bad\nexport type Any = a\nexport pure func one() -> int = 1\n",
			"q/main.ail":  "module q/main\nimport q/bad (one)\nexport pure func main() -> int = one()\n"},
			types.TC_UNBOUND_TYPE_003, "bad.ail:2:"), // the file name only: Windows separators
		c("control: declarations using parameters, local (forward) and imported lowercase types", shapes(
			"module q/main\nimport q/shapes (point, label, px)\n"+
				"type seg2 = {a: pt, b: point}\ntype Tagged[t] = T(t, label) | N\ntype pt = {x: int, y: int}\n"+
				"export pure func main() -> int = match T(1, \"s\") { T(n, _) => n, N => 0 }\n")),
		// ---- with #2288 (record rows) and #2289 (literal annotations) ----
		c("2211+2288: a lowercase record type rejects a missing field like an uppercase one", solo(
			"module q/main\ntype point = {x: int, y: int}\npure func mk() -> point = {x: 1}\nexport pure func main() -> int = mk().x\n"),
			"record field mismatch", "{x, y}"),
		c("2211+2288: a lowercase record type rejects an extra field", solo(
			"module q/main\ntype point = {x: int, y: int}\npure func px(p: point) -> int = p.x\nexport pure func main() -> int = px({x: 1, y: 2, z: 3})\n"),
			"record field mismatch"),
		c("2211+2288: std/sem's sem_frame is enforced at a call", solo(
			"module q/main\nimport std/sem (get_simhash)\nexport pure func main() -> int = get_simhash({simhash: 7})\n"),
			"record field mismatch", "simhash"),
		c("control: std/sem's frames through their declared type", solo(
			"module q/main\nimport std/sem (sem_frame, make_frame_at, bump_version)\nimport std/bytes (fromString)\n"+
				"pure func touch(f: sem_frame) -> sem_frame = bump_version(f)\n"+
				"export pure func main() -> int = touch(make_frame_at(\"k\", \"c\", fromString(\"\"), 0)).ver\n")),
		c("2211+2289: a literal annotated with a lowercase declared type is enforced", solo(
			"module q/main\ntype cnt = int\nexport pure func main() -> int {\n  let inc = func(c: cnt) -> cnt { c + 1 };\n  match inc(\"s\") { _ => 0 }\n}\n"),
			"int vs string"),
		c("2240+2289: a literal annotated with the enclosing function's [T]", solo(
			"module q/main\npure func app[T](x: T) -> T {\n  let g = func(y: T) -> T { y };\n  g(x)\n}\n"+
				"export pure func main() -> int = app(5) + match app(\"s\") { _ => 0 }\n")),
		c("control: a literal annotated with an implicit lowercase variable", solo(
			"module q/main\nexport pure func main() -> int {\n  let ident = func(y: a) -> a { y };\n  ident(5) + ident(6)\n}\n")),
		// ---- a type alias may not refer to itself (no ADT on the cycle) ----
		c("alias cycle: a list of itself", solo(
			"module q/main\ntype L[a] = [L[a]]\nexport pure func main() -> int = 0\n"),
			types.TC_TYPE_ALIAS_CYCLE_001, "type alias 'L' refers to itself", "write `type L[a] = L([L[a]])`"),
		c("alias cycle: a branching tuple of itself", solo(
			"module q/main\ntype T2[a] = (T2[a], T2[a])\nexport pure func main() -> int = 0\n"),
			types.TC_TYPE_ALIAS_CYCLE_001, "type alias 'T2' refers to itself", "write `type T2[a] = T2((T2[a], T2[a]))`"),
		c("alias cycle: through an applied type", solo(
			"module q/main\ntype W[a] = Option[W[a]]\nexport pure func main() -> int = 0\n"),
			types.TC_TYPE_ALIAS_CYCLE_001, "type alias 'W' refers to itself"),
		c("alias cycle: mutual", solo(
			"module q/main\ntype A = B\ntype B = (int, A)\nexport pure func main() -> int = 0\n"),
			types.TC_TYPE_ALIAS_CYCLE_001, "type alias 'A' refers to itself through B"),
		c("alias cycle: in an imported module fails there", map[string]string{
			"q/prime.ail": "module q/prime\nexport pure func main() -> int = 1\n",
			"q/cyc.ail":   "module q/cyc\nexport type L = [L]\nexport pure func one() -> int = 1\n",
			"q/main.ail":  "module q/main\nimport q/cyc (one)\nexport pure func main() -> int = one()\n"},
			types.TC_TYPE_ALIAS_CYCLE_001, "cyc.ail:2:"), // the file name only: Windows separators
		c("control: recursive ADT, aliases and records over it, a self-referential record", shapes(
			"module q/main\nimport q/shapes (px)\nimport std/list (length)\n"+
				"type Tree[a] = Leaf | Node(Tree[a], a, Tree[a])\ntype Forest = [Tree[int]]\n"+
				"type Holder = {root: Tree[int], name: string}\ntype Rose = {label: string, kids: [Rose]}\n"+
				"pure func size(t: Tree[int]) -> int = match t { Leaf => 0, Node(l, _, r) => size(l) + 1 + size(r) }\n"+
				"pure func count(r: Rose) -> int = 1 + length(r.kids)\n"+
				"export pure func main() -> int {\n  let h: Holder = {root: Node(Leaf, 1, Leaf), name: \"h\"};\n"+
				"  let fs: Forest = [h.root];\n  size(h.root) + length(fs) + count({label: \"a\", kids: [{label: \"b\", kids: []}]})\n}\n")),
		c("control: an imported alias of a recursive ADT", map[string]string{
			"q/prime.ail": "module q/prime\nimport q/trees (leaf)\nexport pure func main() -> int = 1\n",
			"q/trees.ail": "module q/trees\nexport type Tree = Leaf | Node(Tree, Tree)\nexport type Forest = [Tree]\nexport pure func leaf() -> Tree = Leaf\n",
			"q/main.ail":  "module q/main\nimport q/trees (Forest, leaf)\nimport std/list (length)\npure func n(f: Forest) -> int = length(f)\nexport pure func main() -> int = n([leaf(), leaf()])\n"}),
		// ---- #2211: readings that stay type variables, with a warning ----
		w("2211: a value imported by name used as a type stays a variable and warns", shapes(
			"module q/main\nimport q/shapes (origin)\n"+
				"pure func twice(f: (origin) -> origin, x: origin) -> origin = f(f(x))\n"+
				"export pure func main() -> int = twice(\\n. n + 1, 1) + match twice(\\s. s, \"s\") { _ => 0 }\n"),
			"'origin' is treated as a type variable", "imported by name from q/shapes, but as a value"),
		w("2211: an implicit variable named like an imported module's type stays a variable and warns", shapes(
			"module q/main\nimport q/shapes (px)\n"+
				"pure func keep(x: label, y: label) -> label = x\n"+
				"export pure func main() -> int = keep(5, 6) + match keep(\"s\", \"t\") { _ => 0 }\n"),
			"'label' is treated as a type variable", "import q/shapes (label)"),
		c("2211: a lowercase type parameter shadows an imported module's type, with no warning", shapes(
			"module q/main\nimport q/shapes (px)\n"+
				"pure func keep[label](x: label) -> label = x\n"+
				"type Holder[point] = H(point) | E\n"+
				"pure func un(h: Holder[int]) -> int = match h { H(n) => n, E => 0 }\n"+
				"export pure func main() -> int = keep(5) + un(H(1)) + match keep(\"s\") { _ => 0 }\n")),
		c("2211: a lowercase type parameter shadows the same type imported by name", shapes(
			"module q/main\nimport q/shapes (px, label)\n"+
				"pure func keep[label](x: label) -> label = x\n"+
				"export pure func main() -> int = keep(5) + match keep(\"s\") { _ => 0 }\n")),
		// ---- #2211 negative controls ----
		c("control: the lowercase type used correctly", shapes(
			"module q/main\nimport q/shapes (point, origin, px)\n"+
				"pure func idp(p: point) -> point = p\n"+
				"export pure func main() -> int = px(idp(origin()))\n")),
		c("control: local lowercase types used correctly", solo(
			"module q/main\ntype point = {x: int, y: int}\ntype seg = {a: point, b: point}\n"+
				"pure func start(s: seg) -> point = s.a\n"+
				"export pure func main() -> int = start({a: {x: 1, y: 2}, b: {x: 3, y: 4}}).x\n")),
		c("control: implicit lowercase type variables stay polymorphic", solo(
			"module q/main\npure func first(p: (a, b)) -> a = match p { (x, _) => x }\n"+
				"pure func foldl(f: (acc, elem) -> acc, z: acc, xs: [elem]) -> acc = match xs { [] => z, h :: t => foldl(f, f(z, h), t) }\n"+
				"export pure func main() -> int = first((1, \"s\")) + foldl(\\s x. s + x, 0, [1, 2]) + match first((\"s\", 1)) { _ => 0 }\n")),
		// ---- #2240: an uppercase type parameter is a type variable ----
		c("2240 repro: id[T] applied to an int", solo(
			"module q/main\npure func id[T](x: T) -> T = x\nexport pure func main() -> int = id(5)\n")),
		c("2240: id[T] is polymorphic (used at two types)", solo(
			"module q/main\npure func id[T](x: T) -> T = x\n"+
				"export pure func main() -> int = id(5) + match id(\"s\") { _ => 0 }\n")),
		c("2240: multi-parameter, nested in types and in the body", solo(
			"module q/main\npure func mapPair[A, B](f: (A) -> B, p: (A, A)) -> (B, B) {\n"+
				"  let g: (A) -> B = f;\n  match p { (x, y) => (g(x), g(y)) }\n}\n"+
				"export pure func main() -> int = match mapPair(\\n. n + 1, (1, 2)) { (a, b) => a + b }\n")),
		c("2240: a type declaration's uppercase parameter", solo(
			"module q/main\ntype Box[T] = Box(T)\n"+
				"pure func unbox[T](b: Box[T]) -> T = match b { Box(v) => v }\n"+
				"export pure func main() -> int = unbox(Box(3)) + match unbox(Box(\"s\")) { _ => 0 }\n")),
		c("2240: the parameter shadows a declared type of the same name", solo(
			"module q/main\ntype T = Tee(int)\npure func id[T](x: T) -> T = x\n"+
				"export pure func main() -> int = id(5)\n")),
		c("2240: outside its list, the uppercase name is the declared type again", solo(
			"module q/main\ntype T = Tee(int)\npure func id[T](x: T) -> T = x\n"+
				"pure func un(t: T) -> int = match t { Tee(n) => n }\n"+
				"export pure func main() -> int = un(5)\n"), "int"),
		c("2240: a type parameter may not be named like a built-in type", solo(
			"module q/main\ntype Res[int] = Ok(int) | Err(string)\nexport pure func main() -> int = 0\n"),
			"PAR_TYPE_PARAM_BUILTIN", "Rename the type parameter"),
		c("2240: a type parameter cannot be applied (no higher-kinded types)", solo(
			"module q/main\npure func same[m](x: m[int]) -> m[int] = x\nexport pure func main() -> int = 0\n"),
			"PAR_TYPE_PARAM_APPLIED", "type parameter 'm' cannot be applied"),
	}
}

func TestTypeNameScope_Cold(t *testing.T) {
	for _, c := range typeNameScopeCases() {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			writeNominalFiles(t, root, c.files)
			res, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: "q/main.ail"})
			assertTypeNameVerdict(t, c, res, err)
		})
	}
}

func TestTypeNameScope_Warm(t *testing.T) {
	for _, c := range typeNameScopeCases() {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
			writeNominalFiles(t, root, c.files)
			if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "q/prime.ail"}); err != nil {
				t.Fatalf("prime compile: %v", err)
			}
			var (
				res Result
				err error
			)
			out := capturePipelineStderr(func() {
				res, err = Run(Config{Mode: ModeCheck, DebugCompile: true}, Source{Filename: "q/main.ail"})
			})
			assertTypeNameVerdict(t, c, res, err)
			if _, ok := c.files["q/shapes.ail"]; ok && !strings.Contains(out, "[CACHE] q/shapes: SKIP") {
				t.Errorf("warm run did not serve q/shapes from the cache; the verdict was not measured warm")
			}
		})
	}
}

func assertTypeNameVerdict(t *testing.T, c tnCase, res Result, err error) {
	t.Helper()
	if c.reject != nil {
		if err == nil {
			t.Fatal("accepted a program whose annotation names a type it does not satisfy")
		}
		for _, want := range c.reject {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("rejection must contain %q:\n%v", want, err)
			}
		}
		return
	}
	if err != nil {
		t.Fatalf("must type-check: %v", err)
	}
	var got []string
	for _, w := range res.Warnings {
		if cw, ok := w.(interface{ Code() string }); ok && cw.Code() == types.TC_UNBOUND_TYPE_002 {
			got = append(got, w.String())
		}
	}
	if c.warn == nil {
		if len(got) > 0 {
			t.Fatalf("unexpected TC_UNBOUND_TYPE_002:\n%s", strings.Join(got, "\n"))
		}
		return
	}
	if len(got) != 1 {
		t.Fatalf("want one TC_UNBOUND_TYPE_002, got %d:\n%s", len(got), strings.Join(got, "\n"))
	}
	for _, want := range c.warn {
		if !strings.Contains(got[0], want) {
			t.Errorf("warning must contain %q:\n%s", want, got[0])
		}
	}
}
