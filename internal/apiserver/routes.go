package apiserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/petermattis/goid"
	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/eval"
)

// goroutineID returns the current goroutine ID for debug logging.
// Uses goid package (~3ns) instead of runtime.Stack (~500ns).
func goroutineID() int {
	return int(goid.Get())
}

// RouteEntry represents a custom route defined by a @route annotation.
type RouteEntry struct {
	Method     string   // "GET", "POST", etc.
	Path       string   // "/general/v0/general"
	Module     string   // module path
	Function   string   // function name
	IsRaw      bool     // @raw: pass full HttpRequest record instead of parsed args
	IsNowrap   bool     // @nowrap: skip FunctionCallResponse envelope, return raw JSON
	ParamNames []string // parameter names for named JSON binding
	ParamTypes []string // parameter type strings for zero-value padding
}

// extractParamInfo populates ExportInfo.ParamNames and ExportInfo.ParamTypes
// from the parsed AST for all exported functions. This enables named JSON
// parameter binding and zero-value padding for missing parameters.
func extractParamInfo(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		if !fn.IsExport {
			continue
		}
		names := make([]string, len(fn.Params))
		types := make([]string, len(fn.Params))
		for i, p := range fn.Params {
			names[i] = p.Name
			types[i] = paramTypeToString(p.Type)
		}
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].ParamNames = names
				modInfo.Exports[i].ParamTypes = types
				break
			}
		}
	}
}

// paramTypeToString converts an ast.Type to a simple type name string
// used for zero-value padding of missing parameters.
func paramTypeToString(t ast.Type) string {
	if t == nil {
		return "unknown"
	}
	switch v := t.(type) {
	case *ast.SimpleType:
		return v.Name
	case *ast.ListType:
		return "list"
	case *ast.ArrayType:
		return "array"
	case *ast.RecordType:
		return "record"
	case *ast.LabelledType:
		// IFC label/refinement — delegate to base type for name extraction.
		return v.Base.String()
	default:
		return "unknown"
	}
}

// extractRouteAnnotations populates ExportInfo.RouteMethod/RoutePath from
// @route annotations found in the parsed AST.
func extractRouteAnnotations(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		routeAnn := fn.GetAnnotation("route")
		if routeAnn == nil || len(routeAnn.Args) < 2 {
			continue
		}
		methodLit, ok1 := routeAnn.Args[0].(*ast.Literal)
		pathLit, ok2 := routeAnn.Args[1].(*ast.Literal)
		if !ok1 || !ok2 || methodLit.Kind != ast.StringLit || pathLit.Kind != ast.StringLit {
			continue
		}
		method := methodLit.Value.(string)
		path := pathLit.Value.(string)
		isRaw := fn.GetAnnotation("raw") != nil
		isNowrap := fn.GetAnnotation("nowrap") != nil

		// Find matching export and set route info
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].RouteMethod = method
				modInfo.Exports[i].RoutePath = path
				modInfo.Exports[i].IsRaw = isRaw
				modInfo.Exports[i].IsNowrap = isNowrap
				modInfo.Exports[i].IsNoExpose = false // @route overrides @noexpose
				flags := ""
				if isRaw {
					flags += " raw"
				}
				if isNowrap {
					flags += " nowrap"
				}
				if flags != "" {
					log.Printf("    Route: %s %s -> %s (%s)", method, path, fn.Name, strings.TrimSpace(flags))
				} else {
					log.Printf("    Route: %s %s -> %s", method, path, fn.Name)
				}
				break
			}
		}
	}
}

