// Package quorum implements the mission fleet's design-doc quorum review
// (M-MISSION-FLEET-AB, Phase B). It composes N independent off-Anthropic
// frontier reviewers (via the shipped internal/ai handlers + the models.yml
// resolver) into a single reject-by-default verdict, with graceful N-1
// degradation that always NAMES an absent reviewer (never a silent pass).
//
// It deliberately rides the already-shipped primitives (Critical Principle 1,
// and the design doc's BINDING redundancy audit): text providers in
// internal/ai/{openai,gemini}, the eval_harness models.yml registry, and the
// Handler.CallJson surface. It does NOT rebuild the stalled `ailang exec`
// unification path.
package quorum

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Verdict is a reviewer's binary judgement. Reject-by-default: a reviewer
// that cannot articulate a strong objection still must return an explicit
// pass; there is no implicit/empty state.
type Verdict string

const (
	// VerdictPass means the reviewer found no blocking objection.
	VerdictPass Verdict = "pass"
	// VerdictReject means the reviewer found a blocking objection; the
	// strongest one MUST be populated in ReviewResult.StrongestObjection.
	VerdictReject Verdict = "reject"
)

// ReviewResult is one reviewer's structured verdict on a design doc.
//
// The schema is enforced two ways: the provider is asked for JSON conforming
// to reviewSchema, AND ValidateReviewResult re-checks the parsed result so a
// malformed or gate-violating response is a hard error, never a silent pass
// (Critical Principle 2 — no silent fallbacks).
type ReviewResult struct {
	// Verdict is "pass" or "reject". Any other value is a hard error.
	Verdict Verdict `json:"verdict"`
	// StrongestObjection is the single most important reason to reject or,
	// on a pass, the closest thing to a concern the reviewer could find.
	// REQUIRED and non-empty on a reject; the reject-by-default lever.
	StrongestObjection string `json:"strongest_objection"`
	// Catch is free-text: the specific thing the reviewer would flag to the
	// doc author (a premise gap, a Conflict-Surface omission, an axiom
	// violation, etc.). Required non-empty.
	Catch string `json:"catch"`
}

// reviewSchema is the JSON schema handed to CallJson so schema-supporting
// providers enforce structure server-side. ValidateReviewResult re-checks it
// regardless (providers vary in enforcement strength).
const reviewSchema = `{
  "type": "object",
  "properties": {
    "verdict": {"type": "string", "enum": ["pass", "reject"]},
    "strongest_objection": {"type": "string"},
    "catch": {"type": "string"}
  },
  "required": ["verdict", "strongest_objection", "catch"]
}`

// systemPrompt is the reject-by-default reviewer instruction. It scores
// against the design-doc-creator hard gates (premise verification, Conflict
// Surface, axiom compliance) and forces a strongest-objection field so
// reviewers cannot rubber-stamp (the LGTM-bias risk in the design doc's risk
// table).
const systemPrompt = `You are an adversarial design-doc reviewer for the AILANG mission. Your job is to REJECT by default.

Score the document against these hard gates:
1. PREMISE VERIFICATION — is every factual claim about the codebase (files, APIs, behavior) verified, or asserted? Unverified premises are grounds to reject.
2. CONFLICT SURFACE — does the doc identify what existing machinery it overlaps/conflicts with, and justify not reusing it? A missing conflict-surface analysis is grounds to reject.
3. AXIOM COMPLIANCE — does it respect the mission axioms: minimal frozen core, route-to-extension bias, no silent fallbacks, bounded waits, deterministic behavior?

You MUST return a single JSON object with exactly these fields:
- "verdict": "pass" or "reject"
- "strongest_objection": the SINGLE most important reason this doc should not proceed as written. On a reject this MUST be a concrete, specific objection (never empty, never "looks fine"). On a pass, state the closest concern you could find.
- "catch": the specific thing you would flag to the author to fix or verify.

Do not praise. Do not summarize the doc. Output ONLY the JSON object, no prose, no code fences.`

// BuildPrompt returns the user prompt for reviewing a design doc.
func BuildPrompt(docPath, docBody string) string {
	var b strings.Builder
	b.WriteString("Review this AILANG mission design doc")
	if docPath != "" {
		b.WriteString(" (" + docPath + ")")
	}
	b.WriteString(". Apply the reject-by-default rubric.\n\n---\n")
	b.WriteString(docBody)
	b.WriteString("\n---\n")
	return b.String()
}

// ParseReviewResult parses a provider's JSON response into a ReviewResult and
// validates it. A malformed or gate-violating response is a hard error.
func ParseReviewResult(raw string) (*ReviewResult, error) {
	trimmed := stripCodeFences(strings.TrimSpace(raw))
	var r ReviewResult
	if err := json.Unmarshal([]byte(trimmed), &r); err != nil {
		return nil, fmt.Errorf("reviewer returned non-JSON or malformed response: %w (raw: %.200q)", err, trimmed)
	}
	if err := ValidateReviewResult(&r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ValidateReviewResult enforces the reject-by-default contract:
//   - verdict must be exactly "pass" or "reject";
//   - strongest_objection must be non-empty (ALWAYS — a reject with no
//     objection is the LGTM-bias failure we are guarding against, and a pass
//     must still name its closest concern);
//   - catch must be non-empty.
//
// Any violation is an error, never a coerced pass (Critical Principle 2).
func ValidateReviewResult(r *ReviewResult) error {
	switch r.Verdict {
	case VerdictPass, VerdictReject:
	default:
		return fmt.Errorf("reviewer verdict %q is not one of {pass, reject}", r.Verdict)
	}
	if strings.TrimSpace(r.StrongestObjection) == "" {
		return fmt.Errorf("reviewer verdict %q has empty strongest_objection (reject-by-default requires a stated objection; a missing one is not a pass)", r.Verdict)
	}
	if strings.TrimSpace(r.Catch) == "" {
		return fmt.Errorf("reviewer verdict %q has empty catch field", r.Verdict)
	}
	return nil
}

// stripCodeFences removes a leading ```json / ``` fence and trailing ``` if a
// provider wrapped the JSON despite the instruction not to.
func stripCodeFences(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// Drop first line (```json or ```)
	if nl := strings.IndexByte(s, '\n'); nl >= 0 {
		s = s[nl+1:]
	}
	if idx := strings.LastIndex(s, "```"); idx >= 0 {
		s = s[:idx]
	}
	return strings.TrimSpace(s)
}
