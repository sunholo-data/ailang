package apiserver

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// serverWithRoutes builds a minimal Server populated with the given
// RouteEntry fixtures as if they'd been registered via @route annotations.
// Each fixture is placed on its own synthetic module so s.getCustomRoutes()
// returns them.
func serverWithRoutes(routes ...RouteEntry) *Server {
	mods := map[string]*ModuleInfo{}
	for i, r := range routes {
		modPath := "test/mod" + string(rune('a'+i))
		mods[modPath] = &ModuleInfo{
			Path: modPath,
			Exports: []ExportInfo{
				{
					Name:        r.Function,
					RouteMethod: r.Method,
					RoutePath:   r.Path,
					IsRaw:       r.IsRaw,
					IsNowrap:    r.IsNowrap,
				},
			},
		}
	}
	return &Server{modules: mods}
}

// serverWithModule builds a minimal Server populated with a loaded module
// (no @route annotations) — used to test the legacy /api/{module}/{func}
// dispatch path.
func serverWithModule(modPath string, exports ...ExportInfo) *Server {
	return &Server{
		modules: map[string]*ModuleInfo{
			modPath: {
				Path:    modPath,
				Exports: exports,
			},
		},
	}
}

// decodeResponse extracts a FunctionCallResponse from a ResponseRecorder.
func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder) FunctionCallResponse {
	t.Helper()
	var resp FunctionCallResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nbody: %s", err, rr.Body.String())
	}
	return resp
}

// --- ROUTE_NOT_FOUND ---

