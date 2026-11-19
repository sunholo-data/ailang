package eval_harness

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDriverPiOpenRouterRungsAreHeld closes D-FLEET-35: every OpenRouter model the
// mission driver routes through pi must have BOTH a pi-lane registry row and a canonical
// pi config entry, with one output cap between them.
//
// Without the row, nothing holds the cap. config_preflight.go checks the installed pi
// config only for tasks that carry registry settings, and
// TestPiModelsConfigMatchesRegistry checks only slugs that have a pi row. Without the
// canonical entry, the cap exists only on the machine someone hand-edited. Three
// routed rungs (z-ai/glm-5.3, moonshotai/kimi-k3, minimax/minimax-m3) had caps only in
// the rig's installed ~/.pi/agent/models.json (attended 10-07 workaround). A pi model
// missing from that file is not an error to pi: it falls back to its built-in catalogue
// and sends an uncapped request, which OpenRouter refuses with a 402 "can only afford".
func TestDriverPiOpenRouterRungsAreHeld(t *testing.T) {
	rungs := driverPiOpenRouterRungs(t, filepath.Clean("../../tools/launchd/mission-control.sh"))
	if len(rungs) == 0 {
		t.Fatal("found no pi:openrouter/ rung in the driver's fallback exports — the parser no longer sees the chains it guards")
	}

	raw, err := os.ReadFile(filepath.Clean(piModelsCanonicalPath))
	if err != nil {
		t.Fatalf("read canonical pi config: %v", err)
	}
	var piCfg piModelsConfig
	if err := json.Unmarshal(raw, &piCfg); err != nil {
		t.Fatalf("parse canonical pi config: %v", err)
	}
	canonical := map[string]int{}
	for _, m := range piCfg.Providers["openrouter"].Models {
		canonical[m.ID] = m.MaxTokens
	}

	c, err := LoadModelsConfig("../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load models.yml: %v", err)
	}
	rows := map[string][]string{} // "openrouter/<slug>" -> pi rows naming it
	caps := map[string]int{}
	for name, m := range c.Models {
		if m.AgentCLI == nil || *m.AgentCLI != "pi" || m.AgentModelName == nil {
			continue
		}
		rows[*m.AgentModelName] = append(rows[*m.AgentModelName], name)
		caps[name] = m.MaxOutputTokens
	}

	for _, slug := range rungs {
		maxTokens, inCanonical := canonical[slug]
		if !inCanonical || maxTokens == 0 {
			t.Errorf("driver rung pi:openrouter/%s has no maxTokens entry in %s — its cap lives only in an installed config", slug, piModelsCanonicalPath)
		}
		names := rows["openrouter/"+slug]
		if len(names) == 0 {
			t.Errorf("driver rung pi:openrouter/%s has no pi registry row (agent_model_name openrouter/%s) — neither the drift test nor the preflight holds its cap", slug, slug)
			continue
		}
		for _, n := range names {
			if caps[n] <= 0 || (inCanonical && caps[n] != maxTokens) {
				t.Errorf("registry row %s declares max_output_tokens %d, canonical pi config maxTokens %d for %s", n, caps[n], maxTokens, slug)
			}
		}
	}
}

var piOpenRouterRung = regexp.MustCompile(`pi:openrouter/([^,"}\s]+)`)

// driverPiOpenRouterRungs returns the OpenRouter slugs named by pi rungs on the driver's
// non-comment `export` lines (the fallback chains), in file order, deduplicated.
func driverPiOpenRouterRungs(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open driver: %v", err)
	}
	defer f.Close()
	seen := map[string]bool{}
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		for _, m := range piOpenRouterRung.FindAllStringSubmatch(line, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				out = append(out, m[1])
			}
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan driver: %v", err)
	}
	return out
}
