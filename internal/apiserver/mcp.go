package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo/ailang/internal/apiserver/schema"
	"github.com/sunholo/ailang/internal/embed"
)

// MCPServer wraps an apiserver.Server to expose its functions as MCP tools.
type MCPServer struct {
	server    *Server
	mcpServer *mcp.Server
}

// NewMCPServer creates an MCP server from an apiserver.Server.
// All loaded modules' exported functions are registered as MCP tools.
func NewMCPServer(srv *Server) *MCPServer {
	mcpSrv := mcp.NewServer(&mcp.Implementation{
		Name:    "ailang-api",
		Version: "0.8.1",
	}, nil)

	ms := &MCPServer{
		server:    srv,
		mcpServer: mcpSrv,
	}

	ms.registerTools()
	ms.registerResources()

	return ms
}

// registerTools registers each exported function as an MCP tool.
func (ms *MCPServer) registerTools() {
	modules := ms.server.GetModules()

	for modPath, modInfo := range modules {
		for _, export := range modInfo.Exports {
			if export.Arity < 0 {
				continue // skip non-function exports
			}

			toolName := modPath + "." + export.Name
			// Replace path separators with dots for valid tool names.
			toolName = strings.ReplaceAll(toolName, "/", ".")

			desc := export.Name
			if export.Type != "" {
				desc = fmt.Sprintf("%s(%s)", export.Name, export.Type)
			}
			if export.Pure {
				desc += " [pure]"
			}

			fs := schema.FromTypeString(export.Type)
			inputSchema := buildMCPInputSchema(fs)

			// Capture loop variables for closure.
			capturedMod := modPath
			capturedFunc := export.Name

			tool := &mcp.Tool{
				Name:        toolName,
				Description: desc,
				InputSchema: inputSchema,
			}

			ms.mcpServer.AddTool(tool, ms.makeToolHandler(capturedMod, capturedFunc))
		}
	}
}

// makeToolHandler creates a ToolHandler that calls the AILANG function.
func (ms *MCPServer) makeToolHandler(modulePath, funcName string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract args from the request arguments (json.RawMessage).
		var args []any

		if len(req.Params.Arguments) > 0 {
			var argMap map[string]any
			if err := json.Unmarshal(req.Params.Arguments, &argMap); err == nil {
				if argsRaw, ok := argMap["args"]; ok {
					if argsSlice, ok := argsRaw.([]any); ok {
						args = argsSlice
					}
				}
			}
		}

		// Call the AILANG function.
		result, callErr := ms.server.engine.Call(modulePath, funcName, args...)
		if callErr != nil {
			return mcpError(fmt.Sprintf("function call failed: %v", callErr)), nil
		}

		// Convert result to Go value.
		goResult, err := embed.ToGo(result)
		if err != nil {
			return mcpError(fmt.Sprintf("result conversion failed: %v", err)), nil
		}

		// Marshal to JSON for text content.
		resultJSON, err := json.MarshalIndent(goResult, "", "  ")
		if err != nil {
			return mcpError(fmt.Sprintf("result serialization failed: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(resultJSON)},
			},
		}, nil
	}
}

// registerResources registers MCP resources for module introspection.
func (ms *MCPServer) registerResources() {
	ms.mcpServer.AddResource(&mcp.Resource{
		URI:         "ailang://meta/modules",
		Name:        "AILANG Modules",
		Description: "List of all loaded AILANG modules and their exports",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		modules := ms.server.GetModules()
		data, _ := json.MarshalIndent(modules, "", "  ")
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "ailang://meta/modules",
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	})
}

// RunStdio runs the MCP server on stdio transport (blocking).
func (ms *MCPServer) RunStdio(ctx context.Context) error {
	log.Println("Starting MCP server on stdio transport...")
	return ms.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

// HTTPHandler returns an HTTP handler for the MCP server using streamable HTTP transport.
func (ms *MCPServer) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return ms.mcpServer },
		nil,
	)
}

// buildMCPInputSchema creates a JSON Schema map for MCP tool input from a FunctionSchema.
// The MCP SDK accepts any JSON-serializable value for InputSchema.
func buildMCPInputSchema(fs *schema.FunctionSchema) map[string]any {
	if fs.Arity == 0 {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"args": map[string]any{
				"type":     "array",
				"items":    fs.Parameters,
				"minItems": fs.Arity,
				"maxItems": fs.Arity,
			},
		},
		"required": []string{"args"},
	}
}

// mcpError creates an MCP error result.
func mcpError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
		IsError: true,
	}
}
