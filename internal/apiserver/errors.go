package apiserver

import "net/http"

// Router error codes. These are stable identifiers AI agents and SDKs can
// match on to decide how to recover from a router-layer failure.
//
// These codes cover errors produced by the serve-api router and dispatch
// machinery BEFORE user code runs. Runtime errors from user code, coercion
// failures, and Result.Err → HTTP status mapping are covered by sibling
// sprints (M-DX-SERVE-API-ERROR-STATUS, M-DX-SERVE-API-COERCION).
const (
	// ErrCodeRouteNotFound is emitted when a request path does not match any
	// registered @route on a route-driven server (i.e. the server has at
	// least one @route registered). Carries did-you-mean suggestions.
	ErrCodeRouteNotFound = "ROUTE_NOT_FOUND"

	// ErrCodeModuleNotLoaded is emitted only on legacy (zero-@route) servers
	// when a /api/{module}/{func} dispatch targets a module that isn't
	// loaded. On route-driven servers, unmatched paths return
	// ErrCodeRouteNotFound instead.
	ErrCodeModuleNotLoaded = "MODULE_NOT_LOADED"

	// ErrCodeFunctionNotFound is emitted when the module is loaded but the
	// requested function does not exist OR is hidden via @noexpose. The
	// code is intentionally shared with hidden-export errors so @noexpose
	// remains indistinguishable from "genuinely missing" to external callers.
	ErrCodeFunctionNotFound = "FUNCTION_NOT_FOUND"

	// ErrCodeMethodNotAllowed is emitted when the request method doesn't
	// match the @route method (or isn't POST/GET for the catch-all handler).
	ErrCodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
)

// RouterErrorDetail is the structured, AI-first error envelope emitted by
// the serve-api router layer. It's attached to FunctionCallResponse.ErrorDetail
// so clients can either match on the flat `error` string (backward compat)
// or on the structured `error_detail.code` field (preferred).
//
// The envelope intentionally mirrors the shape that docparse and other
// serve-api consumers already use for their own typed errors, so AI agents
// can use a single error-handling path across all error sources.
type RouterErrorDetail struct {
	// Code is a stable identifier (e.g. "ROUTE_NOT_FOUND"). Clients match
	// on this rather than the human-readable Message.
	Code string `json:"code"`

	// Message is a human-readable description of what went wrong. Always
	// mirrors the flat FunctionCallResponse.Error field for backward compat.
	Message string `json:"message"`

	// Retryable indicates whether the same request could succeed if retried
	// without modification. Router-layer errors are almost always false.
	Retryable bool `json:"retryable"`

	// SuggestedFix is an optional human-readable hint for how to recover.
	// For ROUTE_NOT_FOUND, this is populated with "Did you mean ...?" when
	// a close route match exists. May be empty.
	SuggestedFix string `json:"suggested_fix,omitempty"`

	// AvailableRoutes is an optional list of registered routes that might
	// be relevant to the failed request (e.g. routes sharing a path prefix).
	// Capped at 10 entries. Each entry is formatted as "METHOD /path".
	AvailableRoutes []string `json:"available_routes,omitempty"`
}

// writeRouterError writes a typed router error response. It populates BOTH
// the flat FunctionCallResponse.Error string (for backward compatibility
// with existing clients that parse the top-level `error` field) AND the
// structured ErrorDetail envelope (for AI agents and new clients).
//
// Callers should provide:
//   - status: the HTTP status code to return
//   - code: one of the ErrCode* constants
//   - msg: the human-readable error message (will appear in both fields)
//   - suggestedFix: optional recovery hint (pass "" if none)
//   - available: optional list of relevant routes (pass nil if none)
//
// The module and func fields of FunctionCallResponse are left empty by this
// helper; callers that know them (e.g. the legacy module/func dispatch path)
// should use writeJSON directly with a manually-constructed response.
func writeRouterError(w http.ResponseWriter, status int, code, msg, suggestedFix string, available []string) {
	writeJSON(w, status, FunctionCallResponse{
		Error: msg,
		ErrorDetail: &RouterErrorDetail{
			Code:            code,
			Message:         msg,
			Retryable:       false,
			SuggestedFix:    suggestedFix,
			AvailableRoutes: available,
		},
	})
}

// writeRouterErrorWithDispatch is like writeRouterError but also populates
// the Module and Func fields of the response. Used by the legacy
// /api/{module}/{func} dispatch path so existing clients that inspect those
// fields continue to work.
func writeRouterErrorWithDispatch(w http.ResponseWriter, status int, code, msg, suggestedFix, module, fn string) {
	writeJSON(w, status, FunctionCallResponse{
		Module: module,
		Func:   fn,
		Error:  msg,
		ErrorDetail: &RouterErrorDetail{
			Code:         code,
			Message:      msg,
			Retryable:    false,
			SuggestedFix: suggestedFix,
		},
	})
}
