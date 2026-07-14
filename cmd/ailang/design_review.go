package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

// runDesignReview implements `ailang design-review` (M-MISSION-FLEET-AB Phase B1):
// a single off-Anthropic reviewer verdict on a design doc, via the shipped
// internal/ai handlers + the models.yml resolver. It does NOT rebuild the
// stalled `ailang exec` unification — it is a thin caller over the quorum
// package's reviewer primitive.
//
// Usage:
//
//	ailang design-review <doc.md> --reviewer gpt5-6-sol [--json] [--max-cost-usd 0.10]
//	ailang design-review --reviewer gemini-3-1-pro < doc.md   (reads stdin)
func runDesignReview() {
	fs := flag.NewFlagSet("design-review", flag.ExitOnError)
	reviewer := fs.String("reviewer", "", "reviewer model id from models.yml (e.g. gpt5-6-sol, gemini-3-1-pro)")
	asJSON := fs.Bool("json", false, "emit the reviewer's structured JSON verdict")
	maxCost := fs.Float64("max-cost-usd", quorum.DefaultMaxCostUSD, "per-reviewer budget cap in USD")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, designReviewHelp)
	}
	_ = fs.Parse(hoistFlags(os.Args[2:]))

	if *reviewer == "" {
		fmt.Fprintln(os.Stderr, "design-review: --reviewer <model> is required")
		fs.Usage()
		os.Exit(2)
	}

	docPath, docBody, err := readDoc(fs.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "design-review: %v\n", err)
		os.Exit(1)
	}

	outcome := quorum.RunReviewer(*reviewer, docPath, docBody, *maxCost)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(outcome)
	} else {
		printOutcome(outcome)
	}

	// Exit non-zero when the reviewer could not produce a verdict — a missing
	// reviewer is never a silent pass (Principle 2). Present pass/reject both
	// exit 0; the caller reads the verdict field.
	if !outcome.Present {
		os.Exit(1)
	}
}

// hoistFlags reorders args so leading positional operands (e.g. a doc path)
// come AFTER the flags, letting Go's flag package parse `<doc> --reviewer X`
// the same as `--reviewer X <doc>`. Flags that take a value (--reviewer X,
// --max-cost-usd 0.1) keep their value adjacent; `--flag=value` and boolean
// flags are single tokens. Everything after a bare `--` is treated as
// positional verbatim.
func hoistFlags(args []string) []string {
	valueFlags := map[string]bool{
		"--reviewer": true, "-reviewer": true,
		"--reviewers": true, "-reviewers": true,
		"--max-cost-usd": true, "-max-cost-usd": true,
		"--artifact-dir": true, "-artifact-dir": true,
		"--mission-log": true, "-mission-log": true,
		"--controller-verdict": true, "-controller-verdict": true,
		"--controller-note": true, "-controller-note": true,
	}
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			// Consume a following value token for space-separated value flags.
			if valueFlags[a] && !strings.Contains(a, "=") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}

// readDoc resolves the doc body from a positional path or, if none given,
// stdin.
func readDoc(args []string) (path, body string, err error) {
	if len(args) > 0 && args[0] != "" && args[0] != "-" {
		path = args[0]
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return "", "", fmt.Errorf("read %s: %w", path, rerr)
		}
		return path, string(b), nil
	}
	b, rerr := io.ReadAll(os.Stdin)
	if rerr != nil {
		return "", "", fmt.Errorf("read stdin: %w", rerr)
	}
	if len(b) == 0 {
		return "", "", fmt.Errorf("no design doc: pass a path or pipe the doc on stdin")
	}
	return "(stdin)", string(b), nil
}

func printOutcome(o *quorum.ReviewerOutcome) {
	if !o.Present {
		fmt.Printf("reviewer %s ABSENT (%s): %s\n", o.Model, o.AbsentReason, o.Err)
		return
	}
	fmt.Printf("reviewer %s → %s ($%.4f)\n", o.Model, o.Result.Verdict, o.CostUSD)
	fmt.Printf("  strongest_objection: %s\n", o.Result.StrongestObjection)
	fmt.Printf("  catch: %s\n", o.Result.Catch)
}

const designReviewHelp = `ailang design-review — single off-Anthropic reviewer verdict on a design doc (fleet Phase B)

USAGE:
  ailang design-review <doc.md> --reviewer <model> [flags]
  ailang design-review --reviewer <model> < doc.md

FLAGS:
  --reviewer <model>     reviewer model id from models.yml (gpt5-6-sol, gemini-3-1-pro, ...)  [required]
  --json                 emit the reviewer's structured JSON verdict
  --max-cost-usd <n>     per-reviewer budget cap in USD (default 0.10)

BEHAVIOR:
  Runs ONE reject-by-default reviewer via the shipped internal/ai handlers +
  the models.yml resolver. OpenAI reviewers use OPENAI_API_KEY; Google/Gemini
  reviewers use Vertex ADC (NOT GEMINI_API_KEY). A missing verdict (auth,
  unreachable, budget, or malformed response) exits non-zero and is NEVER a
  silent pass.

SEE ALSO:
  ailang design-quorum — compose N reviewers into one quorum verdict.
`
