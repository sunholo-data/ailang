package builtins

import (
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/types"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaEmbeddingSurface(t *testing.T) {
	for _, name := range []string{"_ollama_embed", "_ollama_embed_media", "_ollama_embed_result", "_ollama_embed_media_result"} {
		spec, ok := GetSpec(name)
		if !ok {
			t.Errorf("missing %s", name)
			continue
		}
		if spec.Effect != "AI" || spec.IsPure {
			t.Errorf("effect for %s = %s", name, spec.Effect)
		}
		typ := spec.Type().(*types.TFunc2)
		if _, ok := typ.EffectRow.Labels["AI"]; !ok {
			t.Errorf("type lacks AI: %v", typ)
		}
		media := strings.Contains(name, "media")
		args := []eval.Value{&eval.StringValue{Value: "m"}, &eval.StringValue{Value: "hi"}}
		if media {
			args = append(args, &eval.BytesValue{}, &eval.BytesValue{})
		}
		ctx := effects.NewEffContext(nil)
		if v, err := spec.Impl(ctx, args); err == nil {
			t.Fatalf("%s denied cap became %v", name, v)
		}
		ctx.Grant(effects.NewCapability("AI"))
		ctx.AI = effects.NewAIContext(effects.NewStubAIHandler())
		zero := 0
		ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"AI": &zero}))
		if v, err := spec.Impl(ctx, args); err == nil {
			t.Fatalf("%s denied budget became %v", name, v)
		}
		ctx.SetBudget(nil)
		ctx.SetOperatorBudget(effects.NewOperatorBudget(map[string]int{"AI": 0}))
		if v, err := spec.Impl(ctx, args); err == nil {
			t.Fatalf("%s operator denial became %v", name, v)
		}
		ctx.SetOperatorBudget(nil)
		one := 1
		ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"AI": &one}))
		// Emulate runtime wrapper authorization plus the normal builtin charge scope.
		if err := ctx.RequireCapWithBudget("AI", ""); err != nil {
			t.Fatal(err)
		}
		ctx.BeginBudgetChargeScope()
		v, err := spec.Impl(ctx, args)
		ctx.EndBudgetChargeScope()
		if err != nil {
			t.Fatal(err)
		}
		if ctx.Budget.Used("AI") != 1 {
			t.Fatal("double charged", name)
		}
		if strings.HasSuffix(name, "result") {
			tag := v.(*eval.TaggedValue)
			if tag.CtorName != "Ok" || len(tag.Fields[0].(*eval.ListValue).Elements) != 8 {
				t.Fatal(v)
			}
		}
	}
}
func TestOllamaEmbeddingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"error":"model missing"}`))
	}))
	defer server.Close()
	t.Setenv("OLLAMA_HOST", server.URL)
	for _, name := range []string{"_ollama_embed", "_ollama_embed_media", "_ollama_embed_result", "_ollama_embed_media_result"} {
		spec, ok := GetSpec(name)
		if !ok {
			t.Errorf("missing %s", name)
			continue
		}
		ctx := effects.NewEffContext(nil)
		ctx.Grant(effects.NewCapability("AI"))
		args := []eval.Value{&eval.StringValue{Value: "m"}, &eval.StringValue{Value: "hi"}}
		if strings.Contains(name, "media") {
			args = append(args, &eval.BytesValue{}, &eval.BytesValue{})
		}
		v, err := spec.Impl(ctx, args)
		if strings.HasSuffix(name, "result") {
			if err != nil {
				t.Fatal(err)
			}
			tag := v.(*eval.TaggedValue)
			prefix := "_ollama_embed: Ollama embed failed: "
			if strings.Contains(name, "media") {
				prefix = "_ollama_embed_media: Ollama embed failed: "
			}
			if tag.CtorName != "Err" || !strings.HasPrefix(tag.Fields[0].(*eval.StringValue).Value, prefix) {
				t.Fatal(v)
			}
		} else if err == nil {
			t.Fatal("legacy failure didn't abort")
		}
	}
}
