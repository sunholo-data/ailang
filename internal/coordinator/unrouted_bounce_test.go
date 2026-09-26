package coordinator

import (
	"strings"
	"testing"
)

// A component that answers messages with messages carries the burden of proof.
// One legitimate message once produced 591 notifications and 59 job executions
// in 96 minutes (2026-08-31), so every guard here has an arm.

func TestSuggestInbox_CatchesTheRealTypos(t *testing.T) {
	known := []string{
		"pkg:sunholo/email", "pkg:sunholo/ailang_parse", "design-doc-creator",
		"sprint-planner", "sprint-executor", "coordinator", "pkg:sunholo/motoko_ext_*",
	}
	tests := []struct {
		wanted, want string
		why          string
	}{
		// The measured ones, 2026-09-08..10.
		{"pkg:sunholo/email_parse", "pkg:sunholo/email", "one suffix away from a real inbox"},
		{"sprint-plannr", "sprint-planner", "a dropped letter"},
		{"design-doc-creatr", "design-doc-creator", "a dropped vowel"},
		// Not typos. A wrong suggestion sends someone to the wrong agent, which
		// is worse than none — these must stay empty.
		{"ailang-core", "", "a different name entirely, not a near miss"},
		{"controlplane", "", "no close registered name"},
		// The asymmetry, pinned. `pkg:sunholo/ailang` was really sent on
		// 2026-09-09 meaning the LANGUAGE; `pkg:sunholo/ailang_parse` is the
		// document parser. Suggesting it would route the report to the wrong
		// agent, which is worse than saying nothing.
		{"pkg:sunholo/ailang", "", "a registered name that EXTENDS this one is a different product, not a fix"},
		{"", "", "nothing to suggest for an empty inbox"},
	}
	for _, tt := range tests {
		t.Run(tt.wanted, func(t *testing.T) {
			if got := SuggestInbox(tt.wanted, known); got != tt.want {
				t.Errorf("SuggestInbox(%q) = %q, want %q (%s)", tt.wanted, got, tt.want, tt.why)
			}
		})
	}
}

func TestSuggestInbox_NeverSuggestsAWildcard(t *testing.T) {
	// A pattern is not a name a sender can use, so offering it would be advice
	// that cannot be followed.
	got := SuggestInbox("pkg:sunholo/motoko_ext_1", []string{"pkg:sunholo/motoko_ext_*"})
	if got != "" {
		t.Errorf("suggested the pattern %q; a wildcard is not a usable inbox name", got)
	}
}

func TestIsUnroutedBounce_RecognisesOurOwnOutput(t *testing.T) {
	// THE loop guard. A bounce is delivered to the sender's inbox, that inbox may
	// itself be unrouted, and without this the reply bounces the bounce forever.
	if !isUnroutedBounce(&Message{Kind: bounceMessageType}) {
		t.Error("a bounce must be recognised by its message type")
	}
	if !isUnroutedBounce(&Message{Kind: strings.ToUpper(bounceMessageType)}) {
		t.Error("type matching must not be case-sensitive")
	}
	// The fallback, for a store that did not round-trip the type.
	if !isUnroutedBounce(&Message{Title: bounceTitlePrefix + "anything"}) {
		t.Error("a bounce must be recognised by its generated title prefix")
	}
	if isUnroutedBounce(&Message{Kind: "feedback", Title: "a real report"}) {
		t.Error("an ordinary message must not be mistaken for a bounce")
	}
	if isUnroutedBounce(nil) {
		t.Error("nil must not be treated as a bounce")
	}
}

func TestBounceUnroutedMessage_GuardsRefuseBeforeSending(t *testing.T) {
	registry := NewAgentRegistry()
	registry.SetTriageOnlyInboxes([]string{"public-feedback"})

	// A daemon with a registry but NO message store: every case below must be
	// refused by a guard before it ever reaches the store, so a nil store here
	// is a deliberate tripwire rather than a missing fixture.
	d := &Daemon{agentRegistry: registry}

	tests := []struct {
		name  string
		msg   *Message
		inbox string
	}{
		{"our own bounce", &Message{From: "daneel", Kind: bounceMessageType, Title: "x"}, "daneel"},
		{"a declared triage inbox", &Message{From: "mcp-public", Title: "x"}, "public-feedback"},
		{"no sender to answer", &Message{From: "", Title: "x"}, "ailang-core"},
		{"nil message", nil, "ailang-core"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Would panic on d.msgStore if a guard failed to stop it.
			if d.bounceUnroutedMessage(tt.msg, tt.inbox) {
				t.Errorf("expected no bounce for %s", tt.name)
			}
		})
	}
}

func TestBounceBody_SaysWhatHappenedAndWhatToDo(t *testing.T) {
	original := &Message{ID: "inbox_123", Title: "std/regex docs are wrong", From: "clients-planner"}
	body := bounceBody(original, "pkg:sunholo/ailang", "pkg:sunholo/ailang_parse",
		[]string{"design-doc-creator", "sprint-planner"})

	// The three things the sender cannot see for themselves.
	for _, want := range []string{
		"FILED BUT NOT DISPATCHED", // that nothing happened
		"pkg:sunholo/ailang",       // which name failed
		"no agent serves this",     // why
		"inbox_123",                // which message, so it can be resent
		"ailang messages inboxes",  // how to find a name that works
	} {
		if !strings.Contains(body, want) {
			t.Errorf("bounce body is missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, `Did you mean "pkg:sunholo/ailang_parse"`) {
		t.Errorf("a suggestion must be offered when one exists:\n%s", body)
	}
}

func TestBounceBody_OmitsTheSuggestionWhenThereIsNone(t *testing.T) {
	// Silence beats a guess. "Did you mean ''?" would be worse than no line.
	body := bounceBody(&Message{ID: "i1", Title: "t"}, "ailang-core", "", nil)
	if strings.Contains(body, "Did you mean") {
		t.Errorf("no suggestion should be offered when none is close:\n%s", body)
	}
	if !strings.Contains(body, "FILED BUT NOT DISPATCHED") {
		t.Error("the notice itself must still be delivered")
	}
}
