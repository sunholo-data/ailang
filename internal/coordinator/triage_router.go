package coordinator

import (
	"github.com/sunholo-data/ailang/internal/messaging"
)

// TriageConfig controls the coordinator's auto-triage router (M-MSG-TRIAGE-ROUTER).
// It is parsed from the `coordinator.triage` block of ~/.ailang/config.yaml and
// is opt-in: the router does nothing unless Enabled is true.
type TriageConfig struct {
	// Enabled turns the router on. Off by default — existing deployments see
	// zero behaviour change until they explicitly enable it.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// IntakeInboxes are the inboxes whose unread messages are triaged.
	// Empty defaults to {"user", "claude-code"}.
	IntakeInboxes []string `yaml:"intake_inboxes" json:"intake_inboxes,omitempty"`

	// PromoteInbox is where promoted messages are forwarded.
	// Empty defaults to "design-doc-creator".
	PromoteInbox string `yaml:"promote_inbox" json:"promote_inbox,omitempty"`

	// PromoteCategories are the message categories eligible for promotion.
	// Empty defaults to {"bug", "feature"}.
	PromoteCategories []string `yaml:"promote_categories" json:"promote_categories,omitempty"`

	// NoiseAgents are sender IDs whose uncategorized messages are dropped
	// (e.g. automated eval runs). Empty defaults to {"eval-suite"}.
	NoiseAgents []string `yaml:"noise_agents" json:"noise_agents,omitempty"`

	// ClusterSlot is the envelope slot used to group related messages.
	// Empty defaults to "intent".
	ClusterSlot string `yaml:"cluster_slot" json:"cluster_slot,omitempty"`

	// SimilarityThreshold is the cosine cutoff for clustering. Zero defaults to 0.75.
	SimilarityThreshold float64 `yaml:"similarity_threshold" json:"similarity_threshold,omitempty"`

	// PollIntervalSecs is the triage tick cadence. Zero defaults to 120.
	PollIntervalSecs int `yaml:"poll_interval_secs" json:"poll_interval_secs,omitempty"`
}

// normalized returns a copy of the config with empty/zero fields filled with
// defaults, so the router and classifier can rely on populated values.
func (c TriageConfig) normalized() TriageConfig {
	if len(c.IntakeInboxes) == 0 {
		c.IntakeInboxes = []string{"user", "claude-code"}
	}
	if c.PromoteInbox == "" {
		c.PromoteInbox = "design-doc-creator"
	}
	if len(c.PromoteCategories) == 0 {
		c.PromoteCategories = []string{"bug", "feature"}
	}
	if len(c.NoiseAgents) == 0 {
		c.NoiseAgents = []string{"eval-suite"}
	}
	if c.ClusterSlot == "" {
		c.ClusterSlot = "intent"
	}
	if c.SimilarityThreshold == 0 {
		c.SimilarityThreshold = 0.75
	}
	if c.PollIntervalSecs == 0 {
		c.PollIntervalSecs = 120
	}
	return c
}

// Decision is the triage outcome for a single message. The zero value is
// DecisionHold — the safe default (leave the message in place for a human).
type Decision int

const (
	DecisionHold    Decision = iota // leave in the intake inbox (no action)
	DecisionPromote                 // forward to PromoteInbox (design-doc-creator)
	DecisionDrop                    // noise — leave in place, never promote
)

func (d Decision) String() string {
	switch d {
	case DecisionPromote:
		return "promote"
	case DecisionDrop:
		return "drop"
	default:
		return "hold"
	}
}

// classify decides what to do with a single inbox message. Category is the
// authoritative signal: a categorized bug/feature (that is not itself a
// duplicate) is promoted regardless of sender; an uncategorized message from a
// known noise agent is dropped; everything else is held for a human to triage.
//
// classify is pure (no IO) so it is exhaustively table-testable. Callers should
// pass a normalized config.
func classify(msg messaging.InboxMessage, cfg TriageConfig) Decision {
	// Promotion wins over the noise filter: an explicitly categorized bug or
	// feature is actionable even if it came from an otherwise-noisy sender.
	if msg.DupOf == "" && stringInSlice(cfg.PromoteCategories, msg.Category) {
		return DecisionPromote
	}
	if stringInSlice(cfg.NoiseAgents, msg.FromAgent) {
		return DecisionDrop
	}
	return DecisionHold
}

func stringInSlice(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
