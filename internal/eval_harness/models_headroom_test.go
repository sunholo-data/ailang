package eval_harness

import (
	"sort"
	"testing"
)

// validThinkingStates is the closed vocabulary for ModelConfig.DefaultThinking.
var validThinkingStates = map[string]bool{
	"on": true, "off": true, "always_on": true, "none": true, "unknown": true,
}

// TestModels_DefaultThinkingIsExplicit (M-EVAL-TOKEN-HEADROOM P2) requires every
// entry to declare its default thinking state.
//
// Why this is a hard gate rather than a nice-to-have: the harness sends no
// `thinking` control, so each model runs at its VENDOR default — and those
// disagree (Opus 5 / Sonnet 5 think, Opus 4.8 does not, Fable 5 cannot be
// stopped, gemini-3-5-flash-lite has no thinking at all). Without this column a
// reader cannot tell whether a row's output-token count reflects the answer or
// the reasoning, which is exactly how GLM-5.2's truncation was misread as a
// capability regression in June 2026.
//
// "unknown" is a permitted value ON PURPOSE. A guess is worse than an admission
// here: a wrong label silently corrupts an efficiency comparison, whereas
// "unknown" correctly withholds the row from one.
func TestModels_DefaultThinkingIsExplicit(t *testing.T) {
	c, err := LoadModelsConfig("../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	var missing, invalid []string
	for name, m := range c.Models {
		switch {
		case m.DefaultThinking == "":
			missing = append(missing, name)
		case !validThinkingStates[m.DefaultThinking]:
			invalid = append(invalid, name+"="+m.DefaultThinking)
		}
	}
	sort.Strings(missing)
	sort.Strings(invalid)

	if len(missing) > 0 {
		t.Errorf("%d entries have no default_thinking (add one of on/off/always_on/none/unknown): %v",
			len(missing), missing)
	}
	if len(invalid) > 0 {
		t.Errorf("%d entries have an out-of-vocabulary default_thinking: %v", len(invalid), invalid)
	}
}

// TestModels_CloudHeadroomEqualised (M-EVAL-TOKEN-HEADROOM P3) pins the headroom
// policy: every CLOUD entry sits at the 65536 target, or at a lower declared
// ceiling that is annotated as such.
//
// Unequal headroom is a silent, DIRECTIONAL confound — a thinking model on a
// small budget looks like a weak model. Equalising is also what converts running
// out of tokens from a harness artifact into real efficiency signal.
//
// Local (ollama) rows are exempt: their ceiling is VRAM-bound, not a policy choice.
func TestModels_CloudHeadroomEqualised(t *testing.T) {
	const target = 65536

	// Entries permitted below target, with the reason. Anything not listed here
	// must sit exactly at the target.
	ceilingLimited := map[string]int{
		// Anthropic family cap for the non-streaming Messages API as encoded in
		// this repo. 2.3% under target; not worth a hard 400 to close.
		"claude-opus-5": 64000, "claude-opus-4-8": 64000, "claude-opus-4-7": 64000,
		"claude-opus-4-6": 64000, "claude-opus-4-5": 64000, "claude-fable-5": 64000,
		"claude-fable-5-1": 64000,
		"claude-sonnet-5":  64000, "claude-sonnet-4-6": 64000, "claude-sonnet-4-5": 64000,
		"claude-haiku-4-5": 64000, "opencode-sonnet-4-6": 64000, "opencode-haiku": 64000,
		"pi-claude-sonnet-4-6": 64000, "pi-claude-haiku-4-5": 64000,
		"motoko-claude-sonnet-4-6": 64000, "motoko-claude-haiku-4-5": 64000,
		"motoko-or-sonnet-5": 64000,
		// Hard provider ceilings verified via the OpenRouter endpoints API
		// (2026-07-27): max_completion_tokens across ALL upstreams.
		"or-deepseek-v3": 16384, "or-qwen-2-5-72b": 16384,
		// Tencent Hy4 preview (2026-09-01): /endpoints reports ONE upstream
		// (Tencent first-party) at max_completion_tokens 64000, so 65536 is
		// unreachable — and with a single route there is no provider order to
		// pin that would raise it.
		"or-hy4-preview": 64000,

		// HARNESS ceiling, not a provider one — and the only one here proven by
		// reading the actual request rather than a vendor doc. pi-ai's
		// buildBaseOptions (dist/providers/simple-options.js:4) computes
		// Math.min(model.maxTokens, 32000) whenever the caller passes no explicit
		// maxTokens, and pi-coding-agent never does for main agent turns. So every
		// openai-compat pi lane is capped at 32000 regardless of what any config
		// says, and pi exposes neither a max-tokens flag nor a request hook to
		// raise it.
		//
		// Measured on the wire 2026-08-26 via OpenRouter Broadcast: declaring
		// 20000 sent max_completion_tokens=20000, declaring 65536 sent 32000 —
		// i.e. exactly min(declared, 32000). These rows now declare the truth;
		// restoring 65536 would only reinstate a fiction CI was pinning.
		// Re-verify with scripts/check_pi_wire_budget.sh.
		"pi-gpt5-4": 32000, "pi-gemini-3-flash-preview": 32000,
		"pi-or-deepseek-v4-flash": 32000,
	}

	c, err := LoadModelsConfig("../modelreg/models.yml")
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	var offTarget []string
	for name, m := range c.Models {
		if m.Provider == "ollama" {
			continue // VRAM-bound, out of scope
		}
		want := target
		if v, ok := ceilingLimited[name]; ok {
			want = v
		}
		if m.MaxOutputTokens != want {
			offTarget = append(offTarget, name)
		}
	}
	sort.Strings(offTarget)

	if len(offTarget) > 0 {
		t.Errorf("%d cloud entries are off-policy (want %d, or a documented ceiling): %v\n"+
			"If a model genuinely cannot reach %d, add it to ceilingLimited WITH the verified "+
			"provider ceiling — do not silently lower the row.",
			len(offTarget), target, offTarget, target)
	}
}
