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
		inbox   string
		msgType string
		want    inboxBucket
		why     string
	}{
		{"pkg:sunholo/ailang_parse", "request", bucketRoutable, "exact agent registered"},
		{"pkg:sunholo/motoko_ext_abi", "request", bucketRoutable, "wildcard family member"},
		{"public-feedback", "feedback", bucketTriage, "declared human-triage"},
		{"user", "notification", bucketTriage, "declared human-triage"},
		{"nobody-watches-this", "notification", bucketUnroutable, "no agent, not declared"},
		{"", "notification", bucketUnroutable, "empty inbox is unroutable, never routable"},
	}
	for _, tc := range tests {
		t.Run(tc.inbox+"/"+tc.msgType, func(t *testing.T) {
			if got := classifyInbox(reg, tc.inbox, tc.msgType); got != tc.want {
				t.Errorf("classifyInbox(%q, %q) = %v, want %v (%s)", tc.inbox, tc.msgType, got, tc.want, tc.why)
			}
		})
	}
}

// TestClassifyInboxAgentOutputIsNotUndelivered is the bug this bucket exists for.
//
// An agent files its own completion and approval_request into its own inbox.
// Judged by inbox alone they are "routable and unread", so the verdict counted
// them as work that was never dispatched: on 2026-09-14 prod read
// "DEGRADED 87 filed, routable, and never dispatched" when the real number was
// 4 — the other 83 were the agents' own output. A number that is never zero on a
// healthy plane trains the reader to skip it.
func TestClassifyInboxAgentOutputIsNotUndelivered(t *testing.T) {
	reg := coordinator.NewAgentRegistry()
	if err := reg.Register(&coordinator.AgentConfig{
		ID: "design-doc-creator", Inbox: "design-doc-creator", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	for _, tc := range []struct {
		msgType string
		want    inboxBucket
	}{
		{"completion", bucketResult},
		{"approval_request", bucketResult},
		{"response", bucketResult},
		{"inbox_unrouted_notice", bucketResult},
		{"request", bucketRoutable},
		{"handoff", bucketRoutable},
		{"notification", bucketRoutable},
		{"feedback", bucketRoutable},
	} {
		if got := classifyInbox(reg, "design-doc-creator", tc.msgType); got != tc.want {
			t.Errorf("a %q on an agent inbox = %v, want %v", tc.msgType, got, tc.want)
		}
	}
}

// TestClassifyInboxBounceToUnservedInboxStaysAGap: the result-type shortcut must
// apply ONLY on an inbox an agent serves.
//
// Measured 2026-09-13: five UNDELIVERED notices were addressed to "ailang",
// which is itself unregistered — a bounce that bounced. Bucketing every
// inbox_unrouted_notice as "output" would hide exactly that.
func TestClassifyInboxBounceToUnservedInboxStaysAGap(t *testing.T) {
	reg := coordinator.NewAgentRegistry()
	if got := classifyInbox(reg, "ailang", "inbox_unrouted_notice"); got != bucketUnroutable {
		t.Errorf("bounce to an unserved inbox = %v, want bucketUnroutable", got)
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

	if got := classifyInbox(reg, "contested", "request"); got != bucketRoutable {
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
