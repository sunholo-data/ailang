package ollama

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil/wiregolden"
)

// The wire parity suite. testdata/wire/parity.json was captured while this
// package still spoke to Ollama through github.com/ollama/ollama/api
// (v0.40.15): every request body, header and path that SDK put on the wire,
// and every result or error string the callers saw for a fixed set of
// replies. The local client in ./wire must reproduce all of it byte for byte.
// Callers branch on error STRINGS (ai.ClassifyError, telemetry), so the error
// texts are part of the contract, not decoration.

// parityEnv pins every environment knob the native paths read, so the goldens
// do not depend on the machine (the rig exports several of these globally).
func parityEnv(t *testing.T, extra map[string]string) {
	t.Helper()
	for _, k := range []string{
		config.EnvRigLease, config.EnvOllamaHost, config.EnvOllamaNumCtx,
		config.EnvOllamaTemperature, config.EnvOllamaMaxTokens, config.EnvOllamaNativeTools,
		config.EnvOllamaV1Stream, config.EnvOllamaHTTPTimeoutSec,
	} {
		t.Setenv(k, "")
	}
	t.Setenv(config.EnvOllamaLogRequests, filepath.Join(t.TempDir(), "requests.jsonl"))
	for k, v := range extra {
		t.Setenv(k, v)
	}
}

// respView is the caller-visible part of an ai.Response (wall-clock fields
// excluded: they are not wire data).
type respView struct {
	Text         string        `json:"text"`
	Model        string        `json:"model"`
	FinishReason string        `json:"finish_reason,omitempty"`
	ToolCalls    []ai.ToolCall `json:"tool_calls,omitempty"`
	InputTokens  int           `json:"input_tokens"`
	OutputTokens int           `json:"output_tokens"`
	TotalTokens  int           `json:"total_tokens"`
}

type parityCase struct {
	name    string
	env     map[string]string
	replies []wiregolden.Reply
	refused bool // point the client at a closed server
	call    func(ctx context.Context, c *Client) (any, error)
}

func runParity(t *testing.T, golden string, cases []parityCase) {
	t.Helper()
	got := map[string]wiregolden.Outcome{}
	for _, pc := range cases {
		t.Run(pc.name, func(t *testing.T) {
			parityEnv(t, pc.env)
			srv := wiregolden.NewServer(t, pc.replies...)
			if pc.refused {
				srv.Close()
			}
			c, err := NewClient(WithFrozenEndpoint(srv.URL))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			res, err := pc.call(context.Background(), c)
			out := wiregolden.Outcome{Requests: srv.Exchanges()}
			if err != nil {
				out.Error = wiregolden.ErrorText(err, srv.URL)
				out.ErrorCode = ai.ClassifyError(err).Code
			} else {
				out.Result = res
			}
			got[pc.name] = out
		})
	}
	wiregolden.Check(t, golden, got)
}

func generate(req *ai.Request) func(context.Context, *Client) (any, error) {
	return func(ctx context.Context, c *Client) (any, error) {
		r, err := c.Generate(ctx, req)
		return view(r), err
	}
}

func step(req *ai.Request) func(context.Context, *Client) (any, error) {
	return func(ctx context.Context, c *Client) (any, error) {
		r, err := c.Step(ctx, req)
		return view(r), err
	}
}

func view(r *ai.Response) any {
	if r == nil {
		return nil
	}
	return respView{
		Text: r.Text, Model: r.Model, FinishReason: r.FinishReason, ToolCalls: r.ToolCalls,
		InputTokens: r.InputTokens, OutputTokens: r.OutputTokens, TotalTokens: r.TotalTokens,
	}
}

// stallBudget is the AILANG_OLLAMA_HTTP_TIMEOUT_SEC the stall scenarios run
// under. The golden expects the deadline to cut the BODY read ("context
// deadline exceeded"); on a loaded machine a 1s budget occasionally expired
// before the response headers arrived instead, which is a different (also
// correct) error text. 3s keeps the scenario on the body-read path.
const stallBudget = "3"

const (
	genChunk1 = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:00.123456789Z","response":"Hel","done":false}`
	genChunk2 = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:00.2Z","response":"lo <&> é","thinking":"hmm","done":false,"surprise":{"x":[1,2]}}`
	genDone   = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:01Z","response":"","done":true,"done_reason":"stop","context":[1,2,3],"total_duration":123456789,"load_duration":1000,"prompt_eval_count":42,"prompt_eval_duration":5000,"eval_count":7,"eval_duration":6000}`
)

func genReq(mut func(*ai.Request)) *ai.Request {
	r := &ai.Request{
		Model:        "ollama:qwen3:8b",
		SystemPrompt: "You are terse. <b>&amp;</b>",
		UserPrompt:   "Say hello   \"quoted\" \\ tab\t",
	}
	if mut != nil {
		mut(r)
	}
	return r
}

