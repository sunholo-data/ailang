package types

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// M-IFC-DATA-CALLABLE-PRUNE: a value whose declared type holds no function is
// never applied, so it must not carry a callable (and its demands) along.

// The specstore 0.3.0 shape-first audit, reduced (reported 2026-11-05): mapE's
// result was assumed to possibly BE shapeRow (the imported-callee fallback), so
// a list of strings carried mark's <client> demand, and indexing it with a
// <structure> int was refused.
const shapeFirstSrc = `module t
import std/list (mapE, filter, length, nth, range)
type Cell = { text: string<client> }
func mark(raw: string<client>) -> string<structure> ! {Declassify} { if raw == "" then "" else "x" }
func shapeRow(r: [Cell]) -> [string<structure>] ! {Declassify} = mapE(\c. mark(c.text), r)
type S = { rows: [[string<structure>]] }
func shapeAll(rs: [[Cell]]) -> S ! {Declassify} = { rows: mapE(shapeRow, rs) }
func at(xs: [[string<structure>]], k: int) -> [string<structure>] = match nth(xs, k) { Some(r) => r, None => [] }
func pick(s: S) -> int = length(filter(\k. length(at(s.rows, k)) > 0, range(0, length(s.rows))))
export func go(rs: [[Cell]]) -> int ! {Declassify} = pick(shapeAll(rs))
`

func TestIFCPrune_DeclassifiedDataCarriesNoDemand(t *testing.T) {
	errs, _ := ifcWork(t, shapeFirstSrc)
	if len(errs) != 0 {
		t.Fatalf("declassified shapes indexed by a <structure> int were refused: %v", errs)
	}
}

// The same data bound by an annotated let and passed to a function-free
// parameter. (Unannotated, rows would keep the conservative callable: the
// imported length(rows) is assumed to possibly return its function argument,
// since this pass has no result types for imported functions. See the design
// doc's residual.)
func TestIFCPrune_ParameterTypePrunes(t *testing.T) {
	src := shapeFirstSrc + `export func go2(rs: [[Cell]]) -> int ! {Declassify} = {
  let rows: [[string<structure>]] = mapE(shapeRow, rs);
  length(filter(\k. length(at(rows, k)) > 0, range(0, length(rows))))
}
`
	errs, _ := ifcWork(t, src)
	if len(errs) != 0 {
		t.Fatalf("an unannotated list of shapes indexed by a <structure> int was refused: %v", errs)
	}
}

// The real demand is kept: <structure> data passed straight to the
// declassifier is still refused.
func TestIFCPrune_KeepsDirectDemand(t *testing.T) {
	src := strings.Replace(shapeFirstSrc,
		"export func go(rs: [[Cell]]) -> int ! {Declassify} = pick(shapeAll(rs))",
		`export func go(rs: [[Cell]]) -> int ! {Declassify} = pick(shapeAll(rs))
func bad(s: string<structure>) -> string<structure> ! {Declassify} = mark(s)`, 1)
	errs, _ := ifcWork(t, src)
	if len(errs) != 1 || !strings.Contains(errs[0].Message, `parameter "raw"`) || !strings.HasPrefix(errs[0].Position, "fold.ail:11:") {
		t.Fatalf("want the one real violation at mark(s), got %v", errs)
	}
}

// A list whose type holds functions keeps its callables: the sinks applied
// through map are still checked (#1719 list-element route).
func TestIFCPrune_FunctionListKeepsDemands(t *testing.T) {
	src := `module t
import std/list (map)
func secret() -> string<client> = "k"
func sink(s: string{not client}) -> int = 0
func run(fs: [(string{not client}) -> int]) -> [int] = map(\f. f(secret()), fs)
func use() -> [int] = run([sink])
`
	errs, _ := ifcWork(t, src)
	if len(errs) == 0 {
		t.Fatal("client data reached a {not client} sink through a list of functions unreported")
	}
}

func TestIFCPrune_DataOnlyDecisions(t *testing.T) {
	file := parseFoldModule(t, `module t
type Cell = { text: string<client>, n: int }
type S = { rows: [[Cell]] }
type Box[a] = { v: a }
type Tree = Leaf | Node(Tree, int, Tree)
type Fn = { f: (int) -> int }
type Color = Red | Green
func f(a: string, b: [[Cell]], c: S, d: Box[string], e: Box[(int) -> int], g: Tree, h: Fn, i: Color, j: Option[string], k: (int) -> int, l: (string, [int])) -> () = ()
`)
	ts := newIFCTypes(file)
	want := map[string]bool{"a": true, "b": true, "c": true, "d": true, "e": false, "g": true, "h": false, "i": true, "j": false, "k": false, "l": true}
	for _, p := range file.Funcs[0].Params {
		if got := ts.dataOnly(p.Type); got != want[p.Name] {
			t.Errorf("dataOnly(%s: %s) = %v, want %v", p.Name, p.Type, got, want[p.Name])
		}
	}
	if ts.dataOnly(nil) || ts.dataOnly(&ast.TypeVar{Name: "a"}) {
		t.Error("an unknown type must not count as function-free")
	}
}

// DX note 1 (2026-11-05): the limit named neither the function nor the site.
func TestIFCLimit_NamesFunction(t *testing.T) {
	msg := ifcAnalysisLimit{reason: "summary size", where: "describeTable"}.diagnostic().Message
	if !strings.Contains(msg, "while analysing function describeTable") {
		t.Fatalf("limit diagnostic does not name the function: %s", msg)
	}
}
