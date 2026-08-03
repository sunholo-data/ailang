package effects

import (
	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// encodeStreamChunk converts a Go ai.StreamChunk variant into the matching
// AILANG `StreamChunk` ADT. Mirrors the type definitions in std/ai.ail
// (see M-AI-STEP-STREAMING design doc for shape contract).
//
//	ai.StreamContentDelta{Text}  → ContentDelta(string)
//	ai.StreamThinkingDelta{Text} → ThinkingDelta(string)  (v0.18.8)
//	ai.StreamUsage{...}          → Usage({input_tokens, output_tokens,
//	                                     cache_read_input_tokens,
//	                                     cache_creation_input_tokens})
func encodeStreamChunk(chunk ai.StreamChunk) eval.Value {
	switch c := chunk.(type) {
	case ai.StreamContentDelta:
		return &eval.TaggedValue{
			CtorName: "ContentDelta",
			Fields:   []eval.Value{&eval.StringValue{Value: c.Text}},
		}
	case ai.StreamThinkingDelta:
		return &eval.TaggedValue{
			CtorName: "ThinkingDelta",
			Fields:   []eval.Value{&eval.StringValue{Value: c.Text}},
		}
	case ai.StreamUsage:
		usageRec := &eval.RecordValue{
			Fields: map[string]eval.Value{
				"input_tokens":                &eval.IntValue{Value: c.InputTokens},
				"output_tokens":               &eval.IntValue{Value: c.OutputTokens},
				"cache_read_input_tokens":     &eval.IntValue{Value: c.CacheReadInputTokens},
				"cache_creation_input_tokens": &eval.IntValue{Value: c.CacheCreationInputTokens},
			},
		}
		return &eval.TaggedValue{
			CtorName: "Usage",
			Fields:   []eval.Value{usageRec},
		}
	default:
		return nil
	}
}

// makeOkStringResult builds Ok(string) — for callResult / callJsonResult.
func makeOkStringResult(s string) eval.Value {
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{&eval.StringValue{Value: s}},
	}
}

// makeOkStepResult builds Ok(StepResult record) — for step.
func makeOkStepResult(resp *ai.Response) eval.Value {
	// Build the assistant Message record.
	msgRec := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"role":         &eval.StringValue{Value: "assistant"},
			"content":      &eval.StringValue{Value: resp.Text},
			"tool_calls":   encodeToolCalls(resp.ToolCalls),
			"tool_call_id": &eval.StringValue{Value: ""},
		},
	}
	stepResult := &eval.RecordValue{
		Fields: map[string]eval.Value{
			"message":                     msgRec,
			"tool_calls":                  encodeToolCalls(resp.ToolCalls),
			"finish_reason":               &eval.StringValue{Value: resp.FinishReason},
			"input_tokens":                &eval.IntValue{Value: resp.InputTokens},
			"output_tokens":               &eval.IntValue{Value: resp.OutputTokens},
			"cache_read_input_tokens":     &eval.IntValue{Value: resp.CacheReadInputTokens},
			"cache_creation_input_tokens": &eval.IntValue{Value: resp.CacheCreationInputTokens},
		},
	}
	return &eval.TaggedValue{
		CtorName: "Ok",
		Fields:   []eval.Value{stepResult},
	}
}

// encodeToolCalls builds an AILANG list[ToolCall] from a Go slice.
func encodeToolCalls(calls []ai.ToolCall) eval.Value {
	elems := make([]eval.Value, 0, len(calls))
	for _, c := range calls {
		elems = append(elems, &eval.RecordValue{
			Fields: map[string]eval.Value{
				"id":        &eval.StringValue{Value: c.ID},
				"name":      &eval.StringValue{Value: c.Name},
				"arguments": &eval.StringValue{Value: c.Arguments},
			},
		})
	}
	return &eval.ListValue{Elements: elems}
}

// makeAIErrorResultRecord builds Err(AIError record) for any AI op that
// surfaces a typed failure. AIError shape: {code, message, retryable},
// matching std/ai/streaming.AIError byte-for-byte.
func makeAIErrorResultRecord(e *ai.AIError) eval.Value {
	if e == nil {
		// Defensive: should never happen, but if it does emit an
		// internal-coded record so downstream consumers see something
		// meaningful instead of a nil-deref.
		e = ai.NewAIError(ai.CodeInternal, "nil AIError surfaced from effect op", false)
	}
	return &eval.TaggedValue{
		CtorName: "Err",
		Fields: []eval.Value{
			&eval.RecordValue{
				Fields: map[string]eval.Value{
					"code":      &eval.StringValue{Value: e.Code},
					"message":   &eval.StringValue{Value: e.Message},
					"retryable": &eval.BoolValue{Value: e.Retryable},
				},
			},
		},
	}
}
