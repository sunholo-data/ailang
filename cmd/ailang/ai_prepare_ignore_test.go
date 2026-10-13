package main

import (
	"slices"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// `ailang run` binds its AI handler through prepareRunAIBinding, not
// setupAIHandler; the registry's OpenRouter host exclusions must be installed
// on that path too (2026-10-13: they were not, and motoko's GLM calls still
// reached inference-net).
func TestPrepareRunAIBinding_InstallsOpenRouterIgnore(t *testing.T) {
	openrouter.SetProviderIgnore(nil)
	t.Cleanup(func() { openrouter.SetProviderIgnore(nil) })
	if _, err := prepareRunAIBinding("openrouter/z-ai/glm-5.3-flash"); err != nil {
		t.Fatalf("prepareRunAIBinding: %v", err)
	}
	if got := openrouter.ProviderIgnoreFor("z-ai/glm-5.3-flash"); !slices.Contains(got, "inference-net") {
		t.Fatalf("after the run binding, glm-5.3-flash ignore = %v, want inference-net", got)
	}
}
