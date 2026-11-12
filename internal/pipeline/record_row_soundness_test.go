package pipeline

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/types"
)

// #2210 and #2221: record rows were never bound, the occurs check had no record
// cases, and every field access shared one row named "r". One record
// unification (internal/types/unification_record_rows.go) fixes them; these
// are the issues' programs plus the row-polymorphic programs that must keep
// working.

// checkModule type-checks one module named m (cache off) and returns the error.
func checkModule(t *testing.T, src string) error {
	t.Helper()
	_, err := writeAndCheck(t, map[string]string{"m.ail": src}, "m.ail")
	return err
}

func wantCode(t *testing.T, err error, code, detail string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s (%s), got a passing check", code, detail)
	}
	if !strings.Contains(err.Error(), code) || !strings.Contains(err.Error(), detail) {
		t.Fatalf("expected %s mentioning %q, got: %v", code, detail, err)
	}
}

// #2210: a declared closed return flows through a lambda parameter. The shim
// `{score | ρ}` never had ρ bound, so `best` exported the shim and
// `best().bogus` type-checked, then failed at run time.
func TestRecordRows_2210_DeclaredReturnThroughLambda(t *testing.T) {
	src := `module m

export func best() -> {label: string, score: float} {
  let f = \acc. if acc.score > 1.0 then acc else acc
  in f({label: "none", score: 0.0})
}

export func main() -> () ! {IO} {
  println(best().bogus)
}
`
	wantCode(t, checkModule(t, src), types.TC_REC_001, "field 'bogus' not found in record {label, score}")
}

// #2210, the foldl form the issue names.
func TestRecordRows_2210_ThroughFoldl(t *testing.T) {
	src := `module m

import std/list (foldl)

export func top(xs: [{label: string, score: float}]) -> {label: string, score: float} =
  foldl(\acc x. if x.score > acc.score then x else acc, {label: "none", score: 0.0}, xs)

export func main() -> () ! {IO} {
  println(top([]).bogus)
}
`
	wantCode(t, checkModule(t, src), types.TC_REC_001, "field 'bogus'")
}

// #2210 across a module boundary: the exported type is the closed record, so
// the importer's access to a missing field is rejected.
func TestRecordRows_2210_AcrossModules(t *testing.T) {
	lib := `module q/lib

export func best() -> {label: string, score: float} {
  let f = \acc. if acc.score > 1.0 then acc else acc
  in f({label: "none", score: 0.0})
}
`
	main := `module q/main

import q/lib (best)

export func main() -> () ! {IO} {
  println(best().bogus)
}
`
	_, err := writeAndCheck(t, map[string]string{"q/lib.ail": lib, "q/main.ail": main}, "q/main.ail")
	wantCode(t, err, types.TC_REC_001, "field 'bogus'")
}

// #2221 part 1: an infinite type through a record. Each program used to run
// past 20 s; each must now fail the occurs check promptly.
func TestRecordRows_2221_OccursCheckTerminates(t *testing.T) {
	cases := map[string]string{
		"self_eq": `module m

export func main() -> () ! {IO} {
  let h = \x. {self: x} == x;
  println("x")
}
`,
		"letrec_inner": `module m

export func main() -> () ! {IO} {
  letrec f = \x. {inner: f(x)} in println("x")
}
`,
		"through_open_param": `module m

func g(p: {self: a, ...}) -> a = p.self

export func main() -> () ! {IO} {
  let h = \x. g(x) == x;
  println("x")
}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			done := make(chan error, 1)
			go func() { done <- checkModule(t, src) }()
			select {
			case err := <-done:
				wantCode(t, err, types.TC_OCCURS, "occurs check failed")
			case <-time.After(30 * time.Second):
				t.Fatal("check did not terminate within 30s")
			}
		})
	}
}

// #2221 part 2: a record update keeps the other fields. `upd` returned the
// shim `{x: int | ρ}` with ρ unbound, so `.y` was a free type and `+ 1` was
// accepted against a string field.
func TestRecordRows_2221_UpdateKeepsOtherFields(t *testing.T) {
	src := `module m

export func main() -> () ! {IO} {
  let upd = \q. {q | x: 1};
  println(show(upd({x: 0, y: "s"}).y + 1))
}
`
	err := checkModule(t, src)
	if err == nil || !strings.Contains(err.Error(), "Num[string]") {
		t.Fatalf("expected No instance for Num[string], got: %v", err)
	}
}

// A record UPDATE has its own fresh row too (review mutant M17: the update's
// row back to the literal "r"). With a shared row, the update of q met p's
// user-written `| r`, or two updates on different records met each other.
func TestRecordRows_UpdateRowsAreFresh(t *testing.T) {
	cases := map[string]string{
		"update_meets_user_r": `module m

func f(p: {x: int | r}, q: {y: int, z: string}) -> int {
  let q2 = {q | y: 1};
  p.x + q2.y
}

export func main() -> () ! {IO} = println(show(f({x: 1, w: true}, {y: 2, z: "s"})))
`,
		"two_updates_two_records": `module m

export func main() -> () ! {IO} {
  let g = \p q. { let a = {p | x: 1}; let b = {q | y: 2}; a.x + b.y };
  println(show(g({x: 0, a: true}, {y: 0, b: "s"})))
}
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if err := checkModule(t, src); err != nil {
				t.Fatalf("valid program rejected: %v", err)
			}
		})
	}
}

