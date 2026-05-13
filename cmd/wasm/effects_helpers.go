package main

import (
	"github.com/sunholo-data/ailang/internal/ai"
)

// Pure-Go conversion helpers for the WASM ai.step bridge.
//
// These are split out of effects.go (which is //go:build js && wasm only) so
// they can be unit-tested on the host without a syscall/js dependency. The
// remaining helpers (jsToResponse, jsToToolCalls, jsToStreamChunk, jsGetString,
// jsGetInt) consume js.Value inputs and stay in effects.go behind the WASM
// build tag.
//
// Each function returns []interface{} (a JS-array-compatible slice that
// js.ValueOf can wrap directly) and never returns nil — empty inputs become
// empty slices, matching the on-wire shape OpenAI / Anthropic expect on
// assistant-without-tool-call messages.

// messagesToJSCompat converts []ai.Message to a slice that js.ValueOf can
// wrap as a JS array of objects. Each message becomes an object with
// {role, content, tool_calls, tool_call_id}. Empty tool_calls serializes
// as a JS empty array (not null).
func messagesToJSCompat(msgs []ai.Message) []interface{} {
	out := make([]interface{}, len(msgs))
	for i, m := range msgs {
		tcOut := make([]interface{}, len(m.ToolCalls))
		for j, tc := range m.ToolCalls {
			tcOut[j] = map[string]interface{}{
				"id":        tc.ID,
				"name":      tc.Name,
				"arguments": tc.Arguments,
			}
		}
		out[i] = map[string]interface{}{
			"role":         m.Role,
			"content":      m.Content,
			"tool_calls":   tcOut,
			"tool_call_id": m.ToolCallID,
		}
	}
	return out
}

// toolsToJSCompat converts []ai.ToolSchema to a JS-array-compatible slice.
// Parameters is the raw JSON Schema string; the JS shim is responsible for
// parsing it before sending to the provider.
func toolsToJSCompat(tools []ai.ToolSchema) []interface{} {
	out := make([]interface{}, len(tools))
	for i, t := range tools {
		out[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters,
		}
	}
	return out
}

// cacheBreakpointsToJSCompat converts []ai.CacheBreakpoint similarly.
// Empty input → JS empty array (not null).
func cacheBreakpointsToJSCompat(bps []ai.CacheBreakpoint) []interface{} {
	out := make([]interface{}, len(bps))
	for i, bp := range bps {
		out[i] = map[string]interface{}{
			"position": bp.Position,
			"ttl":      bp.TTL,
		}
	}
	return out
}
