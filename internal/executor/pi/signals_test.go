package pi

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-PI-HARNESS-UPGRADE M3: the two signals 0.84+ added, wired.

func TestExecuteStreaming_ReasoningDisjointFromOutput(t *testing.T) {
	// reasoning.ndjson: assistant message_end usage {output:41, reasoning:8}
	// and totalTokens == input+output+cacheRead+cacheWrite with reasoning NOT
	// added (V33) — so reasoning is INSIDE output on the wire, and the
	// executor contract (Result.ReasonTokens disjoint from OutputTokens)
	// requires the subtraction (D5).
	res := runFake(t, loadFixtureLines(t, "v0_85_1/reasoning.ndjson"))
	if res.ReasonTokens != 8 {
		t.Fatalf("ReasonTokens = %d, want 8", res.ReasonTokens)
	}
	if res.OutputTokens != 41-8 {
		t.Fatalf("OutputTokens = %d, want 33 (usage.output 41 − reasoning 8)", res.OutputTokens)
	}
	if res.OutputTokens+res.ReasonTokens != 41 {
		t.Fatalf("identity broken: output %d + reason %d != usage.output 41", res.OutputTokens, res.ReasonTokens)
	}
}

func TestExecuteStreaming_NoReasoningFieldLeavesOutputWhole(t *testing.T) {
	// The 0.73.1 wire has no usage.reasoning at all: output must be untouched
	// and ReasonTokens must be 0 ("not reported"), not negative or guessed.
	res := runFake(t, loadFixtureLines(t, "v0_73_1/fizzbuzz.ndjson"))
	if res.ReasonTokens != 0 {
		t.Fatalf("ReasonTokens = %d on a wire without the field, want 0", res.ReasonTokens)
	}
	if res.OutputTokens <= 0 {
		t.Fatalf("OutputTokens = %d, want the fixture's usage.output", res.OutputTokens)
	}
}

func TestExecuteStreaming_RawStopReasonBanked(t *testing.T) {
	res := runFake(t, loadFixtureLines(t, "v0_85_1/tool_use.ndjson"))
	// Last settled rawStopReason in the tool_use capture is "stop" (final
	// turn); the intermediate ones are "tool_calls".
	if got := res.ProviderData["pi_raw_stop_reason"]; got != "stop" {
		t.Fatalf("pi_raw_stop_reason = %v, want \"stop\"", got)
	}
	if res.FinishReason != executor.FinishStop {
		t.Fatalf("FinishReason = %q, want stop", res.FinishReason)
	}
}

func TestExecuteStreaming_RawStopReasonConsultedOnlyWhenUnrecognised(t *testing.T) {
	// stopReason is pi's normalised value and stays authoritative. Only when
	// pi passes through something we do not know does the provider's own
	// rawStopReason get a vote.
	var events []string
	for _, ln := range loadFixtureLines(t, "v0_85_1/fizzbuzz.ndjson") {
		if strings.Contains(ln, `"type":"message_end"`) || strings.Contains(ln, `"type":"turn_end"`) {
			ln = strings.ReplaceAll(ln, `"stopReason":"stop"`, `"stopReason":"somethingNew"`)
			ln = strings.ReplaceAll(ln, `"rawStopReason":"stop"`, `"rawStopReason":"length"`)
		}
		events = append(events, ln)
	}
	res := runFake(t, events)
	if res.FinishReason != executor.FinishLength {
		t.Fatalf("FinishReason = %q; an unrecognised stopReason must fall back to the provider's rawStopReason (length)", res.FinishReason)
	}
}

// Dual-version replay: the fields both wires carry must produce the same
// Result shape on 0.73.1 and 0.85.1 — the parser is one parser, not two.
func TestExecuteStreaming_DualVersionReplay(t *testing.T) {
	for _, v := range []string{"v0_73_1", "v0_85_1"} {
		t.Run(v, func(t *testing.T) {
			res := runFake(t, loadFixtureLines(t, v+"/tool_use.ndjson"))
			if !res.Success {
				t.Fatalf("%s: Success=false Error=%q", v, res.Error)
			}
			if res.ToolCallCount < 2 || res.ToolCalls["write"] < 1 {
				t.Fatalf("%s: tool calls = %d %v, want >=2 incl. write", v, res.ToolCallCount, res.ToolCalls)
			}
			if res.InputTokens <= 0 || res.OutputTokens <= 0 || res.NumTurns < 2 {
				t.Fatalf("%s: tokens/turns not summed: in=%d out=%d turns=%d", v, res.InputTokens, res.OutputTokens, res.NumTurns)
			}
			if res.FinishReason != executor.FinishStop {
				t.Fatalf("%s: FinishReason = %q", v, res.FinishReason)
			}
			if _, drift := res.ProviderData["pi_unknown_events"]; drift {
				t.Fatalf("%s: unknown events on a known wire: %v", v, res.ProviderData["pi_unknown_events"])
			}
		})
	}
}
