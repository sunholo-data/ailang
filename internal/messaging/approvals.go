package messaging

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Approval represents an approval request for effect-gated actions
type Approval struct {
	ID              string    `json:"id"`
	ThreadID        string    `json:"thread_id"`
	ThreadTitle     string    `json:"thread_title,omitempty"` // Title of the associated thread
	InstanceID      string    `json:"instance_id"`
	CreatedAt       time.Time `json:"created_at"`
	EffectDeltaJSON string    `json:"effect_delta_json"`
	Proposal        string    `json:"proposal"`
	Impact          string    `json:"impact"`
	EstimatedCost   float64   `json:"estimated_cost"`
	Status          string    `json:"status"`
	ReviewedBy      string    `json:"reviewed_by,omitempty"`
	ReviewedAt      time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes     string    `json:"review_notes,omitempty"`
	CapabilityToken string    `json:"capability_token,omitempty"`
	TokenExpiresAt  time.Time `json:"token_expires_at,omitempty"`
}

// EffectDelta represents the effect capabilities being requested
type EffectDelta struct {
	CapType     string   `json:"cap_type"`     // e.g., "FS", "IO", "Net"
	Paths       []string `json:"paths"`        // e.g., ["src/", "docs/"]
	BudgetDelta float64  `json:"budget_delta"` // e.g., 0.50 (50 cents)
}

// CapabilityToken represents a signed capability token
type CapabilityToken struct {
	ThreadID   string    `json:"thread_id"`
	InstanceID string    `json:"instance_id"`
	ApprovalID string    `json:"approval_id"`
	Effects    string    `json:"effects"` // JSON-encoded EffectDelta
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Signature  string    `json:"signature"`
}

// CreateApproval creates a new approval request
func (s *Store) CreateApproval(threadID, instanceID string, effectDelta *EffectDelta, proposal, impact string, estimatedCost float64) (*Approval, error) {
	now := time.Now()
	approvalID := fmt.Sprintf("approval_%d_%s", now.UnixMilli(), generateRandomID(12))

	// Serialize effect delta to JSON
	effectJSON, err := json.Marshal(effectDelta)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal effect delta: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO approvals (
			id, thread_id, instance_id, created_at,
			effect_delta_json, proposal, impact, estimated_cost,
			status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, approvalID, threadID, instanceID, now.UnixMilli(),
		string(effectJSON), proposal, impact, estimatedCost,
		"pending")

	if err != nil {
		return nil, fmt.Errorf("failed to create approval: %w", err)
	}

	return &Approval{
		ID:              approvalID,
		ThreadID:        threadID,
		InstanceID:      instanceID,
		CreatedAt:       now,
		EffectDeltaJSON: string(effectJSON),
		Proposal:        proposal,
		Impact:          impact,
		EstimatedCost:   estimatedCost,
		Status:          "pending",
	}, nil
}

// GetApproval retrieves an approval by ID
func (s *Store) GetApproval(approvalID string) (*Approval, error) {
	var approval Approval
	var createdAtMs int64
	var reviewedAtMs sql.NullInt64
	var tokenExpiresAtMs sql.NullInt64
	var reviewedBy, reviewNotes, capabilityToken sql.NullString

	err := s.db.QueryRow(`
		SELECT id, thread_id, instance_id, created_at,
		       effect_delta_json, proposal, impact, estimated_cost,
		       status, reviewed_by, reviewed_at, review_notes,
		       capability_token, token_expires_at
		FROM approvals
		WHERE id = ?
	`, approvalID).Scan(
		&approval.ID, &approval.ThreadID, &approval.InstanceID, &createdAtMs,
		&approval.EffectDeltaJSON, &approval.Proposal, &approval.Impact, &approval.EstimatedCost,
		&approval.Status, &reviewedBy, &reviewedAtMs, &reviewNotes,
		&capabilityToken, &tokenExpiresAtMs,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("approval not found: %s", approvalID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get approval: %w", err)
	}

	approval.CreatedAt = time.UnixMilli(createdAtMs)
	if reviewedBy.Valid {
		approval.ReviewedBy = reviewedBy.String
	}
	if reviewedAtMs.Valid {
		approval.ReviewedAt = time.UnixMilli(reviewedAtMs.Int64)
	}
	if reviewNotes.Valid {
		approval.ReviewNotes = reviewNotes.String
	}
	if capabilityToken.Valid {
		approval.CapabilityToken = capabilityToken.String
	}
	if tokenExpiresAtMs.Valid {
		approval.TokenExpiresAt = time.UnixMilli(tokenExpiresAtMs.Int64)
	}

	return &approval, nil
}

// GetApprovalsByStatus retrieves approvals by status, including thread titles
func (s *Store) GetApprovalsByStatus(status string, limit int) ([]Approval, error) {
	query := `
		SELECT a.id, a.thread_id, COALESCE(t.title, '') as thread_title, a.instance_id, a.created_at,
		       a.effect_delta_json, a.proposal, a.impact, a.estimated_cost,
		       a.status, a.reviewed_by, a.reviewed_at, a.review_notes,
		       a.capability_token, a.token_expires_at
		FROM approvals a
		LEFT JOIN threads t ON a.thread_id = t.id
		WHERE a.status = ?
		ORDER BY a.created_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query approvals: %w", err)
	}
	defer rows.Close()

	// Initialize as empty slice (not nil) so JSON marshals to [] instead of null
	approvals := []Approval{}
	for rows.Next() {
		var approval Approval
		var createdAtMs int64
		var reviewedAtMs sql.NullInt64
		var tokenExpiresAtMs sql.NullInt64
		var reviewedBy, reviewNotes, capabilityToken sql.NullString

		err := rows.Scan(
			&approval.ID, &approval.ThreadID, &approval.ThreadTitle, &approval.InstanceID, &createdAtMs,
			&approval.EffectDeltaJSON, &approval.Proposal, &approval.Impact, &approval.EstimatedCost,
			&approval.Status, &reviewedBy, &reviewedAtMs, &reviewNotes,
			&capabilityToken, &tokenExpiresAtMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}

		approval.CreatedAt = time.UnixMilli(createdAtMs)
		if reviewedBy.Valid {
			approval.ReviewedBy = reviewedBy.String
		}
		if reviewedAtMs.Valid {
			approval.ReviewedAt = time.UnixMilli(reviewedAtMs.Int64)
		}
		if reviewNotes.Valid {
			approval.ReviewNotes = reviewNotes.String
		}
		if capabilityToken.Valid {
			approval.CapabilityToken = capabilityToken.String
		}
		if tokenExpiresAtMs.Valid {
			approval.TokenExpiresAt = time.UnixMilli(tokenExpiresAtMs.Int64)
		}

		approvals = append(approvals, approval)
	}

	return approvals, rows.Err()
}

// ApproveApproval approves an approval request and generates a capability token
func (s *Store) ApproveApproval(approvalID, reviewedBy string, reviewNotes string, tokenDuration time.Duration) error {
	// Get the approval
	approval, err := s.GetApproval(approvalID)
	if err != nil {
		return err
	}

	if approval.Status != "pending" {
		return fmt.Errorf("approval %s is not pending (status: %s)", approvalID, approval.Status)
	}

	// Generate capability token
	token, expiresAt, err := generateCapabilityToken(approval.ThreadID, approval.InstanceID, approvalID, approval.EffectDeltaJSON, tokenDuration)
	if err != nil {
		return fmt.Errorf("failed to generate capability token: %w", err)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE approvals
		SET status = 'approved',
		    reviewed_by = ?,
		    reviewed_at = ?,
		    review_notes = ?,
		    capability_token = ?,
		    token_expires_at = ?
		WHERE id = ?
	`, reviewedBy, now.UnixMilli(), reviewNotes, token, expiresAt.UnixMilli(), approvalID)

	if err != nil {
		return fmt.Errorf("failed to approve approval: %w", err)
	}

	return nil
}

