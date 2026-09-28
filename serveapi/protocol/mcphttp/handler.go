// Package mcphttp serves the tools subset of MCP over stateless Streamable HTTP
// using only the standard library and serveapi/protocol. It exists so a
// consumer whose build must stay free of the MCP SDK (and its OAuth stack) can
// still project host tools over MCP without writing its own JSON-RPC codec
// (ailang#885). serveapi.Server.MCPHandler delegates here; there is one
// implementation.
//
// Scope: initialize, ping, tools/list, tools/call and notifications, on POST
// only. There are no sessions, no GET stream and no server-initiated messages.
// Supported protocol versions are listed in SupportedVersions; anything else
// is refused rather than served.
package mcphttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/serveapi/protocol"
	"github.com/sunholo-data/ailang/serveapi/protocol/hostcall"
)

// SupportedVersions are the MCP protocol versions this handler negotiates,
// newest first. 2026-07-28 (per-request _meta negotiation) is deliberately not
// supported yet.
var SupportedVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

// defaultVersion is what the spec says to assume when a request carries no
// MCP-Protocol-Version header (2025-06-18 transports, "Protocol Version Header").
const defaultVersion = "2025-03-26"

// Config wires the handler to the embedding host. Every field is required.
type Config struct {
	Agent    protocol.AgentInfo
	Resolver protocol.SessionResolver
	Tools    protocol.ToolSource
	Invoker  protocol.Invoker
	// Runner bounds every host call. There is no unbounded default.
	Runner *hostcall.Runner
}

type handler struct{ config Config }

// NewHandler validates config and returns the MCP endpoint handler.
func NewHandler(config Config) (http.Handler, error) {
	switch {
	case config.Resolver == nil:
		return nil, errors.New("mcphttp: session resolver is required")
	case config.Tools == nil:
		return nil, errors.New("mcphttp: tool source is required")
	case config.Invoker == nil:
		return nil, errors.New("mcphttp: invoker is required")
	case config.Runner == nil:
		return nil, errors.New("mcphttp: runner is required")
	case strings.TrimSpace(config.Agent.Name) == "" || strings.TrimSpace(config.Agent.Version) == "":
		return nil, errors.New("mcphttp: agent name and version are required")
	}
	return &handler{config: config}, nil
}

// request is everything dispatch needs for one POST.
type request struct {
	ctx     context.Context
	session protocol.Session
	surface *protocol.AuthorizedSurface
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodyBytes+1))
	if err != nil || len(body) > MaxRequestBodyBytes {
		protocol.WriteMCPEnvelope(w, protocol.RequestID(body), "invalid MCP request body")
		return
	}
	id := protocol.RequestID(body)

	// Authorization and the caller's surface come first, exactly as before this
	// package existed: an unauthorized caller learns nothing about the transport.
	req, ok := h.authorize(w, r, id)
	if !ok {
		return
	}
	if !isJSONContent(r) {
		writeTransportError(w, http.StatusBadRequest, errorResponse(id, codeInvalidRequest, "Content-Type must be application/json"))
		return
	}
	if !acceptsBoth(r) {
		writeTransportError(w, http.StatusBadRequest, errorResponse(id, codeInvalidRequest, "Accept must list both application/json and text/event-stream"))
		return
	}
	version := r.Header.Get("MCP-Protocol-Version")
	if version == "" {
		version = defaultVersion
	} else if !supported(version) {
		writeTransportError(w, http.StatusBadRequest, errorResponse(id, codeInvalidRequest,
			"unsupported MCP-Protocol-Version %q (supported: %s)", version, strings.Join(SupportedVersions, ", ")))
		return
	}
	h.serveBody(w, req, id, version, bytes.TrimSpace(body))
}

func (h *handler) authorize(w http.ResponseWriter, r *http.Request, id json.RawMessage) (request, bool) {
	session, err := hostcall.Run(r.Context(), h.config.Runner, func(ctx context.Context) (protocol.Session, error) {
		return h.config.Resolver.ResolveSession(ctx, r)
	})
	if err != nil {
		if status := protocol.AuthorizationStatus(err); status != 0 {
			http.Error(w, err.Error(), status)
			return request{}, false
		}
		protocol.WriteMCPEnvelope(w, id, protocol.CallbackMessage(err))
		return request{}, false
	}
	descriptors, err := hostcall.Run(r.Context(), h.config.Runner, func(ctx context.Context) ([]protocol.ToolDescriptor, error) {
		return h.config.Tools.Tools(ctx, session)
	})
	if err != nil {
		protocol.WriteMCPEnvelope(w, id, protocol.CallbackMessage(err))
		return request{}, false
	}
	surface, err := protocol.CallerSurface(descriptors)
	if err != nil {
		protocol.WriteMCPEnvelope(w, id, err.Error())
		return request{}, false
	}
	return request{ctx: r.Context(), session: session, surface: surface}, true
}

func (h *handler) serveBody(w http.ResponseWriter, req request, id json.RawMessage, version string, body []byte) {
	if len(body) > 0 && body[0] == '[' {
		// Batches exist only in the 2025-03-26 transport; 2025-06-18 removed them.
		if version != "2025-03-26" {
			writeTransportError(w, http.StatusBadRequest, errorResponse(nil, codeInvalidRequest,
				"batch requests are not supported under protocol version %s", version))
			return
		}
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil {
			writeTransportError(w, http.StatusBadRequest, errorResponse(nil, codeParseError, "invalid JSON: %v", err))
			return
		}
		if len(batch) == 0 {
			writeTransportError(w, http.StatusBadRequest, errorResponse(nil, codeInvalidRequest, "empty batch"))
			return
		}
		h.serveMessages(w, req, id, batch, true)
		return
	}
	h.serveMessages(w, req, id, []json.RawMessage{body}, false)
}

func (h *handler) serveMessages(w http.ResponseWriter, req request, id json.RawMessage, raws []json.RawMessage, batch bool) {
	var responses []response
	for _, raw := range raws {
		var msg message
		if err := json.Unmarshal(raw, &msg); err != nil {
			writeTransportError(w, http.StatusBadRequest, errorResponse(nil, codeParseError, "invalid JSON: %v", err))
			return
		}
		if msg.JSONRPC != "2.0" {
			writeTransportError(w, http.StatusBadRequest, errorResponse(msg.ID, codeInvalidRequest, "expected jsonrpc: \"2.0\""))
			return
		}
		switch {
		case msg.isNotification(), msg.isResponse():
			continue // accepted; this stateless server has nothing to do with them
		case !msg.isRequest():
			writeTransportError(w, http.StatusBadRequest, errorResponse(msg.ID, codeInvalidRequest, "not a JSON-RPC request, notification or response"))
			return
		}
		resp, hostErr := h.dispatch(req, msg)
		if hostErr != nil {
			// A failed host call answers the whole POST with the frozen envelope,
			// never a partial stream (TestEmbeddedMCPFrozenCallbackEnvelopes).
			protocol.WriteMCPEnvelope(w, id, protocol.CallbackMessage(hostErr))
			return
		}
		responses = append(responses, resp)
	}
	switch {
	case len(responses) == 0:
		writeAccepted(w)
	case batch:
		writeSSE(w, responses)
	default:
		writeSSE(w, responses[0])
	}
}

func supported(version string) bool {
	for _, v := range SupportedVersions {
		if v == version {
			return true
		}
	}
	return false
}
