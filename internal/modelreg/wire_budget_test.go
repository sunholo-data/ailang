package modelreg

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// unequalWireBudgets are the api_names whose rows still disagree on
// max_output_tokens. All four are the same pattern: the pi rows declare 32000
// for a clamp measured on pi 0.79 (2026-08-26) that pi 0.85.1 no longer
// applies (Broadcast, 2026-10-16: pi sent ~943k for glm-5.3-flash). Each needs
// a wire measurement and a decision, as glm-5.3-flash got. This list may only
// shrink: the test fails when a listed api_name becomes consistent, so the
// entry is removed in the same change.
var unequalWireBudgets = map[string]bool{
	"deepseek/deepseek-v4-flash-0731": true,
	"deepseek/deepseek-v4.1-flash":    true,
	"minimax/minimax-m3":              true,
	"moonshotai/kimi-k3":              true,
}

// TestModels_SameAPINameSameWireBudget: an output cap and a reasoning effort
// are properties of the model, not of the harness that calls it. Rows sharing
// an OpenRouter api_name with different values mean two harnesses send two
// different requests for "the same model" — which is how every glm-5.3-flash
// pi-vs-motoko A/B before 2026-10-16 compared 32000-no-effort against
// 943k-effort-high and read the difference as harness quality.
func TestModels_SameAPINameSameWireBudget(t *testing.T) {
	c, err := LoadModelsConfigBytes(embeddedModelsYAML)
	if err != nil {
		t.Fatalf("load embedded models.yml: %v", err)
	}
	caps := map[string]map[int][]string{}
	efforts := map[string]map[string][]string{}
	for name, m := range c.Models {
		if m.Provider != "openrouter" || m.APIName == "" {
			continue
		}
		if caps[m.APIName] == nil {
			caps[m.APIName] = map[int][]string{}
			efforts[m.APIName] = map[string][]string{}
		}
		caps[m.APIName][m.MaxOutputTokens] = append(caps[m.APIName][m.MaxOutputTokens], name)
		efforts[m.APIName][m.ReasoningEffort] = append(efforts[m.APIName][m.ReasoningEffort], name)
	}

	for api, byCap := range caps {
		if len(efforts[api]) > 1 {
			t.Errorf("rows for %q declare different reasoning_effort: %s", api, describe(efforts[api]))
		}
		if len(byCap) <= 1 {
			if unequalWireBudgets[api] {
				t.Errorf("%q now declares one max_output_tokens on every row — remove it from unequalWireBudgets", api)
			}
			continue
		}
		if unequalWireBudgets[api] {
			continue
		}
		byStr := map[string][]string{}
		for v, rows := range byCap {
			byStr[fmt.Sprint(v)] = rows
		}
		t.Errorf("rows for %q declare different max_output_tokens: %s — one model, one wire budget", api, describe(byStr))
	}
	for api := range unequalWireBudgets {
		if _, ok := caps[api]; !ok {
			t.Errorf("unequalWireBudgets lists %q, which no openrouter row names — remove it", api)
		}
	}
}

func describe(m map[string][]string) string {
	var parts []string
	for v, rows := range m {
		sort.Strings(rows)
		if v == "" {
			v = "(unset)"
		}
		parts = append(parts, v+": "+strings.Join(rows, ", "))
	}
	sort.Strings(parts)
	return strings.Join(parts, "; ")
}
