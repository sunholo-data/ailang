package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// piModelsConfig mirrors the subset of ~/.pi/agent/models.json this test asserts
// on. Canonical copy: tools/pi-extensions/models.mission.json.
type piModelsConfig struct {
	Providers map[string]struct {
		Models []struct {
			ID            string `json:"id"`
			MaxTokens     int    `json:"maxTokens"`
			ContextWindow int    `json:"contextWindow"`
			Reasoning     *bool  `json:"reasoning"`
		} `json:"models"`
	} `json:"providers"`
}

const piModelsCanonicalPath = "../../tools/pi-extensions/models.mission.json"

// TestPiModelsConfigMatchesRegistry closes the gap that made models.yml's
// declared budget a fiction on the pi lane.
//
// pi has NO max-tokens CLI flag: it reads maxTokens from its own models.json and
// falls back to 16384 for any model that omits it (model-registry.js). Our four
// OpenRouter models were registered as bare {"id": ...}, so the mission executor
// and every pi agent-mode eval ran at 16384 — 4x under the 65536 this registry
// declares and TestModels_CloudHeadroomEqualised pins — with reasoning tokens
// eating the same budget. That is invisible from the Go side: buildPiArgs cannot
// forward task.MaxOutputTokens, and CI was green the whole time.
//
// So the invariant is asserted where it can be: every OpenRouter model in the
// canonical pi config must declare the same budget as the models.yml entry that
// names it. Ollama rows are out of scope — their ceiling is VRAM-bound, and the
// three local harnesses (pi/opencode/motoko) currently disagree by design.
func TestPiModelsConfigMatchesRegistry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Clean(piModelsCanonicalPath))
	if err != nil {
		t.Fatalf("read canonical pi config: %v", err)
	}
	var piCfg piModelsConfig
	if err := json.Unmarshal(raw, &piCfg); err != nil {
		t.Fatalf("parse canonical pi config: %v", err)
	}

	c, err := LoadModelsConfig("models.yml")
	if err != nil {
		t.Fatalf("load models.yml: %v", err)
	}

	// api_name -> declared max_output_tokens, for openrouter entries only.
	declared := map[string][]struct {
		key    string
		budget int
	}{}
	for name, m := range c.Models {
		if m.Provider != "openrouter" {
			continue
		}
		declared[m.APIName] = append(declared[m.APIName], struct {
			key    string
			budget int
		}{name, m.MaxOutputTokens})
	}

	or, ok := piCfg.Providers["openrouter"]
	if !ok || len(or.Models) == 0 {
		t.Fatal("canonical pi config declares no openrouter models — the file this test guards is empty or restructured")
	}

	checked := 0
	for _, m := range or.Models {
		// ":floor" / ":free" are OpenRouter routing suffixes, not part of the
		// model id the registry knows.
		slug := m.ID
		if i := strings.Index(slug, ":"); i >= 0 {
			slug = slug[:i]
		}
		entries, ok := declared[slug]
		if !ok {
			// Registered in pi but not in models.yml: nothing to drift against.
			continue
		}
		checked++

		if m.MaxTokens == 0 {
			t.Errorf("pi model %q declares no maxTokens — pi will silently use its 16384 default while models.yml declares %d",
				m.ID, entries[0].budget)
			continue
		}
		for _, e := range entries {
			if m.MaxTokens != e.budget {
				t.Errorf("pi model %q maxTokens = %d, but models.yml entry %q declares max_output_tokens = %d — the wire budget wins, so the registry value is a fiction",
					m.ID, m.MaxTokens, e.key, e.budget)
			}
		}
		if m.ContextWindow == 0 {
			t.Errorf("pi model %q declares no contextWindow — pi will use its 128000 default, which drives compaction against the wrong ceiling", m.ID)
		}
		if m.Reasoning == nil || !*m.Reasoning {
			t.Errorf("pi model %q is not marked reasoning:true — pi then refuses every --thinking level and can send no reasoning control at all", m.ID)
		}
	}

	if checked == 0 {
		var ids []string
		for _, m := range or.Models {
			ids = append(ids, m.ID)
		}
		sort.Strings(ids)
		t.Fatalf("no pi openrouter model matched a models.yml api_name — the test asserted nothing. pi ids: %v", ids)
	}
}
