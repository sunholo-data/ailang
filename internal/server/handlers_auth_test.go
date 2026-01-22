package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sunholo/ailang/internal/server/auth"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
	}{
		{
			name:     "Valid bearer token",
			header:   "Bearer valid-token-123",
			expected: "valid-token-123",
		},
		{
			name:     "Empty header",
			header:   "",
			expected: "",
		},
		{
			name:     "Missing Bearer",
			header:   "Basic abc123",
			expected: "",
		},
		{
			name:     "Malformed header",
			header:   "BearerNoSpace",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			token := extractBearerToken(req)
			assert.Equal(t, tt.expected, token)
		})
	}
}

func TestWriteErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()

	writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Missing token")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, "unauthorized", resp.Error)
	assert.Equal(t, http.StatusUnauthorized, resp.Status)
	assert.Equal(t, "Missing token", resp.Message)
}

func TestGetUserFromContext(t *testing.T) {
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		Role:        "Viewer",
	}

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	retrieved, err := GetUserFromContext(req)
	require.NoError(t, err)
	assert.Equal(t, user.FirebaseUID, retrieved.FirebaseUID)
	assert.Equal(t, user.Email, retrieved.Email)
}

func TestGetUserFromContextMissing(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	_, err := GetUserFromContext(req)
	assert.Error(t, err)
}

func TestRequireAuthMiddleware(t *testing.T) {
	// Test with user in context
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
	}

	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "OK", w.Body.String())
}

func TestRequireAuthMiddlewareNoUser(t *testing.T) {
	handler := RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRequireApproverMiddleware(t *testing.T) {
	// Test with Approver role
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		Role:        "Approver",
	}

	handler := RequireApprover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireApproverMiddlewareNotApprover(t *testing.T) {
	// Test with Viewer role
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		Role:        "Viewer",
	}

	handler := RequireApprover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRequireWorkspaceAccessMiddleware(t *testing.T) {
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		WorkspaceID: "workspace-123",
	}

	handler := RequireWorkspaceAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with matching workspace
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Workspace-ID", "workspace-123")
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireWorkspaceAccessMiddlewareForbidden(t *testing.T) {
	user := &auth.User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		WorkspaceID: "workspace-123",
	}

	handler := RequireWorkspaceAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Test with different workspace
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Workspace-ID", "workspace-456")
	ctx := context.WithValue(req.Context(), UserContextKey, user)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestErrorResponseFields(t *testing.T) {
	resp := ErrorResponse{
		Error:   "test_error",
		Status:  400,
		Message: "Test message",
	}

	assert.Equal(t, "test_error", resp.Error)
	assert.Equal(t, 400, resp.Status)
	assert.Equal(t, "Test message", resp.Message)
}

func TestContextKeyType(t *testing.T) {
	// Verify context key is correct type
	var key AuthContextKey = "user"
	assert.Equal(t, AuthContextKey("user"), key)
	assert.Equal(t, UserContextKey, AuthContextKey("user"))
}
