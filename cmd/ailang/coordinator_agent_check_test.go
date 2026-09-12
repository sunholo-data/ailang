package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Each case here is a failure that actually happened while adding one agent on
// 2026-09-11, and each one was silent at the time.

func TestCheckCoherence_CatchesFlagsThatCannotFire(t *testing.T) {
	tests := []struct {
		name  string
		agent coordinator.AgentConfig
		want  checkState
		says  string
	}{
		{
			// A deploy key is SSH-only; auto-merge is a GraphQL mutation.
			name:  "auto_merge with a deploy key",
			agent: coordinator.AgentConfig{SSHKeySecret: "k", SSHHostAlias: "a", AutoMerge: true, ArtifactPatterns: []string{"**/*.md"}},
			want:  stateFail,
			says:  "cannot use the GitHub API",
		},
		{
			// skip_approval + merge_branch is the real direct-push trigger.
			name:  "auto_merge with direct push",
			agent: coordinator.AgentConfig{SkipApproval: true, MergeBranch: "dev", AutoMerge: true, ArtifactPatterns: []string{"**/*.md"}},
			want:  stateFail,
			says:  "pushes DIRECTLY",
		},
		{
			// The scope guard refuses everything with no declaration, so
			// auto-merge would silently never fire.
			name:  "auto_merge with no artifact patterns",
			agent: coordinator.AgentConfig{ID: "x", AutoMerge: true},
			want:  stateFail,
			says:  "bound nothing",
		},
		{
			// The alias IS the bound; without it the key is not scoped.
			name:  "deploy key with no host alias",
			agent: coordinator.AgentConfig{SSHKeySecret: "k"},
			want:  stateFail,
			says:  "bound",
		},
		{
			name:  "a coherent deploy-key agent",
			agent: coordinator.AgentConfig{SSHKeySecret: "k", SSHHostAlias: "a", SkipApproval: true, MergeBranch: "dev"},
			want:  statePass,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkCoherence(&tt.agent)
			if got.State != tt.want {
				t.Fatalf("state = %v, want %v (detail: %s)", got.State, tt.want, got.Detail)
			}
			if tt.says != "" && !strings.Contains(got.Detail, tt.says) {
				t.Errorf("detail should explain %q, got: %s", tt.says, got.Detail)
			}
		})
	}
}

func TestDiffAgents_ReportsBehaviouralDrift(t *testing.T) {
	base := coordinator.AgentConfig{
		Workspace: "o/r", MergeBranch: "dev", Model: "m",
		SSHKeySecret: "s", SSHHostAlias: "a", ArtifactPatterns: []string{"docs/**"},
	}
	if d := diffAgents(&base, &base); d != "" {
		t.Errorf("identical configs must not report drift, got %q", d)
	}

	// The exact drift that shipped: the bucket had the agent without its
	// deploy-key settings, so it would have used the fleet token and failed.
	live := base
	live.SSHKeySecret = ""
	if d := diffAgents(&live, &base); !strings.Contains(d, "ssh_key_secret") {
		t.Errorf("a missing ssh_key_secret must be reported, got %q", d)
	}

	other := base
	other.MergeBranch = "main"
	if d := diffAgents(&other, &base); !strings.Contains(d, "merge_branch") {
		t.Errorf("a branch difference must be reported, got %q", d)
	}

	if d := diffAgents(nil, &base); d == "" {
		t.Error("a missing side must be reported, not treated as in sync")
	}
}

func TestUnknownConfigKeys_NamesInertAgentKeys(t *testing.T) {
	// `push_branch` reads as decisive and AgentConfig has no such field, so YAML
	// dropped it silently and the entry looked configured.
	cfg := []byte(`
coordinator:
  agents:
    - id: someone
      inbox: someone
      workspace: o/r
      push_branch: dev
`)
	keys, err := coordinator.UnknownConfigKeys(cfg)
	if err != nil {
		t.Fatalf("strict parse: %v", err)
	}
	joined := strings.Join(keys, " ")
	if !strings.Contains(joined, "push_branch") {
		t.Errorf("an inert agent key must be named, got %v", keys)
	}

	// And a clean config must be quiet, or the check gets ignored.
	clean := []byte(`
coordinator:
  agents:
    - id: someone
      inbox: someone
      workspace: o/r
      merge_branch: dev
`)
	keys, err = coordinator.UnknownConfigKeys(clean)
	if err != nil {
		t.Fatalf("strict parse: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("a clean config must report nothing, got %v", keys)
	}
}
