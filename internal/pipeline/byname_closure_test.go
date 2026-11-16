package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The candidates of a constructor matched by name (#2353) — and the #1478
// foreign-constructor and arity checks that read the same table — come from
// the module's own import closure, never from the siblings the program
// happens to load (#2362 review, round 3). A library's verdict, its error and
// its cached interface therefore do not depend on what else is loaded, the
// root's import order, or which root compiled it first.

// byNameFixture writes files under a fresh root, makes it the working
// directory with a cache dir inside it, and returns a check function.
func byNameFixture(t *testing.T, files map[string]string) func(entry string) string {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("AILANG_CACHE_DIR", filepath.Join(root, "cache"))
	for path, src := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return func(entry string) string {
		_, err := runModuleWithCacheDependencies(t.Context(), Config{Mode: ModeCheck}, Source{Filename: entry}, productionCacheDependencies())
		if err == nil {
			return "ok"
		}
		return err.Error()
	}
}

const (
	bnShape = "module lib/s\nexport type Shape = Circle(int) | Rect(int, int) | Dot\n" +
		"export pure func mks(i: int) -> Shape = if i == 0 then Circle(5) else if i == 1 then Rect(2, 3) else Dot\n"
	bnV = "module lib/v\nimport std/option (Option, Some, None)\n" +
		"export pure func oo(i: int) -> Option[int] = if i > 0 then Some(i) else None\n" +
		"export pure func osx(i: int) -> Option[string] = if i > 0 then Some(\"s\") else None\n"
	// lib/q declares Circle and Some too; lib/u reaches it.
	bnQ = "module lib/q\nexport type Q = Circle(int) | Some(int)\nexport pure func qq(i: int) -> int = i\n"
	bnU = "module lib/u\nimport lib/q (qq)\nexport pure func uu(i: int) -> int = qq(i)\n"
)

// Review shapes sa/sb/sc/sd: the same library checked with and without an
// unrelated transitive dependency (lib/u → lib/q) that declares a
// same-named constructor. Same verdict either way.
func TestByNameVerdictIndependentOfTransitiveNamesakes(t *testing.T) {
	bodies := map[string]string{
		"sa unannotated parameter":         "export pure func fa(x) -> int = match x { Circle(r) => r, _ => 0 }\n",
		"sb let-bound helper at two types": "export pure func fb(i: int) -> string { let isS = \\o. match o { Some(_) => \"y\", _ => \"n\" }; \"${isS(oo(i))}${isS(osx(i))}\" }\n",
		"sc lambda over a list of a":       "export pure func fc(xs) -> int = length(filter(\\v. match v { Circle(_) => true, _ => false }, xs))\n",
		"sd contract let at two types":     "export pure func fd(i: int) -> int\nrequires { let g = \\q. match q { Some(_) => true, _ => false } in g(oo(i)) || g(osx(i)) }\n{ i }\n",
		"typed value":                      "export pure func fe(i: int) -> int = match ss(i) { Circle(r) => r, _ => 0 }\n",
	}
	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			verdict := func(withQ bool) string {
				w := "module lib/w\nimport lib/s (mks)\nexport pure func ss(i: int) = mks(i)\n"
				if withQ {
					w = "module lib/w\nimport lib/s (mks)\nimport lib/u (uu)\nexport pure func ss(i: int) = mks(i)\nexport pure func wu(i: int) -> int = uu(i)\n"
				}
				check := byNameFixture(t, map[string]string{
					"lib/s.ail": bnShape, "lib/v.ail": bnV, "lib/q.ail": bnQ, "lib/u.ail": bnU, "lib/w.ail": w,
					"lib/p.ail": "module lib/p\nimport lib/w (ss)\nimport lib/v (oo, osx)\nimport std/list (filter, length)\n" + body,
				})
				return check("lib/p.ail")
			}
			without, with := verdict(false), verdict(true)
			if (without == "ok") != (with == "ok") || codeOf(without) != codeOf(with) {
				t.Errorf("verdict depends on an unrelated transitive module:\nwithout lib/q: %s\nwith lib/q:    %s", without, with)
			}
		})
	}
}

// codeOf returns the first diagnostic code in msg ("" when none).
func codeOf(msg string) string {
	for _, f := range strings.FieldsFunc(msg, func(r rune) bool { return r == ' ' || r == ':' || r == '\n' }) {
		if strings.HasPrefix(f, "TC_") {
			return f
		}
	}
	return ""
}

// Review shape sib_*: lib/aq declares Circle and is imported by the ROOT
// only, a sibling of lib/p. The root's import order must not change lib/p's
// verdict or its error text.
func TestByNameVerdictIndependentOfRootImportOrder(t *testing.T) {
	p := "module lib/p\nimport lib/w (ss)\nexport pure func fa(x) -> int = match x { Circle(r) => r, _ => 0 }\n"
	run := func(aqFirst bool) string {
		imports := "import lib/p (fa)\nimport lib/aq (qv)\n"
		if aqFirst {
			imports = "import lib/aq (qv)\nimport lib/p (fa)\n"
		}
		check := byNameFixture(t, map[string]string{
			"lib/s.ail": bnShape, "lib/p.ail": p,
			"lib/w.ail":  "module lib/w\nimport lib/s (mks)\nexport pure func ss(i: int) = mks(i)\n",
			"lib/aq.ail": "module lib/aq\nexport type Q = Circle(int) | QQ\nexport pure func qv(i: int) -> int = i\n",
			"main.ail":   "module main\n" + imports + "import lib/w (ss)\nexport pure func summary() -> string = show([fa(ss(0)), qv(7)])\n",
		})
		return check("main.ail")
	}
	pFirst, aqFirst := run(false), run(true)
	if pFirst != aqFirst {
		t.Errorf("lib/p's result depends on the root's import order:\np first:  %s\naq first: %s", pFirst, aqFirst)
	}
	if !strings.Contains(pFirst, "TC_MATCH_007") || strings.Contains(pFirst, "lib/aq") {
		t.Errorf("want TC_MATCH_007 naming lib/s's Shape only (lib/aq is not in lib/p's closure), got %s", pFirst)
	}
}

