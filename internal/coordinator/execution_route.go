package coordinator

import "fmt"

// ExecutionRoute is the resolved cloud execution authority for one agent.
// Its fields are intentionally private: callers can inspect a route, but cannot
// create or partially rewrite one without passing through ResolveExecutionRoute.
type ExecutionRoute struct {
	agentID         string
	provider        string
	executorVariant string
	model           string
}

func (r ExecutionRoute) AgentID() string         { return r.agentID }
func (r ExecutionRoute) Provider() string        { return r.provider }
func (r ExecutionRoute) ExecutorVariant() string { return r.executorVariant }
func (r ExecutionRoute) Model() string           { return r.model }

// PermanentDispatchError identifies a dispatch request that cannot become valid
// by retrying. Route/config errors use this type so callers can terminalize them
// before reserving capacity or invoking an external service.
type PermanentDispatchError struct {
	AgentID         string
	Provider        string
	ExecutorVariant string
	Err             error
}

func (e *PermanentDispatchError) Error() string {
	if e == nil {
		return "permanent dispatch error"
	}
	reason := "invalid execution route"
	if e.Err != nil {
		reason = e.Err.Error()
	}
	return fmt.Sprintf("permanent dispatch error for agent %q (provider=%q, executor_variant=%q): %s",
		e.AgentID, e.Provider, e.ExecutorVariant, reason)
}

func (e *PermanentDispatchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var cloudProviderVariants = map[string]map[string]struct{}{
	"claude": {
		"default": {},
		"go":      {},
	},
	"gemini": {
		"gemini":    {},
		"gemini-go": {},
	},
	"codex": {
		"codex":    {},
		"codex-go": {},
	},
	"opencode": {"opencode": {}},
	"pi":       {"pi": {}},
	"motoko":   {"motoko": {}},
}

// NormalizeExecutionVariant preserves the sole legacy variant default: an
// empty Claude variant means the default Claude image. No other provider gets
// a fallback cell.
func NormalizeExecutionVariant(provider, variant string) string {
	if provider == "claude" && variant == "" {
		return "default"
	}
	return variant
}

// ValidateExecutionRoute applies the complete Cloud Run provider/image matrix.
// It is shared by config validation and the dispatcher guard.
func ValidateExecutionRoute(agentID, provider, variant string) error {
	normalized := NormalizeExecutionVariant(provider, variant)
	variants, providerKnown := cloudProviderVariants[provider]
	_, pairAllowed := variants[normalized]
	if providerKnown && pairAllowed {
		return nil
	}

	reason := fmt.Errorf("unsupported provider/executor_variant pair")
	if !providerKnown {
		reason = fmt.Errorf("unknown cloud provider")
	} else if normalized == "" {
		reason = fmt.Errorf("executor_variant is required for provider %q", provider)
	}
	return &PermanentDispatchError{
		AgentID:         agentID,
		Provider:        provider,
		ExecutorVariant: variant,
		Err:             reason,
	}
}

// ResolveExecutionRoute resolves provider, variant, and model once for the
// requested agent. The agent ID is supplied separately so a missing or
// mismatched registry lookup can still produce an attributable permanent error.
func ResolveExecutionRoute(agentID string, agent *AgentConfig) (ExecutionRoute, error) {
	if agent == nil {
		return ExecutionRoute{}, &PermanentDispatchError{
			AgentID: agentID,
			Err:     fmt.Errorf("agent is not registered"),
		}
	}
	if agent.ID != agentID {
		return ExecutionRoute{}, &PermanentDispatchError{
			AgentID: agentID,
			Err:     fmt.Errorf("registry returned agent %q", agent.ID),
		}
	}

	variant := NormalizeExecutionVariant(agent.Provider, agent.ExecutorVariant)
	if err := ValidateExecutionRoute(agent.ID, agent.Provider, variant); err != nil {
		return ExecutionRoute{}, err
	}
	model, err := ResolveModel(agent)
	if err != nil {
		return ExecutionRoute{}, &PermanentDispatchError{
			AgentID:         agent.ID,
			Provider:        agent.Provider,
			ExecutorVariant: variant,
			Err:             fmt.Errorf("resolve model: %w", err),
		}
	}

	return ExecutionRoute{
		agentID:         agent.ID,
		provider:        agent.Provider,
		executorVariant: variant,
		model:           model,
	}, nil
}

// ValidateCloudExecutionRoutes validates every literal and pipeline-expanded
// cloud agent. Local agents use host-specific executors and are intentionally
// outside the Cloud Run image matrix.
func ValidateCloudExecutionRoutes(cfg *CoordinatorConfig) error {
	if cfg == nil {
		return fmt.Errorf("coordinator config is nil")
	}

	agents := append([]*AgentConfig(nil), cfg.Agents...)
	expanded, err := cfg.ExpandPipelines()
	if err != nil {
		return err
	}
	agents = append(agents, expanded...)
	for _, agent := range agents {
		if agent == nil {
			return &PermanentDispatchError{Err: fmt.Errorf("agent config is nil")}
		}
		if agent.ResolveLane() == LaneLocal {
			continue
		}
		if _, err := ResolveExecutionRoute(agent.ID, agent); err != nil {
			return err
		}
	}
	return nil
}
