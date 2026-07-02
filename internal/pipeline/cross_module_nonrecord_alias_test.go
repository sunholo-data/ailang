package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// M-XMOD-ALIAS regression pack.
//
// Cross-module transparent type aliases to NON-record targets (`type Id = int`,
// `type Row = Json`) must expand across module boundaries, exactly as record
// aliases already do (M-TYPE-ALIAS / M-TRANSITIVE-ALIAS-ENV-IMPORT). Pre-fix the
// interface builder only registered `*ast.RecordType` aliases, so an imported
// non-record alias stayed a nominal TCon and failed to unify with its target.
//
// The fixtures below intentionally cover both directions:
//   - POSITIVE: non-record aliases must now unify with their target cross-module.
//   - NEGATIVE (non-regression): nominal ADTs / newtypes must STAY nominal — the
//     fix keys off *ast.TypeAlias only, which is a disjoint parser node from the
//     *ast.AlgebraicType that sum types and `Ctor(T)` newtypes produce.

// checkModules writes the given {relpath: content} files under a temp dir and
// runs the pipeline in ModeCheck against main.ail. Returns the compile error
// (nil on success).
func checkModules(t *testing.T, files map[string]string) error {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "ailang-xmod-alias-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	for rel, content := range files {
		full := filepath.Join(tempDir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(originalDir)

	src := Source{Filename: "main.ail"}
	cfg := Config{Mode: ModeCheck}
	_, compileErr := Run(cfg, src)
	return compileErr
}

// TestXModAlias_NonRecordAliasInField is the core regression: module A declares
// `type Id = int` (a non-record alias) inside a record alias `Box = { items: [Id] }`,
// and main builds a Box from a plain [int]. Pre-fix this fails to unify [int] with
// [Id] because Id crosses the module boundary as a nominal TCon. This is the exact
// shape of the sunholo/duckdb `Json vs Row` failure that broke eparse.
func TestXModAlias_NonRecordAliasInField(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Id = int
export type Box = { items: [Id] }
`,
		"main.ail": `module main

import pkg_a/types (Box)

export func mk(xs: [int]) -> Box {
    { items: xs }
}

export func main() -> int {
    let b = mk([1, 2, 3]);
    0
}
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("non-record alias in record field failed to type-check: %v", err)
	}
}

// TestXModAlias_NonRecordTargetKinds covers the non-record alias target kinds
// that the parser accepts in alias position — TCon (`int`, and by extension a
// named type like `Json`) and TList (`[string]`) — each declared in module A and
// used against its underlying type in main. Pre-fix both fail cross-module.
//
// NOTE: tuple-type aliases (`type Pair = (int, string)`) and function-type
// aliases (`type Pred = (int) -> bool`) are omitted: neither PARSES even within a
// single module (PAR_TYPE_BODY_EXPECTED / PAR_NO_PREFIX_PARSE) — separate
// parser-level gaps, out of scope for M-XMOD-ALIAS.
func TestXModAlias_NonRecordTargetKinds(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type UserId = int
export type Names  = [string]
`,
		"main.ail": `module main

import pkg_a/types (UserId, Names)

export func useId(x: int) -> UserId { x }
export func useNames(xs: [string]) -> Names { xs }

export func main() -> int { 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("non-record alias target kinds failed to type-check: %v", err)
	}
}

// TestXModAlias_TupleAndFuncAliasTargets (M-PARSER-ALIAS-TARGETS): tuple- and
// function-type aliases must parse in alias-target position and — composing with
// M-XMOD-ALIAS — cross module boundaries. Pre-fix these do not PARSE
// (parseTypeDeclBody had no LPAREN case).
func TestXModAlias_TupleAndFuncAliasTargets(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Pair = (int, string)
export type Pred = (int) -> bool
`,
		"main.ail": `module main

import pkg_a/types (Pair, Pred)

export func usePair(p: (int, string)) -> Pair { p }
export func usePred(f: (int) -> bool) -> Pred { f }

export func main() -> int { 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("tuple/function alias targets failed: %v", err)
	}
}

