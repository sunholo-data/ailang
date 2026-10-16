package builtins

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

func TestAIEmbedBuiltinRegistrationAndAuthority(t *testing.T) {
	// Exercise the registry's actual implementation: denied calls never reach
	// lazy initialization, while authorized unconfigured calls return typed Err.
	spec, ok := GetSpec("_ai_embed")
	if !ok {
		t.Fatal("embedding builtin missing")
	}
	if spec.Module != "std/ai" || spec.Effect != "AI" || spec.NumArgs != 1 || spec.IsPure || spec.Metadata == nil {
		t.Fatalf("wrong metadata: %+v", spec)
	}
	typ := spec.Type().String()
	if !strings.Contains(typ, "float") || !strings.Contains(typ, "Result") || !strings.Contains(typ, "AI") {
		t.Fatalf("wrong type: %s", typ)
	}
	ctx := effects.NewEffContext(nil)
	initialized := false
	ctx.AI = effects.NewLazyAIContext(func() (*effects.AIContext, error) {
		initialized = true
		return effects.NewAIContext(effects.NewStubAIHandler()), nil
	})
	args := []eval.Value{&eval.StringValue{Value: "hello"}}
	if _, err := spec.Impl(ctx, args); err == nil || initialized {
		t.Fatal("unauthorized embed reached provider")
	}
	ctx.Grant(effects.NewCapability("AI"))
	v, err := spec.Impl(ctx, args)
	if err != nil || v.(*eval.TaggedValue).CtorName != "Err" {
		t.Fatalf("%v %v", v, err)
	}
}

func TestAIEmbedLegacyOllamaSurface(t *testing.T) {
	spec, ok := GetSpec("_ollama_embed")
	if !ok || spec.Effect != "IO" || spec.NumArgs != 2 {
		t.Fatalf("legacy surface changed: %+v", spec)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"test","embeddings":[[0.25,0.75]]}`))
	}))
	defer server.Close()
	t.Setenv("OLLAMA_HOST", server.URL)
	v, err := spec.Impl(effects.NewEffContext(nil), []eval.Value{&eval.StringValue{Value: "test"}, &eval.StringValue{Value: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	list, ok := v.(*eval.ListValue)
	if !ok || len(list.Elements) != 2 || list.Elements[0].(*eval.FloatValue).Value != 0.25 {
		t.Fatalf("legacy result changed: %v", v)
	}
}
