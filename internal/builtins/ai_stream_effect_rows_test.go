package builtins

import (
	"github.com/sunholo-data/ailang/internal/types"
	"testing"
)

func TestAIStreamBuiltinCallbackEffects(t *testing.T) {
	for name, build := range map[string]func() types.Type{
		"stream":   makeAIStepWithStreamType,
		"recorded": makeAIStepWithStreamRecordedType,
	} {
		t.Run(name, func(t *testing.T) {
			fn := build().(*types.TFunc2)
			cb := fn.Params[4].(*types.TFunc2)
			if fn.EffectRow.Tail == nil || cb.EffectRow.Tail == nil {
				t.Fatalf("stream and callback must share an open row: %s", fn)
			}
			if fn.EffectRow.Tail.Name != cb.EffectRow.Tail.Name {
				t.Fatalf("callback effects would escape the outer row: %s", fn)
			}
			if _, ok := fn.EffectRow.Labels["AI"]; !ok {
				t.Fatalf("streaming itself must still require AI: %s", fn)
			}
		})
	}
}
