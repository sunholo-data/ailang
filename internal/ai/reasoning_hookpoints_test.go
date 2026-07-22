package ai_test

// M0 acceptance test (M-AI-REASONING-EFFORT, fork-a mandatory gate).
//
// Proves that EVERY in-scope request constructor — Generate, Step, and
// StreamStep on all four providers (openai, gemini, anthropic, openrouter) —
// invokes the ONE shared resolver (ai.ResolveReasoning) BEFORE any HTTP
// dispatch. The proof: an invalid Request.ReasoningEffort must be rejected with
// the typed ai.ErrInvalidReasoningEffort and ZERO HTTP hits reach the wire.
//
// This is network-free: each client is pointed at a base URL whose transport
// FAILS the test if it is ever hit. A resolver that ran before marshal/dispatch
// means the failing transport is never reached.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/anthropic"
	"github.com/sunholo-data/ailang/internal/ai/gemini"
	"github.com/sunholo-data/ailang/internal/ai/openai"
	"github.com/sunholo-data/ailang/internal/ai/openrouter"
)

// failingServer returns an httptest server that fails the test if it receives
// any request, plus its URL. Used to prove no dispatch occurs.
func failingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("HTTP request dispatched to %s — reasoning validation must occur BEFORE dispatch", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
}

// invalidReq builds a request with an invalid typed ReasoningEffort. Messages
// is populated so Step/StreamStep have a turn to (attempt to) send.
func invalidReq(model string) *ai.Request {
	return &ai.Request{
		Model:           model,
		SystemPrompt:    "sys",
		UserPrompt:      "hi",
		Messages:        []ai.Message{{Role: "user", Content: "hi"}},
		ReasoningEffort: "bogus", // not one of ""/off/low/medium/high
	}
}

// assertInvalidReasoning asserts err is the typed ErrInvalidReasoningEffort
// wrapped in a non-retryable *ai.AIError.
func assertInvalidReasoning(t *testing.T, path string, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected ErrInvalidReasoningEffort, got nil", path)
	}
	if !errors.Is(err, ai.ErrInvalidReasoningEffort) {
		t.Fatalf("%s: error = %v, want ErrInvalidReasoningEffort", path, err)
	}
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("%s: error is not *ai.AIError: %T", path, err)
	}
	if aiErr.Code != ai.CodeSchemaValidation || aiErr.Retryable {
		t.Fatalf("%s: want non-retryable SchemaValidation, got code=%q retryable=%v", path, aiErr.Code, aiErr.Retryable)
	}
}

func TestAllRequestPathsInvokeResolver(t *testing.T) {
	ctx := context.Background()
	noChunk := func(ai.StreamChunk) {}

	t.Run("openai", func(t *testing.T) {
		srv := failingServer(t)
		defer srv.Close()
		// Exercise both Responses and Chat routing by forcing each API type.
		for _, apiType := range []openai.APIType{openai.APIResponses, openai.APIChatCompletions} {
			c := openai.NewClient("k", openai.WithBaseURL(srv.URL), openai.WithAPIType(apiType))
			_, err := c.Generate(ctx, invalidReq("test"))
			assertInvalidReasoning(t, "openai.Generate", err)
		}
		c := openai.NewClient("k", openai.WithBaseURL(srv.URL))
		_, err := c.Step(ctx, invalidReq("test"))
		assertInvalidReasoning(t, "openai.Step", err)
		_, err = c.StreamStep(ctx, invalidReq("test"), noChunk)
		assertInvalidReasoning(t, "openai.StreamStep", err)
	})

	t.Run("gemini", func(t *testing.T) {
		srv := failingServer(t)
		defer srv.Close()
		c := gemini.NewClient("k", gemini.WithBaseURL(srv.URL))
		_, err := c.Generate(ctx, invalidReq("gemini-x"))
		assertInvalidReasoning(t, "gemini.Generate", err)
		_, err = c.Step(ctx, invalidReq("gemini-x"))
		assertInvalidReasoning(t, "gemini.Step", err)
		_, err = c.StreamStep(ctx, invalidReq("gemini-x"), noChunk)
		assertInvalidReasoning(t, "gemini.StreamStep", err)
	})

	t.Run("anthropic", func(t *testing.T) {
		srv := failingServer(t)
		defer srv.Close()
		c := anthropic.NewClient("k", anthropic.WithBaseURL(srv.URL))
		_, err := c.Generate(ctx, invalidReq("claude-x"))
		assertInvalidReasoning(t, "anthropic.Generate", err)
		_, err = c.Step(ctx, invalidReq("claude-x"))
		assertInvalidReasoning(t, "anthropic.Step", err)
		_, err = c.StreamStep(ctx, invalidReq("claude-x"), noChunk)
		assertInvalidReasoning(t, "anthropic.StreamStep", err)
	})

	t.Run("openrouter", func(t *testing.T) {
		srv := failingServer(t)
		defer srv.Close()
		c := openrouter.NewClient("k", openrouter.WithBaseURL(srv.URL))
		_, err := c.Generate(ctx, invalidReq("x/y"))
		assertInvalidReasoning(t, "openrouter.Generate", err)
		_, err = c.Step(ctx, invalidReq("x/y"))
		assertInvalidReasoning(t, "openrouter.Step", err)
		_, err = c.StreamStep(ctx, invalidReq("x/y"), noChunk)
		assertInvalidReasoning(t, "openrouter.StreamStep", err)
	})
}
