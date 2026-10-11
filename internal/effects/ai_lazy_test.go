package effects

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

type lazySurfaceHandler struct {
	*StubAIHandler
	model, options string
	route          *trace.ResolvedRoute
}

func (h *lazySurfaceHandler) LastRoutingMetadata() *trace.ResolvedRoute { return h.route }
func (h *lazySurfaceHandler) Step(m string, _ []ai.Message, _ []ai.ToolSchema) (*ai.Response, error) {
	h.model = m
	return &ai.Response{Text: "step"}, nil
}
func (h *lazySurfaceHandler) StepWithCache(m string, ms []ai.Message, ts []ai.ToolSchema, _ []ai.CacheBreakpoint) (*ai.Response, error) {
	return h.Step(m, ms, ts)
}
func (h *lazySurfaceHandler) StepWithStream(m string, ms []ai.Message, ts []ai.ToolSchema, _ []ai.CacheBreakpoint, cb func(ai.StreamChunk)) (*ai.Response, error) {
	if cb != nil {
		cb(ai.StreamContentDelta{Text: "chunk"})
	}
	return h.Step(m, ms, ts)
}
func (h *lazySurfaceHandler) CallImage(_, path, options string) (string, error) {
	h.options = options
	return path, nil
}
func (h *lazySurfaceHandler) CallImageBase64(_, options string) (string, error) {
	h.options = options
	return "image", nil
}
func (h *lazySurfaceHandler) CallImageWithRefs(p string, _ []ai.ImagePart, path, options string) (string, error) {
	return h.CallImage(p, path, options)
}
func (h *lazySurfaceHandler) CallImageBase64WithRefs(p string, _ []ai.ImagePart, options string) (string, error) {
	return h.CallImageBase64(p, options)
}
func (h *lazySurfaceHandler) CallSpeech(_, _, _, options string) ([]byte, error) {
	h.options = options
	return []byte{1, 2}, nil
}

func lazyOperations(c *AIContext) map[string]func() error {
	return map[string]func() error{
		"call":       func() error { _, e := c.Call("hi"); return e },
		"json":       func() error { _, e := c.CallJson("hi", "{}"); return e },
		"step":       func() error { _, e := c.Step("friendly", nil, nil); return e },
		"cache":      func() error { _, e := c.StepWithCache("friendly", nil, nil, nil); return e },
		"stream":     func() error { _, e := c.StepWithStream("friendly", nil, nil, nil, nil); return e },
		"image":      func() error { _, e := c.CallImage("hi", "out", `{"model":"friendly"}`); return e },
		"base64":     func() error { _, e := c.CallImageBase64("hi", `{"model":"friendly"}`); return e },
		"refs":       func() error { _, e := c.CallImageWithRefs("hi", nil, "out", `{"model":"friendly"}`); return e },
		"refsBase64": func() error { _, e := c.CallImageBase64WithRefs("hi", nil, `{"model":"friendly"}`); return e },
		"speech":     func() error { _, e := c.CallSpeech("hi", "voice", "style", "{}"); return e },
	}
}

func TestLazyAIContext_AllSurfacesAndMetadata(t *testing.T) {
	h := &lazySurfaceHandler{StubAIHandler: NewStubAIHandler(), route: &trace.ResolvedRoute{}}
	initialized := 0
	c := NewLazyAIContext(func() (*AIContext, error) {
		initialized++
		return NewAIContext(h).WithModelResolver(func(m string) (string, error) { return "api-" + m, nil }), nil
	})
	if c.LastRoutingMetadata() != nil || initialized != 0 {
		t.Fatal("metadata initialized provider")
	}
	for name, op := range lazyOperations(c) {
		t.Run(name, func(t *testing.T) {
			if err := op(); err != nil {
				t.Fatal(err)
			}
		})
	}
	if initialized != 1 || c.LastRoutingMetadata() != h.route {
		t.Fatal("initialization/routing capability lost")
	}
	if h.model != "api-friendly" {
		t.Fatalf("resolver lost: %s", h.model)
	}
	// Every image variant receives the resolved per-call model.
	for _, name := range []string{"image", "base64", "refs", "refsBase64"} {
		if err := lazyOperations(c)[name](); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(h.options, "api-friendly") {
			t.Fatalf("%s options=%s", name, h.options)
		}
	}
	chunks := 0
	if _, err := c.StepWithStream("friendly", nil, nil, nil, func(ai.StreamChunk) { chunks++ }); err != nil || chunks != 1 {
		t.Fatalf("stream callback: %d %v", chunks, err)
	}
}

