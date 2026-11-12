// Package auth provides Firestore-based access control and role lookups.
package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
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
