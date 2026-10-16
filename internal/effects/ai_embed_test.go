package effects

import (
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func embedArgs(media bool) []eval.Value {
	a := []eval.Value{&eval.StringValue{Value: "test-model"}, &eval.StringValue{Value: "hello"}}
	if media {
		a = append(a, &eval.BytesValue{Value: []byte("private-image")}, &eval.BytesValue{})
	}
	return a
}
func TestEmbedAuthorityAndBudget(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[1,2]]}`))
	}))
	defer s.Close()
	t.Setenv("OLLAMA_HOST", s.URL)
	for _, op := range []string{"embedOllama", "embedOllamaMedia"} {
		ctx := NewEffContext(nil)
		a := embedArgs(op == "embedOllamaMedia")
		if _, err := Call(ctx, "AI", op, a); err == nil {
			t.Fatal("missing cap accepted")
		}
		ctx.Grant(NewCapability("AI"))
		ctx.SetBudget(NewBudgetContext(map[string]*int{"AI": intPtr(0)}))
		if _, err := Call(ctx, "AI", op, a); err == nil {
			t.Fatal("zero budget accepted")
		}
		if calls != 0 {
			t.Fatal("unauthorized network")
		}
		ctx.SetBudget(NewBudgetContext(map[string]*int{"AI": intPtr(1)}))
		v, err := Call(ctx, "AI", op, a)
		if err != nil {
			t.Fatal(err)
		}
		if len(v.(*eval.ListValue).Elements) != 2 || ctx.Budget.Used("AI") != 1 {
			t.Fatal("vector/budget", v, ctx.Budget)
		}
		calls = 0
	}
}
func TestEmbedStubAndLazy(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings":[[1,2]]}`))
	}))
	defer s.Close()
	t.Setenv("OLLAMA_HOST", s.URL)
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewAIContext(NewStubAIHandler())
	for _, op := range []string{"embedOllama", "embedOllamaMedia"} {
		a := embedArgs(op == "embedOllamaMedia")
		v, err := Call(ctx, "AI", op, a)
		if err != nil {
			t.Fatal(err)
		}
		again, err := Call(ctx, "AI", op, a)
		if err != nil || !reflect.DeepEqual(v, again) {
			t.Fatal("not deterministic")
		}
		xs := v.(*eval.ListValue).Elements
		if len(xs) != 8 {
			t.Fatal(len(xs))
		}
		sum := 0.0
		for _, x := range xs {
			f := x.(*eval.FloatValue).Value
			sum += f * f
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Fatal(sum)
		}
		a[1] = &eval.StringValue{Value: "different"}
		other, err := Call(ctx, "AI", op, a)
		if err != nil || reflect.DeepEqual(v, other) {
			t.Fatal("not input dependent")
		}
	}
	ctx.AI = NewLazyAIContext(func() (*AIContext, error) { t.Fatal("embedding initialized chat provider"); return nil, ErrNoAIHandler })
	if _, err := Call(ctx, "AI", "embedOllama", embedArgs(false)); err != nil {
		t.Fatal(err)
	}
}

func TestEmbedTrace(t *testing.T) {
	for _, fail := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if fail {
				w.WriteHeader(404)
				_, _ = w.Write([]byte(`{"error":"missing"}`))
			} else {
				_, _ = w.Write([]byte(`{"embeddings":[[1]]}`))
			}
		}))
		t.Setenv("OLLAMA_HOST", server.URL)
		for _, op := range []string{"embedOllama", "embedOllamaMedia"} {
			ctx := NewEffContext(nil)
			ctx.Grant(NewCapability("AI"))
			ctx.Trace = trace.NewCollector()
			a := embedArgs(op == "embedOllamaMedia")
			a[1] = &eval.StringValue{Value: strings.Repeat("x", 1000)}
			_, err := Call(ctx, "AI", op, a)
			if (err != nil) != fail {
				t.Fatal(err)
			}
			count := 0
			for _, event := range ctx.Trace.Events() {
				if event.Effect == nil {
					continue
				}
				count++
				for _, arg := range event.Effect.Args {
					if len(arg) > 300 {
						t.Fatal("unbounded trace arg")
					}
				}
				b, _ := json.Marshal(event)
				if strings.Contains(string(b), "private-image") || strings.Contains(string(b), "707269766174652d696d616765") {
					t.Fatal("raw media in trace", string(b))
				}
			}
			if count == 0 {
				t.Fatal("no AI trace")
			}
		}
		server.Close()
	}
}
