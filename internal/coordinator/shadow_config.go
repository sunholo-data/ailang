package coordinator

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// ShadowConfig runs a sample of an agent's tasks a second time on the other
// lane harness, with no side effects, so the two can be graded against each
// other on real work (M-AILANG-LANE-EXECUTOR-SHADOW, ratified 2026-10-13).
//
// The primary task is untouched. The shadow is a child task `<id>-shadow`
// carrying ShadowOf, dispatched to Provider/ExecutorVariant, and it may write
// nothing outside its own run directory: no push, no PR, no approval card, no
// handoff.
type ShadowConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Provider is the shadow's executor CLI. Required, and it must differ from
	// the primary's: shadowing a harness with itself measures nothing (D2).
	Provider string `yaml:"provider" json:"provider"`
	// ExecutorVariant picks the shadow's Cloud Run Job, and therefore its image.
	// Required, and it must be able to run Provider.
	ExecutorVariant string `yaml:"executor_variant" json:"executor_variant"`
	// Model overrides the shadow's model. Empty means the primary's model (D3),
	// which is what makes the comparison about the harness.
	Model string `yaml:"model" json:"model,omitempty"`
	// SampleRate is the fraction of eligible tasks shadowed, in [0, 1].
	SampleRate float64 `yaml:"sample_rate" json:"sample_rate"`
	// MaxCostUSD caps one shadow run. Zero means no per-run cap beyond the
	// shadow key's own monthly cap (D7).
	MaxCostUSD float64 `yaml:"max_cost_usd" json:"max_cost_usd,omitempty"`
	// OfferAlternative opts the agent in to a draft PR when the shadow wins and
	// the primary failed (D5). Off by default: report only.
	OfferAlternative bool `yaml:"offer_alternative" json:"offer_alternative,omitempty"`
	// ReviewerModel is the off-Anthropic model (a models.yml id on an openai,
	// google or ollama lane, as design-quorum admits) that breaks ties and
	// disagreements between the grade layers. Empty disables the reviewer: the
	// layered verdict stands.
	ReviewerModel string `yaml:"reviewer_model" json:"reviewer_model,omitempty"`
	// ReviewerMaxCostUSD caps one review, checked before the call. Zero means
	// the default, defaultShadowReviewerMaxCostUSD.
	ReviewerMaxCostUSD float64 `yaml:"reviewer_max_cost_usd" json:"reviewer_max_cost_usd,omitempty"`
}

// defaultShadowReviewerMaxCostUSD caps a review when the agent sets none.
const defaultShadowReviewerMaxCostUSD = 0.25

// reviewerBudget is the per-review cap in force.
func (s *ShadowConfig) reviewerBudget() float64 {
	if s.ReviewerMaxCostUSD > 0 {
		return s.ReviewerMaxCostUSD
	}
	return defaultShadowReviewerMaxCostUSD
}

// validateShadowConfig returns the config errors in an agent's shadow block,
// or nil when the agent has none. A block that is present but disabled is
// still validated, so it cannot be switched on later in a broken state.
func validateShadowConfig(agent *AgentConfig) []error {
	s := agent.Shadow
	if s == nil {
		return nil
	}
	fail := func(format string, args ...any) error {
		return ProviderDeclarationError{
			AgentID: agent.ID, Variant: s.ExecutorVariant, Declared: s.Provider,
			Reason: "shadow: " + fmt.Sprintf(format, args...),
		}
	}
	var errs []error
	if policy := agent.GetEffectiveToolPolicy(); policy != executor.ToolProfileAILANGOnly {
		errs = append(errs, fail("only ailang_only agents can be shadowed in v1, and this one runs tool_policy %q", policy))
	}
	if s.SampleRate < 0 || s.SampleRate > 1 {
		errs = append(errs, fail("sample_rate %v is outside [0, 1]", s.SampleRate))
	}
	if s.MaxCostUSD < 0 {
		errs = append(errs, fail("max_cost_usd %v is negative", s.MaxCostUSD))
	}
	if s.ReviewerMaxCostUSD < 0 {
		errs = append(errs, fail("reviewer_max_cost_usd %v is negative", s.ReviewerMaxCostUSD))
	}
	if m := strings.ToLower(s.ReviewerModel); strings.Contains(m, "claude") || strings.Contains(m, "anthropic") {
		errs = append(errs, fail("reviewer_model %q is an Anthropic model; the reviewer must be off-Anthropic", s.ReviewerModel))
	}
	if s.Provider == "" || s.ExecutorVariant == "" {
		return append(errs, fail("provider and executor_variant are both required"))
	}
	if primary := ResolveDispatchProvider(agent, ""); primary == s.Provider {
		errs = append(errs, fail("provider %q is the primary's own; a shadow must run the other harness", s.Provider))
	}
	errs = append(errs, shadowVariantErrors(s, fail)...)
	return errs
}

// shadowVariantErrors applies the same variant/provider rule as the agent's own
// declaration: the variant is the image, and the image decides which CLI exists.
func shadowVariantErrors(s *ShadowConfig, fail func(string, ...any) error) []error {
	if err := RetiredVariantError(s.ExecutorVariant); err != nil {
		return []error{fail("%v", err)}
	}
	if _, known := variantProviders[s.ExecutorVariant]; !known {
		return []error{fail("unknown executor_variant %q", s.ExecutorVariant)}
	}
	if derived, ok := ProviderForVariant(s.ExecutorVariant); ok && derived != s.Provider {
		return []error{fail("image agent-%s runs %q and cannot run %q",
			variantImage(s.ExecutorVariant), derived, s.Provider)}
	}
	return nil
}
