// Package accessapproval implements the explicit-local, durable access approval
// lifecycle. The CLI is a trusted operator boundary, never a worker capability.
// Only the fixed dummy operation is registered; production secret resolution is
// intentionally unconfigured. Legacy synchronous secret APIs are unchanged.
package accessapproval

import "fmt"

// Input is the JSON command protocol. Handlers validate fields for each command.
type Input struct {
	ID              string         `json:"id,omitempty"`
	Owner           string         `json:"owner,omitempty"`
	Worker          string         `json:"worker,omitempty"`
	Task            string         `json:"task,omitempty"`
	Attempt         string         `json:"attempt,omitempty"`
	Operation       string         `json:"operation,omitempty"`
	ResourceAlias   string         `json:"resource_alias,omitempty"`
	Purpose         string         `json:"purpose,omitempty"`
	Arguments       map[string]any `json:"arguments,omitempty"`
	ArgumentsDigest string         `json:"arguments_digest,omitempty"`
	PolicyVersion   string         `json:"policy_version,omitempty"`
	Version         int            `json:"version,omitempty"`
	ExpiresAt       int64          `json:"expires_at,omitempty"`
	Actor           string         `json:"actor,omitempty"`
	Decision        string         `json:"decision,omitempty"`
	IdempotencyKey  string         `json:"idempotency_key,omitempty"`
	ClaimID         string         `json:"claim_id,omitempty"`
	Reason          string         `json:"reason,omitempty"`
	Limit           int            `json:"limit,omitempty"`
	Offset          int            `json:"offset,omitempty"`
	Status          string         `json:"status,omitempty"`
}

// Request is the canonical v1 record. All times are UTC Unix seconds. Version
// binds immutable request content; Revision is the durable mutation counter.
type Request struct {
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Owner           string         `json:"owner"`
	Worker          string         `json:"worker"`
	Task            string         `json:"task"`
	Attempt         string         `json:"attempt"`
	Operation       string         `json:"operation"`
	ResourceAlias   string         `json:"resource_alias"`
	Purpose         string         `json:"purpose"`
	Arguments       map[string]any `json:"arguments"`
	ArgumentsDigest string         `json:"arguments_digest"`
	PolicyVersion   string         `json:"policy_version"`
	Version         int            `json:"version"`
	Revision        int            `json:"revision"`
	CreatedAt       int64          `json:"created_at"`
	ExpiresAt       int64          `json:"expires_at"`
	Decision        string         `json:"decision"`
	DecisionBy      string         `json:"decision_by,omitempty"`
	DecisionAt      int64          `json:"decision_at,omitempty"`
	DecisionKey     string         `json:"decision_key,omitempty"`
	ClaimUntil      int64          `json:"claim_until,omitempty"`
	CancelReason    string         `json:"cancel_reason,omitempty"`
	Execution       Execution      `json:"execution"`
}

type Execution struct {
	State       string         `json:"state"`
	ClaimID     string         `json:"claim_id,omitempty"`
	ClaimKey    string         `json:"claim_key,omitempty"`
	ClaimUntil  int64          `json:"claim_until,omitempty"`
	StartedAt   int64          `json:"started_at,omitempty"`
	CompletedAt int64          `json:"completed_at,omitempty"`
	Result      map[string]any `json:"result,omitempty"`
	Error       string         `json:"error,omitempty"`
}

type List struct {
	Requests []*Request `json:"requests"`
	Total    int        `json:"total"`
	Limit    int        `json:"limit"`
	Offset   int        `json:"offset"`
}

// Error is deliberately safe for machine-readable stderr (no secret values).
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string        { return e.Code + ": " + e.Message }
func fail(code, message string) error { return &Error{Code: code, Message: message} }
func conflict(message string) error   { return fail("conflict", message) }
func validateText(name, value string, max int) error {
	if value == "" || len(value) > max {
		return fail("invalid_input", fmt.Sprintf("%s must contain 1..%d bytes", name, max))
	}
	return nil
}
