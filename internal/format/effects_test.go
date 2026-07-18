package format

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
)

// effects_test.go guards the empty-effect-row `! {}` case (controller-found
// defect). The parser produces a NON-nil empty slice for `! {}` but nil for no
// annotation; ast.FormatEffects collapses both to "", which dropped `! {}` from
// formatted output and broke round-trip verification. formatEffectRow preserves
// the distinction, and these tests fail loudly if a future change regresses it.
//
// Note: the corpus tree (testdata/corpus) has no comment-free `! {}` example, so
// the corpus round-trip walk never exercised this path — these dedicated
// fixtures are the guard.

// TestFormatEffectRow_NilEmptyNonEmpty unit-checks the shared helper directly.
func TestFormatEffectRow_NilEmptyNonEmpty(t *testing.T) {
	cases := []struct {
		name    string
		effects []ast.EffectAnnotation
		want    string
	}{
		{"nil_no_annotation", nil, ""},
		{"non_nil_empty_pure_row", []ast.EffectAnnotation{}, "! {}"},
		{"single_effect", []ast.EffectAnnotation{{Name: "IO"}}, "! {IO}"},
		{"multiple_effects", []ast.EffectAnnotation{{Name: "IO"}, {Name: "FS"}}, "! {IO, FS}"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatEffectRow(tc.effects); got != tc.want {
				t.Errorf("formatEffectRow(%#v) = %q, want %q", tc.effects, got, tc.want)
			}
		})
	}
}

// TestEmptyEffectRow_RoundTrip covers the round-trip + idempotence properties for
// `! {}` across every node the parser can attach an empty effect row to, and
// asserts the formatted output actually CONTAINS `! {}` so a regression that
// drops it fails loudly (not silently via a still-passing round-trip on nil).
func TestEmptyEffectRow_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		// Equation-form function body (the design doc's motivating V2-V5 idiom).
		{"func_equation_body", "module m\nexport func main() -> int ! {} = let x = 1; let y = 2; x + y"},
		// Block-form function body.
		{"func_block_body", "module m\nfunc f() -> int ! {} { 42 }"},
		// Function type in a parameter annotation (funcTypeString site).
		{"func_type_param", "module m\nfunc apply(g: (int) -> int ! {}, x: int) -> int = g(x)"},
		// Func literal with an explicit empty row (funcLit site).
		{"func_lit_empty_row", "module m\nfunc h() = func(x: int) -> int ! {} { x }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := assertIdempotentAndRoundTrips(t, tc.src, "test://"+tc.name)
			if !strings.Contains(out, "! {}") {
				t.Errorf("formatted output dropped empty effect row `! {}` for %s:\n%s", tc.name, out)
			}
		})
	}
}

// TestEmptyEffectRow_CanonicalGolden pins the exact canonical form so a change to
// spacing/placement of `! {}` is caught, and documents the preserved output.
func TestEmptyEffectRow_CanonicalGolden(t *testing.T) {
	// Single-expression equation body keeps the `! {} = expr` form on one line,
	// which pins the exact spacing/placement of the preserved empty effect row.
	src := "module m\nexport func f() -> int ! {} = 1"
	prog := parseProg(t, src, "test://golden")
	out, err := Source(prog, Options{})
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	const want = "module m\n\nexport func f() -> int ! {} = 1\n"
	if string(out) != want {
		t.Errorf("canonical form mismatch:\n--- got ---\n%q\n--- want ---\n%q", string(out), want)
	}
}