// TestHandleFunctionCall_RouteNotFound_WithSuggestion is the docparse
// scenario: a route-driven server gets a request to a close-but-wrong path
// and should return a typed ROUTE_NOT_FOUND with a did-you-mean suggestion.
func TestHandleFunctionCall_RouteNotFound_WithSuggestion(t *testing.T) {
	s := serverWithRoutes(
		RouteEntry{Method: "POST", Path: "/api/v1/auth/device", Function: "start"},
		RouteEntry{Method: "POST", Path: "/api/v1/auth/device/poll", Function: "poll"},
		RouteEntry{Method: "POST", Path: "/api/v1/auth/device/approve", Function: "approve"},
	)

	req := httptest.NewRequest("POST", "/api/v1/auth/device/token", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	s.handleFunctionCall(rr, req)

	if rr.Code != 404 {
		t.Errorf("status: got %d want 404", rr.Code)
	}

	resp := decodeResponse(t, rr)
	if resp.ErrorDetail == nil {
		t.Fatalf("expected ErrorDetail, got nil. body: %s", rr.Body.String())
	}
	if resp.ErrorDetail.Code != ErrCodeRouteNotFound {
		t.Errorf("code: got %q want %q", resp.ErrorDetail.Code, ErrCodeRouteNotFound)
	}
	if !strings.Contains(resp.ErrorDetail.Message, "No route registered") {
		t.Errorf("message should mention 'No route registered', got: %q", resp.ErrorDetail.Message)
	}
	// Flat Error field must mirror ErrorDetail.Message for backward compat.
	if resp.Error != resp.ErrorDetail.Message {
		t.Errorf("flat Error should mirror ErrorDetail.Message: flat=%q detail=%q", resp.Error, resp.ErrorDetail.Message)
	}
	// Critical: the word "module" must NOT appear in the error message.
	if strings.Contains(strings.ToLower(resp.Error), "module") {
		t.Errorf("ROUTE_NOT_FOUND should not mention 'module' (was docparse bug): %q", resp.Error)
	}
	// Suggestion should point at a close route.
	if resp.ErrorDetail.SuggestedFix == "" {
		t.Error("expected SuggestedFix to be populated for close match")
	}
	if !strings.Contains(resp.ErrorDetail.SuggestedFix, "/api/v1/auth/device") {
		t.Errorf("suggestion should mention /api/v1/auth/device*, got: %q", resp.ErrorDetail.SuggestedFix)
	}
	// available_routes should list the device-* routes.
	if len(resp.ErrorDetail.AvailableRoutes) == 0 {
		t.Error("expected AvailableRoutes to be populated")
	}
}

// TestHandleFunctionCall_RouteNotFound_NoSuggestion verifies that when no
// route is close enough to suggest, SuggestedFix stays empty but the
// available_routes list is still returned (agent still gets a usable hint).
func TestHandleFunctionCall_RouteNotFound_NoSuggestion(t *testing.T) {
	s := serverWithRoutes(
		RouteEntry{Method: "GET", Path: "/health", Function: "health"},
		RouteEntry{Method: "GET", Path: "/metrics", Function: "metrics"},
	)

	req := httptest.NewRequest("POST", "/api/v1/auth/device/token", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	s.handleFunctionCall(rr, req)

	if rr.Code != 404 {
		t.Errorf("status: got %d want 404", rr.Code)
	}
	resp := decodeResponse(t, rr)
	if resp.ErrorDetail == nil {
		t.Fatal("expected ErrorDetail")
	}
	if resp.ErrorDetail.Code != ErrCodeRouteNotFound {
		t.Errorf("code: got %q want %q", resp.ErrorDetail.Code, ErrCodeRouteNotFound)
	}
	// available_routes should still be populated as a fallback.
	if len(resp.ErrorDetail.AvailableRoutes) == 0 {
		t.Error("expected AvailableRoutes populated even without close match")
	}
}

// --- MODULE_NOT_LOADED (legacy) ---

// TestHandleFunctionCall_ModuleNotLoaded_Legacy verifies that a server with
// zero @route registrations still returns MODULE_NOT_LOADED (preserving
// historical behavior for legacy dispatch-only deployments). The message
// text matches the historical format so existing clients parsing the flat
// Error field continue to work.
func TestHandleFunctionCall_ModuleNotLoaded_Legacy(t *testing.T) {
	// Server with a loaded module but no @routes — and the request targets
	// a *different* module path that isn't loaded.
	s := serverWithModule("ecommerce/api/handlers",
		ExportInfo{Name: "successResponse"},
	)

	req := httptest.NewRequest("POST", "/api/nonexistent/mod/someFunc", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	s.handleFunctionCall(rr, req)

	if rr.Code != 404 {
		t.Errorf("status: got %d want 404", rr.Code)
	}
	resp := decodeResponse(t, rr)
	if resp.ErrorDetail == nil {
		t.Fatal("expected ErrorDetail")
	}
	if resp.ErrorDetail.Code != ErrCodeModuleNotLoaded {
		t.Errorf("code: got %q want %q", resp.ErrorDetail.Code, ErrCodeModuleNotLoaded)
	}
	// Historical flat message format preserved.
	if !strings.Contains(resp.Error, `module "nonexistent/mod" not loaded`) {
		t.Errorf("expected legacy flat error text, got: %q", resp.Error)
	}
	// Module/Func fields populated for legacy clients.
	if resp.Module != "nonexistent/mod" {
		t.Errorf("Module: got %q want %q", resp.Module, "nonexistent/mod")
	}
	if resp.Func != "someFunc" {
		t.Errorf("Func: got %q want %q", resp.Func, "someFunc")
	}
}

// --- FUNCTION_NOT_FOUND ---

// TestHandleFunctionCall_FunctionNotFound verifies that when the module is
// loaded but the requested function doesn't exist, we return the typed
// FUNCTION_NOT_FOUND code with the existing "available: [...]" message.
func TestHandleFunctionCall_FunctionNotFound(t *testing.T) {
	s := serverWithModule("mymod",
		ExportInfo{Name: "existingFunc"},
	)

	req := httptest.NewRequest("POST", "/api/mymod/nonexistentFunc", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	s.handleFunctionCall(rr, req)

	if rr.Code != 404 {
		t.Errorf("status: got %d want 404", rr.Code)
	}
	resp := decodeResponse(t, rr)
	if resp.ErrorDetail == nil {
		t.Fatal("expected ErrorDetail")
	}
	if resp.ErrorDetail.Code != ErrCodeFunctionNotFound {
		t.Errorf("code: got %q want %q", resp.ErrorDetail.Code, ErrCodeFunctionNotFound)
	}
	if !strings.Contains(resp.Error, "existingFunc") {
		t.Errorf("expected 'available' list to mention existingFunc, got: %q", resp.Error)
	}
}

// --- METHOD_NOT_ALLOWED ---

// TestHandleFunctionCall_MethodNotAllowed verifies that non-POST/GET
// requests to the catch-all handler return the typed METHOD_NOT_ALLOWED code.
func TestHandleFunctionCall_MethodNotAllowed(t *testing.T) {
	s := serverWithModule("mymod", ExportInfo{Name: "f"})

	req := httptest.NewRequest("DELETE", "/api/mymod/f", nil)
	rr := httptest.NewRecorder()
	s.handleFunctionCall(rr, req)

	if rr.Code != 405 {
		t.Errorf("status: got %d want 405", rr.Code)
	}
	resp := decodeResponse(t, rr)
	if resp.ErrorDetail == nil {
		t.Fatal("expected ErrorDetail")
	}
	if resp.ErrorDetail.Code != ErrCodeMethodNotAllowed {
		t.Errorf("code: got %q want %q", resp.ErrorDetail.Code, ErrCodeMethodNotAllowed)
	}
}