func TestWireParity_Generate(t *testing.T) {
	ok := []wiregolden.Reply{{Lines: []string{genChunk1, genChunk2, genDone}}}
	cases := []parityCase{
		{name: "basic", replies: ok, call: generate(genReq(nil))},
		{name: "json_format_maxtokens_temperature", replies: ok, call: generate(genReq(func(r *ai.Request) {
			r.ResponseFormat, r.MaxTokens, r.Temperature = "json", 100, 0.3
		}))},
		{name: "json_schema_format", replies: ok, call: generate(genReq(func(r *ai.Request) {
			r.ResponseFormat = "json"
			r.ResponseSchema = "{ \"type\" : \"object\",\n \"properties\": {\"z\": {\"type\":\"string\", \"pattern\":\"<a&b>\"}, \"a\": {\"type\":\"integer\"}} }"
		}))},
		{name: "num_ctx_env", env: map[string]string{config.EnvOllamaNumCtx: "65536"}, replies: ok, call: generate(genReq(nil))},
		{name: "err_404_json", replies: []wiregolden.Reply{{Status: 404, Body: `{"error":"model 'qwen3:8b' not found"}`}}, call: generate(genReq(nil))},
		{name: "err_500_text", replies: []wiregolden.Reply{{Status: 500, Body: "boom: runner exploded"}}, call: generate(genReq(nil))},
		// The one scenario that departs from the SDK on purpose: it returned an
		// empty successful result here; the local client reports the 500.
		{name: "err_500_empty_body", replies: []wiregolden.Reply{{Status: 500, Body: ""}}, call: generate(genReq(nil))},
		{name: "err_401_signin", replies: []wiregolden.Reply{{Status: 401, Body: `{"error":"unauthorized","signin_url":"https://ollama.com/connect?x=1"}`}}, call: generate(genReq(nil))},
		{name: "err_429_json", replies: []wiregolden.Reply{{Status: 429, Body: `{"error":"too many requests"}`}}, call: generate(genReq(nil))},
		{name: "err_midstream_error_line", replies: []wiregolden.Reply{{Lines: []string{genChunk1, `{"error":"an error was encountered while running the model: unexpected EOF"}`, genDone}}}, call: generate(genReq(nil))},
		{name: "err_garbage_line", replies: []wiregolden.Reply{{Lines: []string{genChunk1, `this is not json`}}}, call: generate(genReq(nil))},
		{name: "err_blank_line", replies: []wiregolden.Reply{{Lines: []string{genChunk1, ``, genDone}}}, call: generate(genReq(nil))},
		{name: "err_bad_created_at", replies: []wiregolden.Reply{{Lines: []string{`{"model":"m","created_at":"yesterday","response":"x","done":false}`}}}, call: generate(genReq(nil))},
		{name: "err_wrong_field_type", replies: []wiregolden.Reply{{Lines: []string{`{"model":"m","response":5,"done":false}`}}}, call: generate(genReq(nil))},
		{name: "err_line_over_8MB", replies: []wiregolden.Reply{{Lines: []string{genChunk1, `{"response":"` + strings.Repeat("a", 8_000_000) + `"}`}}}, call: generate(genReq(nil))},
		{name: "err_refused", refused: true, call: generate(genReq(nil))},
		{name: "err_stall_deadline", env: map[string]string{config.EnvOllamaHTTPTimeoutSec: stallBudget}, replies: []wiregolden.Reply{{Lines: []string{genChunk1}, Stall: true}}, call: generate(genReq(nil))},
		{name: "no_trailing_newline", replies: []wiregolden.Reply{{Body: genChunk1 + "\n" + genDone}}, call: generate(genReq(nil))},
		{name: "crlf_framing", replies: []wiregolden.Reply{{Body: genChunk1 + "\r\n" + genDone + "\r\n"}}, call: generate(genReq(nil))},
	}
	runParity(t, "testdata/wire/generate.json", cases)
}

