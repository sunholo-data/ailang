package main

import (
	"fmt"
	"os"
	"strings"
)

// Who may resolve an approval unattended.
//
// Authority is a property of WHO THE SESSION IS, not of how good the row looks.
// The operator's rule (Mark, attended 2026-09-08): "the default is none, it
// always needs approval, but controller sessions and high-end models in mission
// loops like fable/astra have the authority to do approvals unattended."
//
// So the grant is identity-first, and the identity has to come from something
// the DRIVER sets, not something the session says about itself. Two exist:
//
//	MISSION_CONTROL_ACTIVE=1   exported by tools/launchd/mission-control.sh once the
//	                           controller plan is resolved post-degradation
//	CONTROLLER_ID              "<provider>:<model>", e.g. claude:claude-fable-5-1,
//	                           codex:gpt-5.6-sol, pi:ollama/glm-5.3:cloud
//
// Those come off the plist and the driver's own ladder walk, so a demoted
// controller (the fallback chain drops to glm-5.3 in a dry-out) loses the grant
// automatically — which is the point. The rung that is trusted to decide is the
// rung the driver actually got, not the one the plist hoped for.
//
// TWO RULES THAT ARE NOT PREFERENCES:
//
//  1. A ROLE IS NOT A CONTROLLER. A designer, planner, executor or evaluator
//     spawned by the loop gets no authority however strong its model, because
//     the agent that produced a change must never be the one that approves it.
//     Generator-not-equal-judge is already a non-negotiable property of this
//     loop; an approval is the last place to give it up.
//  2. AN UNREVIEWABLE ROW IS NOBODY'S TO APPROVE UNATTENDED. A row whose
//     executor recorded no diff cannot be reviewed by an agent at all — it
//     cannot go and look at the PR — and approving what you cannot see is the
//     defect #921 was filed for. Authority covers reviewable rows; the rest are
//     reported as needing the operator. `AILANG_APPROVAL_POLICY=always` is the
//     deliberate override.
//
// WHAT THIS IS NOT: enforcement. Every caller drives the same CLI against the
// same store, and any session can export any variable, so this decides what a
// session is TOLD it may do. It is a routing rule for a fleet that follows its
// configuration, not a control against one that does not — and saying so beats
// implying a boundary that is not there.

// defaultAuthorityModels are the rungs trusted to decide unattended.
//
// Matched as substrings against CONTROLLER_ID so a version bump does not
// silently revoke a grant (claude-fable-5-1 → claude-fable-5-2 keeps "fable").
// Deliberately NOT the whole ladder: codex:gpt-5.6-sol and the pi/glm rungs are
// the loop's dry-out fallbacks — able to keep the mission breathing, not chosen
// for judgment.
var defaultAuthorityModels = []string{"fable", "astra", "opus"}

// approvalAuthority is the resolved answer to "may THIS session decide?"
type approvalAuthority struct {
	Granted bool
	// Identity is who the session resolved as, for the banner and the audit
	// line. An approval recorded without saying which controller made it is a
	// decision with no author.
	Identity string
	// Reason states the grant or the refusal in the operator's terms.
	Reason string
	// RequireReviewable withholds rows with no visible diff even under a grant.
	// False only under an explicit `always`.
	RequireReviewable bool
	// RequireEvaluatorPass additionally demands an independent PASS verdict.
	// Only `evaluated` sets it: a controller grant is trust in the DECIDER, and
	// layering a verdict requirement on top would silently cover nothing on the
	// package lanes, which have no evaluator edge and so never carry a verdict.
	RequireEvaluatorPass bool
}

// authorityModels reads the allowlist, defaulting to fable/astra/opus.
func authorityModels() []string {
	raw := strings.TrimSpace(os.Getenv("AILANG_APPROVAL_AUTHORITY_MODELS"))
	if raw == "" {
		return defaultAuthorityModels
	}
	var out []string
	for _, m := range strings.Split(raw, ",") {
		if m = strings.ToLower(strings.TrimSpace(m)); m != "" {
			out = append(out, m)
		}
	}
	if len(out) == 0 {
		return defaultAuthorityModels
	}
	return out
}

