package mcphttp

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

type toolJSON struct {
	Name         string                    `json:"name"`
	Title        string                    `json:"title,omitempty"`
	Description  string                    `json:"description,omitempty"`
	InputSchema  json.RawMessage           `json:"inputSchema"`
	OutputSchema json.RawMessage           `json:"outputSchema,omitempty"`
	Annotations  *protocol.ToolAnnotations `json:"annotations,omitempty"`
}

type contentJSON struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callResultJSON struct {
	Content           []contentJSON   `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
}

// dispatch answers one JSON-RPC request. A non-nil error means a host callback
// failed; the caller turns that into the frozen envelope for the whole POST.
func (h *handler) dispatch(req request, msg message) (response, error) {
	switch msg.Method {
	case "initialize":
		return h.initialize(msg), nil
	case "ping":
		return resultResponse(msg.ID, struct{}{}), nil
	case "tools/list":
		return resultResponse(msg.ID, map[string]any{"tools": listTools(req.surface)}), nil
	case "tools/call":
		return h.callTool(req, msg)
	default:
		return errorResponse(msg.ID, codeMethodNotFound, "method not found: %q", msg.Method), nil
	}
}

func (h *handler) initialize(msg message) response {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(msg.Params, &params)
	negotiated := SupportedVersions[0]
	if supported(params.ProtocolVersion) {
		negotiated = params.ProtocolVersion
	}
	return resultResponse(msg.ID, map[string]any{
		"protocolVersion": negotiated,
		// Stateless: the tool list can never change mid-session, and this server
		// never sends log messages, so neither is advertised.
		"capabilities": map[string]any{"tools": map[string]any{"listChanged": false}},
		"serverInfo":   map[string]any{"name": h.config.Agent.Name, "version": h.config.Agent.Version},
	})
}

func listTools(surface *protocol.AuthorizedSurface) []toolJSON {
	all := surface.All()
	tools := make([]toolJSON, 0, len(all))
	for _, d := range all {
		tools = append(tools, toolJSON{Name: d.Name, Title: d.Title, Description: d.Description, InputSchema: d.InputSchema, OutputSchema: d.OutputSchema, Annotations: d.Annotations})
	}
	return tools
}

func (h *handler) callTool(req request, msg message) (response, error) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return errorResponse(msg.ID, codeInvalidParams, "invalid params: %v", err), nil
	}
	if _, ok := req.surface.Lookup(params.Name); !ok {
		return errorResponse(msg.ID, codeInvalidParams, "unknown tool %q", params.Name), nil
	}
	result, err := hostcall.Run(req.ctx, h.config.Runner, func(ctx context.Context) (protocol.InvocationResult, error) {
		return h.config.Invoker.Invoke(ctx, req.session, protocol.Invocation{Name: params.Name, Arguments: params.Arguments})
	})
	if err != nil {
		var coded protocol.JSONRPCError
		if errors.As(err, &coded) {
			if code, message := coded.JSONRPCError(); code != 0 && message != "" {
				return errorResponse(msg.ID, code, "%s", message), nil
			}
		}
		return response{}, err
	}
	// Fail loud and distinguishably, as the A2A handler does: no result at all is
	// a different host mistake from malformed JSON.
	if len(result.Value) == 0 {
		return errorResponse(msg.ID, -32603, "host callback returned no result"), nil
	}
	if !json.Valid(result.Value) {
		return errorResponse(msg.ID, -32603, "host callback returned invalid JSON"), nil
	}
	return resultResponse(msg.ID, callResultJSON{
		Content:           []contentJSON{{Type: "text", Text: string(result.Value)}},
		StructuredContent: result.Value,
	}), nil
}
