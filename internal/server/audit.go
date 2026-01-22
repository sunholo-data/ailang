// audit.go provides audit logging functionality for tracking important operations.
package server

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sunholo/ailang/internal/server/auth"
)

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	ID          string                 `firestore:"id"`
	UserID      string                 `firestore:"user_id"`
	UserEmail   string                 `firestore:"user_email"`
	WorkspaceID string                 `firestore:"workspace_id"`
	Action      string                 `firestore:"action"`
	Resource    string                 `firestore:"resource"`
	ResourceID  string                 `firestore:"resource_id"`
	Status      string                 `firestore:"status"` // "success", "failure"
	Details     map[string]interface{} `firestore:"details"`
	Timestamp   time.Time              `firestore:"timestamp"`
	IP          string                 `firestore:"ip_address"`
	UserAgent   string                 `firestore:"user_agent"`
}

// AuditLogger provides audit logging functionality.
type AuditLogger struct {
	firestore *firestore.Client
}

// NewAuditLogger creates a new AuditLogger.
func NewAuditLogger(fs *firestore.Client) *AuditLogger {
	return &AuditLogger{
		firestore: fs,
	}
}

// LogAction logs an audit entry for a user action.
func (al *AuditLogger) LogAction(ctx context.Context, user *auth.User, action, resource, resourceID string, details map[string]interface{}) error {
	if al.firestore == nil {
		// If Firestore is not available, skip logging
		return nil
	}

	entry := AuditLogEntry{
		ID:          fmt.Sprintf("%d-%s-%s", time.Now().UnixNano(), user.FirebaseUID, resourceID),
		UserID:      user.FirebaseUID,
		UserEmail:   user.Email,
		WorkspaceID: user.WorkspaceID,
		Action:      action,
		Resource:    resource,
		ResourceID:  resourceID,
		Status:      "success",
		Details:     details,
		Timestamp:   time.Now(),
	}

	// Add to Firestore
	_, err := al.firestore.Collection("audit_logs").Doc(entry.ID).Set(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// LogApproval logs an approval action.
func (al *AuditLogger) LogApproval(ctx context.Context, user *auth.User, taskID, action string) error {
	return al.LogAction(ctx, user, "approval", "task", taskID, map[string]interface{}{
		"approval_action": action, // "approved" or "rejected"
	})
}

// LogRoleChange logs a role change for a user.
func (al *AuditLogger) LogRoleChange(ctx context.Context, user *auth.User, targetUserID, oldRole, newRole string) error {
	return al.LogAction(ctx, user, "role_change", "user", targetUserID, map[string]interface{}{
		"old_role": oldRole,
		"new_role": newRole,
	})
}

// LogWorkspaceAccess logs workspace access events.
func (al *AuditLogger) LogWorkspaceAccess(ctx context.Context, user *auth.User, workspaceID string) error {
	return al.LogAction(ctx, user, "workspace_access", "workspace", workspaceID, map[string]interface{}{
		"role": user.Role,
	})
}

// LogFailedAttempt logs a failed authentication or authorization attempt.
func (al *AuditLogger) LogFailedAttempt(ctx context.Context, email, reason string) error {
	if al.firestore == nil {
		return nil
	}

	entry := AuditLogEntry{
		ID:        fmt.Sprintf("%d-failed-%s", time.Now().UnixNano(), email),
		UserEmail: email,
		Action:    "authentication_failed",
		Resource:  "auth",
		Status:    "failure",
		Details: map[string]interface{}{
			"reason": reason,
		},
		Timestamp: time.Now(),
	}

	_, err := al.firestore.Collection("audit_logs").Doc(entry.ID).Set(ctx, entry)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs for a user or workspace.
func (al *AuditLogger) GetAuditLogs(ctx context.Context, workspaceID string, limit int) ([]*AuditLogEntry, error) {
	if al.firestore == nil {
		return []*AuditLogEntry{}, nil
	}

	query := al.firestore.Collection("audit_logs").
		Where("workspace_id", "==", workspaceID).
		OrderBy("timestamp", firestore.Desc).
		Limit(limit)

	iter := query.Documents(ctx)
	defer iter.Stop()

	var entries []*AuditLogEntry
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}

		var entry AuditLogEntry
		if err := doc.DataTo(&entry); err != nil {
			continue
		}
		entries = append(entries, &entry)
	}

	return entries, nil
}
