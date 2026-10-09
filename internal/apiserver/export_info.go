package apiserver

// ExportInfo describes a single exported function from an AILANG module.
type ExportInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`                   // human-readable type signature
	Pure        bool     `json:"pure"`                   // declared effect row is empty (set from the AST at load; see extractMCPToolMetaAnnotations)
	Arity       int      `json:"arity"`                  // number of parameters (-1 if not a function)
	ParamNames  []string `json:"param_names,omitempty"`  // parameter names in order (for named JSON binding)
	ParamTypes  []string `json:"param_types,omitempty"`  // parameter type strings in order (for zero-value padding)
	ParamZeros  []any    `json:"-"`                      // per param: zero of its declared record type, nil otherwise (param_zero.go)
	RouteMethod string   `json:"route_method,omitempty"` // custom HTTP method from @route annotation
	RoutePath   string   `json:"route_path,omitempty"`   // custom URL path from @route annotation
	IsRaw       bool     `json:"is_raw,omitempty"`       // @raw annotation: pass full HttpRequest record
	IsNowrap    bool     `json:"is_nowrap,omitempty"`    // @nowrap annotation: skip FunctionCallResponse envelope
	IsNoExpose  bool     `json:"is_no_expose,omitempty"` // @noexpose annotation: hide from HTTP endpoints
	IsNoMCP     bool     `json:"is_no_mcp,omitempty"`    // @nomcp annotation: hide from the MCP tool surface only (HTTP/OpenAPI/A2A unaffected)
	MCPName     string   `json:"mcp_name,omitempty"`     // @mcp_name annotation: explicit MCP tool name override
	MCPTitle    string   `json:"mcp_title,omitempty"`    // @mcp_title annotation: MCP display title
	MCPHints    []string `json:"mcp_hints,omitempty"`    // @mcp_hints annotation: MCP behaviour hints (validated at registration)
	HasMCPHints bool     `json:"-"`                      // @mcp_hints present (an empty list is still a declaration)
	MCPAuth     string   `json:"mcp_auth,omitempty"`     // @mcp_auth: "oauth2" gates the tool on the listed surface; "" / "noauth" = open
	MCPSecret   []string `json:"-"`                      // @mcp_secret: params dropped (zero-bound) on the listed surface
	IsAgentOnly bool     `json:"-"`                      // @mcp_agent_only: absent from the listed surface
	// @mcp_file: params that take an OpenAI file object (openai/fileParams).
	// Validated at load (extractMCPFileAnnotations): a declared, non-secret
	// param typed as the four-string file record.
	MCPFile []string `json:"-"`
	// MCP Apps (M-MCP-FILE-HANDOFF F1c, extractMCPUIAnnotations):
	// @mcp_ui_resource: this function's HTML is the widget resource UIResource,
	// with CSP connectDomains UIConnect ("self" = the server's public URL).
	UIResource string   `json:"-"`
	UIConnect  []string `json:"-"`
	MCPUI      string   `json:"-"` // @mcp_ui: the tool renders this ui:// resource
	IsAppOnly  bool     `json:"-"` // @mcp_app_only: visibility ["app"], hidden from the model
	// @mcp_token_verifier: the server's one Bearer-token verifier. Never a
	// tool (IsNoMCP) and, without @route, never an HTTP endpoint (IsNoExpose):
	// exposed, it would be a token-guessing oracle.
	IsTokenVerifier      bool     `json:"-"`
	VerifierSigOK        bool     `json:"-"`                     // declared signature is exactly (string) -> bool
	Optional             []string `json:"optional,omitempty"`    // @optional annotation: params not required on MCP (absent/null → zero value)
	DocComment           string   `json:"doc_comment,omitempty"` // doc comment (-- lines) preceding the function
	IsWS                 bool     `json:"is_ws,omitempty"`       // @route("WS", ...): a WebSocket route, off every HTTP/MCP/A2A surface
	Effects              []string `json:"-"`                     // declared effect row (WS registration check)
	WSReq                []string `json:"-"`                     // WS routes: the declared req record fields, sorted
	ResponseHeadersIssue string   `json:"-"`                     // incompatible declared @route response headers
	WSReqIssue           string   `json:"-"`                     // WS routes: why the declared req record is refused ("" = accepted)
}
