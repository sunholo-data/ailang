// handlers_auth.go provides authentication and authorization middleware for HTTP handlers.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/sunholo/ailang/internal/server/auth"
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

		// Verify token
		claims, err := srv.tokenVerifier.VerifyToken(r.Context(), token)
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "invalid_token", "Invalid or expired token")
			return
		}

		// Create user object
		user := auth.NewUserFromClaims(claims)

		// Load user role from Firestore
		if srv.accessControl != nil {
			// For now, we'll set a default workspace. This should be enhanced based on the request.
			workspaceID := r.Header.Get("X-Workspace-ID")
			if workspaceID == "" {
				// Try to extract from URL or use default
				workspaceID = "default"
			}

			// Try email-based lookup first (preferred), then fall back to UID
			var role *auth.WorkspaceRole
			var err error

			if user.Email != "" {
				role, err = srv.accessControl.GetUserRoleByEmail(r.Context(), user.Email, workspaceID)
			}

			if err != nil || role == nil {
				// Fall back to UID-based lookup
				role, err = srv.accessControl.GetUserRole(r.Context(), user.FirebaseUID, workspaceID)
			}

			if err != nil || role == nil {
				// User not in access control, treat as Guest
				user.Role = "Guest"
			} else {
				user.Role = role.Role
				user.WorkspaceID = role.WorkspaceID
			}
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuthMiddleware extracts user info if a token is present but doesn't reject unauthenticated requests.
// Use this for endpoints that work for both authenticated and unauthenticated users.
func (srv *Server) OptionalAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			// No token - continue without user context
			next.ServeHTTP(w, r)
			return
		}

		// Try to verify token
		claims, err := srv.tokenVerifier.VerifyToken(r.Context(), token)
		if err != nil {
			// Invalid token - continue without user context (don't reject)
			next.ServeHTTP(w, r)
			return
		}

		// Create user object
		user := auth.NewUserFromClaims(claims)

		// Add user to request context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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

// WorkspaceAccessContextKey is used to store workspace access info in HTTP requests.
const WorkspaceAccessContextKey AuthContextKey = "workspace_access"

// WorkspaceAccessInfo contains the user's workspace access information for this request.
type WorkspaceAccessInfo struct {
	RequestedWorkspace   string   // The workspace requested in query/header
	AccessibleWorkspaces []string // All workspaces the user can access
	Role                 string   // User's role in the requested workspace (Viewer/Approver)
	IsPublicOnly         bool     // True if user is unauthenticated (public workspaces only)
}

// RequireWorkspaceAccess is a middleware that checks if the user has access to the specified workspace.
// For authenticated users, checks against Firestore permissions.
// For unauthenticated users, only allows access to public workspaces.
func (srv *Server) RequireWorkspaceAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Extract workspace ID from header or URL parameter
		workspaceID := r.Header.Get("X-Workspace-ID")
		if workspaceID == "" {
			workspaceID = r.URL.Query().Get("workspace")
		}
		if workspaceID == "" {
			workspaceID = r.URL.Query().Get("workspace_id")
		}

		// Get user from context (may be nil for unauthenticated)
		user := getUserFromContext(r)

		accessInfo := &WorkspaceAccessInfo{
			RequestedWorkspace: workspaceID,
		}

		// If no workspace service configured, pass through
		if srv.workspaceService == nil {
			ctx = context.WithValue(ctx, WorkspaceAccessContextKey, accessInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if user == nil || user.Email == "" {
			// Unauthenticated user - only public workspaces
			accessInfo.IsPublicOnly = true

			// If specific workspace requested, check if it's public
			if workspaceID != "" && workspaceID != "public" {
				workspace, err := srv.workspaceService.GetWorkspace(ctx, workspaceID)
				if err != nil || !workspace.IsPublic {
					writeErrorResponse(w, http.StatusForbidden, "forbidden", "Workspace requires authentication")
					return
				}
			}

			// Get list of public workspaces for filtering
			publicWorkspaces, err := srv.workspaceService.ListAccessibleWorkspaces(ctx, "")
			if err == nil {
				for _, ws := range publicWorkspaces {
					accessInfo.AccessibleWorkspaces = append(accessInfo.AccessibleWorkspaces, ws.ID)
				}
			}
		} else {
			// Authenticated user - check permissions
			accessInfo.IsPublicOnly = false

			// Get accessible workspaces
			accessible, err := srv.workspaceService.ListAccessibleWorkspaces(ctx, user.Email)
			if err == nil {
				for _, ws := range accessible {
					accessInfo.AccessibleWorkspaces = append(accessInfo.AccessibleWorkspaces, ws.ID)
				}
			}

			// If specific workspace requested, verify access
			if workspaceID != "" && workspaceID != "public" {
				hasAccess, role, err := srv.workspaceService.HasWorkspaceAccess(ctx, user.Email, workspaceID)
				if err != nil || !hasAccess {
					writeErrorResponse(w, http.StatusForbidden, "forbidden", "Access to workspace denied")
					return
				}
				accessInfo.Role = role
			}
		}

		ctx = context.WithValue(ctx, WorkspaceAccessContextKey, accessInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetWorkspaceAccessFromContext retrieves workspace access info from request context.
func GetWorkspaceAccessFromContext(r *http.Request) *WorkspaceAccessInfo {
	access, ok := r.Context().Value(WorkspaceAccessContextKey).(*WorkspaceAccessInfo)
	if !ok {
		return nil
	}
	return access
}

// HealthHandler returns a simple health check response.
func (srv *Server) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// WhoAmIHandler returns information about the current authenticated user.
func WhoAmIHandler(w http.ResponseWriter, r *http.Request) {
	user, err := GetUserFromContext(r)
	if err != nil {
		writeErrorResponse(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           user.ID,
		"email":        user.Email,
		"name":         user.Name,
		"role":         user.Role,
		"workspace_id": user.WorkspaceID,
	})
}
