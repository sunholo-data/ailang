package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sunholo-data/ailang/internal/embed"
	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// MCP Apps serving (see mcp_ui.go for the spec quotes).

// publicBaseHeader carries the request's public origin from the HTTP layer
// to MCP handlers, which see the request headers but not r.Host. The HTTP
// wrapper always overwrites it, so a client cannot choose it, and _headers
// bindings never see it.
const publicBaseHeader = "X-Ailang-Mcp-Public-Base"

// withPublicBase stamps publicBaseHeader on every MCP HTTP request.
func withPublicBase(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set(publicBaseHeader, protocol.PublicBaseURL(r))
		next.ServeHTTP(w, r)
	})
}

// uiResource is one @mcp_ui_resource served on this MCP surface.
type uiResource struct {
	modPath string
	export  ExportInfo
}

// registerUIResources adds every @mcp_ui_resource as an MCP resource.
func (ms *MCPServer) registerUIResources() {
	modules := ms.server.GetModules()
	var found []uiResource
	for _, info := range modules {
		if info == nil {
			continue
		}
		for _, e := range info.Exports {
			if e.UIResource != "" {
				found = append(found, uiResource{modPath: info.Path, export: e})
			}
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].export.UIResource < found[j].export.UIResource })
	for _, r := range found {
		if ms.uiResources == nil {
			ms.uiResources = map[string]uiResource{}
		}
		ms.uiResources[r.export.UIResource] = r
		ms.mcpServer.AddResource(&mcp.Resource{
			URI:         r.export.UIResource,
			Name:        r.export.Name,
			Title:       r.export.MCPTitle,
			Description: r.export.DocComment,
			MIMEType:    uiResourceMIME,
		}, ms.readUIResource(r))
	}
}

// uiResourceMeta is the resource's _meta: {"ui": {"csp": {"connectDomains":
// [...], "resourceDomains": []}}}, with "self" resolved from the request.
func uiResourceMeta(e ExportInfo, header http.Header) (mcp.Meta, error) {
	domains := make([]string, 0, len(e.UIConnect))
	for _, d := range e.UIConnect {
		if d == uiSelf {
			base := header.Get(publicBaseHeader)
			if base == "" {
				return nil, fmt.Errorf("%s: connect domain %q needs an HTTP request to resolve (not available on stdio); name the origin instead", e.UIResource, uiSelf)
			}
			d = base
		}
		domains = append(domains, d)
	}
	return mcp.Meta{"ui": map[string]any{
		"csp": map[string]any{"connectDomains": domains, "resourceDomains": []string{}},
	}}, nil
}

func requestHeader(req mcp.Request) http.Header {
	if req == nil {
		return nil
	}
	if extra := req.GetExtra(); extra != nil {
		return extra.Header
	}
	return nil
}

// readUIResource calls the HTML function and returns it as the widget.
func (ms *MCPServer) readUIResource(r uiResource) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		meta, err := uiResourceMeta(r.export, requestHeader(req))
		if err != nil {
			return nil, err
		}
		args := make([]any, len(r.export.ParamNames)) // unit params bind nil
		res, err := ms.server.engine.CallPreserveFloats(r.modPath, r.export.Name, args...)
		if err != nil {
			return nil, fmt.Errorf("%s: %s failed: %v", r.export.UIResource, r.export.Name, err)
		}
		v, err := embed.ToGo(res)
		if err != nil {
			return nil, err
		}
		html, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("%s: %s returned %T, want string", r.export.UIResource, r.export.Name, v)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: r.export.UIResource, MIMEType: uiResourceMIME, Text: html, Meta: meta,
		}}}, nil
	}
}

// uiResourcesListMiddleware puts each UI resource's _meta.ui (with "self"
// resolved for this request) on its resources/list entry. Entries are copied:
// the SDK's registered resources are shared across requests. On stdio, an
// entry whose "self" cannot be resolved is listed without _meta; reading it
// fails with the reason.
func (ms *MCPServer) uiResourcesListMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		if err != nil || method != "resources/list" || len(ms.uiResources) == 0 {
			return res, err
		}
		lr, ok := res.(*mcp.ListResourcesResult)
		if !ok {
			return res, err
		}
		header := requestHeader(req)
		out := *lr
		out.Resources = make([]*mcp.Resource, len(lr.Resources))
		for i, rsc := range lr.Resources {
			out.Resources[i] = rsc
			ui, isUI := ms.uiResources[rsc.URI]
			if !isUI {
				continue
			}
			if meta, mErr := uiResourceMeta(ui.export, header); mErr == nil {
				cp := *rsc
				cp.Meta = meta
				out.Resources[i] = &cp
			}
		}
		return &out, nil
	}
}
