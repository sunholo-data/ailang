package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/mission"
	"github.com/sunholo-data/ailang/internal/mission/quorum"
)

// runDesignQuorum implements `ailang design-quorum` (M-MISSION-FLEET-AB Phase B2):
// it runs N reviewers from different vendors in parallel into one quorum verdict,
// records a machine JSON artifact + an optional mission-log markdown block, and
// degrades gracefully to N-1 (naming any absent reviewer). The Claude
// controller's own review is IN-SESSION (not an API call) and can be folded in
// via flags.
//
// Roster (Mark, attended 2026-09-25): one seat per vendor — gpt6-astra (OpenAI),
// gemini-3-1-pro (Google), oc-glm-5-3 (Z-AI) and claude-sonnet-5@claude-p
// (Anthropic, subscription). The AUTHOR'S vendor sits out: "if anthropic does the
// design, it doesn't review; if astra does the design, it's not on the quorum."
// Benched seats come back, labelled, only when every seated reviewer is absent.
// See quorum/seating.go. This retires the hand-applied "substitute gpt5-6-sol on
// astra's turn" workaround, which still had OpenAI reviewing OpenAI.
//
// History: OpenAI seat gpt5-6-sol -> gpt6-astra (2026-09-05); Z-AI seat
// oc-glm-5-2 -> oc-glm-5-3 (2026-09-25).
func runDesignQuorum() {
	fs := flag.NewFlagSet("design-quorum", flag.ExitOnError)
	reviewers := fs.String("reviewers", defaultQuorumRoster, "comma-separated reviewer roster (models.yml ids, or "+quorum.ClaudeReviewerID+")")
	author := fs.String("author", os.Getenv("MISSION_DESIGN_AUTHOR"), "model or lane that WROTE the doc (e.g. claude:claude-opus-5-5, codex:gpt-6-astra); its vendor sits out")
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

	authorID := *author
	if authorID == "" {
		// The Claude controller is the usual author in an attended session; assume
		// it rather than let Anthropic review its own doc, and say so.
		authorID = "claude (assumed — pass --author)"
		fmt.Fprintln(os.Stderr, "design-quorum: no --author given; assuming a Claude author, so the Anthropic seat sits out")
	} else if quorum.VendorOf(authorID) == "" {
		fmt.Fprintf(os.Stderr, "design-quorum: author %q is not a recognised vendor — nobody sits out\n", authorID)
	}
	seated, benched := quorum.SeatReviewers(models, authorID)
	fmt.Fprintf(os.Stderr, "design-quorum: author %s — seated %s; sitting out %s\n", authorID, strings.Join(seated, ","), csvOrNone(benched))

	isoTS := time.Now().UTC().Format(time.RFC3339)
	runner := quorumSeatRunner
	result := quorum.RunQuorum(docPath, docBody, isoTS, seated, *maxCost, controller, runner)
	if quorum.RecallBenched(result, benched, docPath, docBody, *maxCost, runner) {
		fmt.Fprintf(os.Stderr, "design-quorum: every independent reviewer was absent — recalled the author's vendor (%s), labelled same-vendor\n", strings.Join(benched, ","))
	}

	// Always write the machine artifact (seeds Phase E).
	artPath, aerr := quorum.WriteJSONArtifact(*artifactDir, result)
	if aerr != nil {
		fmt.Fprintf(os.Stderr, "design-quorum: %v\n", aerr)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "quorum artifact: %s\n", artPath)

	// LOUD, non-blocking: a reviewer billed with zero reported tokens leaves the
	// Gate-3 chain ledger unreconcilable, and a zero there reads exactly like a
	// free call. Name it rather than let it pass silently (#708). This never
	// changes the exit code — an unreconcilable reviewer must not wedge a
	// quorum the loop depends on.
	for _, gap := range result.TokenAccountingGaps() {
		fmt.Fprintf(os.Stderr, "design-quorum: TOKEN ACCOUNTING GAP — %s\n", gap)
	}

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
  ailang design-quorum --author codex:gpt-6-astra < doc.md

FLAGS:
  --reviewers <csv>          reviewer roster (default gpt6-astra,gemini-3-1-pro,oc-glm-5-3,claude-sonnet-5@claude-p)
  --max-cost-usd <n>         per-reviewer budget cap in USD (default 0.30)
  --author <model>           the doc's author; its vendor sits out (see below)
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

  AUTHOR'S VENDOR SITS OUT: pass --author (or MISSION_DESIGN_AUTHOR) with the
  model that wrote the doc. Reviewers from that vendor are benched, and come
  back — labelled same-vendor — only if every other reviewer is absent. With no
  author given, a Claude author is assumed. The Anthropic seat
  (claude-sonnet-5@claude-p) runs "claude -p" on the subscription, never the
  API key, with tools, settings and MCP off; it is absent (quota) when the
  Anthropic bucket is over its mission ration.

EXIT CODES:
  0 = proceed   3 = blocked   1/2 = IO or usage error

SEE ALSO:
  ailang design-review — a single reviewer verdict.
`

// defaultQuorumRoster is one seat per vendor; the author's vendor sits out.
const defaultQuorumRoster = "gpt6-astra,gemini-3-1-pro,oc-glm-5-3," + quorum.ClaudeReviewerID

// quorumSeatRunner runs one seat: the Anthropic seat over the subscription (unless
// the Anthropic bucket is over its mission ration — the ration is what keeps
// attended sessions' headroom), every other seat through models.yml.
func quorumSeatRunner(model, docPath, docBody string, maxCostUSD float64) *quorum.ReviewerOutcome {
	if !strings.HasSuffix(model, quorum.ClaudeReviewerSuffix) {
		return quorum.RunReviewer(model, docPath, docBody, maxCostUSD)
	}
	if a := mission.ObserveAnthropicQuota(time.Now()); a.Blocked() {
		return &quorum.ReviewerOutcome{
			Model:        model,
			AbsentReason: quorum.ReasonQuota,
			Err:          "Anthropic bucket blocked by the mission ration: " + a.Reason,
		}
	}
	return quorum.RunClaudeSubscriptionReviewer(strings.TrimSuffix(model, quorum.ClaudeReviewerSuffix), docPath, docBody)
}

func csvOrNone(xs []string) string {
	if len(xs) == 0 {
		return "none"
	}
	return strings.Join(xs, ",")
}
