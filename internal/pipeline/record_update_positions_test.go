package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

// M2_RECORD_UPDATE_FIX / #327 — position × callee-kind audit matrix.
//
// Root cause (found via the preserved deontic engine.ail artifact): the
// elaborator's intra-module call-graph builder (internal/elaborate/scc.go
// findReferences) was NOT an exhaustive AST traversal. It handled record
// LITERALS (*ast.Record) but had no case for *ast.RecordUpdate, *ast.Array,
// or match GUARD clauses. A module-local function referenced ONLY from an
// un-traversed position produced no call-graph edge, so Tarjan's SCC ordering
// could emit the callee's Let AFTER the caller's Let. The core type checker
// (typechecker_core.go CheckCoreProgram) threads globalEnv strictly in decl
// order, so the caller was checked while the callee was not yet bound →
// "undefined variable: <localFn>" for a function that plainly exists.
//
// Why clean-room repros passed: the bug only surfaces when the wrong
// topological order actually occurs. It reliably occurs when the local callee
// is DEFINED AFTER its caller and referenced ONLY from the position under
// test (no other edge pins the ordering). Every cell below defines the callee
// after the user for exactly this reason.
//
// The matrix is the shared regression net for the "resolution diverges by
// syntactic position" bug family (#323 was the pattern-position member).
//
// Callee kinds:
//   - localFunc  : a module-local `func` (needs an SCC edge → the trigger)
//   - importedFunc: resolved as VarGlobal at elaboration (no SCC needed)
//   - constructor: resolved via the $adt factory (no SCC needed)
//   - lambdaBound: a function parameter, bound in the local type env
//
// Every cell asserts type-check SUCCESS. Before the fix, the
// (recordUpdateField × localFunc), (array × localFunc) and
// (matchGuard × localFunc) cells are RED; all others are GREEN.

// twoModuleCheck writes a `types` sibling and an `engine` entry module into a
// temp package and runs the full pipeline in check mode (mirrors
// `ailang check --package .`). Returns the pipeline error (nil on success).
func twoModuleCheck(t *testing.T, engineSrc string) error {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "ailang-recupd-pos")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	manifest := `[package]
name = "audit/recupd"
version = "0.1.0"
edition = "1"

[exports]
modules = ["audit/recupd/types", "audit/recupd/engine"]
`

	// The sibling module provides: a record-alias State whose field is itself
	// an imported-ADT-bearing shape (mirrors deontic's term: Term), an ADT with
	// a unary constructor, an imported function, and a State builder. This is
	// the cross-module package shape from the bug report.
	typesSrc := `module audit/recupd/types

export type Box = Empty | Full(int)

export type State = {
  xs: [int],
  flag: bool,
  box: Box
}

export pure func mk() -> State = { xs: [], flag: true, box: Empty }

export pure func imp(xs: [int]) -> [int] = xs

-- Wrap an [int] back into a State via a record LITERAL. Positions under test
-- route their [int] result through this so every fixture returns a State while
-- isolating the single syntactic position being audited. (Record-literal
-- traversal is correct, so this wrapper never masks a real red elsewhere.)
export pure func wrap(xs: [int]) -> State = { xs: xs, flag: true, box: Empty }
`

	if err := os.WriteFile(filepath.Join(tempDir, "ailang.toml"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "types.ail"), []byte(typesSrc), 0644); err != nil {
		t.Fatalf("write types: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "engine.ail"), []byte(engineSrc), 0644); err != nil {
		t.Fatalf("write engine: %v", err)
	}

	src := Source{Filename: "engine.ail"}
	cfg := Config{Mode: ModeCheck, RelaxModules: true}
	_, runErr := Run(cfg, src)
	return runErr
}

// engineHeader is the common preamble for every engine fixture.
const engineHeader = `module audit/recupd/engine

import ./types (State, mk, imp, wrap, Box, Empty, Full)
import std/array as A
`

// calleeDefs supplies, per callee kind, (a) the expression that invokes the
// callee producing an [int], and (b) any trailing local definition needed.
//
// CRITICAL: for localFunc the definition is emitted AFTER the user function
// so that a MISSING call-graph edge forces the callee into a later decl. This
// is what makes the buggy cells actually red.
type calleeKind struct {
	name string
	// intExpr yields an [int] value referencing the callee (for positions that
	// consume an [int]).
	intExpr string
	// boolExpr yields a bool value referencing the callee (for if/guard).
	boolExpr string
	// trailer is extra top-level source appended AFTER the user function.
	trailer string
}

func localFuncKind() calleeKind {
	return calleeKind{
		name:     "localFunc",
		intExpr:  "helper(s.xs)",
		boolExpr: "isBig(s.xs)",
		trailer: `
func helper(xs: [int]) -> [int] = xs
func isBig(xs: [int]) -> bool = match xs { [] => false, _ :: _ => true }
`,
	}
}

func importedFuncKind() calleeKind {
	return calleeKind{
		name:     "importedFunc",
		intExpr:  "imp(s.xs)",
		boolExpr: "s.flag",
		trailer:  "",
	}
}

func constructorKind() calleeKind {
	// Constructor produces a Box, used in the box field. For [int] positions we
	// still need an [int]; constructors are resolved as $adt factories with no
	// SCC edge, so we exercise them via the box field only where a Box is
	// expected, and use a trivial [int] elsewhere. To keep the callee-kind on
	// the SAME [int] positions as the others, we wrap Full(_) in a helper-free
	// expression that yields [int]: match on the constructed Box.
	return calleeKind{
		name:     "constructor",
		intExpr:  "(match Full(1) { Empty => s.xs, Full(_) => s.xs })",
		boolExpr: "(match Full(1) { Empty => false, Full(_) => true })",
		trailer:  "",
	}
}

