package effects

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestAIEmbedTypedAndLazy(t *testing.T) {
	e := &countedEmbedder{}
	for _, lazy := range []bool{false, true} {
		c := NewAIContext(nil).WithEmbedder(e)
		if lazy {
			resolved := c
			c = NewLazyAIContext(func() (*AIContext, error) { return resolved, nil })
		}
		v, err := aiEmbed(&EffContext{AI: c}, []eval.Value{&eval.StringValue{Value: "hello"}})
		if err != nil || v.(*eval.TaggedValue).CtorName != "Ok" {
			t.Fatalf("%v %v", v, err)
		}
	}
	v, err := aiEmbed(&EffContext{}, []eval.Value{&eval.StringValue{Value: "hello"}})
	if err != nil || v.(*eval.TaggedValue).CtorName != "Err" {
		t.Fatalf("%v %v", v, err)
	}
	if _, err := aiEmbed(&EffContext{}, nil); err == nil {
		t.Fatal("missing argument accepted")
	}
	if _, err := aiEmbed(&EffContext{}, []eval.Value{&eval.IntValue{Value: 1}}); err == nil {
		t.Fatal("wrong argument accepted")
	}
}

type errorEmbedder struct {
	countedEmbedder
	err error
}

func (e *errorEmbedder) Embed(string) ([]float32, error) { return nil, e.err }
func TestAIEmbedErrorsAndBudget(t *testing.T) {
	for _, code := range []string{ai.CodeRateLimit, ai.CodeTimeout} {
		ctx := NewEffContext(nil)
		ctx.Grant(NewCapability("AI"))
		ctx.AI = NewAIContext(nil).WithEmbedder(&errorEmbedder{err: ai.NewAIError(code, "test", true)})
		v, err := Call(ctx, "AI", "embed", []eval.Value{&eval.StringValue{Value: "text"}})
		if err != nil || !strings.Contains(v.String(), code) || !strings.Contains(v.String(), "true") {
			t.Fatalf("typed error lost: %v %v", v, err)
		}
	}
	e := &countedEmbedder{}
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(nil).WithEmbedder(NewCachingEmbedder(e, nil))
	ctx.Budget = NewBudgetContext(map[string]*int{"AI": intPtr(2)})
	for i := 0; i < 2; i++ {
		if _, err := Call(ctx, "AI", "embed", []eval.Value{&eval.StringValue{Value: "same"}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := Call(ctx, "AI", "embed", []eval.Value{&eval.StringValue{Value: "same"}}); err == nil {
		t.Fatal("cache hit bypassed budget")
	}
	if e.calls != 1 {
		t.Fatalf("provider called %d times", e.calls)
	}
}
