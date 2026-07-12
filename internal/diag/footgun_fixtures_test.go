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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/stdlibindex"
	"github.com/sunholo-data/ailang/internal/types"
)

// TestMain wires the stdlib import-suggestion path exactly as the CLI does
// (cmd/ailang/diagnostics_wiring.go) so the stdlib_import_hint fixture below
// sees the same fix-carrying diagnostic an agent's `ailang check` sees. Two
// steps are required and BOTH must run before any fixture executes:
//
//  1. AILANG_STDLIB_PATH must point at the on-disk stdlib (../../std relative to
//     this package's CWD) — stdlibindex scans it lazily via sync.Once, so the
//     env var must be set before stdlibindex.Modules is first called.
//  2. types.ImportSuggester must be wired to stdlibindex.Modules — it is nil in
//     pure library code (only the CLI init() sets it), so without this the hint
//     is empty and the fixture is silently red with a bare "undefined variable".
func TestMain(m *testing.M) {
	_ = os.Setenv("AILANG_STDLIB_PATH", "../../std")
	types.ImportSuggester = stdlibindex.Modules
	os.Exit(m.Run())
}

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
	{
		name:   "reserved_keyword",
		code:   "PAR_RESERVED_KEYWORD",
		fix:    "'exists' is reserved for existential types",
		status: "covered", // promoted by m-diagnostic-coverage (M-DIAG-FIXTURE-PROMOTION)
		src: `module benchmark/solution
export func main() -> int { let exists = 1 in exists }
`,
	},
	{
		name:   "hyphen_in_module",
		code:   "PAR_HYPHEN_IN_MODULE",
		fix:    "Use underscores instead",
		status: "covered", // promoted by m-diagnostic-coverage (M-DIAG-FIXTURE-PROMOTION)
		src: `module benchmark/my-solution
export func main() -> int { 1 }
`,
	},
	{
		// M-SYNTAX-AI-FORGIVING R1 made a well-formed `;`-sequence in a `=`-body
		// VALID (`= let x = 1; x` now parses). PAR017 now fires only for a genuinely
		// MISPLACED `;` — here a stray double `;` where an expression was expected.
		// The diagnostic (and its "only valid inside" fix text) is preserved for that
		// real error; the previously-rejected valid form is not.
		name:   "misplaced_semicolon_in_expr_func",
		code:   "PAR017",
		fix:    "only valid inside",
		status: "covered", // promoted by m-diagnostic-coverage (M-DIAG-FIXTURE-PROMOTION)
		src: `module benchmark/solution
export func f() -> int = let x = 1;; x
export func main() -> int { f() }
`,
	},
	{
		// TYPE error (not parse) — the fix substring comes from the ImportSuggester
		// hint wired in TestMain above. Assert a STABLE substring ("exported by
		// std/list"), NOT the full "std/list, std/option, std/result" comma list,
		// which shifts as stdlib exports change.
		name:   "stdlib_import_hint",
		code:   "undefined variable: map",
		fix:    "exported by std/list",
		status: "covered", // promoted by m-diagnostic-coverage (M-DIAG-FIXTURE-PROMOTION)
		src: `module benchmark/solution
export func main() -> list[int] { map(\x. x, [1,2,3]) }
`,
	},
	{
		// EFFECT error (M-EFFECT-MODE-VALIDATION) — closed mode set. Value
		// outside the schema value set. Assert a STABLE legal-value substring.
		name:   "effect_unknown_mode",
		code:   "EFF_UNKNOWN_MODE",
		fix:    "Allowed values: crypto, os, seeded",
		status: "shipped-this-sprint", // M-EFFECT-MODE-VALIDATION
		src: `module benchmark/solution
export func main() -> int ! {Rand[mode=banana]} { 42 }
`,
	},
	{
		// EFFECT error (M-EFFECT-MODE-VALIDATION) — key outside the schema keys.
		name:   "effect_unknown_param_key",
		code:   "EFF_UNKNOWN_PARAM_KEY",
		fix:    "Allowed keys: mode",
		status: "shipped-this-sprint", // M-EFFECT-MODE-VALIDATION
		src: `module benchmark/solution
export func main() -> int ! {Rand[flavor=hot]} { 42 }
`,
	},
	{
		// EFFECT error (M-EFFECT-MODE-VALIDATION) — param on a schema-less effect.
		// The fix names the tracking doc so the port-sprint unblocks the syntax.
		name:   "effect_params_not_supported",
		code:   "EFF_PARAMS_NOT_SUPPORTED",
		fix:    "m-effect-clock-net-fs-modes",
		status: "shipped-this-sprint", // M-EFFECT-MODE-VALIDATION
		src: `module benchmark/solution
export func main() -> int ! {Clock[mode=pinned]} { 0 }
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

// TestFootgunFixture_MOD014_ModuleLess is the MOD014 footgun contract. Unlike
// the inline-Code table above (which routes through the single-file pipeline),
// MOD014 only fires in the MODULE pipeline, which requires a real filename on
// disk. A file with top-level `func` declarations but no `module` header used to
// silently succeed with exit 0 and no output (the entry never ran). It must now
// fail loudly with a fix-carrying diagnostic naming the canonical module path.
func TestFootgunFixture_MOD014_ModuleLess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nomod.ail")
	src := "import std/io (println)\n" +
		"export func main() -> () ! {IO} { println(\"x\") }\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := pipeline.Run(
		pipeline.Config{Mode: pipeline.ModeCheck},
		pipeline.Source{Code: src, Filename: path},
	)
	if err == nil {
		t.Fatal("MOD014: module-less file with top-level funcs must fail loudly, got nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "MOD014") {
		t.Errorf("expected MOD014 code in diagnostic, got:\n%s", msg)
	}
	// Fix-carrying contract: the message must tell the agent to add a module line.
	if !strings.Contains(msg, "Fix: add 'module ") {
		t.Errorf("MOD014 diagnostic must carry the 'add module' fix, got:\n%s", msg)
	}
}

// TestFootgunFixture_MOD014_BareExpressionPreserved guards the escape hatch: a
// module-less file that is a lone bare expression (`1 + 1`) must NOT trip MOD014
// — that eval path is intentional. MOD014 gates on top-level Funcs only.
func TestFootgunFixture_MOD014_BareExpressionPreserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "expr.ail")
	src := "1 + 1\n"
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := pipeline.Run(
		pipeline.Config{Mode: pipeline.ModeCheck},
		pipeline.Source{Code: src, Filename: path},
	)
	if err != nil && strings.Contains(err.Error(), "MOD014") {
		t.Fatalf("bare-expression module-less file must NOT trip MOD014, got:\n%s", err.Error())
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
