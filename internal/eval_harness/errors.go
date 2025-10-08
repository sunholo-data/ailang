package eval_harness

import (
	"fmt"
	"regexp"
)

// ErrCode represents a categorized error type from AILANG execution
type ErrCode string

const (
	// Parser errors
	PAR_001 ErrCode = "PAR_001" // Parse error (block/semicolon issues)

	// Type checker errors - Records
	TC_REC_001 ErrCode = "TC_REC_001" // Record field not found

	// Type checker errors - Type classes
	TC_INT_001 ErrCode = "TC_INT_001" // Not an instance of Integral
	EQ_001     ErrCode = "EQ_001"     // Wrong Eq dictionary

	// Runtime errors - Capabilities
	CAP_001 ErrCode = "CAP_001" // Capability missing

	// Runtime errors - Module system
	MOD_001 ErrCode = "MOD_001" // Undefined module/entry
)

// RepairHint provides actionable guidance for fixing an error
type RepairHint struct {
	Title string // Short description of the error
	Why   string // Explanation of why the error occurred
	How   string // Concrete steps to fix the error
}

// errorRule defines a pattern-matching rule for error categorization
type errorRule struct {
	Code ErrCode
	Re   *regexp.Regexp
	Hint RepairHint
}

// Rules maps error patterns to categorized error codes and repair hints
var Rules = []errorRule{
	{
		PAR_001,
		regexp.MustCompile(`parse error.*unexpected .* near`),
		RepairHint{
			Title: "Block/semicolon issue",
			Why:   "Parser expects semicolons in `{ ... }` blocks.",
			How:   "Add `;` between expressions inside `{}` or unwrap single-expression blocks.",
		},
	},
	{
		TC_REC_001,
		regexp.MustCompile(`field '([^']+)' not found in record.*\{([^}]*)\}`),
		RepairHint{
			Title: "Record field missing",
			Why:   "Type checker requires the field to exist.",
			How:   "Add the field, or generalize the function type to `{ <field>: T | ρ }`.",
		},
	},
	{
		TC_INT_001,
		regexp.MustCompile(`Float .* is not an instance of Integral|mod not defined for Float`),
		RepairHint{
			Title: "Modulo on Float",
			Why:   "`%` requires `Integral` (Int).",
			How:   "Use integers for `%`, or use `/` and `floor` for floats.",
		},
	},
	{
		EQ_001,
		regexp.MustCompile(`Eq dictionary resolution failed|using eq_Int for Float`),
		RepairHint{
			Title: "Float equality dictionary",
			Why:   "The Eq dictionary must match Float.",
			How:   "Annotate as `: float` or ensure both sides are Float.",
		},
	},
	{
		CAP_001,
		regexp.MustCompile(`effect '(\w+)' requires capability`),
		RepairHint{
			Title: "Missing capability",
			Why:   "Effect calls require explicit caps.",
			How:   "Run with `--caps IO,FS,Clock,Net` (only what you need).",
		},
	},
	{
		MOD_001,
		regexp.MustCompile(`entrypoint '(\w+)' not found|module .* not found`),
		RepairHint{
			Title: "Entrypoint/module resolution",
			Why:   "Runner couldn't find your export.",
			How:   "Export a zero-arg `main`, or pass `--entry yourFunc`.",
		},
	},
}

// CategorizeErrorCode matches stderr against error patterns and returns
// the error code and repair hint if a match is found.
// Returns ("", nil) if no pattern matches.
func CategorizeErrorCode(stderr string) (ErrCode, *RepairHint) {
	for _, rule := range Rules {
		if rule.Re.MatchString(stderr) {
			return rule.Code, &rule.Hint
		}
	}
	return "", nil // Unknown error
}

// FormatRepairPrompt creates the repair guidance injection for retry attempts.
// This prompt is appended to the original benchmark prompt to guide the AI
// toward fixing the specific error that occurred.
func FormatRepairPrompt(code ErrCode, hint *RepairHint, benchmarkID, lang string) string {
	return fmt.Sprintf(`Your previous program failed with:
<%s>: %s
Why: %s
How to fix: %s

Please produce a corrected %s program that compiles and runs
for the benchmark "%s". Keep it minimal, single file,
no extra commentary.`, code, hint.Title, hint.Why, hint.How, lang, benchmarkID)
}