// Tool schemas covering every shape the SDK's lossy ToolFunctionParameters /
// ToolProperty decoding treats specially: unknown keywords dropped, struct
// field order imposed, "type" as string or array, nested properties kept in
// key order, free-form items/$defs re-sorted, duplicate keys, null
// properties, an empty schema.
var parityTools = []ai.ToolSchema{
	{Name: "write_file", Description: "Write <a> file & stuff", Parameters: `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","properties":{"path":{"type":"string","description":"file <path>"},"content":{"type":"string","minLength":1}},"required":["path","content"],"additionalProperties":false}`},
	{Name: "shapes", Parameters: `{"type":"object","properties":{"mode":{"type":["string","null"],"enum":["a","b",1,null,{"z":1,"a":2}]},"opts":{"type":"object","properties":{"z":{"type":"integer","minimum":0},"a":{"type":"boolean","default":true}},"required":["z"]},"list":{"type":"array","items":{"type":"object","properties":{"k":{"type":"string"}},"b":2}},"either":{"anyOf":[{"type":"string"},{"type":"number","description":"n"}]},"single":{"type":["string"]}}}`},
	{Name: "empty", Parameters: `{}`},
	{Name: "no_params"},
	{Name: "defs", Parameters: `{"type":"object","$defs":{"X":{"type":"string","z":1,"a":2}},"items":{"q":1,"b":[3,1]},"required":[],"properties":{}}`},
	{Name: "null_props", Parameters: `{"type":"object","properties":null}`},
	{Name: "dupes", Parameters: `{"type":"object","properties":{"b":{"type":"string"},"a":{"type":"string"},"b":{"type":"number"}},"type":"object"}`},
	{Name: "escaped_keys", Parameters: `{"type":"object","properties":{"café":{"type":"string"},"a<b>&c":{"type":"string"},"tab\tkey":{"type":"string"},"bs\bkey":{"type":"string"}}}`},
}

const (
	chatToolChunk = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:00Z","message":{"role":"ASSISTANT","content":"","thinking":"let me think","tool_calls":[{"id":"call_1","function":{"index":0,"name":"write_file","arguments":{"path":"a.ail","content":"x <y> & z","n":1.0,"f":1.5e300,"big":12345678901234567890,"nested":{"z":1,"a":[1,"two",null,true]}}}}]},"done":false}`
	chatToolOdd   = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:00Z","message":{"role":"assistant","content":"","tool_calls":[{"function":{"index":1,"name":"shapes","arguments":null}},{"function":{"name":"empty"}},{"function":{"name":"dupes","arguments":{"k":1,"j":2,"k":3,"café":" ","e":{}}}}]},"done":false}`
	chatText      = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:00Z","message":{"role":"assistant","content":"All <done> & dusted."},"done":false}`
	chatDone      = `{"model":"qwen3:8b","created_at":"2026-11-10T10:00:02Z","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","total_duration":9,"prompt_eval_count":10,"prompt_eval_cached_count":4,"eval_count":5}`
)

// chatReq is a multi-turn conversation exercising every message translation:
// system prepend, tool calls re-attached with good and bad arguments, a tool
// result, and vision input (raw base64, data URI, undecodable).
func chatReq(mut func(*ai.Request)) *ai.Request {
	r := &ai.Request{
		Model:        "ollama/qwen3:8b",
		SystemPrompt: "system <rules> & more",
		Messages: []ai.Message{
			{Role: "user", Content: "look at these", Images: []ai.ImagePart{
				{Source: "UE5HLUJZVEVT", Mime: "image/png"},
				{Source: "data:image/jpeg;base64,SlBFRy1CWVRFUw==", Mime: "image/jpeg"},
				{Source: "!!!not base64!!!", Mime: "image/png"},
			}},
			{Role: "assistant", Content: "calling", ToolCalls: []ai.ToolCall{
				{ID: "c1", Name: "write_file", Arguments: `{"b":1,"a":{"y":2,"x":1},"c":"<>&","n":1.0,"big":12345678901234567890}`},
				{ID: "c2", Name: "empty_args", Arguments: ""},
				{ID: "c3", Name: "null_args", Arguments: "null"},
				{ID: "c4", Name: "array_args", Arguments: "[1,2]"},
				{ID: "c5", Name: "garbage_args", Arguments: "not json"},
				{ID: "c6", Name: "dup_args", Arguments: `{"k":1,"j":2,"k":3}`},
				{ID: "c7", Name: "escaped_args", Arguments: `{"café":"x","tab\tkey":1,"bs\bkey":2,"lt<":3}`},
				{ID: "c8", Name: "string_args", Arguments: `"just a string"`},
			}},
			{Role: "tool", Content: "wrote 3 bytes", ToolCallID: "c1"},
			{Role: "USER", Content: "and now?"},
		},
	}
	if mut != nil {
		mut(r)
	}
	return r
}

