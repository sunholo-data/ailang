package apiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/apiserver/schema"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// MCPServer wraps an apiserver.Server to expose its functions as MCP tools.
type MCPServer struct {
	server     *Server
	mcpServer  *mcp.Server
	feedbackRL *IPRateLimiter // nil = disabled; only applied to submit_feedback
	// verifier is the @mcp_token_verifier function gated tools are checked
	// with (nil when there is none). M-SERVEAPI-DIRECTORY-READY.
	verifier *tokenVerifier
	// listed marks the directory projection served at /mcp/connect/: no
	// @mcp_agent_only tools, and @mcp_secret params neither advertised nor
	// accepted. The agent surface (/mcp/) has listed == false.
	listed bool
}

func mcpError(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
		IsError: true,
	}
}

// NewMCPServer creates an MCP server from an apiserver.Server.
// All loaded modules' exported functions are registered as MCP tools.
//
// Unless suppressed by Config.NoFeedbackTool, the submit_feedback tool is
// registered and rate-limited per-client-IP via env vars
// (AILANG_RATELIMIT_RPM, AILANG_RATELIMIT_BURST). Read-only tools are not
// throttled — they're idempotent and cacheable.
func NewMCPServer(srv *Server) *MCPServer {
	return newMCPServer(srv, false)
}

// NewListedMCPServer builds the directory projection of the same tools
// (M-SERVEAPI-DIRECTORY-READY D3), mounted at /mcp/connect/.
func NewListedMCPServer(srv *Server) *MCPServer {
	return newMCPServer(srv, true)
}

func newMCPServer(srv *Server, listed bool) *MCPServer {
	mcpSrv := mcp.NewServer(&mcp.Implementation{
		Name:    "ailang-api",
		Version: "0.8.1",
	}, &mcp.ServerOptions{
		// Tools and resources are fixed at boot (they are the loaded modules'
		// exports), so never advertise listChanged. Left to the SDK default the
		// capability is inferred as listChanged:true, and every MCP 2025-11-25
		// client (Claude Code 2.1.27x, the go-sdk client) then opens a
		// subscriptions/listen stream that the server holds open until the
		// platform kills it. On Cloud Run that is a permanently in-flight request
		// per connected session: the instance can never go idle, the client
		// re-handshakes every 300s, and a service with zero real traffic bills
		// 86,400 instance-seconds a day (docparse prod, measured 2026-09-21).
		Capabilities: &mcp.ServerCapabilities{
			Logging:   &mcp.LoggingCapabilities{},
			Tools:     &mcp.ToolCapabilities{},
			Resources: &mcp.ResourceCapabilities{},
		},
	})

	ms := &MCPServer{
		server:     srv,
		mcpServer:  mcpSrv,
		feedbackRL: NewIPRateLimiter(feedbackRateLimitRPM(), feedbackRateLimitBurst()),
		listed:     listed,
	}

	ms.registerTools()
	ms.registerResources()
	if !srv.noFeedbackTool {
		if srv.routesOnly {
			log.Printf("MCP tool submit_feedback remains enabled with --routes-only; use --no-feedback-tool to suppress it")
		}
		ms.registerFeedbackTool()
	}

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
	verifier, verifierErr := resolveTokenVerifier(modules)
	if verifierErr != nil {
		log.Printf("  ERROR: %v", verifierErr)
	}
	ms.verifier = verifier

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
			if !loadedExportMember(ms.server.routesOnly, export) {
				continue
			}
			if export.IsNoMCP {
				continue // @nomcp: served over HTTP/OpenAPI/A2A but absent from MCP
			}
			if ms.listed && export.IsAgentOnly {
				continue // @mcp_agent_only: on /mcp/, not on the listed surface
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
	var unhinted []string                         // effectful tools with no @mcp_hints (one summary warning)
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

		if err := validateOptionalParams(export); err != nil {
			// Same posture as an invalid @mcp_name: an author bug, surfaced
			// at registration rather than as a crash on the first call.
			log.Printf("  ERROR: skipping MCP tool registration for %s/%s: %v", c.modPath, export.Name, err)
			continue
		}

		if err := validateMCPAuth(export, ms.server.oauthIssuer, verifier, verifierErr); err != nil {
			log.Printf("  ERROR: skipping MCP tool registration for %s/%s: %v", c.modPath, export.Name, err)
			continue
		}

		hints, err := protocol.ResolveToolHints(export.MCPHints, export.HasMCPHints, export.Pure)
		if err != nil {
			// Same posture as an invalid @mcp_name: an author bug, surfaced at
			// registration — a tool advertising the wrong hints is worse than
			// a missing tool.
			log.Printf("  ERROR: skipping MCP tool registration for %s/%s: @mcp_hints: %v", c.modPath, export.Name, err)
			continue
		}
		if hints == nil {
			unhinted = append(unhinted, toolName)
		}

		tool := &mcp.Tool{
			Name:        toolName,
			Title:       export.MCPTitle,
			Description: desc,
			InputSchema: ms.inputSchemaFor(export),
			Annotations: sdkToolAnnotations(hints, export.MCPTitle),
		}

		ms.mcpServer.AddTool(tool, ms.makeToolHandler(c.modPath, export))
	}
	if len(unhinted) > 0 {
		// Not an error — the tool still works — but MCP directories (Anthropic,
		// OpenAI) refuse tools that declare neither readOnlyHint nor
		// destructiveHint, and serve-api will not guess them for effectful code.
		log.Printf("  WARN: %d effectful MCP tool(s) declare no @mcp_hints and are advertised without annotations: %s",
			len(unhinted), strings.Join(unhinted, ", "))
	}
}

