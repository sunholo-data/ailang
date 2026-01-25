// Package auth provides Firestore-based access control and role lookups.
package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// AccessControlCache stores cached role and workspace membership information.
type AccessControlCache struct {
	firestore *firestore.Client
	roles     map[string]*cachedRole // key: uid:workspace_id
	mutex     sync.RWMutex
	ttl       time.Duration
}

// cachedRole stores a user's role and permissions for a workspace.
type cachedRole struct {
	role      string
	workspace string
	uid       string
	expiresAt time.Time
}

// WorkspaceRole represents a user's role in a workspace.
type WorkspaceRole struct {
	UID         string
	WorkspaceID string
	Role        string // "Viewer" or "Approver"
	Permissions []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewAccessControlCache creates a new access control cache.
func NewAccessControlCache(fs *firestore.Client) *AccessControlCache {
	return &AccessControlCache{
		firestore: fs,
		roles:     make(map[string]*cachedRole),
		ttl:       5 * time.Minute,
	}
}

// GetUserRole retrieves the user's role for a workspace from Firestore.
// It uses caching to reduce latency on repeated lookups.
func (acc *AccessControlCache) GetUserRole(ctx context.Context, uid, workspaceID string) (*WorkspaceRole, error) {
	cacheKey := fmt.Sprintf("%s:%s", uid, workspaceID)

	// Check cache first
	acc.mutex.RLock()
	if cached, exists := acc.roles[cacheKey]; exists {
		acc.mutex.RUnlock()
		if time.Now().Before(cached.expiresAt) {
			return &WorkspaceRole{
				UID:         uid,
				WorkspaceID: workspaceID,
				Role:        cached.role,
			}, nil
		}
		// Cache entry expired, remove it
		acc.mutex.Lock()
		delete(acc.roles, cacheKey)
		acc.mutex.Unlock()
	} else {
		acc.mutex.RUnlock()
	}

	// Query Firestore for access control entry
	doc, err := acc.firestore.Collection("access_control").Doc(cacheKey).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user role: %w", err)
	}

	data := doc.Data()
	role, ok := data["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid role in access control document")
	}

	// Cache the result
	acc.mutex.Lock()
	acc.roles[cacheKey] = &cachedRole{
		role:      role,
		workspace: workspaceID,
		uid:       uid,
		expiresAt: time.Now().Add(acc.ttl),
	}
	acc.mutex.Unlock()

	return &WorkspaceRole{
		UID:         uid,
		WorkspaceID: workspaceID,
		Role:        role,
	}, nil
}

// GetUserRoleByEmail retrieves the user's role by email address.
// This allows configuring access control by email without needing Firebase UIDs.
// Email-based entries use document ID: email:workspace_id
func (acc *AccessControlCache) GetUserRoleByEmail(ctx context.Context, email, workspaceID string) (*WorkspaceRole, error) {
	cacheKey := fmt.Sprintf("email:%s:%s", email, workspaceID)

	// Check cache first
	acc.mutex.RLock()
	if cached, exists := acc.roles[cacheKey]; exists {
		acc.mutex.RUnlock()
		if time.Now().Before(cached.expiresAt) {
			return &WorkspaceRole{
				UID:         email, // Use email as identifier
				WorkspaceID: workspaceID,
				Role:        cached.role,
			}, nil
		}
		// Cache entry expired, remove it
		acc.mutex.Lock()
		delete(acc.roles, cacheKey)
		acc.mutex.Unlock()
	} else {
		acc.mutex.RUnlock()
	}

	// Query Firestore for email-based access control entry
	doc, err := acc.firestore.Collection("access_control").Doc(cacheKey).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user role by email: %w", err)
	}

	data := doc.Data()
	role, ok := data["role"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid role in access control document")
	}

	// Cache the result
	acc.mutex.Lock()
	acc.roles[cacheKey] = &cachedRole{
		role:      role,
		workspace: workspaceID,
		uid:       email,
		expiresAt: time.Now().Add(acc.ttl),
	}
	acc.mutex.Unlock()

	return &WorkspaceRole{
		UID:         email,
		WorkspaceID: workspaceID,
		Role:        role,
	}, nil
}

