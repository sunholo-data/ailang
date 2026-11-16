package pi

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

// D-FLEET-35: the three routed OpenRouter rungs whose caps lived only in the rig's
// installed pi config now have registry rows, so a task built from those rows carries
// settings and the preflight HOLDS the installed config to them. The canonical config
// (what an install copies) must pass, and an installed file that lost the entry must
// refuse instead of letting pi fall back to its uncapped built-in catalogue.
func TestPreflight_HoldsDFleet35Rungs(t *testing.T) {
	c, err := modelreg.LoadModelsConfig("../../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load models.yml: %v", err)
	}
	canonical := "../../../tools/pi-extensions/models.mission.json"
	missing := writePiModels(t, `{"providers":{"openrouter":{"models":[{"id":"z-ai/glm-5.3-flash","maxTokens":131072}]}}}`)
	for _, row := range []string{"pi-or-glm-5-3", "pi-or-kimi-k3", "pi-or-minimax-m3"} {
		m, ok := c.Models[row]
		if !ok || m.AgentModelName == nil {
			t.Errorf("registry row %s missing — nothing holds its cap", row)
			continue
		}
		task := &executor.Task{
			MaxOutputTokens:    m.MaxOutputTokens,
			ReasoningEffort:    m.ReasoningEffort,
			OpenRouterProvider: c.EffectiveOpenRouterProvider(m.APIName),
		}
		if task.MaxOutputTokens == 0 {
			t.Errorf("%s declares no max_output_tokens, so the preflight skips it", row)
			continue
		}
		if err := checkInstalledModelConfig(*m.AgentModelName, task, canonical); err != nil {
			t.Errorf("%s: canonical pi config refused by the preflight: %v", row, err)
		}
		if err := checkInstalledModelConfig(*m.AgentModelName, task, missing); err == nil {
			t.Errorf("%s: an installed config without the model was admitted — pi would send an uncapped request", row)
		}
	}
}
