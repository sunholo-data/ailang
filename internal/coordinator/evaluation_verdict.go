package coordinator

import (
	"fmt"
	"strconv"
	"strings"
)

// M-PIPELINE-RECONCILIATION M1 (decision D1(b), ratified 2026-08-26).
//
// EvaluationVerdict is the CLOSED result type of the sprint-evaluator stage.
// Three kinds, and absence is unrepresentable by construction: an evaluator
// that errors, times out, or emits something unparsable yields UNAVAILABLE
// with a reason — never a missing field. FAIL and UNAVAILABLE block every
// AUTOMATIC progression (auto_merge, auto-approved downstream handoffs) but
// never the human gate: a dead evaluator must not hide work from the person.
//
// Quorum round 1 (2026-08-26) forced this shape: the first draft both claimed
// "an evaluator FAIL is typed and blocks handoff" (axiom table) and said "a
// FAIL does not block — it informs" (solution), while leaving evaluator death
// silent. The closed type is how both contradictions stay fixed.

// VerdictKind enumerates the three (and only three) verdict kinds.
type VerdictKind string

const (
	VerdictPass        VerdictKind = "PASS"
	VerdictFail        VerdictKind = "FAIL"
	VerdictUnavailable VerdictKind = "UNAVAILABLE"
)

// EvaluationVerdict is one evaluator outcome.
type EvaluationVerdict struct {
	Kind   VerdictKind
	Score  int    // 0-100 for PASS/FAIL; 0 for UNAVAILABLE
	Reason string // FAIL reasons or UNAVAILABLE cause
}

// BlocksAutomation reports whether this verdict blocks automatic progression.
// Only PASS clears the way; anything else — including a verdict we could not
// understand — is treated as "a machine should not proceed on this".
func (v EvaluationVerdict) BlocksAutomation() bool {
	return v.Kind != VerdictPass
}

// String renders the canonical form ParseEvaluationVerdict accepts.
func (v EvaluationVerdict) String() string {
	switch v.Kind {
	case VerdictPass:
		return fmt.Sprintf("PASS score=%d", v.Score)
	case VerdictFail:
		if v.Reason != "" {
			return fmt.Sprintf("FAIL score=%d reasons=%s", v.Score, v.Reason)
		}
		return fmt.Sprintf("FAIL score=%d", v.Score)
	default:
		return fmt.Sprintf("UNAVAILABLE reason=%s", v.Reason)
	}
}

// UnavailableVerdict builds the verdict for an evaluator that could not
// deliver: task failure, timeout, or unparsable output.
func UnavailableVerdict(reason string) EvaluationVerdict {
	if reason == "" {
		reason = "no reason recorded"
	}
	return EvaluationVerdict{Kind: VerdictUnavailable, Reason: reason}
}

// ParseEvaluationVerdict parses an evaluator's emitted verdict line.
//
// Accepted forms:
//
//	PASS score=84
//	FAIL score=45 reasons=<free text>
//	UNAVAILABLE reason=<free text>
//
// Anything else — including an empty string — parses to UNAVAILABLE with the
// offending input recorded, because "we could not understand the evaluator" is
// itself a verdict and must reach the approval as one.
func ParseEvaluationVerdict(raw string) EvaluationVerdict {
	s := strings.TrimSpace(raw)
	unparsable := func() EvaluationVerdict {
		return UnavailableVerdict(fmt.Sprintf("unparsable evaluator output: %q", truncateVerdict(s, 200)))
	}

	switch {
	case strings.HasPrefix(s, string(VerdictPass)):
		score, ok := verdictField(s, "score=")
		if !ok {
			return unparsable()
		}
		return EvaluationVerdict{Kind: VerdictPass, Score: score}

	case strings.HasPrefix(s, string(VerdictFail)):
		score, ok := verdictField(s, "score=")
		if !ok {
			return unparsable()
		}
		reason := ""
		if i := strings.Index(s, "reasons="); i >= 0 {
			reason = strings.TrimSpace(s[i+len("reasons="):])
		}
		return EvaluationVerdict{Kind: VerdictFail, Score: score, Reason: reason}

	case strings.HasPrefix(s, string(VerdictUnavailable)):
		reason := "no reason recorded"
		if i := strings.Index(s, "reason="); i >= 0 {
			reason = strings.TrimSpace(s[i+len("reason="):])
		}
		return EvaluationVerdict{Kind: VerdictUnavailable, Reason: reason}

	default:
		return unparsable()
	}
}

// verdictField extracts an integer field like "score=84" from a verdict line.
func verdictField(s, key string) (int, bool) {
	i := strings.Index(s, key)
	if i < 0 {
		return 0, false
	}
	rest := s[i+len(key):]
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0, false
	}
	return n, true
}

func truncateVerdict(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// EvaluationVerdictMarker is the output line an EvaluatesParent agent emits.
const EvaluationVerdictMarker = "EVALUATION_VERDICT:"

// ExtractEvaluationVerdict finds the LAST verdict line in an evaluator's
// output. Absence is UNAVAILABLE: a completed evaluator that emitted no
// verdict did not pass anything.
func ExtractEvaluationVerdict(output string) EvaluationVerdict {
	var last string
	found := false
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, EvaluationVerdictMarker) {
			last = strings.TrimSpace(strings.TrimPrefix(line, EvaluationVerdictMarker))
			found = true
		}
	}
	if !found {
		return UnavailableVerdict("evaluator completed without emitting an " + EvaluationVerdictMarker + " line")
	}
	return ParseEvaluationVerdict(last)
}

// AutoApprovesHandoffTo reports whether a handoff from this agent to target
// skips the approval gate: either the blanket bool, or a per-edge entry.
func (a *AgentConfig) AutoApprovesHandoffTo(target string) bool {
	if a == nil {
		return false
	}
	if a.AutoApproveHandoffs {
		return true
	}
	for _, t := range a.AutoApproveHandoffTo {
		if t == target {
			return true
		}
	}
	return false
}

// AllowsAutomation reports whether this approval may be progressed by
// AUTOMATION (auto_merge, auto-approved downstream handoffs). An empty
// Evaluation means no evaluator stage is configured and does not block —
// making absence block would freeze every non-evaluated flow. Any present
// value must parse to PASS; FAIL, UNAVAILABLE, and garbage all block. The
// HUMAN gate never consults this — a person may approve anything, with the
// verdict displayed.
func (r *ApprovalRequestRecord) AllowsAutomation() bool {
	if r == nil || r.Evaluation == "" {
		return true
	}
	return !ParseEvaluationVerdict(r.Evaluation).BlocksAutomation()
}
