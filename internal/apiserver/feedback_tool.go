package apiserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/feedback"
	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// The per-IP feedback rate limiter (see M-MCP-EDGE-THROTTLE) is configured
// by AILANG_RATELIMIT_RPM / AILANG_RATELIMIT_BURST; the defaults live in
// internal/config's Registry. RPM=0 disables.
func feedbackRateLimitRPM() int { return config.RateLimitRPM() }

func feedbackRateLimitBurst() int { return config.RateLimitBurst() }

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
	// Sends a message to a human-reviewed queue: additive, never destructive,
	// and reaches outside the server (Pub/Sub). Declared so MCP directories,
	// which refuse unannotated tools, accept every serve-api server that keeps
	// the built-in tool on.
	hints, _ := protocol.ResolveToolHints([]string{"openWorld"}, true, false)
	tool := &mcp.Tool{
		Name:        "submit_feedback",
		Title:       "Send feedback",
		Annotations: sdkToolAnnotations(hints, "Send feedback"),
		// Written for a listed connector (Claude/ChatGPT directories): say only
		// what the tool does, ask for nothing beyond the report, and name no
		// internal inboxes, agents or sprints.
		Description: "Send a bug report, feature request, documentation gap or limitation about AILANG to the AILANG team. " +
			"Use it only when the user asks to report something, and leave out personal or confidential information. " +
			"Categories: bug, feature, docs, limitation. Body up to 10 KB, snippet up to 4 KB. " +
			"`contact` is optional, for a reply. `package` (vendor/name) sends the report to that AILANG package's maintainers instead. " +
			"Returns a ticket ID.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"title":          {Type: "string", Description: "Short title for the report"},
				"body":           {Type: "string", Description: "Full description (≤10 KB)"},
				"category":       {Type: "string", Description: "bug | feature | docs | limitation"},
				"ailang_version": {Type: "string", Description: "The reporter's CLI version (free-form, used for triage)"},
				"snippet":        {Type: "string", Description: "Optional code/error snippet (≤4 KB)"},
				"contact":        {Type: "string", Description: "Optional address for a reply. Only if the user gives one"},
				"package":        {Type: "string", Description: "Optional vendor/name (e.g. \"sunholo/auth\") to send the report to that package's maintainers. Empty = the AILANG team."},
				"auto_dispatch":  {Type: "boolean", Description: "Let the package maintainers' automation act on the report before a person reads it. Only if the user asks. Default false: a person triages it first."},
			},
			Required: []string{"title", "body", "category", "ailang_version"},
		},
	}
	tool.Meta = ms.toolMeta(tool.Name, ExportInfo{}) // listed: noauth; /mcp/: nil

	ms.mcpServer.AddTool(tool, ms.handleSubmitFeedback)
}

// handleSubmitFeedback enforces the per-IP rate limit before publishing.
// IP comes from the rightmost X-Forwarded-For entry (Cloud Run's GFE always
// appends the real TCP source, regardless of what the client claims).
func (ms *MCPServer) handleSubmitFeedback(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if ms.feedbackRL != nil && req != nil && req.Extra != nil {
		ip := clientIPFromHeader(req.Extra.Header)
		if !ms.feedbackRL.Allow(ip) {
			return feedbackError("rate_limited", "", "submit_feedback rate limit exceeded for this client; retry after 60s")
		}
	}
	return handleSubmitFeedback(ctx, req)
}

func handleSubmitFeedback(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args struct {
		Title         string `json:"title"`
		Body          string `json:"body"`
		Category      string `json:"category"`
		AILangVersion string `json:"ailang_version"`
		Snippet       string `json:"snippet"`
		Contact       string `json:"contact"`
		Package       string `json:"package"`
		AutoDispatch  bool   `json:"auto_dispatch"`
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
		Package:       args.Package,
		AutoDispatch:  args.AutoDispatch,
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
