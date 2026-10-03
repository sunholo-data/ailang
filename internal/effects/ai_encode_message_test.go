package effects

import (
	"sort"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

// TestMakeOkStepResult_MessageMatchesDeclaredType is the regression test for
// #572: the assistant Message inside a StepResult must carry every field of
// the closed std/ai.Message record (role, content, tool_calls, tool_call_id,
// images). Before the fix `images` was absent, so `result.message.images`
// type-checked but panicked at runtime with "record has no field: images".
// makeOkStepResult is the single constructor behind step, stepWithStream,
// the recorded-stream path and the WASM handler.
func TestMakeOkStepResult_MessageMatchesDeclaredType(t *testing.T) {
	out := makeOkStepResult(&ai.Response{Text: "hi", FinishReason: "stop"})
	tagged, ok := out.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok(...), got %#v", out)
	}
	msg := tagged.Fields[0].(*eval.RecordValue).Fields["message"].(*eval.RecordValue)

	// Must match std/ai.ail `type Message` and builtins.messageRecordType.
	want := []string{"content", "images", "role", "tool_call_id", "tool_calls"}
	got := make([]string, 0, len(msg.Fields))
	for k := range msg.Fields {
		got = append(got, k)
	}
	sort.Strings(got)
	if len(got) != len(want) {
		t.Fatalf("message fields = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("message fields = %v, want %v", got, want)
		}
	}

	images, ok := msg.Fields["images"].(*eval.ListValue)
	if !ok {
		t.Fatalf("images = %T, want *eval.ListValue", msg.Fields["images"])
	}
	if len(images.Elements) != 0 {
		t.Errorf("images len = %d, want 0 (responses carry no vision input)", len(images.Elements))
	}
}
