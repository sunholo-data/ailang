package coordinator

import "testing"

// The lane default, reversed for package inboxes by Mark on 2026-09-17.
//
// A `pkg:` inbox defaults to ailang_only; everything else still defaults to
// full. The default REMOVES authority, which is the only reason defaulting is
// acceptable here — the failure mode is a task that refuses and says why, not
// an agent handed a shell nobody granted.
func TestGetEffectiveToolPolicy_PackageInboxesDefaultToTheLane(t *testing.T) {
	for _, tc := range []struct {
		name  string
		agent *AgentConfig
		want  string
	}{
		{"package inbox, nothing declared", &AgentConfig{ID: "pkg-x", Inbox: "pkg:sunholo/auth"}, "ailang_only"},
		{"package family pattern", &AgentConfig{ID: "pkg-f", Inbox: "pkg:sunholo/motoko_ext_*"}, "ailang_only"},
		// An explicit declaration always wins, in both directions: a package
		// agent that genuinely needs a shell says so and it is visible.
		{"package inbox opting OUT", &AgentConfig{ID: "pkg-y", Inbox: "pkg:sunholo/auth", ToolPolicy: "full"}, "full"},
		{"non-package keeps full", &AgentConfig{ID: "sprint-executor", Inbox: "sprint-executor"}, "full"},
		{"non-package opting IN", &AgentConfig{ID: "daneel", Inbox: "daneel-executor", ToolPolicy: "ailang_only"}, "ailang_only"},
		// Not a package inbox: the prefix is the whole rule, and a name that
		// merely contains it is someone else's inbox.
		{"not a package inbox", &AgentConfig{ID: "z", Inbox: "my-pkg:thing"}, "full"},
	} {
		if got := tc.agent.GetEffectiveToolPolicy(); got != tc.want {
			t.Errorf("%s: GetEffectiveToolPolicy() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestIsPackageInbox(t *testing.T) {
	for in, want := range map[string]bool{
		"pkg:sunholo/auth":         true,
		"pkg:sunholo/motoko_ext_*": true,
		" pkg:sunholo/email ":      true, // config whitespace is not a different inbox
		"sprint-planner":           false,
		"my-pkg:thing":             false,
		"":                         false,
	} {
		if got := IsPackageInbox(in); got != want {
			t.Errorf("IsPackageInbox(%q) = %v, want %v", in, got, want)
		}
	}
}
