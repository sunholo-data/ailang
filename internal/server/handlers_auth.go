// handlers_auth.go provides authentication and authorization middleware for HTTP handlers.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sunholo-data/ailang/internal/server/auth"
)

// AuthContextKey is used to store user context in HTTP requests.
type AuthContextKey string

const UserContextKey AuthContextKey = "user"

// ErrorResponse represents a JSON error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
}

// writeErrorResponse writes a JSON error response with the given status code.
func writeErrorResponse(w http.ResponseWriter, statusCode int, errorMsg, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   errorMsg,
		Status:  statusCode,
		Message: message,
	})
}

// extractBearerToken extracts the Bearer token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// AuthMiddleware verifies Firebase JWT tokens and loads user information.
func (srv *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeErrorResponse(w, http.StatusUnauthorized, "missing_token", "Missing authentication token")
			return
		}

		user, err := srv.firebaseUser(r, token)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// firebaseUser verifies a Firebase ID token and loads the caller's workspace
// role from Firestore access control (Guest when they have none). It needs a
// configured tokenVerifier.
func (srv *Server) firebaseUser(r *http.Request, token string) (*auth.User, error) {
	claims, err := srv.tokenVerifier.VerifyToken(r.Context(), token)
	if err != nil {
		return nil, err
	}
	user := auth.NewUserFromClaims(claims)
	if srv.accessControl == nil {
		return user, nil
	}

	// For now, we'll set a default workspace. This should be enhanced based on the request.
	workspaceID := r.Header.Get("X-Workspace-ID")
	if workspaceID == "" {
		workspaceID = "default"
	}

	// Try email-based lookup first (preferred), then fall back to UID
	var role *auth.WorkspaceRole
	if user.Email != "" {
		role, err = srv.accessControl.GetUserRoleByEmail(r.Context(), user.Email, workspaceID)
	}
	if err != nil || role == nil {
		role, err = srv.accessControl.GetUserRole(r.Context(), user.FirebaseUID, workspaceID)
	}
	if err != nil || role == nil {
		// User not in access control, treat as Guest
		user.Role = "Guest"
	} else {
		user.Role = role.Role
		user.WorkspaceID = role.WorkspaceID
	}
	return user, nil
}

// RequireAuth is a middleware that requires authentication.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userVal := r.Context().Value(UserContextKey)
		if userVal == nil {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		user, ok := userVal.(*auth.User)
		if !ok || user == nil {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireApprover is a middleware that requires the Approver role.
func RequireApprover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userVal := r.Context().Value(UserContextKey)
		if userVal == nil {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		user, ok := userVal.(*auth.User)
		if !ok || user == nil {
			writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		if !user.IsApprover() {
			writeErrorResponse(w, http.StatusForbidden, "forbidden", "This operation requires Approver role")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getUserFromContext retrieves the authenticated user from the request context.
func getUserFromContext(r *http.Request) *auth.User {
	user, ok := r.Context().Value(UserContextKey).(*auth.User)
	if !ok {
		return nil
	}
	return user
}

// GetUserFromContext is exported for use by other packages.
func GetUserFromContext(r *http.Request) (*auth.User, error) {
	user := getUserFromContext(r)
	if user == nil {
		return nil, errors.New("user not found in context")
	}
	return user, nil
}