func lambdaBoundKind() calleeKind {
	// A function parameter bound in the local env. We add an [int]->[int] param
	// `f` and a bool param `b`? Simpler: bind via a let-lambda inside the body
	// is not possible for all positions uniformly, so we thread the callee as a
	// parameter of `go`. The invocation references the parameter directly.
	return calleeKind{
		name:     "lambdaBound",
		intExpr:  "f(s.xs)",
		boolExpr: "match f(s.xs) { [] => false, _ :: _ => true }",
		trailer:  "",
	}
}

// goSignature returns the `go` function signature line for a callee kind.
// lambdaBound needs an extra parameter carrying the function.
func goSignature(k calleeKind) string {
	if k.name == "lambdaBound" {
		return "export pure func go(s: State, f: (([int]) -> [int])) -> State ="
	}
	return "export pure func go(s: State) -> State ="
}

// position builds the body of `go` that places the callee's intExpr/boolExpr
// into a specific syntactic position, always yielding a `State`.
type position struct {
	name string
	// body renders the RHS of `go` (an expression of type State) using the
	// callee's intExpr and boolExpr.
	body func(k calleeKind) string
}

func positions() []position {
	return []position{
		{
			// Callee reference sits directly in a record-LITERAL field.
			name: "recordLiteralField",
			body: func(k calleeKind) string {
				return "{ xs: " + k.intExpr + ", flag: s.flag, box: s.box }"
			},
		},
		{
			// Callee reference sits directly in a record-UPDATE field (the bug).
			name: "recordUpdateField",
			body: func(k calleeKind) string {
				return "({ s | xs: " + k.intExpr + " })"
			},
		},
		{
			// Callee reference is the BODY of a match arm; result routed via wrap.
			name: "matchArmBody",
			body: func(k calleeKind) string {
				return "match s.flag { true => wrap(" + k.intExpr + "), false => s }"
			},
		},
		{
			// Callee reference is a match GUARD expression.
			name: "matchGuard",
			body: func(k calleeKind) string {
				return "match s.box { _ if " + k.boolExpr + " => s, _ => s }"
			},
		},
		{
			// Callee reference is an if CONDITION.
			name: "ifCondition",
			body: func(k calleeKind) string {
				return "if " + k.boolExpr + " then s else s"
			},
		},
		{
			// Callee reference is a let RHS; result routed via wrap.
			name: "letRHS",
			body: func(k calleeKind) string {
				return "let nx = " + k.intExpr + "; wrap(nx)"
			},
		},
		{
			// Callee reference is an ELEMENT of a list literal; the singleton
			// list is destructured by match to recover the [int].
			name: "listElement",
			body: func(k calleeKind) string {
				return "match [" + k.intExpr + "] { [nx] => wrap(nx), _ => s }"
			},
		},
		{
			// Callee reference is an ELEMENT of a tuple; destructured by match.
			name: "tupleElement",
			body: func(k calleeKind) string {
				return "match (" + k.intExpr + ", s.flag) { (nx, _) => wrap(nx) }"
			},
		},
		{
			// Callee reference is an ELEMENT of an ARRAY literal. Arrays were
			// also absent from the call-graph traversal, so this is a third
			// latent member of the family the audit surfaces.
			name: "arrayElement",
			body: func(k calleeKind) string {
				return "let a = #[" + k.intExpr + "]; wrap(A.get(a, 0))"
			},
		},
		{
			// Callee reference is an ARGUMENT to another function call.
			name: "functionArgument",
			body: func(k calleeKind) string {
				return "wrap(imp(" + k.intExpr + "))"
			},
		},
	}
}

func buildEngine(k calleeKind, p position) string {
	// Wrap the body in a block `{ ... }` so positions that need `let`/`;`
	// (letRHS, listElement, tupleElement) parse uniformly. A block containing a
	// single expression is equivalent to that expression.
	return engineHeader + "\n" +
		goSignature(k) + " {\n  " + p.body(k) + "\n}\n" +
		k.trailer
}

// TestRecordUpdatePositions_Matrix is the audit matrix: every callee kind in
// every expression position must type-check. After the findReferences fix all
// cells are green.
func TestRecordUpdatePositions_Matrix(t *testing.T) {
	kinds := []calleeKind{
		localFuncKind(),
		importedFuncKind(),
		constructorKind(),
		lambdaBoundKind(),
	}

	for _, k := range kinds {
		for _, p := range positions() {
			k := k
			p := p
			t.Run(k.name+"/"+p.name, func(t *testing.T) {
				engineSrc := buildEngine(k, p)
				if err := twoModuleCheck(t, engineSrc); err != nil {
					t.Fatalf("callee=%s position=%s must type-check, got:\n%v\n\n--- engine.ail ---\n%s",
						k.name, p.name, err, engineSrc)
				}
			})
		}
	}
}

// TestRecordUpdatePositions_HoistedWorkaroundStillWorks proves the deontic
// package's existing hoisted-let workaround remains valid after the fix (both
// forms must coexist). This is the "keep one hoisted fixture" requirement.
func TestRecordUpdatePositions_HoistedWorkaroundStillWorks(t *testing.T) {
	engineSrc := engineHeader + `
export pure func go(s: State) -> State = {
  let nx = helper(s.xs);
  { s | xs: nx }
}

func helper(xs: [int]) -> [int] = xs
`
	if err := twoModuleCheck(t, engineSrc); err != nil {
		t.Fatalf("hoisted-let form must still type-check, got: %v", err)
	}
}
