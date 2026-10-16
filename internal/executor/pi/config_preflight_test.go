package pi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

func writePiModels(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "models.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

var glmTask = &executor.Task{
	MaxOutputTokens:    131072,
	ReasoningEffort:    "high",
	OpenRouterProvider: map[string]any{"only": []any{"coreweave"}, "sort": "throughput"},
}

const glmMatching = `{"providers":{"openrouter":{"models":[{"id":"z-ai/glm-5.3-flash","maxTokens":131072,
 "thinkingLevelMap":{"off":null,"high":"high"},
 "compat":{"openRouterRouting":{"only":["coreweave"],"sort":"throughput"}}}]}}}`

func TestPreflight_MatchingConfigPasses(t *testing.T) {
	if err := checkInstalledModelConfig("openrouter/z-ai/glm-5.3-flash", glmTask, writePiModels(t, glmMatching)); err != nil {
		t.Fatalf("matching config refused: %v", err)
	}
}

// The 2026-10-16 shape: the installed file lacks the model, so pi would fall
// back to its catalogue. That must refuse, not run.
func TestPreflight_MissingModelRefuses(t *testing.T) {
	p := writePiModels(t, `{"providers":{"openrouter":{"models":[{"id":"z-ai/glm-4.7-flash","maxTokens":32000}]}}}`)
	err := checkInstalledModelConfig("openrouter/z-ai/glm-5.3-flash", glmTask, p)
	if err == nil || !strings.Contains(err.Error(), "built-in catalogue") {
		t.Fatalf("missing model: err = %v, want a refusal naming the catalogue fallback", err)
	}
}

func TestPreflight_EachMismatchIsNamed(t *testing.T) {
	p := writePiModels(t, `{"providers":{"openrouter":{"models":[{"id":"z-ai/glm-5.3-flash","maxTokens":32000,
 "thinkingLevelMap":{"off":null,"high":null}}]}}}`)
	err := checkInstalledModelConfig("openrouter/z-ai/glm-5.3-flash", glmTask, p)
	if err == nil {
		t.Fatal("mismatched config ran")
	}
	for _, want := range []string{"maxTokens 32000", "thinkingLevelMap.high is null", "compat.openRouterRouting null"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error does not name %q: %v", want, err)
		}
	}
}

func TestPreflight_SkipsWhatItCannotJudge(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent.json")
	if err := checkInstalledModelConfig("ollama-rig/qwen3.8:27b", glmTask, missing); err != nil {
		t.Errorf("non-OpenRouter model checked: %v", err)
	}
	if err := checkInstalledModelConfig("openrouter/z-ai/glm-5.3-flash", &executor.Task{}, missing); err != nil {
		t.Errorf("task without registry settings checked: %v", err)
	}
}

func TestPiModelsPath_HonoursAgentDirOverride(t *testing.T) {
	if got, want := piModelsPath([]string{"PI_CODING_AGENT_DIR=/x/agent"}), filepath.Join("/x/agent", "models.json"); got != want {
		t.Errorf("piModelsPath = %q", got)
	}
}
