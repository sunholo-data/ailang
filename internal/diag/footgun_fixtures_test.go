// Package diag hosts the footgun coverage table (footguns.md) and its CI
// enforcement. Each footgun whose diagnostic is CI-enforced (status: covered or
// shipped-this-sprint in footguns.md) has a row here: an .ail snippet, the
// diagnostic code it must produce, and the fix substring its message/suggestion
// must carry. If a diagnostic silently drifts (loses its code or its fix text),
// this test goes red — turning "fix-carrying diagnostics" into a tested contract
// rather than an aspiration (m-diagnostic-coverage, R1.1).
//
// The test drives the real compile pipeline (ModeCheck) so it exercises exactly
// what an agent's `ailang check` sees. It lives in internal/diag (not
// internal/pipeline) so the coverage table and its enforcement sit together;
// internal/diag has no non-test code, so importing internal/pipeline here
// introduces no production dependency.
package diag

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
)

// footgunFixture is one CI-enforced row of footguns.md.
type footgunFixture struct {
	// name is the footguns.md fixture id.
	name string
	// code is the diagnostic code the pipeline MUST emit for this snippet.
	code string
	// fix is a substring that MUST appear in the diagnostic's message or
	// suggestion — the concrete, actionable fix the agent needs. This is the
	// "fix-carrying" contract.
	fix string
	// src is the .ail snippet that triggers the footgun. Inline (per spec:
	// "testdata/ or inline") to keep each contract readable in one place.
	src string
	// status mirrors footguns.md: "covered" (pre-existing gold standard) or
	// "shipped-this-sprint" (added by m-diagnostic-coverage).
	status string
}

// footgunFixtures is the CI-enforced subset of footguns.md. Only covered /
// shipped-this-sprint entries appear here; "inventoried" rows in footguns.md are
// not yet contracts and are intentionally excluded.
var footgunFixtures = []footgunFixture{
	{
		name:   "plusplus_strings",
		code:   "++ operator",
		fix:    "`++` is for lists only",
		status: "covered", // the gold standard
		src: `module benchmark/solution
export func main() -> string { "a" ++ "b" }
`,
	},
	{
		name:   "import_placement",
		code:   "PAR_IMPORT_PLACEMENT",
		fix:    "move this import above the first type/func declaration",
		status: "shipped-this-sprint", // #325
		src: `module benchmark/solution
type Op = Add | Sub
import std/list (map)
export func main() -> int { 1 }
`,
	},
	{
		name:   "import_placement_rule_stated",
		code:   "PAR_IMPORT_PLACEMENT",
		fix:    "imports must appear immediately after the module declaration",
		status: "shipped-this-sprint", // #325 — the rule statement itself
		src: `module benchmark/solution
func helper(x: int) -> int { x }
import std/string (join)
export func main() -> int { 1 }
`,
	},
}

// TestFootgunFixtures enforces that every covered/shipped footgun still produces
// its documented diagnostic code AND carries its fix substring.
func TestFootgunFixtures(t *testing.T) {
	for _, fx := range footgunFixtures {
		t.Run(fx.name, func(t *testing.T) {
			_, err := pipeline.Run(
				pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true},
				pipeline.Source{Code: fx.src},
			)
			if err == nil {
				t.Fatalf("%s (%s): expected a diagnostic, got nil error", fx.name, fx.status)
			}
			msg := err.Error()
			if !strings.Contains(msg, fx.code) {
				t.Errorf("%s: expected diagnostic code %q in output, got:\n%s", fx.name, fx.code, msg)
			}
			if !strings.Contains(msg, fx.fix) {
				t.Errorf("%s: diagnostic must carry the fix substring %q, got:\n%s", fx.name, fx.fix, msg)
			}
		})
	}
}

// TestFootgunConflictSurface is the guard from the m-diagnostic-coverage conflict
// analysis: the import-placement diagnostic must REPLACE the PAR_NO_PREFIX_PARSE
// cascade for a misplaced import, while a genuinely stray token that cannot start
// an expression must STILL produce PAR_NO_PREFIX_PARSE. This proves the new
// diagnostic didn't swallow the general error path.
func TestFootgunConflictSurface(t *testing.T) {
	// Misplaced import: PAR_IMPORT_PLACEMENT, and NO PAR_NO_PREFIX_PARSE.
	misplaced := `module benchmark/solution
type Op = Add
import std/list (map)
export func main() -> int { 1 }
`
	_, err := pipeline.Run(
		pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true},
		pipeline.Source{Code: misplaced},
	)
	if err == nil {
		t.Fatal("expected an error for misplaced import")
	}
	if !strings.Contains(err.Error(), "PAR_IMPORT_PLACEMENT") {
		t.Errorf("misplaced import should yield PAR_IMPORT_PLACEMENT, got:\n%s", err.Error())
	}
	if strings.Contains(err.Error(), "PAR_NO_PREFIX_PARSE") {
		t.Errorf("misplaced import must NOT cascade PAR_NO_PREFIX_PARSE, got:\n%s", err.Error())
	}

	// Genuinely stray token: still PAR_NO_PREFIX_PARSE (general path intact).
	stray := `module benchmark/solution
type Op = Add

]
`
	_, err2 := pipeline.Run(
		pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true},
		pipeline.Source{Code: stray},
	)
	if err2 == nil {
		t.Fatal("expected an error for a stray token")
	}
	if !strings.Contains(err2.Error(), "PAR_NO_PREFIX_PARSE") {
		t.Errorf("stray token should still yield PAR_NO_PREFIX_PARSE, got:\n%s", err2.Error())
	}
	if strings.Contains(err2.Error(), "PAR_IMPORT_PLACEMENT") {
		t.Errorf("stray token must NOT trigger PAR_IMPORT_PLACEMENT, got:\n%s", err2.Error())
	}
}