// A row variable must lack the labels of every record it extends. Here r
// extends both {y: int | r} and {x: int | r}; the call binds r := {x: string}
// through p, which gives q the field x twice (int and string). The two must be
// one field, so the call is TC_REC_004. (The definitions themselves are
// accepted: r is not bound inside them — a lacks constraint at definition time
// is not implemented.) Before, which x a later access saw depended on which map
// was read first: d3 typed `.x + 1` against int while r said string.
func TestRecordRows_RowBoundToAFieldTheRecordHasIsAnError(t *testing.T) {
	cases := map[string]string{
		"d1_access": `module m

func f(p: {y: int | r}, q: {x: int | r}) -> string = p.x

export func main() -> () ! {IO} = println(f({y: 1, x: "s"}, {x: 2}))
`,
		"d3_return": `module m

func f(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q

export func main() -> () ! {IO} = println(show(f({y: 1, x: "s"}, {x: 2}).x + 1))
`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			wantCode(t, checkModule(t, src), types.TC_REC_004, "field 'x' would have two types")
		})
	}
	// The same shape with agreeing types is one field, and is accepted.
	ok := `module m

func f(p: {y: int | r}, q: {x: int | r}) -> {x: int | r} = q

export func main() -> () ! {IO} = println(show(f({y: 1, x: 5}, {x: 2}).x + 1))
`
	if err := checkModule(t, ok); err != nil {
		t.Fatalf("agreeing duplicate rejected: %v", err)
	}
}

// #2221 part 3: a user-written `| r` no longer meets the checker's own "r"
// (and the annotation's row has the same kind value as every other record row).
func TestRecordRows_2221_UserRowNamedR(t *testing.T) {
	src := `module m

func bump(p: {x: int | r}) -> {x: int | r} = {p | x: p.x + 1}

export func main() -> () ! {IO} {
  println(show(bump({x: 1, y: "s"}).x))
}
`
	if err := checkModule(t, src); err != nil {
		t.Fatalf("valid program rejected: %v", err)
	}
}

// Negative controls: row polymorphism, record update and field access through
// lambdas that must keep type-checking.
func TestRecordRows_RowPolymorphicProgramsStillCheck(t *testing.T) {
	src := `module m

import std/list (foldl)

pure func getName(p: {name: string | r}) -> string = p.name

pure func bump(p: {x: int | r}) -> {x: int | r} = {p | x: p.x + 1}

pure func getEmail(u: {email: string, ...}) -> string = u.email

pure func both(p, q) = p.x + q.y

export func best() -> {label: string, score: float} {
  let f = \acc. if acc.score > 1.0 then acc else acc
  in f({label: "none", score: 0.0})
}

export func top(xs: [{label: string, score: float}]) -> {label: string, score: float} =
  foldl(\acc x. if x.score > acc.score then x else acc, {label: "none", score: 0.0}, xs)

export func upd() -> string {
  let u = \q. {q | x: 1};
  let r = u({x: 0, y: "s"});
  "${r.y}${show(r.x)}"
}

export func twoShapes() -> int {
  let getX = \r. r.x;
  getX({x: 1, a: "a"}) + getX({x: 2, b: true, c: 3.0})
}

pure func nameOf(p: {name: string, age: int}) -> string = match p {
  {name, ...} => name
}

export func nested() -> int {
  let g = \o. o.inner.v;
  g({inner: {v: 7, w: "w"}, other: 1})
}

export func main() -> () ! {IO} {
  println(getName({name: "Bob", age: 30}));
  println(show(bump({x: 1, y: "s"}).x));
  println(getEmail({email: "e", name: "T", id: 1}));
  println(show(both({x: 1, z: "a"}, {y: 2, w: true})));
  println(best().label);
  println(top([{label: "a", score: 2.0}]).label);
  println(upd());
  println(show(twoShapes()));
  println(nameOf({name: "N", age: 3}));
  println(show(nested()))
}
`
	if err := checkModule(t, src); err != nil {
		t.Fatalf("row-polymorphic controls rejected: %v", err)
	}
}

