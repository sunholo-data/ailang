package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/mission/quorum"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// The review phase (M-AILANG-LANE-EXECUTOR-SHADOW M6, design layer 3): only
// when the grade layers tie or disagree. An off-Anthropic reviewer, reached the
// way design-quorum reaches one, compares the two diffs against the task and
// must name a winner, its strongest objection to that winner, and a confidence.
// Reject-by-default in spirit: an answer that does not parse, or a reviewer
// that cannot be reached or would exceed its cap, is ungraded — never a silent
// verdict.

// shadowReviewer is the reviewer call: the JSON caller and its pricing.
type shadowReviewer struct {
	call     func(system, user, schema string) (string, float64, error) // raw JSON, provider-reported cost
	estimate func(inputTokens, outputTokens int) float64
	maxOut   int
}

// resolveShadowReviewer reaches a models.yml reviewer through design-quorum's
// resolver (openai, google or ollama lanes only).
func resolveShadowReviewer(model string) (*shadowReviewer, error) {
	caller, mc, err := quorum.ResolveCaller(model)
	if err != nil {
		return nil, err
	}
	return &shadowReviewer{
		// Cost the way design-quorum costs a reviewer: the provider's token
		// counts times the model's registry pricing.
		call: func(system, user, schema string) (string, float64, error) {
			raw, resp, err := caller.CallJSON(system, user, schema)
			return raw, shadowReviewCost(mc.Pricing, resp), err
		},
		estimate: func(in, out int) float64 { return mc.Pricing.Cost(in, out, 0, 0) },
		maxOut:   reviewExpectedOutputTokens,
	}, nil
}

func shadowReviewCost(p modelreg.Pricing, resp *ai.Response) float64 {
	if resp == nil {
		return 0
	}
	return p.Cost(resp.InputTokens, resp.OutputTokens, 0, 0)
}

const (
	// reviewExpectedOutputTokens is the pre-flight output estimate, as in
	// design-quorum: the answer is a small JSON object.
	reviewExpectedOutputTokens = 1024
	reviewDiffCap              = 60000 // chars per diff in the prompt
	reviewDirectiveCap         = 20000
)

const shadowReviewSystem = `You compare two candidate changes, A and B, made by two different coding agents for the same task in an AILANG repository. Judge which change better accomplishes the task: correctness first, then whether it does what was asked and nothing harmful besides. You are given the results of automated checks; weigh them, but they may be incomplete. Start from the assumption that neither is better and change your mind only on evidence in the diffs. Answer only with JSON.`

const shadowReviewSchema = `{"type":"object","properties":{"winner":{"type":"string","enum":["A","B","equivalent"]},"strongest_objection":{"type":"string"},"confidence":{"type":"number"}},"required":["winner","strongest_objection","confidence"]}`

// reviewShadowComparison runs the reviewer and lands the final verdict.
func reviewShadowComparison(ctx context.Context, store coordinator.ShadowComparisonStore, parent, directive, model, maxCost string, resolve func(string) (*shadowReviewer, error)) error {
	c, err := store.GetShadowComparison(ctx, parent)
	if err != nil {
		return fmt.Errorf("shadow review: %w", err)
	}
	review := runShadowReview(c, directive, model, maxCost, resolve)
	return writeShadowPhase(ctx, store, parent, coordinator.ShadowGradeReviewing, func(cur *coordinator.ShadowComparison) {
		cur.Review = review
		cur.Verdict, cur.UngradedReason = coordinator.FinalShadowVerdict(cur)
		cur.GradeState = coordinator.ShadowGradeDone
	})
}

