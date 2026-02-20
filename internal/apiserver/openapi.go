package apiserver

import (
	"encoding/json"
	"net/http"
	"sort"

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

// buildOpenAPISpec creates an OpenAPI 3.1 spec from loaded modules.
func (s *Server) buildOpenAPISpec() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()

	paths := make(map[string]any)

	// Sort module paths for deterministic output.
	modPaths := make([]string, 0, len(s.modules))
	for k := range s.modules {
		modPaths = append(modPaths, k)
	}
	sort.Strings(modPaths)

	for _, modPath := range modPaths {
		modInfo := s.modules[modPath]
		// Sort exports for deterministic output.
		exports := make([]ExportInfo, len(modInfo.Exports))
		copy(exports, modInfo.Exports)
		sort.Slice(exports, func(i, j int) bool { return exports[i].Name < exports[j].Name })

		for _, export := range exports {
			if export.Arity < 0 {
				continue
			}

			pathKey := "/api/" + modPath + "/" + export.Name
			fs := schema.FromTypeString(export.Type)

			operation := map[string]any{
				"operationId": modPath + "." + export.Name,
				"summary":     export.Name + "(" + export.Type + ")",
				"tags":        []string{modPath},
			}

			if export.Pure {
				operation["x-ailang-pure"] = true
			}

			// Request body.
			reqSchema := schema.RequestSchema(fs)
			if fs.Arity > 0 {
				operation["requestBody"] = map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": reqSchema,
						},
					},
				}
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
				"post": operation,
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
			"version":     "0.8.1",
		},
		"servers": []map[string]any{
			{"url": "http://localhost:" + s.port},
		},
		"paths": paths,
	}
}
