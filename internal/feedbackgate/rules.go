package feedbackgate

import (
	"regexp"
	"strings"
)

// Reason codes carried on a Verdict. Structured strings so audit records and
// dashboards can group by reason without parsing prose.
const (
	ReasonPassed                = "passed"
	ReasonNotAuthorized         = "not_authorized_for_dispatch"
	ReasonBodyTooLarge          = "body_too_large"
	ReasonSpamPattern           = "spam_pattern"
	ReasonUntrustedSource       = "untrusted_source"
	ReasonUnknownInbox          = "unknown_inbox"
	ReasonUnknownCategory       = "unknown_category"
	ReasonContactCooldown       = "contact_cooldown"
	ReasonClassifierInjection   = "classifier_prompt_injection"
	ReasonClassifierNotGenuine  = "classifier_not_genuine"
	ReasonClassifierNoValue     = "classifier_no_dispatch_value"
	ReasonClassifierMismatch    = "classifier_category_mismatch"
	ReasonClassifierParseFailed = "classifier_parse_failed"
	ReasonClassifierError       = "classifier_error"
	ReasonBudgetExceeded        = "classifier_budget_exceeded"
)

const agentSenderPrefix = "agent-"

// spamURLThreshold is the number of URLs above which a body is treated as spam.
const spamURLThreshold = 5

// spamBase64MinLen is the minimum length of a contiguous base64-ish blob that
// trips the spam rule (~1KB).
const spamBase64MinLen = 1024

var (
	urlRegexp = regexp.MustCompile(`https?://[^\s]+`)
	// base64Regexp matches a contiguous run of base64-ish characters. Go's RE2
	// caps bounded repeats at 1000, so we match runs of >=200 chars and then
	// length-check the longest match against spamBase64MinLen — a plain
	// {1024,} bound would panic at compile time.
	base64Regexp = regexp.MustCompile(`[A-Za-z0-9+/]{200,}={0,2}`)
)

// applyRules runs the deterministic pre-filter. Rules run in order; first
// match wins. It is pure (no IO) and returns dispatch only when every rule
// passes. cfg is assumed normalized.
func applyRules(in Input, cfg FeedbackGateConfig) Verdict {
	// Rule 1: category must carry the auto: prefix to be dispatch-authorized.
	// Without it the message is filed (a human can still triage it).
	if !hasAutoPrefix(in.Category) {
		return Verdict{Action: ActionFile, Reason: ReasonNotAuthorized}
	}

	// Rule 2: oversized body → reject. (Trim first so trailing whitespace
	// doesn't push a borderline body over.)
	body := strings.TrimSpace(in.Body)
	if len(body) > cfg.MaxBodyBytes {
		return Verdict{Action: ActionReject, Reason: ReasonBodyTooLarge}
	}

	// Rule 3: obvious spam patterns → reject.
	if looksLikeSpam(body) {
		return Verdict{Action: ActionReject, Reason: ReasonSpamPattern}
	}

	// Rule 4: sender must be trusted (mcp-public or agent-*) → else reject.
	if !senderAllowed(in.From, cfg.AllowedSenders) {
		return Verdict{Action: ActionReject, Reason: ReasonUntrustedSource}
	}

	// Rule 5: inbox must be a known routing target (pkg:* or internal) → else
	// reject. An unknown inbox means the message can't be safely routed.
	if !inboxAllowed(in.Inbox) {
		return Verdict{Action: ActionReject, Reason: ReasonUnknownInbox}
	}

	// Rule 6: category (stripped of auto:) must be known → else file for a
	// human to route.
	if !stringInSlice(cfg.KnownCategories, strippedCategory(in.Category)) {
		return Verdict{Action: ActionFile, Reason: ReasonUnknownCategory}
	}

	// All rules passed; the pre-filter would dispatch. Downstream stages
	// (cooldown, classifier) may still override.
	return Verdict{Action: ActionDispatch, Reason: ReasonPassed, Cost: estimatedDispatchCostUSD}
}

// looksLikeSpam applies cheap regex heuristics: too many URLs, or a large
// base64-ish blob embedded in the body.
func looksLikeSpam(body string) bool {
	if len(urlRegexp.FindAllString(body, spamURLThreshold+1)) > spamURLThreshold {
		return true
	}
	for _, run := range base64Regexp.FindAllString(body, -1) {
		if len(run) >= spamBase64MinLen {
			return true
		}
	}
	return false
}

// senderAllowed reports whether from is a trusted sender: any agent-* sender,
// or an explicit member of the allowlist.
func senderAllowed(from string, allowlist []string) bool {
	if strings.HasPrefix(from, agentSenderPrefix) {
		return true
	}
	return stringInSlice(allowlist, from)
}

// inboxAllowed reports whether the routing target is acceptable. Public
// feedback routes to pkg:* inboxes; a small set of internal inboxes are also
// permitted. Anything else is unknown.
func inboxAllowed(inbox string) bool {
	if strings.HasPrefix(inbox, "pkg:") {
		return true
	}
	switch inbox {
	case "coordinator", "design-doc-creator", "user":
		return true
	}
	return false
}

// stringInSlice reports whether s is in xs. Local helper (the coordinator
// package has an identically-named unexported one; keeping our own avoids any
// cross-package coupling).
func stringInSlice(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
