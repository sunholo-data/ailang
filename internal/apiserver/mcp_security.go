package apiserver

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// M-MCP-FILE-HANDOFF F1b (audit G10, sources O5): ChatGPT's mixed-auth
// metadata on the listed surface.
//
// OpenAI Apps SDK, Authentication (https://developers.openai.com/apps-sdk/build/auth,
// fetched 2026-10-07): securitySchemes is a top-level field of each tool
// definition — "Two scheme types are available today, and you can list more
// than one to express optional auth: noauth: The tool is callable
// anonymously; ChatGPT can run it immediately. oauth2: The tool needs an
// OAuth 2.0 access token; include the scopes you will request". Example:
// `securitySchemes: [{ type: "oauth2", scopes: ["docs.write"] }]`.
//
// OpenAI Apps SDK reference (https://developers.openai.com/plugins/reference):
// `_meta["securitySchemes"]` — "Back-compat mirror for clients that only read
// _meta." So the listed surface emits both, with identical values.
//
// serve-api has no per-tool scopes (a token the @mcp_token_verifier accepts is
// enough), so an oauth2 scheme carries no "scopes" key.
//
// The agent surface (/mcp/) is never gated by Bearer tokens, so it declares no
// schemes: tools there are byte-identical to before.

const securitySchemesKey = "securitySchemes"

// securitySchemesFor is the listed-surface scheme list for one export.
func securitySchemesFor(e ExportInfo) []map[string]any {
	if e.MCPAuth == mcpAuthOAuth2 {
		return []map[string]any{{"type": mcpAuthOAuth2}}
	}
	return []map[string]any{{"type": mcpAuthNoAuth}}
}

// toolMeta is the tool descriptor's _meta (nil when there is nothing to say,
// so unannotated tools on /mcp/ marshal exactly as before). It also records
// the listed tool's schemes for the tools/list rewrite.
func (ms *MCPServer) toolMeta(toolName string, e ExportInfo) mcp.Meta {
	var meta mcp.Meta
	if files := fileParamsMeta(e); files != nil {
		meta = mcp.Meta{fileParamsMetaKey: files}
	}
	if m := uiToolMeta(meta, e); m != nil {
		meta = m
	}
	if ms.listed {
		schemes := securitySchemesFor(e)
		if ms.schemes == nil {
			ms.schemes = map[string][]map[string]any{}
		}
		ms.schemes[toolName] = schemes
		if meta == nil {
			meta = mcp.Meta{}
		}
		meta[securitySchemesKey] = schemes
	}
	return meta
}

// securitySchemesMiddleware puts each listed tool's securitySchemes at the top
// level of its tools/list entry. The go-sdk Tool type has no such field, so
// the result is wrapped and re-marshalled with it added.
func (ms *MCPServer) securitySchemesMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		if err != nil || method != "tools/list" || len(ms.schemes) == 0 {
			return res, err
		}
		if lt, ok := res.(*mcp.ListToolsResult); ok {
			return &listedToolsResult{ListToolsResult: lt, schemes: ms.schemes}, nil
		}
		return res, err
	}
}

// listedToolsResult is a tools/list result whose tools carry a top-level
// securitySchemes. Embedding keeps it an mcp.Result.
type listedToolsResult struct {
	*mcp.ListToolsResult
	schemes map[string][]map[string]any
}

func (r *listedToolsResult) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(r.ListToolsResult)
	if err != nil {
		return nil, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return nil, err
	}
	var tools []map[string]json.RawMessage
	if err := json.Unmarshal(top["tools"], &tools); err != nil {
		return nil, err
	}
	for _, tl := range tools {
		var name string
		_ = json.Unmarshal(tl["name"], &name)
		if s, ok := r.schemes[name]; ok {
			sb, err := json.Marshal(s)
			if err != nil {
				return nil, err
			}
			tl[securitySchemesKey] = sb
		}
	}
	if top["tools"], err = json.Marshal(tools); err != nil {
		return nil, err
	}
	return json.Marshal(top)
}
