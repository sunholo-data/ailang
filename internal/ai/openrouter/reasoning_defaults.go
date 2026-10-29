package openrouter

import (
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/ai"
)

// reasoningDefaults maps an OpenRouter wire model ("z-ai/glm-5.3-flash") to the
// reasoning effort sent when the caller chose none. Registered once by the
// command layer from the model registry (modelreg reasoning_effort), like
// providerDefaults.
//
// Why it exists: models.yml reasoning_effort reached pi (as --thinking) and the
// standard-mode harness, but never a program's std/ai step — so motoko, whose
// runtime calls step without reasoning options, ran glm-5.3-flash at the vendor
// default while pi sent effort "high" (OpenRouter Broadcast, 2026-10-16). The
// effort is about the model, so it applies to every caller of the api_name.
var (
	reasoningDefaultsMu sync.RWMutex
	reasoningDefaults   map[string]string
)

// SetReasoningDefaults installs the per-model default reasoning effort. Keys
// are wire model names without the "openrouter/" prefix.
func SetReasoningDefaults(m map[string]string) {
	reasoningDefaultsMu.Lock()
	defer reasoningDefaultsMu.Unlock()
	reasoningDefaults = m
}

// ReasoningDefaultFor reports the installed default effort for a model.
func ReasoningDefaultFor(model string) string {
	reasoningDefaultsMu.RLock()
	defer reasoningDefaultsMu.RUnlock()
	return reasoningDefaults[strings.TrimPrefix(model, "openrouter/")]
}

// resolveReasoning is ai.ResolveReasoning plus the registry default: a caller's
// own reasoning control always wins; only a request that sets none gets the
// model's declared effort. The default goes through the same resolver as a
// caller's effort, so the capability gate applies to it too: a declared effort
// the model is not verified to honour fails loudly rather than shipping.
func resolveReasoning(req *ai.Request) (ai.ReasoningDecision, error) {
	d, err := ai.ResolveReasoning(req, "openrouter", req.Model)
	if err != nil || !d.IsNone() {
		return d, err
	}
	e := ReasoningDefaultFor(req.Model)
	if e == "" {
		return d, nil
	}
	withDefault := *req
	withDefault.ReasoningEffort = e
	return ai.ResolveReasoning(&withDefault, "openrouter", strings.TrimPrefix(req.Model, "openrouter/"))
}