// TestXModAlias_ChainedAlias (M-XMOD-ALIAS-CHAIN): a chained alias Ref -> Id -> int
// expands transitively. `expandAlias` iterates to a fixpoint, so `[int]` unifies
// with `[Ref]` cross-module.
func TestXModAlias_ChainedAlias(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Id  = int
export type Ref = Id
export type Box = { items: [Ref] }
`,
		"main.ail": `module main

import pkg_a/types (Box)

export func mk(xs: [int]) -> Box { { items: xs } }

export func main() -> int { let b = mk([1, 2]); 0 }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("chained non-record alias failed to type-check: %v", err)
	}
}

// TestXModAlias_ChainedRecordAlias: a chain ending in a record (U2 -> Usage -> {…})
// resolves and preserves structural access cross-module.
func TestXModAlias_ChainedRecordAlias(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Usage = { count: int }
export type U2    = Usage

export pure func zero() -> U2 { { count: 0 } }
`,
		"main.ail": `module main

import pkg_a/types (U2, zero)

export func total(u: U2) -> int { u.count }

export func main() -> int { total(zero()) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("chained record alias failed to type-check: %v", err)
	}
}

// TestXModAlias_CyclicAliasTerminates: a self-referential alias `type A = A` must
// not hang the unifier (cycle guard in expandAlias). It should fail gracefully as a
// type error, not loop forever. The test's own timeout would catch a hang.
func TestXModAlias_CyclicAliasTerminates(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type A = A
export type Box = { item: A }
`,
		"main.ail": `module main

import pkg_a/types (Box)

export func mk(x: int) -> Box { { item: x } }

export func main() -> int { 0 }
`,
	}
	// We don't assert on the exact outcome — only that checkModules RETURNS
	// (no infinite loop). A hang would fail the test via the go test timeout.
	_ = checkModules(t, files)
}

// --- Negative / non-regression fixtures -------------------------------------
//
// The fix keys off *ast.TypeAlias ONLY. Sum types and `Ctor(T)` newtypes parse
// to *ast.AlgebraicType (a disjoint node), so they must remain NOMINAL — never
// silently made structural. These two tests would catch a fix that over-reached.

// TestXModAlias_NominalNewtypeStaysNominal: `type Gen = Gen(int)` is a nominal
// newtype. A bare int must NOT unify with Gen cross-module — post-fix this must
// STILL be a type error (it is an *ast.AlgebraicType, not an alias).
func TestXModAlias_NominalNewtypeStaysNominal(t *testing.T) {
	files := map[string]string{
		"pkg_a/types.ail": `module pkg_a/types

export type Gen = Gen(int)
`,
		"main.ail": `module main

import pkg_a/types (Gen)

export func bad(x: int) -> Gen { x }

export func main() -> int { 0 }
`,
	}
	err := checkModules(t, files)
	if err == nil {
		t.Fatal("REGRESSION: nominal newtype `type Gen = Gen(int)` unified with bare int cross-module — the fix leaked nominal→structural")
	}
	if !strings.Contains(err.Error(), "Gen") {
		t.Fatalf("expected a nominal mismatch mentioning Gen, got: %v", err)
	}
}

// TestXModAlias_NominalSumADTStaysNominal: a sum ADT must keep working nominally
// cross-module — constructors resolve, match works, no structural unification is
// introduced. Positive guard that ADT handling is untouched.
func TestXModAlias_NominalSumADTStaysNominal(t *testing.T) {
	files := map[string]string{
		"pkg_a/color.ail": `module pkg_a/color

export type Color = Red | Green | Blue

export func toInt(c: Color) -> int {
    match c { Red => 0, Green => 1, Blue => 2 }
}
`,
		"main.ail": `module main

import pkg_a/color (Color, Red, Green, toInt)

export func main() -> int { toInt(Red) + toInt(Green) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("nominal sum ADT broke cross-module: %v", err)
	}
}

// TestXModAlias_RecordAliasUnchanged: the M-TYPE-ALIAS record path must still
// expand cross-module (record alias field access through an import). Guards that
// adding the non-record case did not disturb the record case.
func TestXModAlias_RecordAliasUnchanged(t *testing.T) {
	files := map[string]string{
		"pkg_a/geo.ail": `module pkg_a/geo

export type Point = { x: int, y: int }

export pure func origin() -> Point { { x: 0, y: 0 } }
`,
		"main.ail": `module main

import pkg_a/geo (Point, origin)

export func mv(p: Point) -> int { p.x + p.y }

export func main() -> int { mv(origin()) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("record alias regressed cross-module: %v", err)
	}
}
