package apiserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/feedback"
)

// registerFeedbackTool registers the Go-side submit_feedback MCP tool.
//
// We register submit_feedback in Go (not via the AILANG mcp_tools/feedback.ail
// path) because it needs Firestore + Pub/Sub access, which the AILANG runtime
// does not yet expose as effects. The AILANG file (mcp_tools/feedback.ail)
// stays around as documentation + offline validation reference but is hidden
// from MCP registration via @noexpose.
//
// Lazy initialization: the underlying feedback.Publisher opens its Firestore
// + Pub/Sub clients on first call. If AILANG_STORAGE != "gcp" or the project
// isn't configured, the FIRST call returns a structured error envelope.
// Subsequent calls reuse the same singleton.
func (ms *MCPServer) registerFeedbackTool() {
	tool := &mcp.Tool{
		Name: "submit_feedback",
		Description: "Anonymous bug report / feature request / docs gap, queued for human review. " +
			"Submissions land in `ailang messages list --inbox public-feedback`. " +
			"Categories: bug, feature, docs, limitation. " +
			"Body limit 10KB, snippet limit 4KB. " +
			"Optional contact field for follow-up; opaque to the server.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"title":          {Type: "string", Description: "Short title for the report"},
				"body":           {Type: "string", Description: "Full description (≤10 KB)"},
				"category":       {Type: "string", Description: "bug | feature | docs | limitation"},
				"ailang_version": {Type: "string", Description: "The reporter's CLI version (free-form, used for triage)"},
				"snippet":        {Type: "string", Description: "Optional code/error snippet (≤4 KB)"},
				"contact":        {Type: "string", Description: "Optional follow-up address (free-form, opaque to the server)"},
			},
			Required: []string{"title", "body", "category", "ailang_version"},
		},
	}

	ms.mcpServer.AddTool(tool, handleSubmitFeedback)
}

func handleSubmitFeedback(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Title         string `json:"title"`
		Body          string `json:"body"`
		Category      string `json:"category"`
		AILangVersion string `json:"ailang_version"`
		Snippet       string `json:"snippet"`
		Contact       string `json:"contact"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return feedbackError("invalid_arguments", "", fmt.Sprintf("could not parse arguments: %v", err))
	}

	feedReq := feedback.Request{
		Title:         args.Title,
		Body:          args.Body,
		Category:      args.Category,
		AILangVersion: args.AILangVersion,
		Snippet:       args.Snippet,
		Contact:       args.Contact,
	}

	// Validate FIRST so bad input always gets a structured field error,
	// even when the publisher (Firestore + Pub/Sub) isn't reachable. This
	// keeps the schema honest for local-mode probing.
	if err := feedback.Validate(feedReq); err != nil {
		if fe, ok := err.(*feedback.FieldError); ok {
			return feedbackError(fe.Code, fe.Field, fe.Detail)
		}
		return feedbackError("invalid_input", "", err.Error())
	}

	pub, err := feedback.Get(ctx)
	if err != nil {
		return feedbackError("publisher_unavailable", "", err.Error())
	}

	res, err := pub.Submit(ctx, feedReq)
	if err != nil {
		// Validation errors carry structured (code, field) info — surface as
		// {error, field, detail} envelope the AILANG-side schema documents.
		if fe, ok := err.(*feedback.FieldError); ok {
			return feedbackError(fe.Code, fe.Field, fe.Detail)
		}
		return feedbackError("publish_failed", "", err.Error())
	}

	body, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return feedbackError("marshal_failed", "", err.Error())
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil
}

func feedbackError(code, field, detail string) (*mcp.CallToolResult, error) {
	payload := map[string]any{"error": code, "detail": detail}
	if field != "" {
		payload["field"] = field
	}
	body, _ := json.MarshalIndent(payload, "", "  ")
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil
}