func TestLazyAIContext_CachedFailureEverySurface(t *testing.T) {
	sentinel := ai.NewAIError(ai.CodeAuthFailed, "missing credential", false)
	initialized := 0
	c := NewLazyAIContext(func() (*AIContext, error) { initialized++; return nil, sentinel })
	for name, op := range lazyOperations(c) {
		if err := op(); !errors.Is(err, sentinel) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if initialized != 1 || c.LastRoutingMetadata() != nil {
		t.Fatal("failure retried or metadata initialized")
	}
}

func TestLazyAIContext_ConcurrentPublication(t *testing.T) {
	var count atomic.Int32
	c := NewLazyAIContext(func() (*AIContext, error) { count.Add(1); return NewAIContext(NewStubAIHandler()), nil })
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := c.Call("hi"); err != nil {
				t.Error(err)
			}
			c.LastRoutingMetadata()
		}()
	}
	wg.Wait()
	if count.Load() != 1 {
		t.Fatal("multiple initialization")
	}
}

func TestLazyAIContext_AuthorityBeforeInitialization(t *testing.T) {
	for _, denied := range []bool{true, false} {
		count := 0
		ctx := NewEffContext(nil)
		ctx.AI = NewLazyAIContext(func() (*AIContext, error) { count++; return NewAIContext(NewStubAIHandler()), nil })
		if !denied {
			ctx.Grant(NewCapability("AI"))
			ctx.Budget = NewBudgetContext(map[string]*int{"AI": intPtr(0)})
		}
		if _, err := Call(ctx, "AI", "call", []eval.Value{&eval.StringValue{Value: "hi"}}); err == nil {
			t.Fatal("unauthorized call succeeded")
		}
		if count != 0 {
			t.Fatal("denied/exhausted AI initialized provider")
		}
	}
}

func TestLazyAIContext_ResultRetainsAuthError(t *testing.T) {
	ctx := NewEffContext(nil)
	ctx.Grant(NewCapability("AI"))
	ctx.AI = NewLazyAIContext(func() (*AIContext, error) { return nil, ai.NewAIError(ai.CodeAuthFailed, "missing credential", false) })
	for _, op := range []string{"callResult", "callJsonResult", "callSpeech"} {
		args := []eval.Value{&eval.StringValue{Value: "hello"}}
		if op == "callJsonResult" {
			args = append(args, &eval.StringValue{Value: "{}"})
		}
		if op == "callSpeech" {
			for i := 0; i < 3; i++ {
				args = append(args, &eval.StringValue{Value: ""})
			}
		}
		v, err := Call(ctx, "AI", op, args)
		if err != nil {
			t.Fatal(err)
		}
		result, ok := v.(*eval.TaggedValue)
		if !ok || result.CtorName != "Err" || !strings.Contains(v.String(), "AuthFailed") {
			t.Fatalf("%s: %v", op, v)
		}
	}
}

func TestLazyAIContext_OuterResolverRetained(t *testing.T) {
	h := &lazySurfaceHandler{StubAIHandler: NewStubAIHandler()}
	c := NewLazyAIContext(func() (*AIContext, error) { return NewAIContext(h), nil }).WithModelResolver(func(m string) (string, error) { return "outer-" + m, nil })
	if _, err := c.Step("model", nil, nil); err != nil {
		t.Fatal(err)
	}
	if h.model != "outer-model" {
		t.Fatalf("outer resolver lost: %s", h.model)
	}
}
