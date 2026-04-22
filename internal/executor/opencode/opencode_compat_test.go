// Package opencode documents the opencode CLI JSON event schema captured from
// live runs with opencode-ai@1.14.20 on 2026-04-22.
//
// # Schema
//
// opencode emits NDJSON via `opencode run --format json`. Each line is a
// self-contained event object. Partial lines do not occur (unlike SSE).
//
// ## Top-level fields (all events)
//
//	{
//	  "type":      string       // "step_start" | "text" | "tool_use" | "step_finish"
//	  "timestamp": int64        // Unix millis
//	  "sessionID": string       // e.g. "ses_24ae068dbffeBRSjSbJYWMtDDR"
//	  "part":      object       // type-specific payload
//	}
//
// ## Event types
//
// ### step_start — begins a reasoning/response step
//
//	"part": { "id", "messageID", "sessionID", "type": "step-start" }
//
// ### text — text token chunk
//
//	"part": {
//	  "id", "messageID", "sessionID",
//	  "type": "text",
//	  "text": string,
//	  "time": { "start": int64, "end": int64 }   // Unix millis
//	}
//
// ### tool_use — tool call (status: "completed" | "running" | "error")
//
//	"part": {
//	  "type": "tool",
//	  "tool": string,       // "write" | "read" | "bash" | etc.
//	  "callID": string,
//	  "state": {
//	    "status": string,
//	    "input":  object,   // tool-specific; e.g. {filePath, content} for write
//	    "output": string,
//	    "title":  string,
//	    "time":   { "start", "end" }
//	  },
//	  "id", "sessionID", "messageID"
//	}
//
// ### step_finish — ends a step; contains token/cost totals
//
//	"part": {
//	  "id", "messageID", "sessionID",
//	  "type": "step-finish",
//	  "reason": "stop" | "tool-calls",
//	  "tokens": {
//	    "total":     int,
//	    "input":     int,     // NEW non-cached input tokens for this step
//	    "output":    int,     // NEW output tokens for this step
//	    "reasoning": int,     // thinking tokens (0 for standard models)
//	    "cache": { "write": int, "read": int }
//	  },
//	  "cost": float64         // USD for this step
//	}
//
// ## Token semantics
//
// Unlike Codex (which emits cumulative running totals), opencode emits PER-STEP
// deltas: each step_finish has the tokens consumed by THAT step only.
// To get conversation totals: sum all step_finish.part.tokens.input and .output.
// To get conversation cost: sum all step_finish.part.cost.
//
// ## Session resumption
//
// The sessionID from any event can be used to resume: `opencode run --session <id>`
// or `opencode run --continue` for the most recent session.
//
// ## Ollama / local model support (IMPORTANT for M-EXEC-EXPAND)
//
// opencode supports custom providers via ~/.config/opencode/opencode.jsonc:
//
//	{
//	  "provider": {
//	    "ollama": {
//	      "npm": "@ai-sdk/openai-compatible",
//	      "name": "Ollama Local",
//	      "options": { "baseURL": "http://localhost:11434/v1" },
//	      "models": { "gemma4:latest": {"name": "Gemma 4"} }
//	    }
//	  }
//	}
//
// Model string format: "provider_id/model_id", e.g. "ollama/gemma4:latest".
// Event schema is identical regardless of provider — the parser in M6 is universal.
//
// ## CLI flags used by the executor
//
//	opencode run <message> \
//	  --format json \
//	  --model <provider/model> \
//	  --dangerously-skip-permissions \
//	  --session <sessionID>           # for resume (iteration > 1)
//	  --dir <workspace>
package opencode

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// opencodeEvent mirrors the top-level wrapper for all opencode NDJSON events.
type opencodeEvent struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	SessionID string          `json:"sessionID"`
	Part      json.RawMessage `json:"part"`
}

// stepFinishPart captures the token/cost payload from a step_finish event.
type stepFinishPart struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
	Tokens struct {
		Total     int `json:"total"`
		Input     int `json:"input"`
		Output    int `json:"output"`
		Reasoning int `json:"reasoning"`
		Cache     struct {
			Write int `json:"write"`
			Read  int `json:"read"`
		} `json:"cache"`
	} `json:"tokens"`
	Cost float64 `json:"cost"`
}

// textPart captures the text payload from a text event.
type textPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Time struct {
		Start int64 `json:"start"`
		End   int64 `json:"end"`
	} `json:"time"`
}

// toolUsePart captures the tool call payload from a tool_use event.
type toolUsePart struct {
	Type   string `json:"type"`
	Tool   string `json:"tool"`
	CallID string `json:"callID"`
	State  struct {
		Status string          `json:"status"`
		Input  json.RawMessage `json:"input"`
		Output string          `json:"output"`
		Title  string          `json:"title"`
	} `json:"state"`
}

