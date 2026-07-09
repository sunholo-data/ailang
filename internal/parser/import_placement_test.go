package parser

import (
	"strings"
	"testing"
)

// TestImportPlacementAfterTypeDecl verifies that an import appearing after a
// type declaration produces the PAR_IMPORT_PLACEMENT diagnostic (which states
// the rule and carries the fix), NOT the old opaque PAR_NO_PREFIX_PARSE cascade.
//
// Regression for #325: claude-haiku-4-5/sonnet lost whole benchmark cells to
// `PAR_NO_PREFIX_PARSE: unexpected token in expression: import` because the
// error never stated that imports must come first.
func TestImportPlacementAfterTypeDecl(t *testing.T) {
	input := `module benchmark/solution

type Op = Add | Sub | Mul

import std/list (map, filter, foldl)
`
	errs := mustParseError(t, input)

	if !hasErrCode(errs, "PAR_IMPORT_PLACEMENT") {
		t.Fatalf("expected PAR_IMPORT_PLACEMENT, got: %v", errs)
	}
	// The placement diagnostic must REPLACE the no-prefix-parse cascade for the
	// import token — not sit alongside it.
	if hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("PAR_NO_PREFIX_PARSE must not appear for a misplaced import; got: %v", errs)
	}
	msg := errs[0].Error()
	if !strings.Contains(msg, "imports must appear immediately after the module declaration") {
		t.Errorf("expected the placement rule in the message, got: %s", msg)
	}
	if !strings.Contains(msg, "move this import above the first type/func declaration") {
		t.Errorf("expected the hoist suggestion in the message, got: %s", msg)
	}
}

// TestImportPlacementAfterFuncDecl verifies the diagnostic fires after a func
// declaration too (not only type decls).
func TestImportPlacementAfterFuncDecl(t *testing.T) {
	input := `module benchmark/solution

func helper(x: int) -> int { x }

import std/list (map)
`
	errs := mustParseError(t, input)
	if !hasErrCode(errs, "PAR_IMPORT_PLACEMENT") {
		t.Fatalf("expected PAR_IMPORT_PLACEMENT after func decl, got: %v", errs)
	}
	if hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Errorf("PAR_NO_PREFIX_PARSE must not appear for a misplaced import; got: %v", errs)
	}
}

// TestImportPlacementMultiple verifies that MULTIPLE misplaced imports each get
// a placement diagnostic (documented choice: one per import) and produce ZERO
// PAR_NO_PREFIX_PARSE errors for those tokens.
func TestImportPlacementMultiple(t *testing.T) {
	input := `module benchmark/solution

type Op = Add | Sub | Mul

import std/list (map, filter, foldl)
import std/string (join)
import std/option (getOrElse)
`
	errs := mustParseError(t, input)

	placement := countErrCode(errs, "PAR_IMPORT_PLACEMENT")
	if placement != 3 {
		t.Errorf("expected 3 PAR_IMPORT_PLACEMENT errors (one per misplaced import), got %d: %v", placement, errs)
	}
	if noprefix := countErrCode(errs, "PAR_NO_PREFIX_PARSE"); noprefix != 0 {
		t.Errorf("expected 0 PAR_NO_PREFIX_PARSE for misplaced imports, got %d: %v", noprefix, errs)
	}
}

// TestImportPlacementValidProgramUntouched verifies that a valid program (imports
// first) parses cleanly — the new diagnostic must fire ONLY on the error path.
func TestImportPlacementValidProgramUntouched(t *testing.T) {
	input := `module benchmark/solution

import std/list (map, filter, foldl)
import std/string (join)

type Op = Add | Sub | Mul

export func main() -> int { 42 }
`
	// Should parse with no errors at all.
	_ = mustParse(t, input)
}

// TestStrayTokenStillNoPrefixParse is the conflict-surface guard: a genuinely
// stray token that cannot start an expression must STILL produce
// PAR_NO_PREFIX_PARSE. The placement fix must not swallow other error paths.
func TestStrayTokenStillNoPrefixParse(t *testing.T) {
	// `]` at declaration level cannot start an expression and is not an import.
	input := `module benchmark/solution

type Op = Add

]
`
	errs := mustParseError(t, input)
	if !hasErrCode(errs, "PAR_NO_PREFIX_PARSE") {
		t.Fatalf("expected PAR_NO_PREFIX_PARSE for a genuinely stray token, got: %v", errs)
	}
	if hasErrCode(errs, "PAR_IMPORT_PLACEMENT") {
		t.Errorf("PAR_IMPORT_PLACEMENT must not fire for non-import stray tokens; got: %v", errs)
	}
}

// hasErrCode reports whether any error's string contains the given diagnostic code.
func hasErrCode(errs []error, code string) bool {
	return countErrCode(errs, code) > 0
}

// countErrCode counts how many errors carry the given diagnostic code.
func countErrCode(errs []error, code string) int {
	n := 0
	for _, e := range errs {
		if pe, ok := e.(*ParserError); ok {
			if pe.Code == code {
				n++
			}
			continue
		}
		if strings.Contains(e.Error(), code) {
			n++
		}
	}
	return n
}
