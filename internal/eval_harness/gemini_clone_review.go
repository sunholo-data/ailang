package eval_harness

// M-GEMINI-REPO-MOUNT Phase 2 — clone-review directive + evidence check.
//
// Split out of gemini_evaluator_bridge.go to keep that file under the 800-line
// CI ceiling. RunGeminiEvaluator (in the sibling file) calls these two helpers
// on the clone-review path; the diff-bundle path is untouched.

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// buildCloneReviewDirective composes the clone-review directive: the canonical
// clone preamble (shared single-source executor.BuildClonePreamble, so the CLI
// and the bridge cannot drift) followed by the reasoning-only evaluator
// instruction and the design doc + sprint plan + acceptance criteria. No diff
// bundle is packed — the sandbox reads the repo it clones.
func buildCloneReviewDirective(repoURL, sha, designDoc, sprintPlan, acceptanceCriteria string) (string, error) {
	preamble, err := executor.BuildClonePreamble(repoURL, sha)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(preamble)
	b.WriteString(evaluatorReasoningInstruction)
	b.WriteString("\n\n=== DESIGN DOC ===\n")
	b.WriteString(strings.TrimSpace(designDoc))
	b.WriteString("\n\n=== SPRINT PLAN ===\n")
	b.WriteString(strings.TrimSpace(sprintPlan))
	if strings.TrimSpace(acceptanceCriteria) != "" {
		b.WriteString("\n\n=== ACCEPTANCE CRITERIA ===\n")
		b.WriteString(strings.TrimSpace(acceptanceCriteria))
	}
	b.WriteString("\n\n")
	return b.String(), nil
}

// sha40Line matches a standalone 40-hex SHA the agent echoed from
// `git rev-parse HEAD`. It is anchored to a line so a SHA mentioned mid-prose
// is not mistaken for the evidence echo.
var sha40Line = regexp.MustCompile(`(?m)^\s*([0-9a-fA-F]{40})\s*$`)

// checkCloneEvidence extracts the echoed `git rev-parse HEAD` from the agent
// output and validates it against the request (M-GEMINI-REPO-MOUNT Phase 2):
//
//   - pinned (wantSHA != ""): the echo must EQUAL wantSHA (case-insensitive);
//   - HEAD review (wantSHA == ""): the echo must be a valid, non-empty 40-hex
//     SHA — proof the agent actually cloned and ran in-sandbox.
//
// It returns the reviewed 40-hex revision on success, or an error describing the
// missing/invalid/mismatched evidence (which the caller stamps as a structured
// degradation — never a clean pass on absent evidence).
func checkCloneEvidence(output, wantSHA string) (string, error) {
	matches := sha40Line.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("no echoed `git rev-parse HEAD` 40-hex SHA found in the agent output (cannot prove which revision was reviewed)")
	}
	// The evidence echo is the LAST standalone 40-hex line (the agent echoes it
	// after cloning, before the verdict).
	got := strings.ToLower(strings.TrimSpace(matches[len(matches)-1][1]))

	if want := strings.ToLower(strings.TrimSpace(wantSHA)); want != "" {
		if got != want {
			return "", fmt.Errorf("echoed HEAD %s does not match the pinned --clone-sha %s (verdict would be on the wrong revision)", got, want)
		}
	}
	return got, nil
}
