package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/apiserver/schema"
	"github.com/sunholo-data/ailang/internal/embed"
)

// MCPServer wraps an apiserver.Server to expose its functions as MCP tools.
type MCPServer struct {
	server     *Server
	mcpServer  *mcp.Server
	feedbackRL *IPRateLimiter // nil = disabled; only applied to submit_feedback
}

// NewMCPServer creates an MCP server from an apiserver.Server.
// All loaded modules' exported functions are registered as MCP tools.
//
// The submit_feedback tool is rate-limited per-client-IP via env vars
// (AILANG_RATELIMIT_RPM, AILANG_RATELIMIT_BURST). Read-only tools are not
// throttled — they're idempotent and cacheable.
func NewMCPServer(srv *Server) *MCPServer {
	mcpSrv := mcp.NewServer(&mcp.Implementation{
		Name:    "ailang-api",
		Version: "0.8.1",
	}, nil)

	ms := &MCPServer{
		server:     srv,
		mcpServer:  mcpSrv,
		feedbackRL: NewIPRateLimiter(feedbackRateLimitRPM(), feedbackRateLimitBurst()),
	}

	ms.registerTools()
	ms.registerResources()
	ms.registerFeedbackTool()

	return ms
}

// registerTools registers each exported function as an MCP tool.
// Respects isExposed() filtering (--routes-only, @noexpose) consistent with
// HTTP handler, OpenAPI spec, and A2A agent card.
//
// Tool name generation is layered:
//  1. @mcp_name("name") author override (validated; invalid names are a hard error).
//  2. Bare function name when globally unique among exposed exports.
//  3. Sanitized "<lastSegment>_<funcName>" fallback for collisions.
//  4. Truncated to 64 chars with deterministic hash suffix if needed.
//
// All names are validated against the strict MCP regex
// `^[a-zA-Z0-9_-]{1,64}$` (Claude Desktop compatible).
func (ms *MCPServer) registerTools() {
	rawModules := ms.server.GetModules()
	// Re-key by RelPath projection (info.Path) so tool name generation
	// and engine dispatch use the URL-shaped module path. The map from
	// GetModules() is keyed by PhysicalPath (the s.modules identity)
	// which is unsuitable for both lastMeaningfulSegment heuristics and
	// engine.Call.
	modules := make(map[string]*ModuleInfo, len(rawModules))
	for _, info := range rawModules {
		if info != nil {
			modules[info.Path] = info
		}
	}
	// Phase 1: dedup by name+type across modules (handles package-loaded duplicates).
	type toolCandidate struct {
		modPath string
		export  ExportInfo
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
			candidate := toolCandidate{modPath, export}

			if existing, ok := best[dedupKey]; ok {
				// Prefer: (1) has @mcp_name override, (2) has doc comment,
				// (3) shorter module path (more likely to be the local file).
				existHasOverride := existing.export.MCPName != ""
				newHasOverride := export.MCPName != ""
				if newHasOverride && !existHasOverride {
					best[dedupKey] = candidate
				} else if newHasOverride == existHasOverride {
					existHasDoc := existing.export.DocComment != ""
					newHasDoc := export.DocComment != ""
					if newHasDoc && !existHasDoc {
						best[dedupKey] = candidate
					} else if newHasDoc == existHasDoc && len(modPath) < len(existing.modPath) {
						best[dedupKey] = candidate
					}
				}
			} else {
				best[dedupKey] = candidate
			}
		}
	}

	// Phase 2: count function-name occurrences across the dedup'd candidate set
	// to decide which functions can use the bare name.
	funcNameCount := make(map[string]int, len(best))
	for _, c := range best {
		funcNameCount[c.export.Name]++
	}

	// Phase 3: register tools with MCP-compliant names.
	usedNames := make(map[string]bool, len(best)) // catch any residual collisions
	for _, c := range best {
		export := c.export

		// Resolve the tool name.
		var toolName string
		if export.MCPName != "" {
			if err := validateMCPName(export.MCPName); err != nil {
				// Hard failure: author-supplied names that violate the regex
				// are a configuration bug — surface immediately.
				log.Printf("  ERROR: skipping MCP tool registration for %s/%s: %v", c.modPath, export.Name, err)
				continue
			}
			toolName = export.MCPName
		} else {
			preferBare := funcNameCount[export.Name] == 1
			toolName = mcpToolName(c.modPath, export.Name, "", preferBare)
		}

		// Defensive check: regex compliance for everything we emit.
		if err := validateMCPName(toolName); err != nil {
			log.Printf("  ERROR: generated MCP tool name failed validation for %s/%s: %v", c.modPath, export.Name, err)
			continue
		}

		// Residual collision: two different (modPath, funcName) pairs produced
		// the same final name. Append a deterministic hash suffix to disambiguate.
		if usedNames[toolName] {
			toolName = truncateWithHash(toolName+"_x", c.modPath, export.Name)
			// Loop until unique (extremely unlikely to iterate more than once).
			for usedNames[toolName] {
				toolName = truncateWithHash(toolName+"_x", c.modPath+"x", export.Name)
			}
		}
		usedNames[toolName] = true

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
//
// Stateless: true disables Mcp-Session-Id validation. Every request gets a
// fresh temporary session, which means clients holding a stale session ID
// (e.g. after a Cloud Run revision rolls) no longer get "session not found"
// 4xx — they just transparently re-handshake. AILANG's MCP tools are all
// read-only lookups (docs_search, stdlib_modules, benchmark_run, ...), so we
// don't need server→client requests, which is the only feature stateless
// mode disables.
func (ms *MCPServer) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return ms.mcpServer },
		&mcp.StreamableHTTPOptions{Stateless: true},
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