// RejectApproval rejects an approval request
func (s *Store) RejectApproval(approvalID, reviewedBy string, reviewNotes string) error {
	approval, err := s.GetApproval(approvalID)
	if err != nil {
		return err
	}

	if approval.Status != "pending" {
		return fmt.Errorf("approval %s is not pending (status: %s)", approvalID, approval.Status)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		UPDATE approvals
		SET status = 'rejected',
		    reviewed_by = ?,
		    reviewed_at = ?,
		    review_notes = ?
		WHERE id = ?
	`, reviewedBy, now.UnixMilli(), reviewNotes, approvalID)

	if err != nil {
		return fmt.Errorf("failed to reject approval: %w", err)
	}

	return nil
}

// generateCapabilityToken generates an HMAC-signed capability token
func generateCapabilityToken(threadID, instanceID, approvalID, effectJSON string, duration time.Duration) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(duration)

	// Create token payload
	token := CapabilityToken{
		ThreadID:   threadID,
		InstanceID: instanceID,
		ApprovalID: approvalID,
		Effects:    effectJSON,
		IssuedAt:   now,
		ExpiresAt:  expiresAt,
	}

	// Serialize token for signing
	tokenBytes, err := json.Marshal(token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal token: %w", err)
	}

	// Sign with HMAC-SHA256
	secret := getSigningSecret()
	h := hmac.New(sha256.New, secret)
	h.Write(tokenBytes)
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	token.Signature = signature

	// Serialize final token
	signedTokenBytes, err := json.Marshal(token)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal signed token: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signedTokenBytes), expiresAt, nil
}

// VerifyCapabilityToken verifies an HMAC-signed capability token
func VerifyCapabilityToken(tokenString string) (*CapabilityToken, error) {
	// Decode base64
	tokenBytes, err := base64.StdEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token encoding: %w", err)
	}

	// Unmarshal token
	var token CapabilityToken
	if err := json.Unmarshal(tokenBytes, &token); err != nil {
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	// Check expiry
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("token expired at %s", token.ExpiresAt.Format(time.RFC3339))
	}

	// Extract signature
	signature := token.Signature
	token.Signature = "" // Clear for verification

	// Recreate expected signature
	tokenBytesForSig, err := json.Marshal(token)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token for verification: %w", err)
	}

	secret := getSigningSecret()
	h := hmac.New(sha256.New, secret)
	h.Write(tokenBytesForSig)
	expectedSignature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Compare signatures
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Restore signature for return
	token.Signature = signature
	return &token, nil
}

// getSigningSecret returns the HMAC signing secret (from env or generated)
func getSigningSecret() []byte {
	// Try to get from environment first
	if secret := os.Getenv("AILANG_TOKEN_SECRET"); secret != "" {
		return []byte(secret)
	}

	// For development/testing, use a deterministic secret
	// In production, this MUST be set via AILANG_TOKEN_SECRET env var
	return []byte("ailang-dev-secret-change-in-production")
}
