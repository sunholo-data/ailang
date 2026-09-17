package pi

import (
	"encoding/json"
	"strings"
	"testing"
)

// Fixture policy (M-PI-HARNESS-UPGRADE M3, D6): testdata holds a pair per
// wire version the fleet has actually run — v0_73_1 (the abandoned
// @mariozechner build the cloud ran until the cutover) and v0_85_1 (the pin).
// Both captured live on the rig 2026-09-16 with the same two directives.
// The 0.70.2 pair pinned a version no plane had run for months and is gone.
//
// The live captures came from ollama/openrouter runs that report zero cost
// and zero cacheWrite. Tests that exist to prove SUMMATION need non-zero
// per-turn values, so they derive them from the pinned fixture with
// patchAssistantUsage rather than trusting a hand-written stream.

// assistantUsagePatch is applied to the i-th assistant message_end's usage.
type assistantUsagePatch struct {
	CacheWrite int
	CostTotal  float64
}

// patchAssistantUsage rewrites the usage of successive assistant message_end
// events (and the matching turn_end, which mirrors the message) so the
// summation tests see known non-zero per-turn buckets. Events are parsed and
// re-marshalled, never string-edited.
func patchAssistantUsage(t *testing.T, events []string, patches []assistantUsagePatch) []string {
	t.Helper()
	out := make([]string, 0, len(events))
	idx := 0
	for _, ln := range events {
		if !strings.Contains(ln, `"role":"assistant"`) || !(strings.Contains(ln, `"type":"message_end"`) || strings.Contains(ln, `"type":"turn_end"`)) {
			out = append(out, ln)
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal([]byte(ln), &ev); err != nil {
			t.Fatalf("patchAssistantUsage: %v", err)
		}
		msg, _ := ev["message"].(map[string]any)
		usage, _ := msg["usage"].(map[string]any)
		if usage == nil || idx/2 >= len(patches) {
			out = append(out, ln)
			if usage != nil {
				idx++
			}
			continue
		}
		p := patches[idx/2] // message_end and turn_end share a patch
		usage["cacheWrite"] = p.CacheWrite
		cost, _ := usage["cost"].(map[string]any)
		if cost != nil {
			cost["total"] = p.CostTotal
		}
		b, err := json.Marshal(ev)
		if err != nil {
			t.Fatalf("patchAssistantUsage: %v", err)
		}
		out = append(out, string(b))
		idx++
	}
	return out
}
