package ollama

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"
	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/ai/openai"
)

// defaultOllamaV1TimeoutSec bounds a single /v1 tool-calling HTTP call. Generous
// enough for a cold 35B model load + one tool-loop turn, but finite so a stalled
// stream (model reload under GPU contention) fails fast instead of hanging
// forever. Override with AILANG_OLLAMA_HTTP_TIMEOUT_SEC ("0" disables the cap).
const defaultOllamaV1TimeoutSec = 300

// ollamaV1Timeout returns the HTTP client timeout for the /v1 delegation path.
// AILANG_OLLAMA_HTTP_TIMEOUT_SEC overrides the default; "0" (or negative) means
// no timeout (restores the legacy unbounded behaviour for debugging).
func ollamaV1Timeout() time.Duration {
	if v := os.Getenv("AILANG_OLLAMA_HTTP_TIMEOUT_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n <= 0 {
				return 0
			}
			return time.Duration(n) * time.Second
		}
	}
	return defaultOllamaV1TimeoutSec * time.Second
}

// resolveOllamaTemperature picks the sampling temperature for the agentic ollama
// path (M-OLLAMA-TEMPERATURE-KNOB). Precedence:
//  1. an explicit reqTemp > 0 (caller's choice always wins);
//  2. AILANG_OLLAMA_TEMPERATURE, if set to a parseable value > 0;
//  3. 0 — meaning "don't send a temperature; let ollama use the model default"
//     (today's behaviour; qwen3.x defaults to 1.0).
//
// The knob exists because qwen3.x's 1.0 default is high-variance and a likely
// cause of non-deterministic 0-tool-call/prose turns under motoko. Off by
// default so it changes nothing until an A/B opts in. A value of exactly 0 via
// the env is treated as "unset" (downstream temperature fields are omitempty, so
// 0 can't be transmitted distinctly anyway) — use a small positive value for
// near-greedy decoding.
// ollamaCallContext bounds a single ollama call with a deadline so a stalled
// stream cannot hang the process forever. The native /api/chat path streams via
// c.client.Chat with NO client timeout — if ollama goes idle and produces no
// data and no error (e.g. a model-reload deadlock, or the env-server RPC wedges),
// the read blocks indefinitely (observed: a 7h motoko hang with ollama idle, NOT
// caught by the /v1 http timeout which only guards the /v1 delegation). This adds
// a per-call ctx deadline covering BOTH paths + Generate. Uses the same
// AILANG_OLLAMA_HTTP_TIMEOUT_SEC budget (default 300s; 0 disables). The returned
// cancel MUST be deferred by the caller.
func ollamaCallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if d := ollamaV1Timeout(); d > 0 {
		return context.WithTimeout(ctx, d)
	}
	return ctx, func() {}
}

func resolveOllamaTemperature(reqTemp float64) float64 {
	if reqTemp > 0 {
		return reqTemp
	}
	if v := os.Getenv("AILANG_OLLAMA_TEMPERATURE"); v != "" {
		if t, err := strconv.ParseFloat(v, 64); err == nil && t > 0 {
			return t
		}
	}
	return 0
}

// defaultOllamaMaxTokens is the FLOOR output budget for the agentic ollama path.
// qwen3.x are heavy reasoners: they emit thousands of tokens of <think> BEFORE the
// tool call, so a small budget truncates (finish_reason=length) before the call is
// produced — which the agent loop then sees as a 0-tool-call no-op. That is the
// dominant motoko disengagement (wire-proven 2026-06-19: median ~13.9k chars of
// reasoning, motoko sent max_tokens=4096, pi sends 16384). 16384 fits the observed
// reasoning (~12.5k tokens) plus the call, and matches pi.
const defaultOllamaMaxTokens = 16384

// resolveOllamaMaxTokens picks the output-token budget for the agentic ollama path.
// Precedence:
//  1. AILANG_OLLAMA_MAX_TOKENS, if a parseable value > 0 (per-model override — the
//     eval adapter can forward the model's declared max_output_tokens here);
//  2. the caller's reqMax when it already meets the floor;
//  3. the floor (defaultOllamaMaxTokens) — so an unset/small default (e.g. motoko's
//     std/ai 4096) can't truncate a reasoning model mid-thought.
func resolveOllamaMaxTokens(reqMax int) int {
	if v := os.Getenv("AILANG_OLLAMA_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if reqMax >= defaultOllamaMaxTokens {
		return reqMax
	}
	return defaultOllamaMaxTokens
}