// Evaluated controls: the values flow through the evaluator unchanged.
func TestRecordRows_EvaluatedControls(t *testing.T) {
	cases := map[string]string{
		`let u = \q. {q | x: 1} in u({x: 0, y: "s"}).y`:                                           "s",
		`let getX = \r. r.x in getX({x: 1, a: "a"}) + getX({x: 2, b: true})`:                      "3",
		`let f = \acc. if acc.score > 1.0 then acc else acc in f({label: "n", score: 0.5}).label`: "n",
		`let g = \o. o.inner.v in g({inner: {v: 7, w: "w"}, other: 1})`:                           "7",
	}
	for code, want := range cases {
		res, err := Run(Config{Mode: ModeEval}, Source{Code: code})
		if err != nil {
			t.Fatalf("%s: %v", code, err)
		}
		if got := res.Value.String(); got != want {
			t.Errorf("%s = %s, want %s", code, got, want)
		}
	}
}

// Rendering of an export over two shim records. Each access has its own rest
// row, so the two parameters no longer claim to share their other fields:
// with a shared row (bound, as it now is) `f({x: 1, z: "a"}, {y: 2, w: true})`
// would be rejected. Replaces TestTyvarHygiene_SharedShimRowRenderingUnchanged,
// which pinned the shared `...a` while rows were never bound.
func TestRecordRows_TwoShimRecordsHaveOwnRows(t *testing.T) {
	src := `module two

export pure func f(a, b) = a.x + b.y

export pure func use() -> int = f({x: 1, z: "a"}, {y: 2, w: true})
`
	res, err := writeAndCheck(t, map[string]string{"two.ail": src}, "two.ail")
	if err != nil {
		t.Fatal(err)
	}
	data, err := res.Interface.ToNormalizedJSON()
	if err != nil {
		t.Fatal(err)
	}
	var rendered struct {
		Funcs []struct{ Name, Type string } `json:"funcs"`
	}
	if err := json.Unmarshal(data, &rendered); err != nil {
		t.Fatal(err)
	}
	const want = "({x: b, ...a},{y: b, ...c})->b"
	for _, f := range rendered.Funcs {
		if f.Name == "f" && f.Type != want {
			t.Fatalf("f renders %s, want %s", f.Type, want)
		}
	}
}

// Why #2221 part 4 (defaulting's own free-variable walk) is not changed here:
// with the shared walk, std/sem's bump_version becomes `{ver: b, ...}` and its
// Num constraint does not reach an importer, so this program would pass check
// and fail at run time. Defaulting to int is what keeps it rejected.
func TestRecordRows_BumpVersionRejectsStringVer(t *testing.T) {
	src := `module m

import std/sem (bump_version)

export func main() -> () ! {IO} {
  println(bump_version({ver: "s", id: "x"}).ver)
}
`
	err := checkModule(t, src)
	if err == nil {
		t.Fatal("bump_version accepted a string ver; it fails at run time")
	}
}

// std/sem's encode_frame reads eight fields. Its exported type named only
// `id` (the shared row hid the rest), so a one-field record type-checked and
// failed at run time with `record has no field: ver`.
func TestRecordRows_EncodeFrameRequiresEveryField(t *testing.T) {
	src := `module m

import std/sem (encode_frame)

export func main() -> () ! {IO} {
  println(encode_frame({id: "x"}))
}
`
	wantCode(t, checkModule(t, src), types.TC_REC_001, "not found in record {id}")
}

// The interface of a module that exports an annotated open record is written
// to the cache. The annotation's row carried a *KRow kind that MarshalKind
// could not encode (CACHE_WRITE_FAILED), so the module was re-checked on every
// run (#2221, "also noted").
func TestRecordRows_OpenRecordExportIsCached(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("AILANG_CACHE_DIR", cacheDir)
	t.Setenv("AILANG_NO_CACHE", "")
	dir := t.TempDir()
	src := "module q/openexp\n\nexport pure func getX(p: {x: int | r}) -> int = p.x\n"
	if err := os.MkdirAll(filepath.Join(dir, "q"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "q", "openexp.ail"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, err := Run(Config{Mode: ModeCheck}, Source{Filename: "q/openexp.ail"}); err != nil {
		t.Fatal(err)
	}
	found := false
	for mod := range collectIfaceJSON(t, cacheDir) {
		if strings.Contains(mod, "openexp") {
			found = true
		}
	}
	if !found {
		t.Fatalf("q/openexp's interface was not written to the cache")
	}
}