// TestOpenCodeEventSchema_FixtureReplay replays the recorded live session and
// asserts the NDJSON structure matches the schema documented in this package.
// This test is the M5 design-freeze gate: if schema parsing fails, M6 (executor
// implementation) cannot proceed.
func TestOpenCodeEventSchema_FixtureReplay(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "opencode_response.jsonl")

	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var (
		stepStarts  int
		texts       int
		toolUses    int
		stepFinish  int
		totalInput  int
		totalOutput int
		totalCost   float64
		lastSession string
		textBuf     string
	)

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}

		var ev opencodeEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			t.Fatalf("parse event: %v\nline: %s", err, line)
		}

		// All events must have the mandatory top-level fields.
		if ev.Type == "" {
			t.Errorf("event missing type field: %s", line)
		}
		if ev.Timestamp == 0 {
			t.Errorf("event missing timestamp: %s", line)
		}
		if ev.SessionID == "" {
			t.Errorf("event missing sessionID: %s", line)
		}
		lastSession = ev.SessionID

		switch ev.Type {
		case "step_start":
			stepStarts++

		case "text":
			texts++
			var p textPart
			if err := json.Unmarshal(ev.Part, &p); err != nil {
				t.Fatalf("parse text part: %v", err)
			}
			if p.Text == "" {
				t.Errorf("text event has empty text field")
			}
			if p.Time.Start == 0 || p.Time.End == 0 {
				t.Errorf("text event missing timing: start=%d end=%d", p.Time.Start, p.Time.End)
			}
			textBuf += p.Text

		case "tool_use":
			toolUses++
			var p toolUsePart
			if err := json.Unmarshal(ev.Part, &p); err != nil {
				t.Fatalf("parse tool_use part: %v", err)
			}
			if p.Tool == "" {
				t.Errorf("tool_use event missing tool name")
			}
			if p.CallID == "" {
				t.Errorf("tool_use event missing callID")
			}
			if p.State.Status == "" {
				t.Errorf("tool_use state missing status")
			}

		case "step_finish":
			stepFinish++
			var p stepFinishPart
			if err := json.Unmarshal(ev.Part, &p); err != nil {
				t.Fatalf("parse step_finish part: %v", err)
			}
			if p.Reason == "" {
				t.Errorf("step_finish missing reason")
			}
			if p.Tokens.Total == 0 {
				t.Errorf("step_finish has zero total tokens")
			}
			totalInput += p.Tokens.Input
			totalOutput += p.Tokens.Output
			totalCost += p.Cost

		default:
			// Forward-compat: unknown event types must not crash the parser.
			t.Logf("WARN: unknown event type %q (schema drift — check opencode changelog)", ev.Type)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}

	// Structural assertions from the 2-session live fixture.
	if stepStarts != 3 {
		t.Errorf("step_start count = %d, want 3", stepStarts)
	}
	if texts != 3 {
		t.Errorf("text event count = %d, want 3", texts)
	}
	if toolUses != 1 {
		t.Errorf("tool_use count = %d, want 1", toolUses)
	}
	if stepFinish != 3 {
		t.Errorf("step_finish count = %d, want 3", stepFinish)
	}

	// Token semantics: opencode emits PER-STEP deltas, not cumulative.
	// Sum of all step_finish.input and .output gives conversation totals.
	// Fixture: step1 (1+26) + step2 (1+133) + step3 (3+25) = 5 in, 184 out.
	if totalInput != 5 {
		t.Errorf("summed input tokens = %d, want 5 (per-step deltas)", totalInput)
	}
	if totalOutput != 184 {
		t.Errorf("summed output tokens = %d, want 184 (per-step deltas)", totalOutput)
	}

	// Total cost: 0.02164725 + 0.02219475 + 0.00203405 = 0.04587605
	const wantCost = 0.04587605
	const epsilon = 1e-6
	if diff := totalCost - wantCost; diff > epsilon || diff < -epsilon {
		t.Errorf("summed cost = %.8f, want %.8f", totalCost, wantCost)
	}

	// Session ID is stable within a session and usable for --session resume.
	if lastSession == "" {
		t.Error("sessionID never populated")
	}

	// Text accumulation sanity.
	if len(textBuf) == 0 {
		t.Error("no text content accumulated")
	}

	t.Logf("schema OK: %d step_starts, %d texts, %d tool_uses, %d step_finishes",
		stepStarts, texts, toolUses, stepFinish)
	t.Logf("token totals (per-step deltas): input=%d output=%d cost=%.8f",
		totalInput, totalOutput, totalCost)
	t.Logf("sessionID for resume: %s", lastSession)
}

// TestOpenCodeTokenSemantics_PerStepDeltas verifies that opencode step_finish
// tokens are per-step deltas (NOT cumulative like Codex). This distinction
// drives the accumulation strategy in the M6 executor: use sum(), not max().
func TestOpenCodeTokenSemantics_PerStepDeltas(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "opencode_response.jsonl")

	f, err := os.Open(fixture)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	var steps []stepFinishPart
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev opencodeEvent
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		if ev.Type != "step_finish" {
			continue
		}
		var p stepFinishPart
		if err := json.Unmarshal(ev.Part, &p); err != nil {
			t.Fatalf("parse step_finish: %v", err)
		}
		steps = append(steps, p)
	}

	// Fixture has 3 step_finish events:
	//   step 1: total=17240, input=1, output=26  (new prompt tokens from step; small)
	//   step 2: total=17357, input=1, output=133
	//   step 3: total=17398, input=3, output=25
	// If cumulative: step3 total (17398) > step1 total (17240). Confirmed PER-STEP.
	// If summing: 1+1+3=5 input (≠ 17398), proving they are not cumulative.
	if len(steps) < 3 {
		t.Fatalf("expected 3 step_finish events, got %d", len(steps))
	}

	// Demonstrate the values are small (per-step), not large running totals.
	for i, s := range steps {
		if s.Tokens.Input > 100 {
			t.Errorf("step %d: input=%d looks like a cumulative total, not a per-step delta", i+1, s.Tokens.Input)
		}
		if s.Tokens.Output > 200 {
			t.Errorf("step %d: output=%d looks like a cumulative total, not a per-step delta", i+1, s.Tokens.Output)
		}
	}

	// The "total" field is the full context window (cache+new), NOT the per-step delta.
	// It grows across steps (17240 → 17357 → 17398) proving it is NOT what to sum.
	if steps[0].Tokens.Total >= steps[1].Tokens.Total {
		t.Errorf("total field should grow across steps (context window), got %d >= %d",
			steps[0].Tokens.Total, steps[1].Tokens.Total)
	}

	t.Logf("confirmed: opencode tokens are per-step deltas; executor must SUM across step_finish events")
}