// sdkToolAnnotations maps the protocol-level hints onto the go-sdk type. The
// title rides along in annotations too, for clients that predate Tool.title.
func sdkToolAnnotations(h *protocol.ToolAnnotations, title string) *mcp.ToolAnnotations {
	if h == nil {
		return nil
	}
	return &mcp.ToolAnnotations{
		Title:           title,
		ReadOnlyHint:    h.ReadOnlyHint,
		DestructiveHint: h.DestructiveHint,
		IdempotentHint:  h.IdempotentHint,
		OpenWorldHint:   h.OpenWorldHint,
	}
}

// headersParam is the reserved parameter name that binds the HTTP request
// headers as a Json object — the same contract as the REST @route path
// (routes_dispatch.go). It is never advertised in a tool's inputSchema and
// never taken from the client's arguments.
const headersParam = "_headers"

// isUnitParamType reports whether a param carries no information: `unit`, or
// the `()` the parser desugars `func f()` into (one param named "_"). Such a
// param is never advertised and binds to nil, which the engine reads as Unit.
// Before this, every zero-arg export advertised a required string "_" and a
// tools/call with {} was rejected as "missing required parameter(s): _".
func isUnitParamType(t string) bool { return t == "unit" || t == "()" }

func paramTypeAt(export ExportInfo, i int) string {
	if i < len(export.ParamTypes) {
		return export.ParamTypes[i]
	}
	return ""
}

// validateOptionalParams checks an export's @optional names against its
// signature: each must be a declared param, must not be the reserved
// _headers param, and must have a type with a zero value to bind when absent.
func validateOptionalParams(export ExportInfo) error {
	for _, name := range export.Optional {
		idx := -1
		for i, p := range export.ParamNames {
			if p == name {
				idx = i
				break
			}
		}
		switch {
		case idx < 0:
			return fmt.Errorf("@optional(%q): no such parameter (params: %s)", name, strings.Join(export.ParamNames, ", "))
		case name == headersParam:
			return fmt.Errorf("@optional(%q): _headers is bound from the request, not the client", name)
		case idx >= len(export.ParamTypes) || zeroValueForType(export.ParamTypes[idx]) == nil:
			typ := "unknown"
			if idx < len(export.ParamTypes) {
				typ = export.ParamTypes[idx]
			}
			return fmt.Errorf("@optional(%q): type %s has no zero value; supported: string, int, float, bool, list, array, record", name, typ)
		}
	}
	return nil
}

