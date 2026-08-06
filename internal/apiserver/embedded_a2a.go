package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// EmbeddedA2AConfig supplies request-scoped host operations without creating
// an internal dependency on the public serveapi package.
type EmbeddedA2AConfig struct {
	AgentName        string
	AgentDescription string
	AgentVersion     string
	Runner           *CallbackRunner
	Resolve          func(context.Context, *http.Request) (any, error)
	Tools            func(context.Context, any) ([]ToolDescriptor, error)
	Invoke           func(context.Context, any, string, json.RawMessage) (json.RawMessage, error)
}

type embeddedA2AHandler struct{ config EmbeddedA2AConfig }

// NewEmbeddedA2AHandler returns a recorder-friendly, request-scoped A2A handler.
func NewEmbeddedA2AHandler(config EmbeddedA2AConfig) http.Handler {
	return &embeddedA2AHandler{config: config}
}

func (h *embeddedA2AHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cardRequest := r.Method == http.MethodGet
	if !cardRequest && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET or POST only"})
		return
	}
	session, surface, err := h.resolveSurface(r)
	if err != nil {
		h.writeCallbackError(w, r, nil, cardRequest, err)
		return
	}
	if cardRequest {
		h.writeCard(w, r, surface)
		return
	}
	h.handleTask(w, r, session, surface)
}

func (h *embeddedA2AHandler) resolveSurface(r *http.Request) (any, *AuthorizedSurface, error) {
	session, err := RunCallback(r.Context(), h.config.Runner, func(ctx context.Context) (any, error) {
		return h.config.Resolve(ctx, r)
	})
	if err != nil {
		return nil, nil, err
	}
	descriptors, err := RunCallback(r.Context(), h.config.Runner, func(ctx context.Context) ([]ToolDescriptor, error) {
		return h.config.Tools(ctx, session)
	})
	if err != nil {
		return nil, nil, err
	}
	surface, err := callerSurface(descriptors)
	return session, surface, err
}

func (h *embeddedA2AHandler) writeCard(w http.ResponseWriter, r *http.Request, surface *AuthorizedSurface) {
	skills := make([]map[string]any, 0, len(surface.tools))
	for _, descriptor := range surface.All() {
		skills = append(skills, map[string]any{
			"id": descriptor.Name, "name": descriptor.Name,
			"description": descriptor.Description, "tags": descriptor.Tags, "examples": descriptor.Examples,
		})
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": h.config.AgentName, "description": h.config.AgentDescription,
		"url": fmt.Sprintf("%s://%s", scheme, r.Host), "version": h.config.AgentVersion,
		"capabilities":      map[string]any{"streaming": false, "pushNotifications": false, "stateTransitionHistory": false},
		"defaultInputModes": []string{"application/json"}, "defaultOutputModes": []string{"application/json"},
		"skills": skills,
	})
}

func (h *embeddedA2AHandler) handleTask(w http.ResponseWriter, r *http.Request, session any, surface *AuthorizedSurface) {
	var req a2aRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		a2aError(w, nil, -32700, "invalid JSON: "+err.Error())
		return
	}
	if req.JSONRPC != "2.0" {
		a2aError(w, req.ID, -32600, "expected jsonrpc: \"2.0\"")
		return
	}
	if req.Method != "tasks/send" {
		a2aError(w, req.ID, -32601, "method not found: "+req.Method)
		return
	}
	var params a2aTaskSendParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		a2aError(w, req.ID, -32602, "invalid params: "+err.Error())
		return
	}
	skillID, _ := params.Metadata["skill_id"].(string)
	if _, ok := surface.Lookup(skillID); !ok {
		a2aError(w, req.ID, -32602, fmt.Sprintf("tool %q is not authorized", skillID))
		return
	}
	arguments := embeddedA2AArguments(params.Message)
	result, err := RunCallback(r.Context(), h.config.Runner, func(ctx context.Context) (json.RawMessage, error) {
		return h.config.Invoke(ctx, session, skillID, arguments)
	})
	if err != nil {
		h.writeCallbackError(w, r, req.ID, false, err)
		return
	}
	var value any
	if err := json.Unmarshal(result, &value); err != nil {
		a2aError(w, req.ID, -32603, "host callback returned invalid JSON")
		return
	}
	taskID := params.ID
	if taskID == "" {
		taskID = "embedded-task"
	}
	a2aResult(w, req.ID, map[string]any{"id": taskID, "status": map[string]any{"state": "completed"},
		"artifacts": []map[string]any{{"parts": []map[string]any{{"type": "data", "data": map[string]any{"result": value}}, {"type": "text", "text": string(result)}}}}})
}

func embeddedA2AArguments(message a2aMessage) json.RawMessage {
	for _, part := range message.Parts {
		if part.Type == "data" && part.Data != nil {
			if args, ok := part.Data["args"]; ok {
				result, _ := json.Marshal(args)
				return result
			}
		}
		if part.Type == "text" && json.Valid([]byte(part.Text)) {
			return json.RawMessage(part.Text)
		}
	}
	return json.RawMessage("[]")
}

func (h *embeddedA2AHandler) writeCallbackError(w http.ResponseWriter, r *http.Request, id json.RawMessage, card bool, err error) {
	if status := authorizationStatus(err); status != 0 {
		http.Error(w, err.Error(), status)
		return
	}
	message := callbackMessage(err)
	if !card {
		a2aError(w, id, -32603, message)
		return
	}
	status := http.StatusInternalServerError
	if message == "host callback timed out" {
		status = http.StatusGatewayTimeout
	} else if message == "host callback capacity exceeded" {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, map[string]string{"error": message})
}
