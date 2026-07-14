package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

// runDesignQuorum implements `ailang design-quorum` (M-MISSION-FLEET-AB Phase B2):
// it composes N off-Anthropic reviewers (default gpt5-6-sol + gemini-3-1-pro)
// in parallel into one quorum verdict, records a machine JSON artifact + an
// optional mission-log markdown block, and degrades gracefully to N-1 (naming
// any absent reviewer). The Claude controller's own review is IN-SESSION (not
// an API call) and can be folded in via flags.
func runDesignQuorum() {
	fs := flag.NewFlagSet("design-quorum", flag.ExitOnError)
	reviewers := fs.String("reviewers", "gpt5-6-sol,gemini-3-1-pro", "comma-separated reviewer model ids from models.yml")
	maxCost := fs.Float64("max-cost-usd", quorum.DefaultMaxCostUSD, "per-reviewer budget cap in USD")
	artifactDir := fs.String("artifact-dir", quorum.ArtifactDir, "directory for the machine JSON artifact")
	logPath := fs.String("mission-log", "", "optional mission log path to append the markdown block")
	ctrlVerdict := fs.String("controller-verdict", "", "the Claude controller's IN-SESSION verdict (pass|reject); NOT an API call")
	ctrlNote := fs.String("controller-note", "", "the controller's in-session note (required if --controller-verdict set)")
	asJSON := fs.Bool("json", false, "print the full quorum result JSON to stdout")
	fs.Usage = func() { fmt.Fprint(os.Stderr, designQuorumHelp) }
	_ = fs.Parse(hoistFlags(os.Args[2:]))

	docPath, docBody, err := readDoc(fs.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "design-quorum: %v\n", err)
		os.Exit(1)
	}

	var controller *quorum.ControllerReview
	if *ctrlVerdict != "" {
		v := quorum.Verdict(strings.ToLower(*ctrlVerdict))
		if v != quorum.VerdictPass && v != quorum.VerdictReject {
			fmt.Fprintf(os.Stderr, "design-quorum: --controller-verdict must be pass or reject\n")
			os.Exit(2)
		}
		if strings.TrimSpace(*ctrlNote) == "" {
			fmt.Fprintf(os.Stderr, "design-quorum: --controller-note is required when --controller-verdict is set (no silent controller pass)\n")
			os.Exit(2)
		}
		controller = &quorum.ControllerReview{Verdict: v, Note: *ctrlNote}
	}

	models := splitCSV(*reviewers)
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "design-quorum: no reviewers specified")
		os.Exit(2)
	}

	isoTS := time.Now().UTC().Format(time.RFC3339)
	result := quorum.RunQuorum(docPath, docBody, isoTS, models, *maxCost, controller, quorum.RunReviewer)

	// Always write the machine artifact (seeds Phase E).
	artPath, aerr := quorum.WriteJSONArtifact(*artifactDir, result)
	if aerr != nil {
		fmt.Fprintf(os.Stderr, "design-quorum: %v\n", aerr)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "quorum artifact: %s\n", artPath)

	// Optionally append the human markdown block to the mission log.
	if *logPath != "" {
		if _, lerr := quorum.AppendMarkdownToLog(*logPath, result); lerr != nil {
			fmt.Fprintf(os.Stderr, "design-quorum: %v\n", lerr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "mission-log block appended: %s\n", *logPath)
	}

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(result)
	} else {
		fmt.Print(quorum.MarkdownBlock(result))
	}

	// Exit code encodes the synthesis: proceed = 0, blocked = 3 (distinct from
	// usage/IO errors) so a caller can gate on it.
	if result.Synthesis.Verdict == quorum.SynthBlocked {
		os.Exit(3)
	}
}

const designQuorumHelp = `ailang design-quorum — N-reviewer quorum verdict on a design doc (fleet Phase B)

USAGE:
  ailang design-quorum <doc.md> [flags]
  ailang design-quorum --reviewers gpt5-6-sol,gemini-3-1-pro < doc.md

FLAGS:
  --reviewers <csv>          reviewer model ids (default gpt5-6-sol,gemini-3-1-pro)
  --max-cost-usd <n>         per-reviewer budget cap in USD (default 0.10)
  --artifact-dir <dir>       machine JSON artifact dir (default .ailang/state/mission-quorum)
  --mission-log <path>       append the human markdown block to this mission log
  --controller-verdict <v>   the Claude controller's IN-SESSION verdict (pass|reject) — NOT an API call
  --controller-note <text>   the controller's in-session note (required with --controller-verdict)
  --json                     print the full quorum result JSON to stdout

BEHAVIOR:
  Runs the reviewers IN PARALLEL (reject-by-default). Synthesis: any present
  reviewer (or the controller) rejects → BLOCKED (objection returns to author);
  all present pass → PROCEED. An unreachable/over-budget/mis-auth reviewer is
  recorded by NAME with its reason and the quorum degrades to N-1 — never a
  silent pass. Always writes a machine JSON artifact; optionally appends a
  mission-log markdown block.

EXIT CODES:
  0 = proceed   3 = blocked   1/2 = IO or usage error

SEE ALSO:
  ailang design-review — a single reviewer verdict.
`
