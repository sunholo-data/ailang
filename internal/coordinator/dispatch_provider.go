package coordinator

// Provider resolution for dispatch.
//
// An agent's executor binary is decided by its executor_variant, which selects
// the container image; the provider names the CLI the wrapper then runs inside
// it. The two must agree, and the dispatcher refuses the job when they do not —
// a guard that exists because a mismatch used to be discoverable only by the
// container, after it had cloned 24,000 files and cut a branch.
//
// That guard was doing its job. What was wrong was the value it received: the
// coordinator resolved the provider from its own plane-wide default and never
// consulted the agent, so an agent declaring provider "codex" was dispatched as
// "pi" and refused every time.

// ResolveDispatchProvider returns the provider to dispatch a task under.
//
// Precedence: the agent's own declaration, then the plane default, then
// "claude". The agent wins because it is the thing that also chose the
// executor_variant, and those two must agree; a plane default that overrides it
// can only ever disagree.
func ResolveDispatchProvider(agent *AgentConfig, planeDefault string) string {
	if agent != nil && agent.Provider != "" {
		return agent.Provider
	}
	if planeDefault != "" {
		return planeDefault
	}
	return "claude"
}

// resolveDispatchProvider is the call-site helper: it looks the agent up and
// applies the precedence above. An unknown agent falls back to the plane
// default, which is the pre-existing behaviour for agents that declare nothing.
func resolveDispatchProvider(registry *AgentRegistry, agentID string, cfg *CoordinatorConfig) string {
	planeDefault := ""
	if cfg != nil {
		planeDefault = cfg.DefaultProvider
	}
	var agent *AgentConfig
	if registry != nil && agentID != "" {
		agent = registry.GetAgentByID(agentID)
	}
	return ResolveDispatchProvider(agent, planeDefault)
}
