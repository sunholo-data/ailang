package pi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// WHY A PREFLIGHT
//
// pi takes its output cap, reasoning levels and OpenRouter routing from its own
// models.json, not from anything we pass: it has no max-tokens flag, and it
// calls OpenRouter with its own client, so AILANG's routing never reaches it.
// A model missing from that file is not an error to pi — it falls back to its
// built-in catalogue. On 2026-10-16 the rig's installed file had drifted from
// tools/pi-extensions/models.mission.json and did not carry z-ai/glm-5.3-flash,
// so pi sent max_completion_tokens 943717, reasoning effort "high" and no
// provider block, while the registry (and motoko) said 32000, no effort and a
// pinned host list. Months of pi-vs-motoko A/Bs compared those two requests.
//
// TestPiModelsConfigMatchesRegistry holds the REPO copy equal to the registry;
// this holds the INSTALLED copy — the one pi reads — equal to the task, and
// refuses to run otherwise. Checked only for OpenRouter models whose task
// carries registry settings, which is where the two have diverged.

// piModelsPath is the models.json pi will load: $PI_CODING_AGENT_DIR (pi's
// own override) or ~/.pi/agent.
func piModelsPath(env []string) string {
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], "PI_CODING_AGENT_DIR="); ok && v != "" {
			return filepath.Join(v, "models.json")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".pi", "agent", "models.json")
}

type installedPiModel struct {
	ID               string             `json:"id"`
	MaxTokens        int                `json:"maxTokens"`
	ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
	Compat           struct {
		OpenRouterRouting map[string]any `json:"openRouterRouting"`
	} `json:"compat"`
}

// checkInstalledModelConfig returns an error naming every way the installed pi
// config would make pi send a different request from the one the registry
// declares for model ("openrouter/<id>").
func checkInstalledModelConfig(model string, task *executor.Task, path string) error {
	provider, id, ok := strings.Cut(model, "/")
	if !ok || provider != "openrouter" {
		return nil
	}
	if task.MaxOutputTokens == 0 && task.ReasoningEffort == "" && task.OpenRouterProvider == nil {
		return nil // no registry settings to hold pi to
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("pi config preflight: cannot read %s, so pi's request for %s cannot be checked against the registry: %w", path, id, err)
	}
	var cfg struct {
		Providers map[string]struct {
			Models []installedPiModel `json:"models"`
		} `json:"providers"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return fmt.Errorf("pi config preflight: %s is not valid JSON: %w", path, err)
	}
	var m *installedPiModel
	for i := range cfg.Providers[provider].Models {
		if cfg.Providers[provider].Models[i].ID == id {
			m = &cfg.Providers[provider].Models[i]
			break
		}
	}
	fix := "install tools/pi-extensions/models.mission.json's entry for it (keeping the installed apiKey)"
	if m == nil {
		return fmt.Errorf("pi config preflight: %s has no %s model %q, so pi would use its built-in catalogue instead of the registry (2026-10-16: max_completion_tokens 943717, effort \"high\", no provider routing) — %s",
			path, provider, id, fix)
	}
	var problems []string
	if task.MaxOutputTokens > 0 && m.MaxTokens != task.MaxOutputTokens {
		problems = append(problems, fmt.Sprintf("maxTokens %d, registry max_output_tokens %d", m.MaxTokens, task.MaxOutputTokens))
	}
	if e := task.ReasoningEffort; e != "" {
		if lvl, ok := m.ThinkingLevelMap[e]; !ok || lvl == nil {
			problems = append(problems, fmt.Sprintf("thinkingLevelMap.%s is null, so --thinking %s sends no effort", e, e))
		}
	}
	if want := task.OpenRouterProvider; want != nil {
		got, _ := json.Marshal(m.Compat.OpenRouterRouting)
		exp, _ := json.Marshal(want)
		if string(got) != string(exp) {
			problems = append(problems, fmt.Sprintf("compat.openRouterRouting %s, registry routing %s", got, exp))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("pi config preflight: %s would make pi send a different request for %q than the registry declares (%s) — %s",
			path, id, strings.Join(problems, "; "), fix)
	}
	return nil
}
