// Package auth provides workspace access control with Firestore backend.
package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// EncodeDocID encodes a workspace ID for use as a Firestore document ID.
// Replaces "/" with "__" since Firestore doesn't allow "/" in document IDs.
func EncodeDocID(id string) string {
	return strings.ReplaceAll(id, "/", "__")
}

// DecodeDocID decodes a Firestore document ID back to a workspace ID.
func DecodeDocID(docID string) string {
	return strings.ReplaceAll(docID, "__", "/")
}

// Workspace represents a workspace with access control metadata.
type Workspace struct {
	ID           string    `json:"id" firestore:"id"`                       // e.g., "sunholo-data/ailang"
	Name         string    `json:"name" firestore:"name"`                   // Human-readable name
	GitHubRepo   string    `json:"github_repo" firestore:"github_repo"`     // GitHub repo (owner/repo)
	IsPublic     bool      `json:"is_public" firestore:"is_public"`         // Anonymous users get Viewer access
	PathPatterns []string  `json:"path_patterns" firestore:"path_patterns"` // File paths that map to this workspace
	CreatedAt    time.Time `json:"created_at" firestore:"created_at"`
	CreatedBy    string    `json:"created_by" firestore:"created_by"`
}

// WorkspaceAccess represents a user's access to a workspace.
type WorkspaceAccess struct {
	Email       string    `json:"email" firestore:"email"`
	WorkspaceID string    `json:"workspace_id" firestore:"workspace_id"`
	Role        string    `json:"role" firestore:"role"` // "Viewer" or "Approver"
	GrantedAt   time.Time `json:"granted_at" firestore:"granted_at"`
	GrantedBy   string    `json:"granted_by" firestore:"granted_by"`
}

// AccessibleWorkspace represents a workspace the user can access with their role.
type AccessibleWorkspace struct {
	Workspace
	Role string `json:"role"` // User's role in this workspace
}

// WorkspaceService provides workspace access control operations.
type WorkspaceService struct {
	firestore *firestore.Client
	cache     *workspaceCache
	mutex     sync.RWMutex
}

// workspaceCache caches workspace metadata lookups.
type workspaceCache struct {
	workspaces map[string]*cachedWorkspace // workspace_id -> workspace metadata
	ttl        time.Duration
}

type cachedWorkspace struct {
	workspace *Workspace
	expiresAt time.Time
}

// NewWorkspaceService creates a new workspace service.
func NewWorkspaceService(fs *firestore.Client) *WorkspaceService {
	return &WorkspaceService{
		firestore: fs,
		cache: &workspaceCache{
			workspaces: make(map[string]*cachedWorkspace),
			ttl:        5 * time.Minute,
		},
	}
}

// ListAccessibleWorkspaces returns all workspaces the user can access.
// For unauthenticated users (email=""), returns only public workspaces.
// For authenticated users, returns public workspaces + workspaces with explicit grants.
func (ws *WorkspaceService) ListAccessibleWorkspaces(ctx context.Context, email string) ([]AccessibleWorkspace, error) {
	var result []AccessibleWorkspace

	// Get all public workspaces
	publicWorkspaces, err := ws.listPublicWorkspaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list public workspaces: %w", err)
	}

	for _, w := range publicWorkspaces {
		role := "Viewer" // Anonymous users get Viewer role on public workspaces
		result = append(result, AccessibleWorkspace{
			Workspace: w,
			Role:      role,
		})
	}

	// If authenticated, also get workspaces with explicit access grants
	if email != "" {
		grantedWorkspaces, err := ws.listGrantedWorkspaces(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to list granted workspaces: %w", err)
		}

		// Merge granted workspaces, preferring higher role if workspace is both public and granted
		publicIDs := make(map[string]int) // workspace_id -> index in result
		for i, w := range result {
			publicIDs[w.ID] = i
		}

		for _, gw := range grantedWorkspaces {
			if idx, exists := publicIDs[gw.ID]; exists {
				// Workspace is public AND user has explicit grant - use granted role if higher
				if gw.Role == "Approver" {
					result[idx].Role = "Approver"
				}
			} else {
				// Workspace is not public, add it
				result = append(result, gw)
			}
		}
	}

	return result, nil
}

// listPublicWorkspaces returns all workspaces with is_public=true.
func (ws *WorkspaceService) listPublicWorkspaces(ctx context.Context) ([]Workspace, error) {
	query := ws.firestore.Collection("workspaces").Where("is_public", "==", true)
	iter := query.Documents(ctx)
	defer iter.Stop()

	var workspaces []Workspace
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating public workspaces: %w", err)
		}

		var w Workspace
		if err := doc.DataTo(&w); err != nil {
			continue // Skip invalid documents
		}
		// ID should be in the document data, but fallback to decoding doc ref ID if not
		if w.ID == "" {
			w.ID = DecodeDocID(doc.Ref.ID)
		}
		workspaces = append(workspaces, w)
	}

	return workspaces, nil
}

