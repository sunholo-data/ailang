// Package auth provides workspace access control with Firestore backend.
package auth

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sunholo/ailang/internal/coordinator"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

// WorkspacesConfig is an alias to coordinator.WorkspacesConfig for convenience.
type WorkspacesConfig = coordinator.WorkspacesConfig

// WorkspaceService provides workspace access control operations.
type WorkspaceService struct {
	firestore *firestore.Client
	config    *WorkspacesConfig
	cache     *workspaceCache
	mutex     sync.RWMutex
}

// workspaceCache caches workspace access checks.
type workspaceCache struct {
	workspaces map[string]*cachedWorkspace       // workspace_id -> workspace metadata
	access     map[string]*cachedWorkspaceAccess // email:workspace_id -> access entry
	ttl        time.Duration
}

type cachedWorkspace struct {
	workspace *Workspace
	expiresAt time.Time
}

type cachedWorkspaceAccess struct {
	hasAccess bool
	role      string
	expiresAt time.Time
}

// NewWorkspaceService creates a new workspace service.
func NewWorkspaceService(fs *firestore.Client, config *WorkspacesConfig) *WorkspaceService {
	if config == nil {
		config = &WorkspacesConfig{
			DefaultWorkspace: "public",
		}
	}
	return &WorkspaceService{
		firestore: fs,
		config:    config,
		cache: &workspaceCache{
			workspaces: make(map[string]*cachedWorkspace),
			access:     make(map[string]*cachedWorkspaceAccess),
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

// HasWorkspaceAccess checks if a user has access to a workspace.
// Returns (hasAccess, role, error).
// For public workspaces, unauthenticated users get Viewer access.
func (ws *WorkspaceService) HasWorkspaceAccess(ctx context.Context, email, workspaceID string) (bool, string, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", email, workspaceID)
	ws.mutex.RLock()
	if cached, exists := ws.cache.access[cacheKey]; exists {
		if time.Now().Before(cached.expiresAt) {
			ws.mutex.RUnlock()
			return cached.hasAccess, cached.role, nil
		}
	}
	ws.mutex.RUnlock()

	// Get workspace to check if public
	workspace, err := ws.GetWorkspace(ctx, workspaceID)
	if err != nil {
		// Workspace doesn't exist
		ws.cacheAccessResult(cacheKey, false, "")
		return false, "", nil
	}

	// If workspace is public, everyone has at least Viewer access
	if workspace.IsPublic {
		// Check if user has explicit higher role
		if email != "" {
			role, err := ws.getUserWorkspaceRole(ctx, email, workspaceID)
			if err == nil && role != "" {
				ws.cacheAccessResult(cacheKey, true, role)
				return true, role, nil
			}
		}
		// Default to Viewer for public workspace
		ws.cacheAccessResult(cacheKey, true, "Viewer")
		return true, "Viewer", nil
	}

	// Private workspace - need explicit grant
	if email == "" {
		ws.cacheAccessResult(cacheKey, false, "")
		return false, "", nil
	}

	role, err := ws.getUserWorkspaceRole(ctx, email, workspaceID)
	if err != nil {
		ws.cacheAccessResult(cacheKey, false, "")
		return false, "", nil
	}

	if role == "" {
		ws.cacheAccessResult(cacheKey, false, "")
		return false, "", nil
	}

	ws.cacheAccessResult(cacheKey, true, role)
	return true, role, nil
}

// getUserWorkspaceRole gets a user's role in a workspace from Firestore.
func (ws *WorkspaceService) getUserWorkspaceRole(ctx context.Context, email, workspaceID string) (string, error) {
	// Encode workspace ID for Firestore document path
	encodedWorkspaceID := EncodeDocID(workspaceID)
	docPath := fmt.Sprintf("workspace_access/%s/users/%s", encodedWorkspaceID, email)
	doc, err := ws.firestore.Doc(docPath).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil // No access
		}
		return "", err
	}

	data := doc.Data()
	role, _ := data["role"].(string)
	return role, nil
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

// GetWorkspaceByPath maps a file path to a workspace ID using configured mappings.
// Returns the default workspace if no mapping matches.
func (ws *WorkspaceService) GetWorkspaceByPath(path string) string {
	if ws.config == nil || len(ws.config.Mappings) == 0 {
		return ws.getDefaultWorkspace()
	}

	// Normalize path separators
	normalizedPath := filepath.ToSlash(path)

	for _, mapping := range ws.config.Mappings {
		if matchPathPattern(mapping.Pattern, normalizedPath) {
			return mapping.Workspace
		}
	}

	return ws.getDefaultWorkspace()
}

// matchPathPattern checks if a path matches a glob-like pattern.
// Supports * for single directory and ** for multiple directories.
func matchPathPattern(pattern, path string) bool {
	// Handle exact match
	if pattern == path {
		return true
	}

	// Handle wildcard patterns
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	return matchParts(patternParts, pathParts)
}

// matchParts recursively matches pattern parts against path parts.
func matchParts(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}

	if pattern[0] == "**" {
		// ** matches zero or more directories
		if len(pattern) == 1 {
			return true // ** at end matches everything
		}
		// Try matching rest of pattern at each position
		for i := 0; i <= len(path); i++ {
			if matchParts(pattern[1:], path[i:]) {
				return true
			}
		}
		return false
	}

	if len(path) == 0 {
		return false
	}

	if pattern[0] == "*" || pattern[0] == path[0] {
		return matchParts(pattern[1:], path[1:])
	}

	// Check if pattern part contains wildcard
	if strings.Contains(pattern[0], "*") {
		if matchWildcard(pattern[0], path[0]) {
			return matchParts(pattern[1:], path[1:])
		}
	}

	return false
}

// matchWildcard matches a pattern with * wildcards against a string.
func matchWildcard(pattern, s string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return s == ""
	}
	if s == "" {
		return pattern == "*"
	}

	// Simple implementation: split on * and check contains
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}

	// First part must be prefix
	if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
		return false
	}

	// Last part must be suffix
	if parts[len(parts)-1] != "" && !strings.HasSuffix(s, parts[len(parts)-1]) {
		return false
	}

	// Middle parts must appear in order
	remaining := s
	if parts[0] != "" {
		remaining = remaining[len(parts[0]):]
	}
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(remaining, parts[i])
		if idx == -1 {
			return false
		}
		remaining = remaining[idx+len(parts[i]):]
	}

	return true
}

