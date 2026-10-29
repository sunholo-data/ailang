package ai

// M-AI-REASONING-EFFORT capability table.
//
// This table records the (provider, model) -> supported reasoning-effort set
// that has been LIVE-VERIFIED. It started EMPTY on purpose: every model in the
// design's Verification Log was NEEDS-LIVE-SMOKE, and per the no-silent-fallback
// axiom an unregistered/unknown model + an explicit reasoning control MUST be
// rejected (fail-loud), never optimistically passed through on the guess that
// the server might ignore the field.
//
// Entries are added ONLY after the parked metered smoke verification (M7)
// confirms a provider/model honors a specific effort semantic exactly. Until
// then the table is empty and checkCapability rejects all non-empty controls.
//
// Follows the CodeModelNoVision / capability-gate precedent (a deterministic
// predicate keyed by provider+model), but for reasoning semantics rather than
// vision input.

// reasoningCapabilities maps a provider name to the set of models that have
// been verified to honor reasoning controls, and for each model the exact set
// of effort values it honors. Empty set of efforts for a registered model would
// mean "registered but honors nothing" (not currently used).
//
// INVARIANT: no entry without a live verification, recorded beside it.
var reasoningCapabilities = map[string]map[string]map[string]bool{
	// VERIFIED LIVE 2026-10-16 (OpenRouter Broadcast, prod observatory): over
	// 1,000 glm-5.3-flash calls carried reasoning:{"effort":"high"} across 7
	// hosts (CoreWeave, Together, Parasail, Friendli, Modal, Crusoe, Decart);
	// every one was accepted and returned reasoning tokens. Only "high" is
	// registered — the one value observed. The model cannot disable reasoning
	// ("Reasoning is mandatory for this endpoint", probed 2026-08-27), so "off"
	// must stay unregistered. This is the registry's declared effort for the
	// model (models.yml reasoning_effort), sent by every harness.
	reasoningProviderOpenRouter: {
		"z-ai/glm-5.3-flash": {ReasoningEffortHigh: true},
	},
	// Example shape once verified (DO NOT add without live smoke):
	//   reasoningProviderGemini: {
	//       "gemini-3-pro": {ReasoningEffortOff: true, ReasoningEffortLow: true, ...},
	//   },
}

// checkCapability returns nil if the provider/model is registered as honoring
// the given effort exactly, otherwise a typed ErrUnsupportedReasoningEffort.
//
// effort must be one of the four non-empty values ("off"/"low"/"medium"/"high");
// an empty effort should never reach here (callers short-circuit on unset).
func checkCapability(provider, model, effort string) *AIError {
	models, ok := reasoningCapabilities[provider]
	if !ok {
		return reasoningError(ErrUnsupportedReasoningEffort,
			"%s: no verified reasoning-capable models registered; model %q cannot honor reasoning effort %q (fail-loud: unknown capability is not an optimistic pass-through)",
			provider, model, effort)
	}
	efforts, ok := models[model]
	if !ok {
		return reasoningError(ErrUnsupportedReasoningEffort,
			"%s: model %q is not registered as reasoning-capable; cannot honor effort %q", provider, model, effort)
	}
	if !efforts[effort] {
		return reasoningError(ErrUnsupportedReasoningEffort,
			"%s: model %q does not honor reasoning effort %q", provider, model, effort)
	}
	return nil
}