func runShadowReview(c *coordinator.ShadowComparison, directive, model, maxCost string, resolve func(string) (*shadowReviewer, error)) *coordinator.ShadowReview {
	out := &coordinator.ShadowReview{Model: model}
	limit, err := strconv.ParseFloat(maxCost, 64)
	if model == "" || err != nil || limit <= 0 {
		out.Ungraded = "reviewer_unavailable: no model or cost cap"
		return out
	}
	r, err := resolve(model)
	if err != nil {
		out.Ungraded = "reviewer_unavailable: " + err.Error()
		return out
	}
	primaryIsA := reviewPrimaryIsA(c.ParentID)
	user := shadowReviewPrompt(c, directive, primaryIsA)
	if est := r.estimate(len(user)/4+len(shadowReviewSystem)/4, r.maxOut); est > limit {
		out.Ungraded = fmt.Sprintf("reviewer_over_budget: estimated $%.4f > cap $%.4f", est, limit)
		return out
	}
	raw, cost, err := r.call(shadowReviewSystem, user, shadowReviewSchema)
	out.CostUSD = cost
	if err != nil {
		out.Ungraded = "reviewer_unavailable: " + err.Error()
		return out
	}
	var ans struct {
		Winner             string  `json:"winner"`
		StrongestObjection string  `json:"strongest_objection"`
		Confidence         float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(stripJSONFences(raw)), &ans); err != nil ||
		strings.TrimSpace(ans.StrongestObjection) == "" || ans.Confidence < 0 || ans.Confidence > 1 {
		out.Ungraded = "reviewer_invalid"
		return out
	}
	switch ans.Winner {
	case "A", "B":
		if (ans.Winner == "A") == primaryIsA {
			out.Winner = coordinator.ShadowSidePrimary
		} else {
			out.Winner = coordinator.ShadowSideShadow
		}
	case "equivalent":
		out.Winner = "equivalent"
	default:
		out.Ungraded = "reviewer_invalid"
		return out
	}
	out.StrongestObjection, out.Confidence = ans.StrongestObjection, ans.Confidence
	return out
}

// reviewPrimaryIsA fixes the A/B order per comparison from its ID: stable on a
// re-run, and not always the primary first, so position bias cannot pass for a
// harness preference.
func reviewPrimaryIsA(parentID string) bool {
	h := fnv.New32a()
	_, _ = h.Write([]byte(parentID))
	return h.Sum32()%2 == 0
}

func shadowReviewPrompt(c *coordinator.ShadowComparison, directive string, primaryIsA bool) string {
	a, b := c.Primary, c.Shadow
	ga, gb := sideChecks(c, true), sideChecks(c, false)
	if !primaryIsA {
		a, b, ga, gb = b, a, gb, ga
	}
	if strings.TrimSpace(directive) == "" {
		directive = "(the task text could not be read; judge the changes on their own merits)"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "## Task\n%s\n\n", capText(directive, reviewDirectiveCap))
	for _, x := range []struct {
		name  string
		run   *coordinator.ShadowRunResult
		check string
	}{{"A", a, ga}, {"B", b, gb}} {
		diff, status := "(no change produced)", "unknown"
		if x.run != nil {
			status = x.run.Status
			if strings.TrimSpace(x.run.Diff) != "" {
				diff = capText(x.run.Diff, reviewDiffCap)
			}
		}
		fmt.Fprintf(&sb, "## Change %s (run status: %s)\nAutomated checks: %s\n```diff\n%s\n```\n\n", x.name, status, x.check, diff)
	}
	return sb.String()
}

// sideChecks summarises the automated layers for one side in plain words.
func sideChecks(c *coordinator.ShadowComparison, primary bool) string {
	var parts []string
	pick := func(m *coordinator.ShadowMechanicalGrade) coordinator.ShadowSideGrade {
		if primary {
			return m.Primary
		}
		return m.Shadow
	}
	describe := func(label string, g coordinator.ShadowSideGrade) string {
		switch {
		case g.Ungraded != "":
			return label + ": not run (" + g.Ungraded + ")"
		case g.Pass:
			return label + ": pass"
		default:
			return label + ": FAIL"
		}
	}
	if c.Mechanical != nil {
		parts = append(parts, describe("check+tests", pick(c.Mechanical)))
	}
	if c.MergedTests != nil {
		parts = append(parts, describe("accepted fix's tests", pick(c.MergedTests)))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "; ")
}

func capText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n… (truncated)"
}

func stripJSONFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