// Review shape orphan: lib/p imports nothing, so no type in its closure
// declares Circle. Two roots load different Circles first; with a shared
// cache, cold and warm, in either order, lib/p gets the same result.
func TestByNameOrphanSameResultUnderEveryRootAndCache(t *testing.T) {
	files := map[string]string{
		"lib/aq.ail": "module lib/aq\nexport type Q = QQ | Circle(int)\nexport pure func qmk(i: int) -> Q = if i > 0 then Circle(i) else QQ\n",
		"lib/s.ail":  bnShape,
		"lib/p.ail":  "module lib/p\nexport pure func fa(x) -> int = match x { Circle(r) => r, _ => 0 }\n",
		"root1.ail":  "module root1\nimport lib/aq (qmk)\nimport lib/p (fa)\nexport pure func summary() -> string = show([fa(qmk(4))])\n",
		"root1b.ail": "module root1b\nimport lib/p (fa)\nimport lib/aq (qmk)\nexport pure func summary() -> string = show([fa(qmk(4))])\n",
		"root2.ail":  "module root2\nimport lib/s (mks)\nimport lib/p (fa)\nexport pure func summary() -> string = show([fa(mks(0))])\n",
	}
	for _, order := range [][]string{
		{"root1.ail", "root2.ail", "root1b.ail", "root1.ail", "root2.ail"},
		{"root2.ail", "root1.ail", "root2.ail", "root1b.ail"},
	} {
		check := byNameFixture(t, files) // a fresh, shared cache per order
		var first string
		for i, entry := range order {
			got := check(entry)
			if !strings.Contains(got, "type error in lib/p ") || !strings.Contains(got, "TC_MATCH_001") {
				t.Fatalf("%v[%d] %s: want lib/p's TC_MATCH_001 (Circle is declared nowhere in its closure), got %s", order, i, entry, got)
			}
			// The library's error is the same whichever root reached it.
			msg := got[strings.Index(got, "type error in lib/p "):]
			if first == "" {
				first = msg
			} else if msg != first {
				t.Errorf("%v[%d] %s: lib/p's error differs:\n%s\nvs\n%s", order, i, entry, msg, first)
			}
		}
	}
}

// Review shape hop3: the import closure is TRANSITIVE. probe imports lib/w,
// which imports lib/m only as a namespace (M), which declares `type Sh =
// Shape` over lib/s: the value's type is lib/s's Shape, three hops away, so
// Circle and Rect are its constructors by name. A closure of direct imports
// only rejected this (TC_MATCH_001) and narrowed the cache-key closure;
// nothing with three hops tested it.
func TestByNameClosureIsTransitive(t *testing.T) {
	check := byNameFixture(t, map[string]string{
		"lib/s.ail": bnShape,
		"lib/m.ail": "module lib/m\nimport lib/s (Shape, mks)\nexport type Sh = Shape\nexport pure func mm(i: int) -> Sh = mks(i)\n",
		"lib/w.ail": "module lib/w\nimport lib/m as M\nexport pure func ww(i: int) = M.mm(i)\n",
		"probe.ail": "module probe\nimport lib/w (ww)\npure func f(i: int) -> int = match ww(i) { Circle(r) => r, Rect(a, b) => a * b, _ => -1 }\nexport pure func summary() -> string = show([f(0), f(1), f(2)])\n",
	})
	if got := check("probe.ail"); got != "ok" {
		t.Fatalf("a by-name match three imports away must check, got %s", got)
	}
}

// Review shape n1_multi (kills the mutant that drops a deferred choice's
// argument equalities): Wrap(int) in lib/a, Wrap(string) in lib/b, neither
// imported; ga returns lib/a's A, so v is an int and `v ++ ["x"]` is a type
// error, not a list concatenation of an int.
func TestByNameDeferredChoiceTypesItsArguments(t *testing.T) {
	check := byNameFixture(t, map[string]string{
		"lib/a.ail": "module lib/a\nexport type A = Wrap(int) | AN\nexport pure func ma(i: int) -> A = Wrap(i)\n",
		"lib/b.ail": "module lib/b\nexport type B = Wrap(string) | BN\nexport pure func mb(i: int) -> B = Wrap(\"s\")\n",
		"lib/w.ail": "module lib/w\nimport lib/a (ma)\nimport lib/b (mb)\nexport pure func ga(i: int) = ma(i)\nexport pure func gb(i: int) = mb(i)\n",
		"main.ail":  "module main\nimport lib/w (ga, gb)\npure func f(i: int) -> [string] = match ga(i) { Wrap(v) => v ++ [\"x\"], _ => [] }\nexport pure func summary() -> string = show(f(1))\n",
	})
	if got := check("main.ail"); !strings.Contains(got, "int") || !strings.Contains(got, "unif") {
		t.Errorf("want a unification error (v is an int), got %s", got)
	}
}