// AddUserByEmail adds a user to a workspace by email with the specified role.
// This is the recommended way to configure access control.
func (acc *AccessControlCache) AddUserByEmail(ctx context.Context, email, workspaceID, role string) error {
	if role != "Viewer" && role != "Approver" {
		return fmt.Errorf("invalid role: %s (must be 'Viewer' or 'Approver')", role)
	}

	cacheKey := fmt.Sprintf("email:%s:%s", email, workspaceID)
	now := time.Now()

	// Add to Firestore
	_, err := acc.firestore.Collection("access_control").Doc(cacheKey).Set(ctx, map[string]interface{}{
		"email":        email,
		"workspace_id": workspaceID,
		"role":         role,
		"created_at":   now,
		"updated_at":   now,
	})
	if err != nil {
		return fmt.Errorf("failed to add user by email: %w", err)
	}

	// Update cache
	acc.mutex.Lock()
	acc.roles[cacheKey] = &cachedRole{
		role:      role,
		workspace: workspaceID,
		uid:       email,
		expiresAt: time.Now().Add(acc.ttl),
	}
	acc.mutex.Unlock()

	return nil
}

// AddUserToWorkspace adds a user to a workspace with the specified role.
func (acc *AccessControlCache) AddUserToWorkspace(ctx context.Context, uid, workspaceID, role string) error {
	if role != "Viewer" && role != "Approver" {
		return fmt.Errorf("invalid role: %s", role)
	}

	cacheKey := fmt.Sprintf("%s:%s", uid, workspaceID)
	now := time.Now()

	// Add to Firestore
	_, err := acc.firestore.Collection("access_control").Doc(cacheKey).Set(ctx, map[string]interface{}{
		"uid":          uid,
		"workspace_id": workspaceID,
		"role":         role,
		"created_at":   now,
		"updated_at":   now,
	})
	if err != nil {
		return fmt.Errorf("failed to add user to workspace: %w", err)
	}

	// Update cache
	acc.mutex.Lock()
	acc.roles[cacheKey] = &cachedRole{
		role:      role,
		workspace: workspaceID,
		uid:       uid,
		expiresAt: time.Now().Add(acc.ttl),
	}
	acc.mutex.Unlock()

	return nil
}

// RemoveUserFromWorkspace removes a user from a workspace.
func (acc *AccessControlCache) RemoveUserFromWorkspace(ctx context.Context, uid, workspaceID string) error {
	cacheKey := fmt.Sprintf("%s:%s", uid, workspaceID)

	// Remove from Firestore
	_, err := acc.firestore.Collection("access_control").Doc(cacheKey).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to remove user from workspace: %w", err)
	}

	// Remove from cache
	acc.mutex.Lock()
	delete(acc.roles, cacheKey)
	acc.mutex.Unlock()

	return nil
}

// UpdateUserRole updates a user's role in a workspace.
func (acc *AccessControlCache) UpdateUserRole(ctx context.Context, uid, workspaceID, newRole string) error {
	if newRole != "Viewer" && newRole != "Approver" {
		return fmt.Errorf("invalid role: %s", newRole)
	}

	cacheKey := fmt.Sprintf("%s:%s", uid, workspaceID)

	// Update in Firestore
	_, err := acc.firestore.Collection("access_control").Doc(cacheKey).Update(ctx, []firestore.Update{
		{Path: "role", Value: newRole},
		{Path: "updated_at", Value: time.Now()},
	})
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}

	// Update cache
	acc.mutex.Lock()
	acc.roles[cacheKey] = &cachedRole{
		role:      newRole,
		workspace: workspaceID,
		uid:       uid,
		expiresAt: time.Now().Add(acc.ttl),
	}
	acc.mutex.Unlock()

	return nil
}

// ListWorkspaceMembers retrieves all members of a workspace.
func (acc *AccessControlCache) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]*WorkspaceRole, error) {
	query := acc.firestore.Collection("access_control").Where("workspace_id", "==", workspaceID)
	iter := query.Documents(ctx)
	defer iter.Stop()

	var members []*WorkspaceRole
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list workspace members: %w", err)
		}

		data := doc.Data()
		uid, _ := data["uid"].(string)
		role, _ := data["role"].(string)

		members = append(members, &WorkspaceRole{
			UID:         uid,
			WorkspaceID: workspaceID,
			Role:        role,
		})
	}

	return members, nil
}

// InvalidateCache clears all cached roles.
func (acc *AccessControlCache) InvalidateCache() {
	acc.mutex.Lock()
	acc.roles = make(map[string]*cachedRole)
	acc.mutex.Unlock()
}

// CacheStats returns the number of cached roles.
func (acc *AccessControlCache) CacheStats() int {
	acc.mutex.RLock()
	defer acc.mutex.RUnlock()
	return len(acc.roles)
}
