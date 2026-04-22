// ailang-microrag-mcp — MCP server frontend for the micro-rag engine.
//
// Exposes two tools any MCP-aware harness (Cursor, Continue, Cline, Claude
// Desktop, etc.) can call when its agent edits files:
//
//	microrag_context_for_file  — JIT knowledge injection on Edit/Write/Read
//	microrag_lint_builtin      — first-use builtin signature nudges
//
// The engine itself lives in internal/microrag and is used identically by the
// `ailang micro-rag` CLI subcommand and the Claude Code bash hooks. This
// binary exists so harnesses without bash hook support can still benefit.
//
// Transport: stdio (the MCP default; what every IDE expects).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/microrag"
	"github.com/sunholo-data/ailang/internal/version"
)

const (
	contextToolName = "microrag_context_for_file"
	lintToolName    = "microrag_lint_builtin"
)

type contextToolArgs struct {
	ToolName string `json:"tool_name"`
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
}

type lintToolArgs struct {
	FilePath string `json:"file_path"`
	Code     string `json:"code"`
}

func main() {
	// Logs MUST go to stderr — stdout is the MCP wire.
	log.SetOutput(os.Stderr)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "ailang-microrag",
		Version: version.Version,
	}, nil)

	registerContextTool(srv)
	registerLintTool(srv)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("ailang-microrag-mcp %s starting on stdio", version.Version)
	if err := srv.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcp run failed: %v", err)
	}
}

func registerContextTool(srv *mcp.Server) {
	tool := &mcp.Tool{
		Name:        contextToolName,
		Description: "Resolve just-in-time AILANG knowledge for a tool call. Returns at most one ≤150-token pointer snippet drawn from the active μRAG corpus, with token-window dedup against this MCP session.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"tool_name": {Type: "string", Description: "Edit | Write | Read | MultiEdit"},
				"file_path": {Type: "string", Description: "Absolute or repo-relative path being touched"},
				"content":   {Type: "string", Description: "Optional file content / diff body for retrieval query"},
			},
			Required: []string{"file_path"},
		},
	}
	srv.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args contextToolArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if args.FilePath == "" {
			return errorResult("file_path is required"), nil
		}

		if !microrag.EnabledFromEnv() {
			return jsonResult(&microrag.ContextResult{State: "disabled", Reason: "env_disabled"})
		}

		cfg, err := microrag.LoadConfig("")
		if err != nil {
			return jsonResult(&microrag.ContextResult{State: "on", Reason: fmt.Sprintf("config_error: %v", err)})
		}

		eng := &microrag.Engine{
			Cfg:        cfg,
			Searcher:   &microrag.CLISearcher{},
			SessionDir: microrag.DefaultSessionDir(),
		}
		res, err := eng.Context(microrag.Request{
			ToolName: args.ToolName,
			FilePath: args.FilePath,
			Content:  args.Content,
		})
		if err != nil {
			return jsonResult(&microrag.ContextResult{State: "on", Reason: fmt.Sprintf("engine_error: %v", err)})
		}
		return jsonResult(res)
	})
}

func registerLintTool(srv *mcp.Server) {
	tool := &mcp.Tool{
		Name:        lintToolName,
		Description: "Scan a code snippet for first-use AILANG builtins (in this MCP session) and return ≤2 short signature nudges. Returns nothing for already-seen names.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"file_path": {Type: "string", Description: "Optional file path tag for telemetry"},
				"code":      {Type: "string", Description: "Code body to scan (truncated to ~8KB)"},
			},
			Required: []string{"code"},
		},
	}
	srv.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args lintToolArgs
		if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
			return errorResult(fmt.Sprintf("invalid arguments: %v", err)), nil
		}
		if args.Code == "" {
			return jsonResult(&microrag.LintResult{State: "on", Reason: "empty_code"})
		}
		if !microrag.EnabledFromEnv() {
			return jsonResult(&microrag.LintResult{State: "disabled", Reason: "env_disabled"})
		}

		linter := &microrag.Linter{
			Resolver:   &microrag.CLIBuiltinResolver{},
			SessionDir: microrag.DefaultSessionDir(),
		}
		res, err := linter.Lint(microrag.LintRequest{FilePath: args.FilePath, Code: args.Code})
		if err != nil {
			return jsonResult(&microrag.LintResult{State: "on", Reason: fmt.Sprintf("lint_error: %v", err)})
		}
		return jsonResult(res)
	})
}

func jsonResult(payload any) (*mcp.CallToolResult, error) {
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return errorResult(fmt.Sprintf("marshal failed: %v", err)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(body)}},
	}, nil
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}
}
