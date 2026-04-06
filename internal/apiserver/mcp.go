package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
// Respects isExposed() filtering (--routes-only, @noexpose) consistent with
// HTTP handler, OpenAPI spec, and A2A agent card.
func (ms *MCPServer) registerTools() {
	modules := ms.server.GetModules()
	// Two-pass dedup: collect candidates first, then register.
	// When a module is loaded both directly and as a package dependency,
	// the same function appears under two module paths. We dedup by
	// name+type and deterministically prefer the entry with a doc comment
	// (or shorter tool name as tiebreaker).
	type toolCandidate struct {
		modPath  string
		export   ExportInfo
		toolName string
	}
	best := make(map[string]toolCandidate) // dedupKey -> best candidate

	for modPath, modInfo := range modules {
		for _, export := range modInfo.Exports {
			if export.Arity < 0 {
				continue
			}
			if !ms.server.isExposed(export) {
				continue
			}

			dedupKey := export.Name + "|" + export.Type
			toolName := portableToolName(modPath, export.Name)
			candidate := toolCandidate{modPath, export, toolName}

			if existing, ok := best[dedupKey]; ok {
				// Prefer: (1) has doc comment, (2) shorter tool name.
				existHasDoc := existing.export.DocComment != ""
				newHasDoc := export.DocComment != ""
				if newHasDoc && !existHasDoc {
					best[dedupKey] = candidate
				} else if newHasDoc == existHasDoc && len(toolName) < len(existing.toolName) {
					best[dedupKey] = candidate
				}
			} else {
				best[dedupKey] = candidate
			}
		}
	}

	for _, c := range best {
		export := c.export
		toolName := c.toolName

		desc := export.DocComment
		if desc == "" {
			desc = export.Name
			if export.Type != "" {
				desc = fmt.Sprintf("%s(%s)", export.Name, export.Type)
			}
			if export.Pure {
				desc += " [pure]"
			}
		}

		inputSchema := buildNamedInputSchema(export)

		capturedMod := c.modPath
		capturedFunc := export.Name

		tool := &mcp.Tool{
			Name:        toolName,
			Description: desc,
			InputSchema: inputSchema,
		}

		capturedParamNames := export.ParamNames
		ms.mcpServer.AddTool(tool, ms.makeToolHandler(capturedMod, capturedFunc, capturedParamNames))
	}
}

// makeToolHandler creates a ToolHandler that calls the AILANG function.
// Accepts both named parameters ({"filepath": "x"}) and legacy positional
// format ({"args": ["x"]}) for backward compatibility.
func (ms *MCPServer) makeToolHandler(modulePath, funcName string, paramNames []string) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args []any

		if len(req.Params.Arguments) > 0 {
			var argMap map[string]any
			if err := json.Unmarshal(req.Params.Arguments, &argMap); err == nil {
				// Try legacy "args" array first for backward compat.
				if argsRaw, ok := argMap["args"]; ok {
					if argsSlice, ok := argsRaw.([]any); ok {
						args = argsSlice
					}
				}
				// If no "args" key and we have param names, resolve named params.
				if len(args) == 0 && len(paramNames) > 0 {
					args = make([]any, len(paramNames))
					for i, name := range paramNames {
						args[i] = argMap[name]
					}
				}
			}
		}

		// Call the AILANG function (preserve floats — JSON has no int/float distinction).
		result, callErr := ms.server.engine.CallPreserveFloats(modulePath, funcName, args...)
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

// buildNamedInputSchema creates a JSON Schema with named parameters from ExportInfo.
// Uses ParamNames and ParamTypes when available; falls back to positional args array.
func buildNamedInputSchema(export ExportInfo) map[string]any {
	if export.Arity <= 0 {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// If we have named parameters, build a proper named schema.
	if len(export.ParamNames) > 0 {
		props := map[string]any{}
		required := make([]string, 0, len(export.ParamNames))
		for i, name := range export.ParamNames {
			prop := map[string]any{
				"type": "string", // default
			}
			if i < len(export.ParamTypes) {
				prop["type"] = ailangTypeToJSONSchema(export.ParamTypes[i])
			}
			props[name] = prop
			required = append(required, name)
		}
		return map[string]any{
			"type":       "object",
			"properties": props,
			"required":   required,
		}
	}

	// Fallback: positional args array (no param names available).
	fs := schema.FromTypeString(export.Type)
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

// ailangTypeToJSONSchema maps AILANG type strings to JSON Schema type strings.
func ailangTypeToJSONSchema(ailangType string) string {
	switch ailangType {
	case "string":
		return "string"
	case "int":
		return "integer"
	case "float":
		return "number"
	case "bool":
		return "boolean"
	case "Json", "record":
		return "object"
	case "list", "array":
		return "array"
	case "bytes":
		return "string" // base64 or file path
	default:
		return "string"
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