// resolveApprovalAuthority answers "may this session decide, and as whom?"
//
// Go gathers the facts the driver published; the AILANG module decides what
// they mean. An unevaluable decision REFUSES with the reason attached — see
// coordinator_approvals_engine.go for why there is deliberately no second Go
// implementation to fall back to.
func resolveApprovalAuthority() approvalAuthority {
	// An explicit policy wins — it is the rollback and the test seam, and it is
	// an operator override rather than an identity question, so it does not go
	// through the decision module.
	switch resolveApprovalPolicy() {
	case approvalPolicyAlways:
		return approvalAuthority{
			Granted:  true,
			Identity: identityLabel(),
			Reason:   "AILANG_APPROVAL_POLICY=always — every pending row, reviewable or not",
			// The one path that does NOT withhold an unreviewable row. Explicit
			// is the whole point: an operator who sets `always` has said so.
			RequireReviewable: false,
		}
	case approvalPolicyEvaluated:
		return approvalAuthority{
			Granted:              true,
			Identity:             identityLabel(),
			Reason:               "AILANG_APPROVAL_POLICY=evaluated — rows an independent evaluator PASSed, with a visible diff",
			RequireReviewable:    true,
			RequireEvaluatorPass: true,
		}
	}

	facts := deciderFacts{
		MissionControlActive: os.Getenv("MISSION_CONTROL_ACTIVE") == "1",
		ControllerID:         strings.TrimSpace(os.Getenv("CONTROLLER_ID")),
		MissionRole:          strings.TrimSpace(os.Getenv("MISSION_ROLE")),
		AttendedGrant:        os.Getenv("AILANG_APPROVAL_CONTROLLER") == "1",
	}

	granted, identity, reason, err := callDecide(facts, authorityModels())
	if err != nil {
		// Refusing is both fail-loud and the safe direction: it lands on the
		// default posture rather than on an unreviewed second opinion.
		return approvalAuthority{
			Identity: identityLabel(),
			Reason:   "authority could not be evaluated, so nothing is granted: " + err.Error(),
		}
	}

	if identity == "" {
		identity = identityLabel()
	}
	if granted {
		reason = reason + " (" + strings.Join(authorityModels(), ", ") + ")"
	}
	return approvalAuthority{
		Granted:  granted,
		Identity: identity,
		Reason:   reason + authorityGrantHint(granted),
		// A grant by identity never waives reviewability; only an explicit
		// `always` does.
		RequireReviewable: true,
	}
}

// authorityGrantHint tells a refused session how the grant is obtained. Only on
// a refusal — a granted session does not need instructions.
func authorityGrantHint(granted bool) string {
	if granted {
		return ""
	}
	return " (a mission controller on " + strings.Join(authorityModels(), "/") +
		" is granted automatically; set AILANG_APPROVAL_CONTROLLER=1 for an attended session)"
}

// identityLabel names the session as well as it can be named, for the audit
// trail. Best effort, and honest about it: an unnamed session says so rather
// than borrowing a label it has not earned.
func identityLabel() string {
	if id := strings.TrimSpace(os.Getenv("CONTROLLER_ID")); id != "" {
		return "controller " + id
	}
	if os.Getenv("CLAUDECODE") == "1" {
		if s := strings.TrimSpace(os.Getenv("CLAUDE_CODE_SESSION_ID")); s != "" {
			return "attended claude-code session " + s
		}
		return "attended claude-code session"
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	return "unidentified session"
}

// coversRow applies an authority to one pending row, in AILANG.
//
// An error here refuses the row rather than covering it: the same direction as
// an unevaluable grant, for the same reason.
func (a approvalAuthority) coversRow(row pendingApprovalJSON) (bool, string) {
	if !a.Granted {
		return false, a.Reason
	}
	covered, err := callCoversRow(row.DiffAvailable, row.Evaluation, a.RequireReviewable, a.RequireEvaluatorPass)
	if err != nil {
		return false, "row coverage could not be evaluated: " + err.Error()
	}
	if !covered {
		switch {
		case a.RequireReviewable && !row.DiffAvailable:
			reason := "no visible diff — cannot approve what cannot be reviewed"
			if row.DiffUnavailable != "" {
				reason = fmt.Sprintf("%s (%s)", reason, row.DiffUnavailable)
			}
			return false, reason
		case row.Evaluation == "":
			return false, "no evaluator verdict yet"
		default:
			return false, "evaluator verdict " + row.Evaluation
		}
	}
	return true, a.Reason
}
