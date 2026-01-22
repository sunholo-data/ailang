// Package auth provides Firebase JWT verification and token caching for the AILANG server.
package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/google/uuid"
)

// TokenCacheTTL is the time-to-live for cached tokens.
const TokenCacheTTL = 5 * time.Minute

// UserClaims represents the claims extracted from a Firebase JWT token.
type UserClaims struct {
	UID       string
	Email     string
	Name      string
	AuthTime  time.Time
	IssuedAt  time.Time
	ExpiresAt time.Time
}

// TokenVerifier provides JWT token verification and caching.
type TokenVerifier struct {
	firebaseAuth *auth.Client
	cache        map[string]*cachedToken
	cacheMutex   sync.RWMutex
}

// cachedToken stores a verified token with expiration information.
type cachedToken struct {
	claims    *UserClaims
	expiresAt time.Time
}

// NewTokenVerifier creates a new TokenVerifier with the given Firebase auth client.
func NewTokenVerifier(firebaseAuth *auth.Client) *TokenVerifier {
	return &TokenVerifier{
		firebaseAuth: firebaseAuth,
		cache:        make(map[string]*cachedToken),
	}
}

// VerifyToken verifies a Firebase JWT token and returns the user claims.
// It uses caching to reduce latency on repeated verifications.
func (tv *TokenVerifier) VerifyToken(ctx context.Context, token string) (*UserClaims, error) {
	// Check cache first
	tv.cacheMutex.RLock()
	if cached, exists := tv.cache[token]; exists {
		tv.cacheMutex.RUnlock()
		if time.Now().Before(cached.expiresAt) {
			return cached.claims, nil
		}
		// Cache entry expired, remove it
		tv.cacheMutex.Lock()
		delete(tv.cache, token)
		tv.cacheMutex.Unlock()
	} else {
		tv.cacheMutex.RUnlock()
	}

	// Verify token with Firebase
	decodedToken, err := tv.firebaseAuth.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims
	email := ""
	if e, ok := decodedToken.Claims["email"].(string); ok {
		email = e
	}

	claims := &UserClaims{
		UID:       decodedToken.UID,
		Email:     email,
		AuthTime:  time.Unix(decodedToken.AuthTime, 0),
		IssuedAt:  time.Unix(decodedToken.IssuedAt, 0),
		ExpiresAt: time.Unix(decodedToken.Expires, 0),
	}

	// Extract optional name claim
	if name, ok := decodedToken.Claims["name"].(string); ok {
		claims.Name = name
	}

	// Cache the verified token
	tv.cacheMutex.Lock()
	tv.cache[token] = &cachedToken{
		claims:    claims,
		expiresAt: time.Now().Add(TokenCacheTTL),
	}
	tv.cacheMutex.Unlock()

	return claims, nil
}

// InvalidateCache clears the token cache.
func (tv *TokenVerifier) InvalidateCache() {
	tv.cacheMutex.Lock()
	tv.cache = make(map[string]*cachedToken)
	tv.cacheMutex.Unlock()
}

// CacheStats returns the number of cached tokens.
func (tv *TokenVerifier) CacheStats() int {
	tv.cacheMutex.RLock()
	defer tv.cacheMutex.RUnlock()
	return len(tv.cache)
}

// User represents an authenticated user with workspace role information.
type User struct {
	ID          string // Unique user identifier (UUID)
	FirebaseUID string // Firebase UID
	Email       string
	Name        string
	Role        string // "Viewer" or "Approver"
	WorkspaceID string
	AuthTime    time.Time
	IssuedAt    time.Time
	ExpiresAt   time.Time
	Permissions []string // Additional permission strings
}

// HasPermission checks if the user has the specified permission.
func (u *User) HasPermission(permission string) bool {
	for _, p := range u.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// IsApprover returns true if the user has the Approver role.
func (u *User) IsApprover() bool {
	return u.Role == "Approver"
}

// IsViewer returns true if the user has the Viewer role.
func (u *User) IsViewer() bool {
	return u.Role == "Viewer"
}

// NewUserFromClaims creates a User from token claims.
func NewUserFromClaims(claims *UserClaims) *User {
	return &User{
		ID:          uuid.New().String(),
		FirebaseUID: claims.UID,
		Email:       claims.Email,
		Name:        claims.Name,
		AuthTime:    claims.AuthTime,
		IssuedAt:    claims.IssuedAt,
		ExpiresAt:   claims.ExpiresAt,
	}
}

// ErrInvalidToken is returned when a token cannot be verified.
var ErrInvalidToken = errors.New("invalid or expired token")

// ErrMissingToken is returned when a token is required but not provided.
var ErrMissingToken = errors.New("missing authentication token")

// ErrInsufficientPermissions is returned when a user lacks required permissions.
var ErrInsufficientPermissions = errors.New("insufficient permissions for this operation")

// ErrUserNotInWorkspace is returned when a user is not a member of a workspace.
var ErrUserNotInWorkspace = errors.New("user is not a member of this workspace")
