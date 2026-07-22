package ai

// M-AI-REASONING-EFFORT capability table.
//
// This table records the (provider, model) -> supported reasoning-effort set
// that has been LIVE-VERIFIED. It starts EMPTY on purpose: every model in the
// design's Verification Log is NEEDS-LIVE-SMOKE, and per the no-silent-fallback
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
// INVARIANT: no entries until live-smoke verified. Keep this literal empty.
var reasoningCapabilities = map[string]map[string]map[string]bool{
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