// extractDocComments reads the source file and extracts -- comment lines
// immediately preceding each exported function declaration. The collected
// comment text (with leading "-- " stripped) is stored in ExportInfo.DocComment.
func extractDocComments(modInfo *ModuleInfo, file *ast.File, filePath string) {
	src, err := os.ReadFile(filePath)
	if err != nil {
		return // best-effort: no source = no doc comments
	}
	lines := strings.Split(string(src), "\n")

	for _, fn := range file.Funcs {
		if !fn.IsExport {
			continue
		}
		// fn.Pos.Line is 1-based. Scan backwards from the line before the func.
		funcLine := fn.Pos.Line - 1 // convert to 0-based index
		if funcLine <= 0 || funcLine > len(lines) {
			continue
		}

		// Skip annotation lines (@route, @raw, etc.) above the func
		scanLine := funcLine - 1
		for scanLine >= 0 {
			trimmed := strings.TrimSpace(lines[scanLine])
			if strings.HasPrefix(trimmed, "@") {
				scanLine--
				continue
			}
			break
		}

		// Collect consecutive -- comment lines
		var commentLines []string
		for scanLine >= 0 {
			trimmed := strings.TrimSpace(lines[scanLine])
			if strings.HasPrefix(trimmed, "--") {
				// Strip "-- " or "--" prefix
				text := strings.TrimPrefix(trimmed, "-- ")
				if text == trimmed {
					text = strings.TrimPrefix(trimmed, "--")
				}
				commentLines = append([]string{text}, commentLines...)
				scanLine--
			} else {
				break
			}
		}

		if len(commentLines) == 0 {
			continue
		}

		doc := strings.Join(commentLines, "\n")
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].DocComment = doc
				break
			}
		}
	}
}

// extractMCPNameAnnotations populates ExportInfo.MCPName from @mcp_name("name")
// annotations found in the parsed AST. Author-supplied names override AILANG's
// auto-generated MCP tool names.
func extractMCPNameAnnotations(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		ann := fn.GetAnnotation("mcp_name")
		if ann == nil || len(ann.Args) < 1 {
			continue
		}
		nameLit, ok := ann.Args[0].(*ast.Literal)
		if !ok || nameLit.Kind != ast.StringLit {
			continue
		}
		mcpName := nameLit.Value.(string)
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name {
				modInfo.Exports[i].MCPName = mcpName
				break
			}
		}
	}
}

// extractNoExposeAnnotations marks exported functions with @noexpose as hidden
// from HTTP endpoints. Functions with @route are never hidden (route overrides noexpose).
func extractNoExposeAnnotations(modInfo *ModuleInfo, file *ast.File) {
	for _, fn := range file.Funcs {
		if fn.GetAnnotation("noexpose") == nil {
			continue
		}
		for i := range modInfo.Exports {
			if modInfo.Exports[i].Name == fn.Name && modInfo.Exports[i].RoutePath == "" {
				modInfo.Exports[i].IsNoExpose = true
				break
			}
		}
	}
}

// isExposed returns true if the export should be visible as an HTTP endpoint,
// considering the server's routesOnly setting and the export's annotations.
func (s *Server) isExposed(exp ExportInfo) bool {
	if exp.IsNoExpose {
		return false
	}
	if s.routesOnly && exp.RoutePath == "" {
		return false
	}
	return true
}

// isValidJSONObjectOrArray checks if a string is a valid JSON object ({...}) or array ([...]).
// Only these compound types are unwrapped — bare strings, numbers, and booleans are NOT.
func isValidJSONObjectOrArray(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	if s[0] != '{' && s[0] != '[' {
		return false
	}
	return json.Valid([]byte(s))
}

// findRouteByPath finds a custom route matching the given URL path.
// Used as a fallback in the catch-all handler for package module routes.
func (s *Server) findRouteByPath(urlPath string) *RouteEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.RoutePath == urlPath {
				return &RouteEntry{
					Method:     exp.RouteMethod,
					Path:       exp.RoutePath,
					Module:     mod.Path,
					Function:   exp.Name,
					IsRaw:      exp.IsRaw,
					IsNowrap:   exp.IsNowrap,
					ParamNames: exp.ParamNames,
					ParamTypes: exp.ParamTypes,
				}
			}
		}
	}
	return nil
}

