package apiserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/apiserver/schema"
)

// handleOpenAPISpec generates and serves an OpenAPI 3.1 spec from loaded modules.
// GET /api/_meta/openapi.json
func (s *Server) handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}

	spec := s.buildOpenAPISpec()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(spec)
}

// handleSwaggerUI serves a Swagger UI page (like FastAPI's /docs).
// GET /api/_meta/docs
func (s *Server) handleSwaggerUI(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, swaggerUIHTML)
}

// handleReDoc serves a ReDoc page (like FastAPI's /redoc).
// GET /api/_meta/redoc
func (s *Server) handleReDoc(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "GET only"})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, redocHTML)
}

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>AILANG API - Swagger UI</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&family=Montserrat:wght@600;700;800&display=swap" rel="stylesheet">
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #0f1419; }
    .swagger-ui { font-family: Inter, -apple-system, BlinkMacSystemFont, sans-serif; }
    .swagger-ui .topbar { background-color: #0f1419; border-bottom: 2px solid #e73c17; }
    .swagger-ui .topbar .download-url-wrapper .select-label { color: #e2e8f0; }
    .swagger-ui .info .title { font-family: Montserrat, sans-serif; color: #e2e8f0; }
    .swagger-ui .info { margin: 30px 0; }
    .swagger-ui .info .title small { background: #e73c17; }
    .swagger-ui .info p, .swagger-ui .info li { color: #a0aec0; }
    .swagger-ui .scheme-container { background: #1a2332; box-shadow: none; }
    .swagger-ui .opblock-tag { font-family: Montserrat, sans-serif; color: #e2e8f0; border-bottom-color: #2d3748; }
    .swagger-ui .opblock.opblock-post { border-color: #e73c17; background: rgba(231,60,23,0.05); }
    .swagger-ui .opblock.opblock-post .opblock-summary-method { background: #e73c17; }
    .swagger-ui .opblock.opblock-get { border-color: #2c7a7b; background: rgba(44,122,123,0.05); }
    .swagger-ui .opblock.opblock-get .opblock-summary-method { background: #2c7a7b; }
    .swagger-ui .btn.execute { background-color: #e73c17; border-color: #e73c17; }
    .swagger-ui .btn.execute:hover { background-color: #c42f0f; }
    .swagger-ui section.models { border-color: #2d3748; }
    .swagger-ui .model-title { font-family: Montserrat, sans-serif; }
    .swagger-ui pre.microlight { background: #1a2332 !important; font-family: JetBrains Mono, monospace; }
    .swagger-ui .wrapper { background: #0f1419; }
    .swagger-ui .opblock .opblock-summary-description { color: #a0aec0; }
    .swagger-ui .opblock-description-wrapper p { color: #cbd5e0; }
    .swagger-ui .response-col_status { color: #e2e8f0; }
    .swagger-ui table thead tr td, .swagger-ui table thead tr th { color: #a0aec0; border-bottom-color: #2d3748; }
    .swagger-ui .parameter__name { color: #e2e8f0; }
    .swagger-ui .parameter__type { color: #2c7a7b; }
    .swagger-ui input[type=text] { background: #1a2332; color: #e2e8f0; border-color: #2d3748; }
    .swagger-ui textarea { background: #1a2332; color: #e2e8f0; border-color: #2d3748; font-family: JetBrains Mono, monospace; }
    .swagger-ui .opblock-body pre { background: #1a2332; color: #e2e8f0; }
    .swagger-ui .responses-inner h4, .swagger-ui .responses-inner h5 { color: #e2e8f0; }
    .swagger-ui .response-col_description { color: #a0aec0; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    SwaggerUIBundle({
      url: '/api/_meta/openapi.json',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
      layout: 'StandaloneLayout',
      deepLinking: true
    });
  </script>
</body>
</html>`

const redocHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>AILANG API - ReDoc</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@400;500&family=Montserrat:wght@600;700;800&display=swap" rel="stylesheet">
  <style>body { margin: 0; }</style>
</head>
<body>
  <redoc spec-url='/api/_meta/openapi.json'
    theme='{
      "colors": {
        "primary": { "main": "#e73c17" },
        "success": { "main": "#2c7a7b" },
        "text": { "primary": "#e2e8f0", "secondary": "#a0aec0" },
        "http": {
          "get": "#2c7a7b",
          "post": "#e73c17",
          "put": "#dd6b20",
          "delete": "#e53e3e"
        }
      },
      "typography": {
        "fontSize": "15px",
        "fontFamily": "Inter, -apple-system, BlinkMacSystemFont, sans-serif",
        "headings": { "fontFamily": "Montserrat, sans-serif", "fontWeight": "700" },
        "code": { "fontFamily": "JetBrains Mono, monospace", "fontSize": "13px" }
      },
      "sidebar": {
        "backgroundColor": "#0f1419",
        "textColor": "#e2e8f0",
        "activeTextColor": "#e73c17",
        "groupItems": { "textTransform": "uppercase" }
      },
      "rightPanel": {
        "backgroundColor": "#1a2332"
      }
    }'
  ></redoc>
  <script src="https://cdn.redoc.ly/redoc/latest/bundles/redoc.standalone.js"></script>
</body>
</html>`

// buildOpenAPISpec creates an OpenAPI 3.1 spec from loaded modules.
func (s *Server) buildOpenAPISpec() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paths := make(map[string]any)

	// Sort module paths (RelPath projections) for deterministic output.
	// s.modules is keyed by PhysicalPath; OpenAPI URLs use RelPath.
	modByRel := make(map[string]*ModuleInfo, len(s.modules))
	modPaths := make([]string, 0, len(s.modules))
	for _, info := range s.modules {
		if info == nil {
			continue
		}
		modByRel[info.Path] = info
		modPaths = append(modPaths, info.Path)
	}
	sort.Strings(modPaths)

	for _, modPath := range modPaths {
		modInfo := modByRel[modPath]
		// Sort exports for deterministic output.
		exports := make([]ExportInfo, len(modInfo.Exports))
		copy(exports, modInfo.Exports)
		sort.Slice(exports, func(i, j int) bool { return exports[i].Name < exports[j].Name })

		for _, export := range exports {
			if export.Arity < 0 {
				continue
			}
			if !s.isExposed(export) {
				continue
			}

			// Use custom route path if @route annotation present, else auto-route
			pathKey := "/api/" + modPath + "/" + export.Name
			httpMethod := "post"
			if export.RoutePath != "" {
				pathKey = export.RoutePath
				httpMethod = strings.ToLower(export.RouteMethod)
			}

			fs := schema.FromTypeString(export.Type)

			operation := map[string]any{
				"operationId": modPath + "." + export.Name,
				"summary":     export.Name + "(" + export.Type + ")",
				"tags":        []string{modPath},
			}

			if export.Pure {
				operation["x-ailang-pure"] = true
			}
			if export.RoutePath != "" {
				operation["x-ailang-route"] = export.RouteMethod + " " + export.RoutePath
			}
			if len(export.ParamNames) > 0 {
				operation["x-ailang-param-names"] = export.ParamNames
			}
			if len(export.ParamTypes) > 0 {
				operation["x-ailang-param-types"] = export.ParamTypes
			}

			// Request body (only for methods that accept a body).
			reqSchema := schema.RequestSchema(fs)
			if fs.Arity > 0 && httpMethod != "get" && httpMethod != "head" {
				reqBody := map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": reqSchema,
						},
					},
				}
				// Add description showing named parameter binding
				if len(export.ParamNames) > 0 {
					reqBody["description"] = fmt.Sprintf(
						"Accepts named JSON: {%s} or positional: {\"args\": [...]}",
						strings.Join(export.ParamNames, ", "),
					)
				}
				operation["requestBody"] = reqBody
			}

			// Response.
			respSchema := schema.ResponseSchema(fs)
			operation["responses"] = map[string]any{
				"200": map[string]any{
					"description": "Successful function call",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": respSchema,
						},
					},
				},
				"400": map[string]any{
					"description": "Invalid arguments",
				},
				"404": map[string]any{
					"description": "Module or function not found",
				},
				"500": map[string]any{
					"description": "Function execution error",
				},
			}

			paths[pathKey] = map[string]any{
				httpMethod: operation,
			}
		}
	}

	// Add meta endpoints.
	paths["/api/_meta/modules"] = map[string]any{
		"get": map[string]any{
			"operationId": "_meta.listModules",
			"summary":     "List all loaded modules and their exports",
			"tags":        []string{"introspection"},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Module listing",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"modules": map[string]any{"type": "array"},
									"count":   map[string]any{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}

	paths["/api/_health"] = map[string]any{
		"get": map[string]any{
			"operationId": "_meta.health",
			"summary":     "Health check",
			"tags":        []string{"introspection"},
			"responses": map[string]any{
				"200": map[string]any{
					"description": "Server is healthy",
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"status":        map[string]any{"type": "string"},
									"modules_count": map[string]any{"type": "integer"},
									"exports_count": map[string]any{"type": "integer"},
								},
							},
						},
					},
				},
			},
		},
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "AILANG API",
			"description": "Auto-generated REST API from AILANG module exports",
			"version":     "0.8.1.1",
		},
		"servers": []map[string]any{
			{"url": "http://localhost:" + s.port},
		},
		"paths": paths,
	}
}
