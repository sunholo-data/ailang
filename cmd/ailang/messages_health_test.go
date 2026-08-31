package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// TestClassifyInbox pins the three-way split the health verdict rests on.
//
// The distinction that matters: "unread" is not by itself a fault. A declared
// human-triage inbox SHOULD accumulate unread messages, and an unrouted inbox is
// a config question. Only a routable message sitting unread means work was filed
// and never dispatched — the number that should always be zero.
func TestClassifyInbox(t *testing.T) {
	reg := coordinator.NewAgentRegistry()
	if err := reg.Register(&coordinator.AgentConfig{
		ID: "pkg-sunholo-ailang-parse", Inbox: "pkg:sunholo/ailang_parse", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := reg.Register(&coordinator.AgentConfig{
		ID: "pkg-motoko-ext-family", Inbox: "pkg:sunholo/motoko_ext_*", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register wildcard: %v", err)
	}
	reg.SetTriageOnlyInboxes([]string{"public-feedback", "user"})

	tests := []struct {
		inbox string
		want  inboxBucket
		why   string
	}{
		{"pkg:sunholo/ailang_parse", bucketRoutable, "exact agent registered"},
		{"pkg:sunholo/motoko_ext_abi", bucketRoutable, "wildcard family member"},
		{"public-feedback", bucketTriage, "declared human-triage"},
		{"user", bucketTriage, "declared human-triage"},
		{"nobody-watches-this", bucketUnroutable, "no agent, not declared"},
		{"", bucketUnroutable, "empty inbox is unroutable, never routable"},
	}
	for _, tc := range tests {
		t.Run(tc.inbox, func(t *testing.T) {
			if got := classifyInbox(reg, tc.inbox); got != tc.want {
				t.Errorf("classifyInbox(%q) = %v, want %v (%s)", tc.inbox, got, tc.want, tc.why)
			}
		})
	}
}

// TestClassifyInboxPrefersAgentOverTriage: if an inbox somehow has BOTH an agent
// and a triage declaration, it is routable — an agent that exists will take the
// work, so counting it as triage would hide a real undelivered message.
func TestClassifyInboxPrefersAgentOverTriage(t *testing.T) {
	reg := coordinator.NewAgentRegistry()
	if err := reg.Register(&coordinator.AgentConfig{
		ID: "a", Inbox: "contested", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reg.SetTriageOnlyInboxes([]string{"contested"})

	if got := classifyInbox(reg, "contested"); got != bucketRoutable {
		t.Errorf("got %v, want bucketRoutable: an agent that exists will take the work", got)
	}
}

// TestEmphasizeIfNonZero: zero must render plainly, non-zero must carry the
// "should be 0" cue — the whole point of the row.
func TestEmphasizeIfNonZero(t *testing.T) {
	if got := emphasizeIfNonZero(0); got != "0" {
		t.Errorf("zero rendered as %q, want plain %q", got, "0")
	}
	got := emphasizeIfNonZero(3)
	if got == "3" {
		t.Error("a non-zero backlog must be emphasized, not rendered plainly")
	}
	if !strings.Contains(got, "should be 0") {
		t.Errorf("non-zero render %q must say what the expected value is", got)
	}
}