// getCustomRoutes returns all custom routes from loaded modules.
func (s *Server) getCustomRoutes() []RouteEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var routes []RouteEntry
	for _, mod := range s.modules {
		for _, exp := range mod.Exports {
			if exp.RoutePath != "" {
				routes = append(routes, RouteEntry{
					Method:     exp.RouteMethod,
					Path:       exp.RoutePath,
					Module:     mod.Path,
					Function:   exp.Name,
					IsRaw:      exp.IsRaw,
					IsNowrap:   exp.IsNowrap,
					ParamNames: exp.ParamNames,
					ParamTypes: exp.ParamTypes,
				})
			}
		}
	}
	return routes
}

// registerCustomRoutes adds custom route handlers to the mux.
// Custom routes are registered BEFORE the catch-all /api/ handler.
// builtinPaths contains paths already registered by buildRoutes(); any @route
// annotation that collides with a built-in path is skipped with a warning
// (Go 1.22+ ServeMux panics on duplicate patterns).
func (s *Server) registerCustomRoutes(mux *http.ServeMux, builtinPaths map[string]bool) {
	routes := s.getCustomRoutes()
	registered := map[string]bool{} // track registered paths to avoid Go 1.22+ duplicate panics
	for _, route := range routes {
		if builtinPaths[route.Path] {
			log.Printf("  WARNING: @route %s %s collides with built-in route, skipping (use built-in handler instead)", route.Method, route.Path)
			continue
		}
		if registered[route.Path] {
			log.Printf("  WARNING: @route %s %s already registered, skipping duplicate from %s", route.Method, route.Path, route.Module)
			continue
		}
		r := route // capture for closure
		handler := func(w http.ResponseWriter, req *http.Request) {
			// Enforce HTTP method
			if req.Method != r.Method && req.Method != "OPTIONS" {
				writeRouterError(w, http.StatusMethodNotAllowed,
					ErrCodeMethodNotAllowed,
					fmt.Sprintf("this endpoint only accepts %s requests", r.Method),
					fmt.Sprintf("Use %s instead of %s", r.Method, req.Method),
					nil)
				return
			}
			s.callFunction(w, req, r.Module, r.Function, callOpts{Raw: r.IsRaw, Nowrap: r.IsNowrap, ParamNames: r.ParamNames, ParamTypes: r.ParamTypes})
		}
		mux.HandleFunc(r.Path, s.corsWrap(s.authMiddleware(handler)))
		registered[route.Path] = true
		log.Printf("  Custom route: %s %s -> %s/%s", r.Method, r.Path, r.Module, r.Function)
	}
}

// buildHttpRequestRecord constructs a map representing an HttpRequest record
// from an http.Request and its already-read body. Used by @raw routes.
// Headers and query are JObject (Json ADT) so handlers can use std/json.get()
// for dynamic key access (e.g., hyphenated header names like "Stripe-Signature").
func buildHttpRequestRecord(r *http.Request, body []byte) map[string]interface{} {
	return map[string]interface{}{
		"body":    string(body),
		"headers": stringMapToJObject(r.Header),
		"method":  r.Method,
		"path":    r.URL.Path,
		"query":   stringMapToJObject(r.URL.Query()),
	}
}

// stringMapToJObject converts an http.Header or url.Values (map[string][]string)
// to a JObject TaggedValue: JObject(List[{key: string, value: JString(string)}]).
func stringMapToJObject(m map[string][]string) *eval.TaggedValue {
	kvPairs := make([]eval.Value, 0, len(m))
	for k, v := range m {
		if len(v) > 0 {
			kvPairs = append(kvPairs, &eval.RecordValue{
				Fields: map[string]eval.Value{
					"key": &eval.StringValue{Value: k},
					"value": &eval.TaggedValue{
						ModulePath: "std/json", TypeName: "Json", CtorName: "JString",
						Fields: []eval.Value{&eval.StringValue{Value: v[0]}},
					},
				},
			})
		}
	}
	return &eval.TaggedValue{
		ModulePath: "std/json", TypeName: "Json", CtorName: "JObject",
		Fields: []eval.Value{&eval.ListValue{Elements: kvPairs}},
	}
}