// logOllamaRequest appends the logical request to AILANG_OLLAMA_LOG_REQUESTS (a
// JSONL file) when that env is set. Best-effort: any error is silently ignored
// (observability must never affect the call). Captures system prompt + messages
// + tools + sampling — enough to diff what a harness sends to the model.
// ollamaLogRequestPath resolves where to dump requests. AILANG_OLLAMA_LOG_REQUESTS
// wins; otherwise a sentinel file at $HOME/.ailang/state/ollama-log-requests whose
// CONTENTS are the dump path. The sentinel exists because some harnesses (motoko's
// bun→ailang process chain) do NOT propagate our custom env to the ailang runtime
// that makes the AI call — but HOME always propagates, so a HOME-relative sentinel
// reaches them. Empty result = logging off.
func ollamaLogRequestPath() string {
	if p := os.Getenv("AILANG_OLLAMA_LOG_REQUESTS"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(home, ".ailang", "state", "ollama-log-requests"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func logOllamaRequest(req *ai.Request) {
	rec := map[string]any{
		"kind":          "request",
		"ts":            time.Now().UTC().Format(time.RFC3339Nano),
		"model":         req.Model,
		"system_prompt": req.SystemPrompt,
		"messages":      req.Messages,
		"tools":         req.Tools,
		"temperature":   req.Temperature,
		"max_tokens":    req.MaxTokens,
	}
	appendOllamaLog(rec)
}

// logOllamaResponse records what the model actually emitted (prose text +
// tool-call names + finish_reason), paired with the preceding request record.
// Without this the dump only showed the INPUT; for diagnosing disengagement we
// need to see the OUTPUT — e.g. a 0-tool-call "stop" with prose text is the
// model answering instead of writing a solution.
func logOllamaResponse(resp *ai.Response) {
	if resp == nil {
		return
	}
	names := make([]string, 0, len(resp.ToolCalls))
	for _, tc := range resp.ToolCalls {
		names = append(names, tc.Name)
	}
	appendOllamaLog(map[string]any{
		"kind":          "response",
		"ts":            time.Now().UTC().Format(time.RFC3339Nano),
		"finish_reason": resp.FinishReason,
		"tool_calls":    names,
		"text":          resp.Text,
	})
}

func appendOllamaLog(rec map[string]any) {
	path := ollamaLogRequestPath()
	if path == "" {
		return
	}
	b, err := json.Marshal(rec)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(b, '\n'))
}

// Step is the multi-turn / tool-aware completion entry point introduced by
// M-AI-TOOL-LOOP (v0.17.0). Ollama's /api/chat endpoint now supports native
// tool/function calling for models that advertise it (e.g. Qwen, Llama 3.1+),
// so this adapter translates req.Tools into Ollama's tool schema, threads tool
// calls + tool results through the conversation, and parses tool_calls back out
// of the response. This is what lets AILANG-native agents (e.g. motoko_agent)
// run a real tool loop against a local Ollama model at $0.
//
// Behaviour:
//   - no Messages AND no Tools → delegate to Generate (legacy single-shot path)
//   - otherwise               → multi-turn chat, tools advertised when present
//   - response carries ToolCalls + FinishReason="tool_calls" when the model
//     decides to call a tool; otherwise FinishReason="stop".
//
// Note: a model whose Modelfile template lacks tool support will simply never
// emit tool_calls (it answers in prose). Ollama returns an error only for a
// malformed request, which we surface via ClassifyError.
// bareModel strips the AILANG provider-routing prefix ("ollama:" or "ollama/")
// from a model string so the Ollama API receives the raw model tag — e.g.
// "ollama:qwen3.5:35b-a3b-mxfp8" → "qwen3.5:35b-a3b-mxfp8". GuessProvider uses
// the prefix to ROUTE to this provider, but Ollama's /api/chat rejects the
// prefixed form as "invalid model name". Only the first segment is stripped, so
// the model's own ":tag" (e.g. ":35b-a3b-mxfp8") is preserved.
func bareModel(m string) string {
	if i := strings.IndexAny(m, ":/"); i > 0 && strings.EqualFold(m[:i], "ollama") {
		return m[i+1:]
	}
	return m
}

func (c *Client) Step(ctx context.Context, req *ai.Request) (*ai.Response, error) {
	// Request observability (M-OLLAMA-LOG-REQUESTS): when AILANG_OLLAMA_LOG_REQUESTS
	// is set, append the exact logical request (model, system prompt, messages,
	// tools, sampling) to that JSONL file BEFORE sending. Endpoint-independent —
	// fires regardless of how the call is routed — so it captures harnesses (e.g.
	// motoko) whose ollama endpoint we cannot redirect for an external tap. This
	// is how we diff what each harness actually sends to the model.
	logOllamaRequest(req)

	// Bound the whole call (native /api/chat has no client timeout — see
	// ollamaCallContext). Covers the /v1 delegation and Generate fallback too.
	ctx, cancel := ollamaCallContext(ctx)
	defer cancel()

	// Single-shot path: no Messages and no Tools → fall back to Generate.
	if len(req.Messages) == 0 && len(req.Tools) == 0 {
		return c.Generate(ctx, req)
	}

	// Tool-calling path: route through Ollama's OpenAI-compatible /v1 endpoint
	// (M-OLLAMA-V1-TOOLCALLING). Small local models (e.g. qwen3.x) reliably emit
	// `tool_calls` over /v1/chat/completions — Ollama's compat layer normalizes
	// the model's tool-call output — but frequently emit ZERO native tool calls
	// over /api/chat, which silently degrades AILANG-native agents (motoko) to
	// non-agentic 0-shot. pi/opencode (the 96%/79% reference harnesses) both drive
	// Ollama via /v1 for exactly this reason. Reuse AILANG's OpenAI provider
	// pointed at the Ollama host's /v1 (dummy key; Ollama ignores auth). Set
	// AILANG_OLLAMA_NATIVE_TOOLS=1 to force the legacy native /api/chat tool path.
	if len(req.Tools) > 0 && os.Getenv("AILANG_OLLAMA_NATIVE_TOOLS") != "1" {
		// CRITICAL: bound the call with an HTTP timeout. The /v1 Step path is
		// non-streaming (io.ReadAll of the full body) and openai.NewClient
		// defaults to http.DefaultClient (Timeout: 0). On a single shared local
		// GPU, a concurrent request makes ollama reload the model mid-request,
		// which stalls the open connection — with no timeout the read blocks
		// FOREVER (observed: a motoko run hung 1h54m while ollama sat idle). The
		// native /api/chat path streams chunk-by-chunk so it surfaces a dropped
		// stream quickly; the /v1 path does not, so we MUST cap it ourselves.
		// Override the ceiling with AILANG_OLLAMA_HTTP_TIMEOUT_SEC (0 disables).
		v1 := openai.NewClient("ollama",
			openai.WithBaseURL(strings.TrimRight(c.endpoint, "/")+"/v1"),
			openai.WithHTTPClient(&http.Client{Timeout: ollamaV1Timeout()}),
		)
		r2 := *req
		r2.Model = bareModel(req.Model)
		// M-OLLAMA-TEMPERATURE-KNOB: qwen3.x's ollama default is temperature 1.0
		// (high variance) — a likely cause of non-deterministic 0-tool-call/prose
		// turns under motoko. Allow an opt-in lower temperature via
		// AILANG_OLLAMA_TEMPERATURE (off by default; req.Temperature still wins).
		r2.Temperature = resolveOllamaTemperature(req.Temperature)
		// Reasoning models truncate (finish=length -> 0 tool calls) when the budget
		// is too small for their <think> output. Floor it (M-OLLAMA-MAX-TOKENS-FLOOR).
		r2.MaxTokens = resolveOllamaMaxTokens(req.MaxTokens)
		// Tool-calling (motoko's path) ALWAYS delegates here, so response logging
		// must wrap this return — not just the native path below.
		resp, err := v1.Step(ctx, &r2)
		if err == nil {
			logOllamaResponse(resp)
		}
		return resp, err
	}

	// Translate req.Messages → ollamaapi.Message[], threading tool calls/results.
	messages := make([]ollamaapi.Message, 0, len(req.Messages)+1)

	// Prepend req.SystemPrompt unless req.Messages already carries a system role.
	hasSystemMsg := false
	for _, m := range req.Messages {
		if m.Role == "system" {
			hasSystemMsg = true
			break
		}
	}
	if req.SystemPrompt != "" && !hasSystemMsg {
		messages = append(messages, ollamaapi.Message{Role: "system", Content: req.SystemPrompt})
	}

	for _, m := range req.Messages {
		om := ollamaapi.Message{Role: m.Role, Content: m.Content}
		// Tool-result message: Ollama has a native "tool" role with tool_call_id.
		if m.Role == "tool" {
			om.ToolCallID = m.ToolCallID
		}
		// Assistant message that issued tool calls: re-attach them so the model
		// sees its own prior calls in the running conversation.
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = make([]ollamaapi.ToolCall, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				oc := ollamaapi.ToolCall{Function: ollamaapi.ToolCallFunction{Name: tc.Name}}
				if tc.Arguments != "" {
					// Best-effort: Ollama wants structured args; a parse failure
					// just yields empty args rather than dropping the whole call.
					_ = json.Unmarshal([]byte(tc.Arguments), &oc.Function.Arguments)
				}
				om.ToolCalls = append(om.ToolCalls, oc)
			}
		}
		messages = append(messages, om)
	}

	options := map[string]interface{}{
		"seed":    int64(42),
		"num_ctx": 8192,
	}
	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	} else {
		options["num_predict"] = 4096
	}
	// M-OLLAMA-TEMPERATURE-KNOB: same opt-in override as the /v1 path above, so
	// the native /api/chat tool path is consistent. Off by default.
	if t := resolveOllamaTemperature(req.Temperature); t > 0 {
		options["temperature"] = t
	}

	chatReq := &ollamaapi.ChatRequest{
		Model:    bareModel(req.Model),
		Messages: messages,
		Options:  options,
	}
	if req.ResponseFormat == "json" {
		if req.ResponseSchema != "" {
			chatReq.Format = json.RawMessage(req.ResponseSchema)
		} else {
			chatReq.Format = json.RawMessage(`"json"`)
		}
	}

	// Advertise tools: translate each ai.ToolSchema → ollamaapi.Tool. The
	// ToolSchema.Parameters is a JSON-Schema string; unmarshal it into Ollama's
	// structured parameter type.
	if len(req.Tools) > 0 {
		tools := make(ollamaapi.Tools, 0, len(req.Tools))
		for _, t := range req.Tools {
			fn := ollamaapi.ToolFunction{Name: t.Name, Description: t.Description}
			if t.Parameters != "" {
				if err := json.Unmarshal([]byte(t.Parameters), &fn.Parameters); err != nil {
					return nil, ai.NewAIError(ai.CodeToolsNotSupported,
						fmt.Sprintf("ollama: invalid JSON-Schema parameters for tool %q: %v", t.Name, err), false)
				}
			}
			tools = append(tools, ollamaapi.Tool{Type: "function", Function: fn})
		}
		chatReq.Tools = tools
	}

	var response strings.Builder
	var toolCalls []ai.ToolCall
	err := c.client.Chat(ctx, chatReq, func(resp ollamaapi.ChatResponse) error {
		response.WriteString(resp.Message.Content)
		// Tool calls arrive in the message (possibly across streamed chunks).
		for _, tc := range resp.Message.ToolCalls {
			toolCalls = append(toolCalls, ai.ToolCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments.String(),
			})
		}
		return nil
	})
	if err != nil {
		return nil, ai.ClassifyError(err)
	}
	// The ollama client can swallow a ctx deadline/cancel mid-stream and return
	// nil (treating a cut stream as end-of-stream). Surface the deadline
	// explicitly so a stalled native stream becomes a timeout error, not a
	// silent empty success (the 7h-hang class).
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ai.ClassifyError(ctxErr)
	}

	finish := "stop"
	if len(toolCalls) > 0 {
		finish = "tool_calls"
	}
	out := &ai.Response{
		Text:         response.String(),
		Model:        req.Model,
		FinishReason: finish,
		ToolCalls:    toolCalls,
		// Ollama doesn't report tokens uniformly across versions; leave at 0.
		InputTokens:  0,
		OutputTokens: 0,
		TotalTokens:  0,
	}
	logOllamaResponse(out)
	return out, nil
}
