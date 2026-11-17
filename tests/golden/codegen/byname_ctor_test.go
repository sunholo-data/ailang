package codegen_golden_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// #2303: a constructor matched by name (#323: neither its type nor the
// constructor is imported) against a value whose type nothing fixes was
// accepted under a catch-all, and the back ends guessed its type: the VM took
// the first loaded ADT declaring the name. With lib/x's `Color = Red | Green | Blue` and lib/y's
// `Light = Green | Red`, `isRed[a](x: a) = match x { Red => 1, _ => 0 }`
// printed [1, 0, 0, 1] on the evaluator and [1, 0, 1, 0] on both VMs (exit 0).
// Such a pattern is now an error on every path,
// by every route that leaves the value's type unfixed. #2353: a by-name
// pattern's type is decided by the value matched, and its arguments are typed
// from that type's constructor — so a pattern nested in one is decided too,
// and the variables it binds have their real types. Patterns whose type IS
// fixed run identically on the evaluator and both VMs.

// checkJSON is the part of `check --json` these tests read.
type checkJSON struct {
	Passed bool `json:"passed"`
	Errors []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

var byNameLibs = map[string]string{
	"lib/x.ail": "module lib/x\nexport type Color = Red | Green | Blue\n" +
		"export pure func mkc(i: int) -> Color = if i == 0 then Red else if i == 1 then Green else Blue\n",
	"lib/y.ail": "module lib/y\nexport type Light = Green | Red\n" +
		"export pure func mkl(i: int) -> Light = if i == 0 then Green else Red\n",
	// Declares no type: Color and Light reach its importers only transitively.
	"lib/w.ail": "module lib/w\nimport lib/x (mkc)\nimport lib/y (mkl)\n" +
		"export pure func c(i: int) -> int = 0\nexport pure func cc(i: int) = mkc(i)\nexport pure func ll(i: int) = mkl(i)\n",
	"lib/z.ail": "module lib/z\nimport lib/w (c)\n" +
		"export pure func isRedZ(x) -> int = match x { Red => 1, _ => 0 }\n",
	"lib/r.ail": "module lib/r\nimport std/result (Result, Ok, Err)\n" +
		"export pure func parse(s: string) -> Result[int, string] = if s == \"1\" then Ok(1) else Err(s)\n",
	// The independent review's nested shapes (PR #2362).
	"lib/b.ail": "module lib/b\nimport lib/x (Color, Red)\nexport type Box[a] = MkBox(a)\nexport type Pair = Pair(Color, int)\n" +
		"export pure func mkbox(c: Color) -> Box[Color] = MkBox(c)\nexport pure func mkpair(c: Color, n: int) -> Pair = Pair(c, n)\n",
	"lib/api.ail": "module lib/api\nimport lib/x (Color, mkc)\nimport std/option (Option, Some, None)\nimport std/result (Result, Ok, Err)\n" +
		"import lib/b (Box, Pair, mkbox, mkpair)\n" +
		"export pure func oc(i: int) -> Option[Color] = if i < 0 then None else Some(mkc(i))\n" +
		"export pure func rc(i: int) -> Result[Color, string] = if i < 0 then Err(\"neg\") else Ok(mkc(i))\n" +
		"export pure func bc(i: int) -> Box[Color] = mkbox(mkc(i))\n" +
		"export pure func pc(i: int) -> Pair = mkpair(mkc(i), i)\n" +
		"export pure func obc(i: int) -> Option[Box[Color]] = Some(mkbox(mkc(i)))\n" +
		"export pure func oo(i: int) -> Option[Option[int]] = if i > 0 then Some(Some(i)) else if i == 0 then Some(None) else None\n" +
		"export pure func oi(i: int) -> Option[int] = if i > 0 then Some(i) else None\n" +
		"export pure func os(i: int) -> Option[string] = if i > 0 then Some(\"s\") else None\n" +
		"export pure func ro(i: int) -> Result[int, string] = if i > 0 then Ok(i) else Err(\"neg\")\n",
	// Pick is declared by two loaded types (lib/q's Q1, lib/q2's Q2).
	"lib/q.ail":  "module lib/q\nexport type Q1 = Pick(int) | Q1none\nexport pure func q1() -> Q1 = Pick(1)\n",
	"lib/q2.ail": "module lib/q2\nexport type Q2 = Pick(string)\nexport pure func q2() -> Q2 = Pick(\"a\")\n",
	"lib/qq.ail": "module lib/qq\nimport lib/q (q1)\nimport lib/q2 (q2)\nexport pure func a1() = q1()\nexport pure func a2() = q2()\n",
}

func writeByNameFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, m := range []map[string]string{byNameLibs, files} {
		for p, src := range m {
			full := filepath.Join(dir, filepath.FromSlash(p))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(src), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

// at renders "file:line:col" of the first `ctor` on line of file's source
// (the error is located at the constructor), or "file:line:" for ctor "".
func at(file, src string, line int, ctor string) string {
	if ctor == "" {
		return fmt.Sprintf("%s:%d:", file, line)
	}
	l := strings.Split(src, "\n")[line-1]
	ident := func(b byte) bool {
		return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
	}
	for i := 0; i+len(ctor) <= len(l); i++ {
		end := i + len(ctor)
		if l[i:end] == ctor && (i == 0 || !ident(l[i-1])) && (end == len(l) || !ident(l[end])) {
			return fmt.Sprintf("%s:%d:%d:", file, line, i+1)
		}
	}
	return fmt.Sprintf("%s:%d: (%s not found)", file, line, ctor)
}

func TestUnresolvedByNameCtorPatternRejectedOnEveryPath(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := ailangBin
	const head = "import lib/w (cc, ll)\n\n"
	const (
		rigid   = "is a type variable of the signature"
		unfixed = "cannot tell which type"
		twoUses = "cannot be generic: all its uses must have one type"
	)
	routes := []struct {
		name, file, src string
		line            int
		ctor            string // the constructor the error points at ("" = line only)
		code, msg       string
	}{
		{"rigid type variable (the issue's program)", "r1.ail",
			"module r1\n" + head + "pure func isRed[a](x: a) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([isRed(cc(0)), isRed(cc(1)), isRed(ll(0)), isRed(ll(1))])\n",
			4, "Red", "TC_MATCH_007", rigid},
		{"unannotated parameter", "r2.ail",
			"module r2\n" + head + "pure func isRed(x) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([isRed(cc(0)), isRed(ll(1))])\n",
			4, "Red", "TC_MATCH_007", unfixed},
		// A let-bound function whose by-name pattern its uses decide stays
		// monomorphic (as for an ambiguous import, #2228): used at two types
		// it is an error naming the constructor; at one it runs (ok7 below).
		{"let-bound lambda used at two types", "r3.ail",
			"module r3\n" + head + "export pure func summary() -> string {\n  let k = \\v. match v { Green => 10, _ => 20 };\n  show([k(cc(1)), k(ll(0))])\n}\n",
			5, "Green", "TC_AMBIGUOUS_CTOR_001", twoUses},
		{"function literal used at two types", "r4.ail",
			"module r4\n" + head + "export pure func summary() -> string {\n  let k = func(v) -> int { match v { Green => 10, _ => 20 } };\n  show([k(cc(1)), k(ll(0))])\n}\n",
			5, "Green", "TC_AMBIGUOUS_CTOR_001", twoUses},
		{"nested in a tuple pattern", "r5.ail",
			"module r5\n" + head + "pure func firstRed(p) -> int = match p { (Red, n) => n, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([firstRed((cc(0), 1)), firstRed((ll(1), 4))])\n",
			4, "Red", "TC_MATCH_007", unfixed},
		{"lambda passed to a HOF on a type variable", "r6.ail",
			"module r6\nimport std/list (filter, length)\n" + head +
				"pure func countRed[a](xs: [a]) -> int = length(filter(\\v. match v { Red => true, _ => false }, xs))\n\n" +
				"export pure func summary() -> string = show([countRed([cc(0), cc(1)]), countRed([ll(0), ll(1)])])\n",
			5, "Red", "TC_MATCH_007", rigid},
		{"another module's function", "lib/z.ail",
			byNameLibs["lib/z.ail"], 3, "Red", "TC_MATCH_007", unfixed},
		{"contract clause", "r8.ail",
			"module r8\n" + head + "pure func keep[a](x: a) -> int\nrequires { match x { Red => true, _ => false } }\n{ 1 }\n\n" +
				"export pure func summary() -> string = show(keep(cc(0)))\n",
			5, "Red", "TC_MATCH_007", rigid},
		{"test body", "r9.ail",
			"module r9\n" + head + "export pure func summary() -> string = show(cc(0))\n\n" +
				"test \"by-name in a test body\" {\n  let k = \\v. match v { Red => 1, _ => 0 };\n  assert k(cc(0)) == 1 && k(ll(1)) == 1\n}\n",
			7, "Red", "TC_AMBIGUOUS_CTOR_001", twoUses},
		{"by-name constructor with arguments, two candidate types, unannotated parameter", "r11.ail",
			"module r11\nimport lib/qq (a1, a2)\n\npure func picked(x) -> int = match x { Pick(c) => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show(picked(a1()))\n",
			4, "Pick", "TC_MATCH_007", unfixed},
		{"a record value (review of #2362: the wording)", "r12.ail",
			"module r12\n" + head + "pure func f(r: {a: int}) -> int = match r { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show(f({a: 1}))\n",
			4, "Red", "TC_MATCH_007", "against a value of type { a: int }, which is not built from constructors here"},
		// #2353: the variables a by-name pattern binds have their real type
		// (before: check passed and every engine crashed at run time).
		{"a by-name pattern's variable used at the wrong type", "r10.ail",
			"module r10\nimport std/list (nth)\n\n" +
				"export pure func summary() -> string {\n  let r: string = match nth([1, 2], 0) { Some(v) => v, None => \"\" };\n  r\n}\n",
			5, "", "", "No instance for Num[string]"}, // the literals 1, 2 meet v: string
	}
	files := map[string]string{}
	for _, r := range routes {
		if r.file != "lib/z.ail" {
			files[r.file] = r.src
		}
	}
	// The cross-module route is reached through an importer.
	files["r7.ail"] = "module r7\nimport lib/w (cc, ll)\nimport lib/z (isRedZ)\n\n" +
		"export pure func summary() -> string = show([isRedZ(cc(0)), isRedZ(ll(1))])\n"
	dir := writeByNameFixture(t, files)
	run := func(args ...string) (string, int) {
		stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
		return filepath.ToSlash(stdout + stderr), code
	}
	for _, r := range routes {
		t.Run(r.name, func(t *testing.T) {
			entry, loc := r.file, at(r.file, r.src, r.line, r.ctor)
			if entry == "lib/z.ail" {
				entry = "r7.ail"
			}
			stdout, _, code := testutil.RunBounded(t, dir, 90*time.Second, bin, "check", "--json", entry)
			var out checkJSON
			if err := json.Unmarshal([]byte(stdout), &out); err != nil || code != 1 || out.Passed || len(out.Errors) == 0 {
				t.Fatalf("check --json: exit %d, %v\n%s", code, err, stdout)
			}
			if e := out.Errors[0]; (r.code != "" && e.Code != r.code && !strings.Contains(e.Message, r.code)) || !strings.Contains(e.Message, r.msg) {
				t.Errorf("check --json: first error %+v, want %s %q", e, r.code, r.msg)
			}
			paths := [][]string{
				{"check", entry},
				{"run", "--quiet", "--entry", "summary", entry},
				{"run", "--quiet", "--bytecode", "--entry", "summary", entry},
				{"run", "--quiet", "--strict-bytecode", "--entry", "summary", entry},
			}
			for _, args := range paths {
				all, code := run(args...)
				if code == 0 || !strings.Contains(all, loc) || !strings.Contains(all, r.code) || !strings.Contains(all, r.msg) ||
					strings.Contains(all, "panic:") {
					t.Errorf("%s: exit %d, want %s %q at %s before anything runs:\n%s", strings.Join(args, " "), code, r.code, r.msg, loc, all)
				}
			}
		})
	}
}

// Negative controls: by-name and imported constructor patterns whose type is
// fixed give the same answer on the evaluator and both VMs.
func TestFixedCtorPatternsAgreeOnEveryEngine(t *testing.T) {
	t.Setenv("AILANG_CACHE_DIR", t.TempDir())
	bin := ailangBin
	cases := []struct {
		name, module, src, want string
	}{
		{"the issue's shape with the type imported", "ok1",
			"module ok1\nimport lib/w (cc, ll)\nimport lib/x (Color)\n\n" +
				"pure func isRed(x) -> int = match x { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string {\n  let k = \\v. match v { Green => 10, _ => 20 };\n" +
				"  show([isRed(cc(0)), isRed(cc(1)), k(cc(1)), k(cc(0))])\n}\n", "[1, 0, 10, 20]"},
		{"by-name on a fixed type: top level, inside Some, guard", "ok2",
			"module ok2\nimport std/list (nth)\nimport std/option (Option, Some, None)\nimport lib/w (cc, ll)\n\n" +
				"export pure func summary() -> string {\n" +
				"  let a = match cc(0) { Red => 1, _ => 0 };\n" +
				"  let b = match ll(1) { Red => 1, _ => 0 };\n" +
				"  let c = match ll(0) { Red => 1, Green => 2 };\n" +
				"  let e = match nth([ll(1)], 0) { Some(Red) => 1, Some(_) => 2, None => 3 };\n" +
				"  let f = match cc(0) { Red if a == 0 => 9, Red => 1, _ => 0 };\n" +
				"  let g = match ll(1) { Red if a == 0 => 9, Red => 1, _ => 0 };\n" +
				"  show([a, b, c, e, f, g])\n}\n", "[1, 1, 2, 1, 1, 1]"},
		{"by-name in a tuple on a fixed type", "ok6",
			"module ok6\nimport lib/w (cc, ll)\n\n" +
				"export pure func summary() -> string = show([match (cc(1), 5) { (Green, n) => n, _ => 0 }, match (ll(0), 6) { (Green, n) => n, _ => 0 }])\n",
			"[5, 6]"},
		{"deferred constructor choice (#2228)", "ok3",
			"module ok3\nimport lib/x (Color, mkc)\nimport lib/y (Light, mkl)\n\n" +
				"pure func f(c: Color) -> int = match c { Red => 1, _ => 0 }\n" +
				"pure func g(l: Light) -> int = match l { Red => 1, _ => 0 }\n\n" +
				"export pure func summary() -> string = show([f(mkc(0)), f(mkc(1)), g(mkl(0)), g(mkl(1))])\n", "[1, 0, 0, 1]"},
		{"alias of an ADT (#2244)", "ok4",
			"module ok4\nimport lib/x (Color, mkc)\n\ntype Hue = Color\n\n" +
				"pure func f(h: Hue) -> int = match h { Red => 1, Green => 2, Blue => 3 }\n\n" +
				"export pure func summary() -> string = show([f(mkc(0)), f(mkc(1)), f(mkc(2))])\n", "[1, 2, 3]"},
		{"prelude Option and Result by name, from library modules", "ok5",
			"module ok5\nimport std/list (nth)\nimport lib/r (parse)\n\n" +
				"export pure func summary() -> string {\n" +
				"  let a = match nth([1, 2], 1) { Some(v) => v + 1, None => 0 };\n" +
				"  let b = match parse(\"1\") { Ok(n) => n, Err(_) => 0 };\n" +
				"  let c = match parse(\"x\") { Ok(n) => n, Err(e) => if e == \"x\" then 7 else 8 };\n" +
				"  show([a, b, c])\n}\n", "[3, 1, 7]"},
		{"let-bound lambda used at one type", "ok7",
			"module ok7\nimport lib/w (cc, ll)\n\n" +
				"export pure func summary() -> string {\n  let k = \\v. match v { Green => 10, _ => 20 };\n  show([k(cc(1)), k(cc(0))])\n}\n",
			"[10, 20]"},
		// The review's over-rejection rows (PR #2362): nested by-name
		// constructors whose type the value's annotated type fixes.
		{"nested by-name constructors (review p01/p02)", "ok8",
			"module ok8\nimport lib/api (oc, rc, bc, pc, obc, oo)\n\n" +
				"pure func fOpt(i: int) -> int = match oc(i) { Some(Red) => 1, Some(_) => 2, _ => 0 }\n" +
				"pure func fRes(i: int) -> int = match rc(i) { Ok(Red) => 1, Ok(_) => 2, _ => 0 }\n" +
				"pure func fBox(i: int) -> int = match bc(i) { MkBox(Red) => 1, _ => 0 }\n" +
				"pure func fPair(i: int) -> int = match pc(i) { Pair(Red, n) => 100 + n, _ => 0 }\n" +
				"pure func fOB(i: int) -> int = match obc(i) { Some(MkBox(Red)) => 1, _ => 0 }\n" +
				"pure func nn(o: Option[Option[int]]) -> int = match o { Some(Some(x)) => x, Some(None) => -1, _ => -2 }\n\n" +
				"export pure func summary() -> string =\n  show([fOpt(0), fOpt(1), fOpt(-1), fRes(0), fRes(1), fBox(0), fBox(1), fPair(0), fPair(1), fOB(0), fOB(1), nn(oo(5)), nn(oo(0)), nn(oo(-1))])\n",
			"[1, 2, 0, 1, 2, 1, 0, 100, 0, 1, 0, 5, -1, -2]"},
		// Review of #2362, round 2: a contract clause's let is an inner let,
		// decided by its use; a by-name helper used at one type runs.
		{"contract lets are decided by use; one-type helpers run", "ok9",
			"module ok9\nimport lib/api (oi, ro)\n\n" +
				"pure func pre(r: Result[int, string]) -> int\nrequires { let g = \\q. match q { Ok(_) => true, _ => false } in g(r) }\n{ 1 }\n" +
				"pure func post(n: int) -> Result[int, string]\nensures { let g = \\q. match q { Ok(v) => v > 0, _ => true } in g(result) }\n{ ro(n) }\n\n" +
				"export pure func summary() -> string {\n" +
				"  let isSome = \\o. match o { Some(_) => true, _ => false };\n" +
				"  let sh = \\o. match o { Some(x) => show(x), _ => \"-\" };\n" +
				"  show([show([isSome(oi(1)), isSome(oi(0))]), sh(oi(1)), sh(oi(0)), show(pre(ro(2))), show(post(2))])\n}\n",
			"[\"[true, false]\", \"1\", \"-\", \"1\", \"Ok(2)\"]"},
	}
	files := map[string]string{}
	for _, c := range cases {
		files[c.module+".ail"] = c.src
	}
	dir := writeByNameFixture(t, files)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, flags := range [][]string{nil, {"--bytecode"}, {"--strict-bytecode"}} {
				args := append(append([]string{"run", "--quiet"}, flags...), "--entry", "summary", c.module+".ail")
				stdout, stderr, code := testutil.RunBounded(t, dir, 90*time.Second, bin, args...)
				if got := strings.TrimSpace(stdout); code != 0 || got != c.want || strings.Contains(stderr, "falling back") {
					t.Errorf("run %v: exit %d, got %q, want %q\n%s", flags, code, got, c.want, stderr)
				}
			}
		})
	}
}
