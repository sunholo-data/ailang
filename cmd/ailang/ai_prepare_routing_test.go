package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// `ailang run` binds its AI handler through prepareRunAIBinding, not
// setupAIHandler; the registry's OpenRouter routing must be installed on that
// path too (2026-10-13: it was not, and motoko's GLM calls reached hosts pi
// never does).
func TestPrepareRunAIBinding_InstallsOpenRouterRouting(t *testing.T) {
	openrouter.SetProviderDefaults(nil)
	t.Cleanup(func() { openrouter.SetProviderDefaults(nil) })
	if _, err := prepareRunAIBinding("openrouter/z-ai/glm-5.3-flash"); err != nil {
		t.Fatalf("prepareRunAIBinding: %v", err)
	}
	if got := openrouter.ProviderDefaultsFor("z-ai/glm-5.3-flash"); got["only"] == nil {
		t.Fatalf("after the run binding, glm-5.3-flash routing = %v, want an `only` pool", got)
	}
}
