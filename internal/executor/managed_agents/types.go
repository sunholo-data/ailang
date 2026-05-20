// Package managed_agents implements an executor for Google Cloud's Vertex AI
// Managed Agents API (Gemini Enterprise Agent Platform).
//
// The Managed Agents API runs the Antigravity agent harness (powered by
// gemini-3-5-flash by default) in a Google-hosted, isolated Linux sandbox per
// interaction, persists state across multi-turn calls, and exposes a single
// REST endpoint with SSE streaming. It replaces our local gemini-cli executor
// (retired in v0.22.0 per M-MANAGED-AGENTS — Google deprecates gemini-cli on
// 2026-06-18).
//
// Endpoint: POST aiplatform.googleapis.com/v1beta1/projects/<p>/locations/global/interactions
// Auth: Application Default Credentials (Bearer token from gcloud)
// Required headers: Content-Type, Authorization, Api-Revision
//
// Verified live against project ailang-dev on 2026-05-20; see testdata/sse_pong.txt
// for the canonical SSE event stream.
package managed_agents

import "encoding/json"

// interactionRequest is the JSON body POSTed to the Managed Agents endpoint.
//
// Required fields:
//   - Stream:      true (SSE streaming is the only supported mode for now)
//   - Background:  true (API rejects sync with "Chiliagon path must set background to true")
//   - Store:       true (retain conversation + sandbox so multi-turn / resume works)
//   - Agent:       "antigravity-preview-05-2026" (only public agent at the moment;
//     others may appear in models.yml as agent_model_name)
//   - Environment: {"type": "remote"} for a fresh sandbox; reuse an existing
//     one by passing a raw env_id string in agent_model_name.
//   - Input:       structured array; one user_input with one text content
//     is enough for the eval harness.
type interactionRequest struct {
	Stream            bool            `json:"stream"`
	Background        bool            `json:"background"`
	Store             bool            `json:"store"`
	Agent             string          `json:"agent"`
	Environment       json.RawMessage `json:"environment"`
	Input             []inputBlock    `json:"input"`
	SystemInstruction string          `json:"system_instruction,omitempty"`
	Interaction       string          `json:"interaction,omitempty"` // For multi-turn resume
}

// inputBlock is one element of the Input array.
type inputBlock struct {
	Type    string         `json:"type"`    // "user_input" for prompts
	Content []contentBlock `json:"content"` // List of typed content pieces
}

// contentBlock is one item inside an input block's Content array.
type contentBlock struct {
	Type string `json:"type"` // "text" for plain text
	Text string `json:"text,omitempty"`
}

// Usage matches the "usage" object inside the interaction.completed event.
// Field names follow the API's snake_case payload exactly.
type Usage struct {
	TotalTokens            int             `json:"total_tokens"`
	TotalInputTokens       int             `json:"total_input_tokens"`
	TotalOutputTokens      int             `json:"total_output_tokens"`
	TotalThoughtTokens     int             `json:"total_thought_tokens"`
	InputTokensByModality  []modalityCount `json:"input_tokens_by_modality,omitempty"`
	OutputTokensByModality []modalityCount `json:"output_tokens_by_modality,omitempty"`
}

type modalityCount struct {
	Modality string `json:"modality"`
	Tokens   int    `json:"tokens"`
}

// completedPayload is the JSON inside an interaction.completed event's data.
type completedPayload struct {
	Interaction struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		EnvironmentID string `json:"environment_id"`
		Usage         Usage  `json:"usage"`
	} `json:"interaction"`
	EventType string `json:"event_type"`
}

// createdPayload is the JSON inside an interaction.created event's data.
type createdPayload struct {
	Interaction struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"interaction"`
	EventType string `json:"event_type"`
}

// stepDeltaPayload is the JSON inside a step.delta event's data.
type stepDeltaPayload struct {
	Index int `json:"index"`
	Delta struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"delta"`
	EventType string `json:"event_type"`
}

// errorPayload is the JSON shape returned on a non-200 response or as an
// error SSE event.
type errorPayload struct {
	Error struct {
		Message string `json:"message"`
		Code    string `json:"code"`
		Status  string `json:"status"`
	} `json:"error"`
}