func TestWireParity_Chat(t *testing.T) {
	native := map[string]string{config.EnvOllamaNativeTools: "1"}
	withTools := func(r *ai.Request) { r.Tools = parityTools }
	cases := []parityCase{
		{name: "multiturn_no_tools", replies: []wiregolden.Reply{{Lines: []string{chatText, chatDone}}}, call: step(chatReq(nil))},
		{name: "native_tools_toolcall_reply", env: native, replies: []wiregolden.Reply{{Lines: []string{chatToolChunk, chatToolOdd, chatDone}}}, call: step(chatReq(withTools))},
		{name: "native_tools_text_reply_temperature_env", env: map[string]string{config.EnvOllamaNativeTools: "1", config.EnvOllamaTemperature: "0.25", config.EnvOllamaNumCtx: "131072"}, replies: []wiregolden.Reply{{Lines: []string{chatText, chatDone}}}, call: step(chatReq(withTools))},
		{name: "system_in_messages_json_schema", replies: []wiregolden.Reply{{Lines: []string{chatText, chatDone}}}, call: step(chatReq(func(r *ai.Request) {
			r.Messages = append([]ai.Message{{Role: "system", Content: "inline system"}}, r.Messages...)
			r.ResponseFormat = "json"
			r.ResponseSchema = "{ \"type\": \"object\" ,\"properties\":{\"b\":{},\"a\":{}}}"
			r.MaxTokens = 77
			r.Temperature = 0.9
		}))},
		{name: "json_no_schema", replies: []wiregolden.Reply{{Lines: []string{chatText, chatDone}}}, call: step(chatReq(func(r *ai.Request) { r.ResponseFormat = "json" }))},
		{name: "bad_tool_schema_type", env: native, call: step(chatReq(func(r *ai.Request) {
			r.Tools = []ai.ToolSchema{{Name: "bad", Parameters: `{"type":"object","properties":{"x":{"type":5}}}`}}
		}))},
		{name: "bad_tool_schema_json", env: native, call: step(chatReq(func(r *ai.Request) {
			r.Tools = []ai.ToolSchema{{Name: "bad", Parameters: `{"type":`}}
		}))},
		{name: "err_args_as_string", env: native, replies: []wiregolden.Reply{{Lines: []string{`{"model":"m","message":{"role":"assistant","content":"","tool_calls":[{"function":{"name":"x","arguments":"{\"a\":1}"}}]},"done":false}`}}}, call: step(chatReq(withTools))},
		{name: "err_400_json", env: native, replies: []wiregolden.Reply{{Status: 400, Body: `{"error":"registry.ollama.ai/library/qwen3:8b does not support tools"}`}}, call: step(chatReq(withTools))},
		{name: "err_503_text", replies: []wiregolden.Reply{{Status: 503, Body: "Service Unavailable"}}, call: step(chatReq(nil))},
		{name: "err_midstream", replies: []wiregolden.Reply{{Lines: []string{chatText, `{"error":"context length exceeded"}`}}}, call: step(chatReq(nil))},
		{name: "err_refused", refused: true, call: step(chatReq(nil))},
		{name: "err_stall_deadline", env: map[string]string{config.EnvOllamaHTTPTimeoutSec: stallBudget}, replies: []wiregolden.Reply{{Lines: []string{chatText}, Stall: true}}, call: step(chatReq(nil))},
	}
	runParity(t, "testdata/wire/chat.json", cases)
}

func TestWireParity_CheckConnection(t *testing.T) {
	check := func(ctx context.Context, c *Client) (any, error) { return "ok", c.CheckConnection(ctx) }
	cases := []parityCase{
		{name: "ok", replies: []wiregolden.Reply{{Body: `{"models":[{"name":"qwen3:8b","model":"qwen3:8b","modified_at":"2026-11-01T00:00:00Z","size":5,"digest":"abc","details":{"format":"gguf","families":["qwen3"]},"capabilities":["completion","tools"]}]}`}}, call: check},
		{name: "ok_empty_body", replies: []wiregolden.Reply{{Body: ""}}, call: check},
		{name: "err_500_json", replies: []wiregolden.Reply{{Status: 500, Body: `{"error":"boom"}`}}, call: check},
		{name: "err_500_text", replies: []wiregolden.Reply{{Status: 500, Body: "plain failure"}}, call: check},
		{name: "err_500_status_field", replies: []wiregolden.Reply{{Status: 500, Body: `{"error":"e","status":"odd","statuscode":7}`}}, call: check},
		{name: "err_401", replies: []wiregolden.Reply{{Status: 401, Body: `{"error":"x","signin_url":"https://ollama.com/connect"}`}}, call: check},
		{name: "err_401_status_field", replies: []wiregolden.Reply{{Status: 401, Body: `{"Status":"401 nope"}`}}, call: check},
		{name: "err_200_garbage", replies: []wiregolden.Reply{{Body: "not json"}}, call: check},
		{name: "err_200_bad_modified_at", replies: []wiregolden.Reply{{Body: `{"models":[{"name":"m","modified_at":"soon"}]}`}}, call: check},
		{name: "err_refused", refused: true, call: check},
	}
	runParity(t, "testdata/wire/list.json", cases)
}
