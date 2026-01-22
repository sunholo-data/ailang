package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenVerifierInvalidateCache(t *testing.T) {
	verifier := NewTokenVerifier(nil)

	// Manually add some cached entries
	verifier.cacheMutex.Lock()
	verifier.cache["test-token"] = &cachedToken{
		claims:    &UserClaims{UID: "test"},
		expiresAt: time.Now().Add(time.Hour),
	}
	verifier.cacheMutex.Unlock()

	assert.Equal(t, 1, verifier.CacheStats())

	verifier.InvalidateCache()
	assert.Equal(t, 0, verifier.CacheStats())
}

func TestUserFromClaims(t *testing.T) {
	claims := &UserClaims{
		UID:      "test-uid",
		Email:    "test@example.com",
		Name:     "Test User",
		AuthTime: time.Now(),
	}

	user := NewUserFromClaims(claims)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "test-uid", user.FirebaseUID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.Name)
}

func TestUserPermissions(t *testing.T) {
	user := &User{
		FirebaseUID: "test-uid",
		Email:       "test@example.com",
		Role:        "Approver",
		Permissions: []string{"approve_tasks", "view_workspace"},
	}

	assert.True(t, user.HasPermission("approve_tasks"))
	assert.True(t, user.HasPermission("view_workspace"))
	assert.False(t, user.HasPermission("delete_workspace"))
	assert.True(t, user.IsApprover())
	assert.False(t, user.IsViewer())

	user.Role = "Viewer"
	assert.False(t, user.IsApprover())
	assert.True(t, user.IsViewer())
}

func TestErrorVariables(t *testing.T) {
	assert.NotNil(t, ErrInvalidToken)
	assert.NotNil(t, ErrMissingToken)
	assert.NotNil(t, ErrInsufficientPermissions)
	assert.NotNil(t, ErrUserNotInWorkspace)
}

func TestUserClaims(t *testing.T) {
	now := time.Now()
	claims := &UserClaims{
		UID:       "uid-123",
		Email:     "user@example.com",
		Name:      "John Doe",
		AuthTime:  now,
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	assert.Equal(t, "uid-123", claims.UID)
	assert.Equal(t, "user@example.com", claims.Email)
	assert.Equal(t, "John Doe", claims.Name)
	assert.Equal(t, now, claims.AuthTime)
}

func TestUserStructure(t *testing.T) {
	user := &User{
		ID:          "user-id",
		FirebaseUID: "firebase-uid",
		Email:       "test@example.com",
		Name:        "Test",
		Role:        "Viewer",
		WorkspaceID: "workspace-1",
		Permissions: []string{"view"},
	}

	assert.Equal(t, "user-id", user.ID)
	assert.Equal(t, "firebase-uid", user.FirebaseUID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test", user.Name)
	assert.Equal(t, "Viewer", user.Role)
	assert.Equal(t, "workspace-1", user.WorkspaceID)
	assert.Len(t, user.Permissions, 1)
}

func TestTokenVerifierStats(t *testing.T) {
	verifier := NewTokenVerifier(nil)
	assert.Equal(t, 0, verifier.CacheStats())

	// Add entries manually
	verifier.cacheMutex.Lock()
	verifier.cache["token1"] = &cachedToken{claims: &UserClaims{}}
	verifier.cache["token2"] = &cachedToken{claims: &UserClaims{}}
	verifier.cacheMutex.Unlock()

	assert.Equal(t, 2, verifier.CacheStats())
}