// getDefaultWorkspace returns the configured default workspace or "public".
func (ws *WorkspaceService) getDefaultWorkspace() string {
	if ws.config != nil && ws.config.DefaultWorkspace != "" {
		return ws.config.DefaultWorkspace
	}
	return "public"
}

// cacheAccessResult caches an access check result.
func (ws *WorkspaceService) cacheAccessResult(key string, hasAccess bool, role string) {
	ws.mutex.Lock()
	ws.cache.access[key] = &cachedWorkspaceAccess{
		hasAccess: hasAccess,
		role:      role,
		expiresAt: time.Now().Add(ws.cache.ttl),
	}
	ws.mutex.Unlock()
}

// GetAccessibleWorkspaceIDs returns a list of workspace IDs the user can access.
// Useful for SQL IN clauses when filtering queries.
func (ws *WorkspaceService) GetAccessibleWorkspaceIDs(ctx context.Context, email string) ([]string, error) {
	workspaces, err := ws.ListAccessibleWorkspaces(ctx, email)
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(workspaces))
	for i, w := range workspaces {
		ids[i] = w.ID
	}
	return ids, nil
}

// CreateWorkspace creates a new workspace in Firestore.
func (ws *WorkspaceService) CreateWorkspace(ctx context.Context, workspace *Workspace) error {
	if workspace.ID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if workspace.Name == "" {
		workspace.Name = workspace.ID
	}
	workspace.CreatedAt = time.Now()

	// Encode workspace ID for Firestore document ID
	docID := EncodeDocID(workspace.ID)
	_, err := ws.firestore.Collection("workspaces").Doc(docID).Set(ctx, workspace)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Invalidate cache
	ws.mutex.Lock()
	delete(ws.cache.workspaces, workspace.ID)
	ws.mutex.Unlock()

	return nil
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

	// Invalidate cache
	cacheKey := fmt.Sprintf("%s:%s", email, workspaceID)
	ws.mutex.Lock()
	delete(ws.cache.access, cacheKey)
	ws.mutex.Unlock()

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

	// Invalidate cache
	cacheKey := fmt.Sprintf("%s:%s", email, workspaceID)
	ws.mutex.Lock()
	delete(ws.cache.access, cacheKey)
	ws.mutex.Unlock()

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
	// Clear all access cache entries for this workspace (pattern: *:workspaceID)
	for key := range ws.cache.access {
		if strings.HasSuffix(key, ":"+workspaceID) {
			delete(ws.cache.access, key)
		}
	}
	ws.mutex.Unlock()

	return nil
}

// InvalidateCache clears all cached data.
func (ws *WorkspaceService) InvalidateCache() {
	ws.mutex.Lock()
	ws.cache.workspaces = make(map[string]*cachedWorkspace)
	ws.cache.access = make(map[string]*cachedWorkspaceAccess)
	ws.mutex.Unlock()
}

// CacheStats returns cache statistics.
func (ws *WorkspaceService) CacheStats() (workspaces, access int) {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	return len(ws.cache.workspaces), len(ws.cache.access)
}
