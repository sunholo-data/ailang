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

	// Import placement (#325): imports after a declaration. Must be categorized
	// BEFORE PAR_001, which otherwise swallows it via the generic "parse errors
	// in ..." pattern and hands back a semicolon-oriented hint that can't repair
	// a placement mistake.
	PAR_IMPORT_PLACEMENT ErrCode = "PAR_IMPORT_PLACEMENT" // Import after a declaration

	// AI usability errors - Wrong language
	WRONG_LANG ErrCode = "WRONG_LANG" // Generated code in wrong programming language

	// AI usability errors - Imperative syntax
	IMPERATIVE ErrCode = "IMPERATIVE" // Used imperative constructs (loop, break, assignment statements)

	// Type checker errors - Records
	TC_REC_001 ErrCode = "TC_REC_001" // Record field not found

	// Type checker errors - Type classes
	TC_INT_001 ErrCode = "TC_INT_001" // Not an instance of Integral
	EQ_001     ErrCode = "EQ_001"     // Wrong Eq dictionary

	// Runtime errors - Capabilities
	CAP_001 ErrCode = "CAP_001" // Capability missing

	// Runtime errors - Module system
	MOD_001 ErrCode = "MOD_001" // Undefined module/entry

	// Contract verification errors (M-CONTRACT-EVAL)
	VERIFY_COUNTEREXAMPLE ErrCode = "VERIFY_COUNTEREXAMPLE" // Z3 found counterexample

	// Source-constraint violations (constrained-construction benchmarks)
	CONSTRAINT_VIOLATION ErrCode = "CONSTRAINT_VIOLATION" // Generated source violates source_constraints
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
	// CRITICAL: WRONG_LANG and IMPERATIVE must be checked BEFORE PAR_001
	// because they also trigger parse errors but need specific repair guidance
	{
		WRONG_LANG,
		regexp.MustCompile(`(?i)(def |class |import json|import sys|function |var |const |#include|using namespace|public static|interface |enum class)`),
		RepairHint{
			Title: "Wrong programming language",
			Why:   "Generated code appears to be Python/JavaScript/C++/Java, not AILANG. AILANG is a pure functional language with ML-style syntax.",
			How:   "Start over with AILANG syntax: 1) Use `let x = expr` for bindings, 2) Use `func name(params) -> Type { body }` for functions, 3) Use recursion instead of loops, 4) No classes, no mutation, no statements. Refer to AILANG examples in the prompt.",
		},
	},
	{
		IMPERATIVE,
		regexp.MustCompile(`(?i)(loop\s*\{|while\s*\(|for\s*\(|break;|continue;|^\s*\w+\s*=\s*[^=]|;\s*\w+\s*=\s*[^=]|let mut )`),
		RepairHint{
			Title: "Imperative syntax not allowed",
			Why:   "Used imperative constructs (loop/while/for/break/assignment statements). AILANG is purely functional - no loops, no mutation, no statements.",
			How:   "Replace imperative code with functional patterns: 1) Use recursion instead of loops, 2) Use `let x = expr in body` instead of `x = expr;`, 3) Use pattern matching instead of break/continue, 4) All variables are immutable.",
		},
	},
	{
		// #325: MUST precede PAR_001 — its `parse errors? in` pattern also matches
		// the PAR_IMPORT_PLACEMENT wrapper, but the generic semicolon hint can't
		// repair a placement mistake. This keys on the specific diagnostic code so
		// the repair hint carries the actual fix (hoist the import).
		PAR_IMPORT_PLACEMENT,
		regexp.MustCompile(`PAR_IMPORT_PLACEMENT|imports must appear immediately after the module declaration`),
		RepairHint{
			Title: "Import placed after a declaration",
			Why:   "In AILANG, all `import` lines must appear immediately after the `module` declaration, before any type/func/let declaration. An import after a declaration is a parse error.",
			How:   "Move every `import ...` line up so they sit directly under the `module ...` line, above the first type or func declaration. Order among imports does not matter.",
		},
	},
	{
		PAR_001,
		regexp.MustCompile(`PAR_NO_PREFIX_PARSE|PAR_UNEXPECTED_TOKEN|parse errors? in|unexpected token`),
		RepairHint{
			Title: "Parse error",
			Why:   "AILANG syntax error - common issues: missing semicolons in blocks, wrong syntax for let/lambda/records.",
			How:   "Check: 1) Use `{ e1; e2; e3 }` for blocks (semicolons between exprs), 2) Use `let x = expr in body` or `let x = expr; rest`, 3) Lambda: `\\x -> body` or `func(x) { body }`, 4) No `=` in function params.",
		},
	},
	{
		TC_REC_001,
		regexp.MustCompile(`field '([^']+)' not found in record|closed row missing labels`),
		RepairHint{
			Title: "Record field missing",
			Why:   "Type checker requires the field to exist in the record.",
			How:   "Add the missing field to the record literal, or use row polymorphism: `{ field: T | ρ }` in type annotation.",
		},
	},
	{
		TC_INT_001,
		regexp.MustCompile(`Float .* is not an instance of Integral|mod not defined for Float`),
		RepairHint{
			Title: "Modulo on Float",
			Why:   "`%` requires `Integral` (Int) type.",
			How:   "Use integers for `%`, or use `/` and `floor` for floats.",
		},
	},
	{
		EQ_001,
		regexp.MustCompile(`Eq dictionary resolution failed|using eq_Int for Float`),
		RepairHint{
			Title: "Float equality dictionary",
			Why:   "The Eq dictionary must match Float type.",
			How:   "Annotate as `: float` or ensure both sides are Float.",
		},
	},
	{
		CAP_001,
		regexp.MustCompile(`no effect context available|effect '(\w+)' requires capability|closed row missing labels: \[(IO|FS|Clock|Net)`),
		RepairHint{
			Title: "Missing capability",
			Why:   "Effect calls require explicit capabilities at runtime.",
			How:   "Declare effects in function signature with explicit type annotation: `let main : Unit -> Unit <IO> = \\() -> { println(...) }`. The type annotation is REQUIRED for effects.",
		},
	},
	{
		MOD_001,
		regexp.MustCompile(`entrypoint '(\w+)' not found|module .* not found`),
		RepairHint{
			Title: "Entrypoint/module resolution",
			Why:   "Runner couldn't find the entry point function.",
			How:   "Export a zero-argument `main` function, the eval harness uses `--entry main`.",
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

// CategorizeErrorWithCode analyzes both generated code and stderr to detect
// AI usability issues like wrong language or imperative syntax.
//
// The WRONG_LANG and IMPERATIVE pattern checks are AILANG-specific: their
// regexes match patterns like `def `, `import json`, `while (`, `for (`
// which are perfectly normal in Python/Go/JS source code. Applying them to
// non-AILANG languages produces false WRONG_LANG/IMPERATIVE categorizations
// on valid code. Callers that don't know the language can use the legacy
// 2-arg path (preserved as CategorizeErrorWithCodeLegacy for the few in-tree
// call sites that don't have language context).
func CategorizeErrorWithCode(code, stderr, language string) (ErrCode, *RepairHint) {
	if language == "ailang" {
		// AILANG-specific: catch the case where the LLM generated Python/JS/etc
		// before even trying to compile, or wrote imperative code.
		for _, rule := range Rules {
			if rule.Code == WRONG_LANG || rule.Code == IMPERATIVE {
				if rule.Re.MatchString(code) {
					return rule.Code, &rule.Hint
				}
			}
		}
	}

	// For non-AILANG languages, fall through to stderr-only categorization.
	// Language-specific code validators (e.g. ValidatePythonCode for the
	// inverse "Python eval got AILANG code" case) are the right pattern here
	// and can be added per language as needed.
	return CategorizeErrorCode(stderr)
}

// ValidatePythonCode checks if Python code accidentally contains AILANG syntax
// Returns error code and hint if AILANG syntax detected, otherwise returns ("", nil)
func ValidatePythonCode(code string) (ErrCode, *RepairHint) {
	// AILANG-specific patterns that should NEVER appear in Python code
	ailangPatterns := regexp.MustCompile(`(?m)(^module\s+|^export\s+func\s+|!\s*\{IO\}|!\s*\{FS\}|!\s*\{Net\}|match\s+\w+\s+\{|\bmatch\s+\w+\s+with\b)`)

	if ailangPatterns.MatchString(code) {
		return WRONG_LANG, &RepairHint{
			Title: "AILANG syntax in Python code",
			Why:   "Generated AILANG code instead of Python. Keywords like 'module', 'export func', '! {IO}', 'match' are AILANG-specific.",
			How:   "Rewrite using Python syntax: Use 'def' for functions, 'import' for modules, no effect annotations, no pattern matching with 'match' keyword (use if/elif or match statement in Python 3.10+). Start with 'def main():' not 'export func main()'.",
		}
	}

	return "", nil
}

// FormatRepairPrompt creates the repair guidance injection for retry attempts.
// This prompt is appended to the original benchmark prompt to guide the AI
// toward fixing the specific error that occurred.
func FormatRepairPrompt(code ErrCode, hint *RepairHint, benchmarkID, lang, failedCode, stderr string) string {
	return fmt.Sprintf(`Your previous program failed with this error:

ERROR:
%s

YOUR PREVIOUS CODE:
%s

DIAGNOSIS:
<%s>: %s
Why: %s
How to fix: %s

Please produce a corrected %s program that fixes this specific error
for the benchmark "%s". Keep it minimal, single file, no extra commentary.`,
		stderr, failedCode, code, hint.Title, hint.Why, hint.How, lang, benchmarkID)
}

// FormatZ3RepairHint creates a RepairHint from Z3 verification output (M-CONTRACT-EVAL)
func FormatZ3RepairHint(verifyStderr string) *RepairHint {
	return &RepairHint{
		Title: "Contract verification failed — Z3 found counterexample",
		Why:   "Your function does not satisfy its ensures clause for all inputs.",
		How:   "Read the counterexample values below and fix the logic error.\n\nZ3 output:\n" + verifyStderr,
	}
}