// makeToolHandler creates a ToolHandler that calls the AILANG function.
// Accepts both named parameters ({"filepath": "x"}) and legacy positional
// format ({"args": ["x"]}) for backward compatibility.
//
// A declared _headers param binds from the tools/call HTTP request headers
// (nil on stdio → empty object). Anything a client sends under that key is
// overwritten, so a caller cannot forge headers through the arguments.
func (ms *MCPServer) makeToolHandler(modulePath string, export ExportInfo) mcp.ToolHandler {
	funcName := export.Name
	paramNames := export.ParamNames
	optional := make(map[string]bool, len(export.Optional))
	for _, name := range export.Optional {
		optional[name] = true
	}
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args []any

		var argMap map[string]any
		if len(req.Params.Arguments) > 0 {
			if err := json.Unmarshal(req.Params.Arguments, &argMap); err == nil {
				// Try legacy "args" array first for backward compat.
				if argsRaw, ok := argMap["args"]; ok {
					if argsSlice, ok := argsRaw.([]any); ok {
						args = argsSlice
					}
				}
			}
		}

		// If no "args" key and we have param names, resolve named params.
		//
		// M-MCP-UNIT-PARAM-BINDING: a declared param the client omits
		// (absent key) or sends as JSON null must NOT bind to nil — the
		// engine converts nil to Unit and the AILANG function then
		// crashes deep in stdlib (e.g. _str_len: expected String, got
		// Unit) before any guard can run. Type-agnostic: an omitted int
		// param would crash the same way. Reject with a structured error
		// naming the missing params in declaration order (deterministic).
		// @optional params bind to their type's zero value instead;
		// _headers is bound below, never from the client.
		if len(args) == 0 && len(paramNames) > 0 {
			var missing []string
			args = make([]any, len(paramNames))
			for i, name := range paramNames {
				if name == headersParam || isUnitParamType(paramTypeAt(export, i)) {
					continue // bound below / nil = Unit
				}
				v, present := argMap[name]
				if !present || v == nil {
					if optional[name] {
						args[i] = zeroValueForType(export.ParamTypes[i])
						continue
					}
					missing = append(missing, name)
					continue
				}
				args[i] = v
			}
			if len(missing) > 0 {
				return mcpError(fmt.Sprintf(
					"missing required parameter(s): %s", strings.Join(missing, ", "),
				)), nil
			}
		}

		// Listed surface: a secret is never taken from the client, named or
		// positional. It binds the zero value (validateSecretParams already
		// required @optional, so one exists).
		if ms.listed {
			for i, name := range paramNames {
				if i < len(args) && isSecretParam(export, name) {
					args[i] = zeroValueForType(export.ParamTypes[i])
				}
			}
		}

		for i, name := range paramNames {
			if name == headersParam && i < len(args) {
				var h http.Header
				if extra := req.GetExtra(); extra != nil {
					h = extra.Header
				}
				args[i] = stringMapToJObject(h)
			}
		}

		// Call the AILANG function (preserve floats — JSON has no int/float distinction).
		result, callErr := ms.server.engine.CallPreserveFloats(modulePath, funcName, args...)
		if callErr != nil && !isCleanExit(callErr) {
			return mcpError(fmt.Sprintf("function call failed: %v", callErr)), nil
		}

		// Convert result to Go value.
		var goResult any
		if callErr == nil {
			var err error
			goResult, err = embed.ToGo(result)
			if err != nil {
				return mcpError(fmt.Sprintf("result conversion failed: %v", err)), nil
			}
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
		optional := make(map[string]bool, len(export.Optional))
		for _, name := range export.Optional {
			optional[name] = true
		}
		for i, name := range export.ParamNames {
			if name == headersParam || isUnitParamType(paramTypeAt(export, i)) {
				continue // bound by the server, never supplied by the client
			}
			prop := map[string]any{
				"type": "string", // default
			}
			if i < len(export.ParamTypes) {
				prop["type"] = ailangTypeToJSONSchema(export.ParamTypes[i])
			}
			props[name] = prop
			if !optional[name] {
				required = append(required, name)
			}
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
