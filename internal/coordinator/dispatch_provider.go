package coordinator

import (
	"fmt"
	"sort"
)

// The executor CLI is DERIVED from the image, not declared alongside it
// (2026-09-03, Mark).
//
// executor_variant selects the Cloud Run Job and therefore the image — the Jobs
// API has no per-execution image override — and each image carries exactly the
// CLIs its Dockerfile installs. agent-pi has `pi` on $PATH; agent-codex-go has
// `codex`. They are genuinely separate containers and neither can invoke the
// other's binary.
//
// So which CLI runs was always a function of which image was chosen. Expressing
// it as a SECOND, independent field meant the two could disagree, and nothing
// forced them to agree — one field picked the box, another picked the tool.
//
// Measured across the live fleet on the day this changed: 34 agents, using four
// variants (pi x31, codex, codex-go, motoko), every one of which admits exactly
// one provider. Zero agents used the eval images, the only ones carrying several
// CLIs and therefore the only place the field could carry information. All 34
// declarations were CORRECT; the dispatcher ignored them and substituted a
// plane-wide default, which is how two pipeline stages became undispatchable.
//
// Deriving makes that state unrepresentable rather than merely detectable. A
// guard can be right and still leave you broken; an impossibility cannot.

// multiCLIVariants are the images that carry several executors, where the
// provider is a genuine choice and must be declared.
var multiCLIVariants = map[string]bool{
	"eval":    true, // agent-eval: claude + gemini + codex + opencode + pi
	"eval-go": true, //   ditto (FROM agent-eval)
}

// ProviderForVariant returns the only CLI an image can run.
//
// ok is false for the multi-CLI images, where the answer is a choice rather than
// a fact, and for unknown variants, which the dispatcher rejects separately.
func ProviderForVariant(variant string) (provider string, ok bool) {
	if multiCLIVariants[variant] {
		return "", false
	}
	allowed, known := variantProviders[variant]
	if !known || len(allowed) != 1 {
		return "", false
	}
	return allowed[0], true
}

// ResolveDispatchProvider returns the provider to dispatch a task under.
//
// The variant decides, because the variant is the image. A declared provider is
// only consulted for the multi-CLI images; anywhere else a declaration can only
// agree (adding nothing) or disagree (being wrong), and ValidateAgentProviders
// rejects the disagreement at config load rather than at dispatch.
func ResolveDispatchProvider(agent *AgentConfig, planeDefault string) string {
	if agent == nil {
		return planeDefault
	}
	if derived, ok := ProviderForVariant(agent.ExecutorVariant); ok {
		return derived
	}
	// Multi-CLI image, or a variant this build does not know: fall back to the
	// declaration, then the plane default.
	if agent.Provider != "" {
		return agent.Provider
	}
	return planeDefault
}

// resolveDispatchProvider is the call-site helper.
func resolveDispatchProvider(registry *AgentRegistry, agentID string, cfg *CoordinatorConfig) string {
	planeDefault := ""
	if cfg != nil {
		planeDefault = cfg.DefaultProvider
	}
	var agent *AgentConfig
	if registry != nil && agentID != "" {
		agent = registry.GetAgentByID(agentID)
	}
	if agent == nil {
		if planeDefault != "" {
			return planeDefault
		}
		return "claude"
	}
	if p := ResolveDispatchProvider(agent, planeDefault); p != "" {
		return p
	}
	return "claude"
}

// ProviderDeclarationError is a config-load complaint about a `provider:` field.
type ProviderDeclarationError struct {
	AgentID  string
	Variant  string
	Declared string
	Derived  string
	Reason   string
}

func (e ProviderDeclarationError) Error() string {
	return fmt.Sprintf("agent %q: %s (executor_variant=%q, declared provider=%q)",
		e.AgentID, e.Reason, e.Variant, e.Declared)
}

// ValidateAgentProviders rejects `provider:` declarations that cannot be honoured.
//
// Two failures, both config errors rather than runtime ones:
//
//   - a declaration that contradicts the image. Previously this dispatched, got
//     refused by the runtime guard, and retried forever.
//   - a MISSING declaration on a multi-CLI image, where nothing can derive the
//     answer. Silently defaulting there would pick an executor by accident.
//
// A declaration that merely agrees with the image is allowed and ignored: it is
// redundant, not wrong, and 34 agent configs carry one today.
func ValidateAgentProviders(agents []*AgentConfig) []error {
	var errs []error
	for _, agent := range agents {
		if agent == nil {
			continue
		}
		derived, ok := ProviderForVariant(agent.ExecutorVariant)
		switch {
		case ok && agent.Provider != "" && agent.Provider != derived:
			errs = append(errs, ProviderDeclarationError{
				AgentID: agent.ID, Variant: agent.ExecutorVariant,
				Declared: agent.Provider, Derived: derived,
				Reason: fmt.Sprintf("image agent-%s runs %q and cannot run %q; drop the provider field or fix the variant",
					variantImage(agent.ExecutorVariant), derived, agent.Provider),
			})
		case multiCLIVariants[agent.ExecutorVariant] && agent.Provider == "":
			errs = append(errs, ProviderDeclarationError{
				AgentID: agent.ID, Variant: agent.ExecutorVariant,
				Reason: "this image carries several executors, so provider cannot be derived and must be declared",
			})
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

func variantImage(variant string) string {
	if variant == "" || variant == "default" {
		return "agent"
	}
	return variant
}
