package coordinator

import (
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
)

// The GetEffective* accessors: what an agent ACTUALLY runs with once
// defaults are filled in. Kept beside, not inside, agent_registry.go (split at
// the 800-line gate); `ailang coordinator agents <id>` prints declared and
// effective side by side for exactly the reason these exist — an absent field
// is not an unset one.

// GetEffectiveToolPolicy is the tool_policy that RUNS: the declared value, or
// executor.ToolProfileFull when none is declared (D6 — the fleet default stays
// full; only a deployment that opts in narrows).
func (a *AgentConfig) GetEffectiveToolPolicy() string {
	if strings.TrimSpace(a.ToolPolicy) == "" {
		return executor.ToolProfileFull
	}
	return strings.TrimSpace(a.ToolPolicy)
}

// GetEffectiveInvokeConfig returns the agent's invoke config, or defaults for known agents.
// Returns nil for unknown agents without explicit config.
//
// Note: Deprecation warnings for using defaults should be logged by the caller,
// as they have logger access. Check if result differs from explicit config.
func (a *AgentConfig) GetEffectiveInvokeConfig() *InvokeConfig {
	if a.Invoke != nil {
		return a.Invoke
	}
	return DefaultInvokeConfig(a.ID)
}

// GetEffectiveOutputMarkers returns the agent's output markers, or defaults for known agents.
func (a *AgentConfig) GetEffectiveOutputMarkers() []string {
	if len(a.OutputMarkers) > 0 {
		return a.OutputMarkers
	}
	return DefaultOutputMarkers(a.ID)
}

// GitIdentity is the git author for an agent's commits.
type GitIdentity struct {
	Name  string `yaml:"name" json:"name"`
	Email string `yaml:"email" json:"email"`
}

// GetEffectiveArtifactPatterns returns the agent's artifact patterns, or defaults
// for known agents. Use it to DISCOVER what changed — never to BOUND what may.
//
// The default for an agent not in DefaultArtifactPatterns is `**/*`, and that is
// correct here: discovery should report every file the agent touched, and a
// narrower default would silently omit real changes from the evidence.
//
// It is wrong for any safety decision, because the widest possible pattern is
// the opposite of a bound. Measured 2026-09-11: the auto-merge scope guard read
// this method, so an agent with auto_merge and no declared patterns got `**/*`
// and the scope half of the guard bounded nothing. Safety decisions read the
// DECLARED field (a.ArtifactPatterns) and treat empty as refuse —
// daemon_tasks_exec.go does this deliberately and says so at the call site.
func (a *AgentConfig) GetEffectiveArtifactPatterns() []string {
	if len(a.ArtifactPatterns) > 0 {
		return a.ArtifactPatterns
	}
	return DefaultArtifactPatterns(a.ID)
}

// GetEffectiveTimeout returns the agent's configured hard ceiling timeout, or the default (60m).
// This is the maximum wall-clock time regardless of activity. Safe to call on nil receiver.
func (a *AgentConfig) GetEffectiveTimeout() time.Duration {
	if a != nil && a.Timeout != "" {
		if d, err := time.ParseDuration(a.Timeout); err == nil && d > 0 {
			return d
		}
	}
	return 60 * time.Minute
}

// GetEffectiveIdleTimeout returns the agent's configured idle timeout, or the default (3m).
// The agent is killed if no streaming events are produced for this duration.
// Safe to call on nil receiver.
func (a *AgentConfig) GetEffectiveIdleTimeout() time.Duration {
	if a != nil && a.IdleTimeout != "" {
		if d, err := time.ParseDuration(a.IdleTimeout); err == nil && d > 0 {
			return d
		}
	}
	return 3 * time.Minute
}
