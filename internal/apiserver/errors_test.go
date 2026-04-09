package apiserver

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRouterErrorDetail_JSONMarshal verifies the envelope marshals with
// error_detail populated and all expected fields present.
func TestRouterErrorDetail_JSONMarshal(t *testing.T) {
	resp := FunctionCallResponse{
		Error: "No route registered for POST /api/v1/auth/device/token",
		ErrorDetail: &RouterErrorDetail{
			Code:         ErrCodeRouteNotFound,
			Message:      "No route registered for POST /api/v1/auth/device/token",
			Retryable:    false,
			SuggestedFix: "Did you mean POST /api/v1/auth/device/poll?",
			AvailableRoutes: []string{
				"POST /api/v1/auth/device",
				"POST /api/v1/auth/device/poll",
			},
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := string(data)
	wantSubstrings := []string{
		`"error":"No route registered for POST /api/v1/auth/device/token"`,
		`"error_detail":{`,
		`"code":"ROUTE_NOT_FOUND"`,
		`"retryable":false`,
		`"suggested_fix":"Did you mean POST /api/v1/auth/device/poll?"`,
		`"available_routes":["POST /api/v1/auth/device","POST /api/v1/auth/device/poll"]`,
	}
	for _, sub := range wantSubstrings {
		if !strings.Contains(got, sub) {
			t.Errorf("response missing %q\nfull: %s", sub, got)
		}
	}
}

// TestRouterErrorDetail_OmittedWhenNil verifies that responses without an
// ErrorDetail don't include the error_detail field at all (backward compat
// for existing clients).
func TestRouterErrorDetail_OmittedWhenNil(t *testing.T) {
	resp := FunctionCallResponse{
		Error:  `module "foo" not loaded`,
		Module: "foo",
		Func:   "bar",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got := string(data)
	if strings.Contains(got, "error_detail") {
		t.Errorf("expected error_detail to be omitted when nil, got: %s", got)
	}
}

// TestWriteRouterError_FlatErrorMirrorsDetail verifies that the flat Error
// field always mirrors ErrorDetail.Message so legacy clients see the same
// text regardless of which field they parse.
func TestWriteRouterError_FlatErrorMirrorsDetail(t *testing.T) {
	rr := httptest.NewRecorder()
	msg := "No route registered for POST /api/x"
	writeRouterError(rr, 404, ErrCodeRouteNotFound, msg, "", nil)

	if rr.Code != 404 {
		t.Errorf("status: got %d want 404", rr.Code)
	}

	var resp FunctionCallResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Error != msg {
		t.Errorf("flat Error: got %q want %q", resp.Error, msg)
	}
	if resp.ErrorDetail == nil {
		t.Fatal("ErrorDetail is nil")
	}
	if resp.ErrorDetail.Code != ErrCodeRouteNotFound {
		t.Errorf("Code: got %q want %q", resp.ErrorDetail.Code, ErrCodeRouteNotFound)
	}
	if resp.ErrorDetail.Message != msg {
		t.Errorf("ErrorDetail.Message: got %q want %q", resp.ErrorDetail.Message, msg)
	}
	if resp.ErrorDetail.Retryable {
		t.Error("router errors should not be retryable")
	}
}

// TestWriteRouterError_EmptyOptionalFieldsOmitted verifies that empty
// SuggestedFix and nil AvailableRoutes are omitted from the JSON output
// (cleaner responses for the no-suggestion case).
func TestWriteRouterError_EmptyOptionalFieldsOmitted(t *testing.T) {
	rr := httptest.NewRecorder()
	writeRouterError(rr, 404, ErrCodeRouteNotFound, "not found", "", nil)

	got := rr.Body.String()
	if strings.Contains(got, "suggested_fix") {
		t.Errorf("empty suggested_fix should be omitted, got: %s", got)
	}
	if strings.Contains(got, "available_routes") {
		t.Errorf("nil available_routes should be omitted, got: %s", got)
	}
}