// listGrantedWorkspaces returns workspaces where the user has explicit access.
func (ws *WorkspaceService) listGrantedWorkspaces(ctx context.Context, email string) ([]AccessibleWorkspace, error) {
	// Query workspace_access collection for this user's grants
	query := ws.firestore.CollectionGroup("users").Where("email", "==", email)
	iter := query.Documents(ctx)
	defer iter.Stop()

	var result []AccessibleWorkspace
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating user access grants: %w", err)
		}

		var access WorkspaceAccess
		if err := doc.DataTo(&access); err != nil {
			continue // Skip invalid documents
		}

		// Get the workspace metadata
		workspaceID := access.WorkspaceID
		workspace, err := ws.GetWorkspace(ctx, workspaceID)
		if err != nil {
			continue // Skip if workspace doesn't exist
		}

		result = append(result, AccessibleWorkspace{
			Workspace: *workspace,
			Role:      access.Role,
		})
	}

	return result, nil
}

// GetWorkspace retrieves a workspace by ID.
func (ws *WorkspaceService) GetWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	// Check cache
	ws.mutex.RLock()
	if cached, exists := ws.cache.workspaces[workspaceID]; exists {
		if time.Now().Before(cached.expiresAt) {
			ws.mutex.RUnlock()
			return cached.workspace, nil
		}
	}
	ws.mutex.RUnlock()

	// Encode workspace ID for Firestore document ID (replaces "/" with "__")
	docID := EncodeDocID(workspaceID)

	// Fetch from Firestore
	doc, err := ws.firestore.Collection("workspaces").Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var workspace Workspace
	if err := doc.DataTo(&workspace); err != nil {
		return nil, err
	}
	// Only set ID from doc ref if not already populated from document data
	// Use DecodeDocID to convert "__" back to "/" in workspace IDs
	if workspace.ID == "" {
		workspace.ID = DecodeDocID(doc.Ref.ID)
	}

	// Cache the result
	ws.mutex.Lock()
	ws.cache.workspaces[workspaceID] = &cachedWorkspace{
		workspace: &workspace,
		expiresAt: time.Now().Add(ws.cache.ttl),
	}
	ws.mutex.Unlock()

	return &workspace, nil
}

// GrantAccess grants a user access to a workspace with the specified role.
func (ws *WorkspaceService) GrantAccess(ctx context.Context, email, workspaceID, role, grantedBy string) error {
	if role != "Viewer" && role != "Approver" {
		return fmt.Errorf("invalid role: %s (must be 'Viewer' or 'Approver')", role)
	}

	access := &WorkspaceAccess{
		Email:       email,
		WorkspaceID: workspaceID,
		Role:        role,
		GrantedAt:   time.Now(),
		GrantedBy:   grantedBy,
	}

	// Encode workspace ID for Firestore document path
	encodedWorkspaceID := EncodeDocID(workspaceID)
	docPath := fmt.Sprintf("workspace_access/%s/users/%s", encodedWorkspaceID, email)
	_, err := ws.firestore.Doc(docPath).Set(ctx, access)
	if err != nil {
		return fmt.Errorf("failed to grant access: %w", err)
	}

	return nil
}

// RevokeAccess removes a user's access to a workspace.
func (ws *WorkspaceService) RevokeAccess(ctx context.Context, email, workspaceID string) error {
	// Encode workspace ID for Firestore document path
	encodedWorkspaceID := EncodeDocID(workspaceID)
	docPath := fmt.Sprintf("workspace_access/%s/users/%s", encodedWorkspaceID, email)
	_, err := ws.firestore.Doc(docPath).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to revoke access: %w", err)
	}

	return nil
}

// SetPublic updates a workspace's public visibility.
func (ws *WorkspaceService) SetPublic(ctx context.Context, workspaceID string, isPublic bool) error {
	// Encode workspace ID for Firestore document ID
	docID := EncodeDocID(workspaceID)
	_, err := ws.firestore.Collection("workspaces").Doc(docID).Update(ctx, []firestore.Update{
		{Path: "is_public", Value: isPublic},
	})
	if err != nil {
		return fmt.Errorf("failed to update workspace visibility: %w", err)
	}

	// Invalidate cache
	ws.mutex.Lock()
	delete(ws.cache.workspaces, workspaceID)
	ws.mutex.Unlock()

	return nil
}
