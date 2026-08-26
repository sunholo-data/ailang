package coordinator

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// M-MESSAGE-PLANE-FAIL-LOUD M2 (decision D2: explicit triage_only).
//
// `public-feedback` has no registered agent ON PURPOSE: submit_feedback documents
// auto_dispatch as "default false — files for human triage", so anonymous input is
// never handed to something that acts on it. Discord is its routing (verified
// delivering 2026-08-26, daemon --env prod, "🌐 External feedback").
//
// But "no agent" and "agent forgotten" were indistinguishable in config, which is
// what sent a 2026-08-26 triage session chasing a phantom routing gap (#900) and
// filing an issue for work that was deliberate. Declaring the intent removes the
// third state.
func TestTriageOnlyInboxes_Parse(t *testing.T) {
	var cfg CoordinatorConfig
	src := `
triage_only_inboxes:
  - public-feedback
  - user
agents:
  - id: some-agent
    inbox: some-inbox
    workspace: org/repo
`
	if err := yaml.Unmarshal([]byte(src), &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(cfg.TriageOnlyInboxes) != 2 {
		t.Fatalf("expected 2 triage-only inboxes, got %d (%v)", len(cfg.TriageOnlyInboxes), cfg.TriageOnlyInboxes)
	}
	if cfg.TriageOnlyInboxes[0] != "public-feedback" {
		t.Errorf("expected public-feedback first, got %q", cfg.TriageOnlyInboxes[0])
	}
}

func TestRegistry_IsTriageOnly(t *testing.T) {
	r := NewAgentRegistry()
	r.SetTriageOnlyInboxes([]string{"public-feedback", "user"})

	if !r.IsTriageOnly("public-feedback") {
		t.Error("public-feedback must be recognised as triage-only")
	}
	if !r.IsTriageOnly("user") {
		t.Error("user must be recognised as triage-only")
	}
	if r.IsTriageOnly("sprint-executor") {
		t.Error("an ordinary agent inbox must NOT be triage-only")
	}
}

// A triage-only inbox must still never dispatch. This pins that D2 changes only
// how the intent is DECLARED, not the refusal behaviour that already shipped in
// resolveInboxAgent.
func TestTriageOnly_StillNeverDispatches(t *testing.T) {
	r := NewAgentRegistry()
	r.SetTriageOnlyInboxes([]string{"public-feedback"})

	if agent := r.GetAgentForInbox("public-feedback"); agent != nil {
		t.Errorf("a triage-only inbox must resolve to no agent, got %q", agent.ID)
	}
}

// The whole point of D2: an inbox that is neither served nor declared is a
// reportable config gap, distinguishable from a deliberate human-triage inbox.
func TestRegistry_UndeclaredUnroutedInboxIsReported(t *testing.T) {
	r := NewAgentRegistry()
	r.SetTriageOnlyInboxes([]string{"public-feedback"})

	if r.IsUndeclaredUnrouted("public-feedback") {
		t.Error("a declared triage-only inbox is NOT a config gap")
	}
	if !r.IsUndeclaredUnrouted("pkg:sunholo/forgotten") {
		t.Error("an inbox with neither an agent nor a triage_only declaration IS a config gap")
	}
}

// Guard: with nothing declared, nothing is treated as triage-only — so the
// accessor cannot silently default to "everything is fine".
func TestTriageOnly_EmptyDeclarationTreatsNothingAsDeclared(t *testing.T) {
	r := NewAgentRegistry()
	r.SetTriageOnlyInboxes(nil)
	if r.IsTriageOnly("public-feedback") {
		t.Error("with no declaration, nothing should be treated as triage-only")
	}
	if !r.IsUndeclaredUnrouted("public-feedback") {
		t.Error("undeclared and unserved must report as a config gap")
	}
}
