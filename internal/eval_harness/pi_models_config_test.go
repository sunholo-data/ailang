package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// piModelsConfig mirrors the subset of ~/.pi/agent/models.json this test asserts
// on. Canonical copy: tools/pi-extensions/models.mission.json.
type piModelsConfig struct {
	Providers map[string]struct {
		Models []struct {
			ID               string             `json:"id"`
			MaxTokens        int                `json:"maxTokens"`
			ContextWindow    int                `json:"contextWindow"`
			Reasoning        *bool              `json:"reasoning"`
			ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
		} `json:"models"`
	} `json:"providers"`
}

const piModelsCanonicalPath = "../../tools/pi-extensions/models.mission.json"

// TestPiModelsConfigMatchesRegistry closes the gap that made models.yml's
// declared budget a fiction on the pi lane.
//
// pi has NO max-tokens CLI flag: it reads maxTokens from its own models.json and
// falls back to 16384 for any model that omits it (model-registry.js). Every
// model here was once registered as a bare {"id": ...}, so the mission executor
// and every pi agent-mode eval ran at 16384 no matter what this registry said —
// 4x under the OpenRouter models' declared 65536, and 2x under the local qwen
// models' 32768. That is invisible from the Go side: buildPiArgs cannot forward
// task.MaxOutputTokens, and CI was green the whole time.
//
// So the invariant is asserted where it can be: every model in the canonical pi
// config must declare the same budget as the PI-LANE models.yml entries that name it.
//
// ⚠️ CORRECTED 2026-08-26 — this test's original premise was too strong, and its
// original claim ("closes the gap that made models.yml's declared budget a fiction")
// was FALSE. Neither file it compares is the wire, and pi-ai clamps the budget
// downstream of BOTH: buildBaseOptions (dist/providers/simple-options.js:4) computes
// Math.min(model.maxTokens, 32000) whenever the caller passes no explicit maxTokens,
// which pi-coding-agent never does for main agent turns. Proven on the wire via
// OpenRouter Broadcast: declaring 20000 sent 20000, declaring 65536 sent 32000.
//
// Two consequences encoded below:
//   - Comparison is scoped to PI-LANE rows. A model's effective budget is
//     HARNESS-dependent, so pi declaring 32000 while opencode/motoko declare 65536
//     for the same slug is CORRECT, not drift.
//   - Only a real request can confirm the clamp constant itself. That check is
//     scripts/check_pi_wire_budget.sh, which reads the request back from the
//     provider's own trace. A config-vs-config test cannot do it, and believing
//     otherwise is what let a 2x understatement sit green in CI for weeks.
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

	type entry struct {
		key    string
		budget int
	}
	// (provider, api_name) -> every registry row that names it.
	declared := map[string]map[string][]entry{}
	for name, m := range c.Models {
		// PI-LANE ROWS ONLY. The same slug legitimately carries different budgets on
		// different harnesses — pi clamps to 32000, our own Go client does not — so
		// comparing pi's config against an opencode or motoko row measures the
		// harness, not drift.
		if m.AgentCLI == nil || *m.AgentCLI != "pi" {
			continue
		}
		if declared[m.Provider] == nil {
			declared[m.Provider] = map[string][]entry{}
		}
		declared[m.Provider][m.APIName] = append(declared[m.Provider][m.APIName], entry{name, m.MaxOutputTokens})
	}

	// Per-provider rules. OpenRouter ids carry routing suffixes (":floor",
	// ":free") that are not part of the registry's api_name; ollama ids contain a
	// colon natively (qwen3.5:35b-a3b-mxfp8), so stripping there would corrupt
	// every lookup.
	providers := []struct {
		name             string
		stripColonSuffix bool
		requireReasoning bool
		defaultBudget    string
	}{
		{"openrouter", true, true, "16384"},
		{"ollama", false, false, "16384"},
	}

	totalChecked := 0
	for _, prov := range providers {
		pcfg, ok := piCfg.Providers[prov.name]
		if !ok || len(pcfg.Models) == 0 {
			t.Errorf("canonical pi config declares no %s models — the file this test guards is empty or restructured", prov.name)
			continue
		}

		checked := 0
		for _, m := range pcfg.Models {
			slug := m.ID
			if prov.stripColonSuffix {
				if i := strings.Index(slug, ":"); i >= 0 {
					slug = slug[:i]
				}
			}
			entries, ok := declared[prov.name][slug]
			if !ok {
				// Registered in pi but not in models.yml: nothing to drift against.
				continue
			}
			checked++
			totalChecked++

			// The registry must not disagree with itself about one model ON ONE
			// HARNESS. Originally this compared across harnesses, which was wrong:
			// the effective budget is harness-dependent (pi clamps at 32000). It is
			// still worth asserting within the pi lane, because two pi rows naming
			// the same slug DO share a wire.
			budgets := map[int][]string{}
			for _, e := range entries {
				budgets[e.budget] = append(budgets[e.budget], e.key)
			}
			if len(budgets) > 1 {
				var lines []string
				for b, keys := range budgets {
					sort.Strings(keys)
					lines = append(lines, strings.Join(keys, ", ")+" -> "+strconv.Itoa(b))
				}
				sort.Strings(lines)
				t.Errorf("models.yml declares MORE THAN ONE max_output_tokens for %q: %v — same model, same harness, same wire, so at most one of these is being honoured",
					m.ID, lines)
			}

			if m.MaxTokens == 0 {
				t.Errorf("pi model %q declares no maxTokens — pi will silently use its %s default while models.yml declares %d",
					m.ID, prov.defaultBudget, entries[0].budget)
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
			if prov.requireReasoning && (m.Reasoning == nil || !*m.Reasoning) {
				t.Errorf("pi model %q is not marked reasoning:true — pi then refuses every --thinking level and can send no reasoning control at all", m.ID)
			}
			// reasoning:true without a thinkingLevelMap is the trap that makes
			// this worse than doing nothing: with no level passed, pi's openrouter
			// compat sends reasoning:{effort:"none"}, DISABLING thinking that was
			// previously on by default. Mapping "off" to null suppresses that.
			if m.Reasoning != nil && *m.Reasoning {
				if m.ThinkingLevelMap == nil {
					t.Errorf("pi model %q sets reasoning:true with no thinkingLevelMap — pi may then send an effort of \"none\" and disable thinking entirely", m.ID)
				} else if off, present := m.ThinkingLevelMap["off"]; !present || off != nil {
					t.Errorf("pi model %q must map thinkingLevelMap.off to null; got %v — a non-null \"off\" is what pi sends when no --thinking level is passed", m.ID, off)
				}
			}
		}

		if checked == 0 {
			var ids []string
			for _, m := range pcfg.Models {
				ids = append(ids, m.ID)
			}
			sort.Strings(ids)
			t.Errorf("no pi %s model matched a models.yml api_name — that arm asserted nothing. pi ids: %v", prov.name, ids)
		}
	}

	if totalChecked == 0 {
		t.Fatal("the test asserted nothing at all")
	}
}
