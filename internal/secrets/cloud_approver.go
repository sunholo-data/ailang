package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CloudSecretApprover is a networked secret-approval gate. It implements
// effects.SecretApprover structurally — this package must NOT import
// internal/effects, which imports us (an import cycle); the assignment site in
// cmd/ailang provides the compile-time interface check.
//
// M-SECRET-REMOTE-APPROVAL-WIRING: secret() runs inside agent-executor jobs, a
// separate process from the coordinator that owns the approval authority. So
// the approver attached to a cloud EffContext is a network client of the
// coordinator, not the in-process coordinator.SecretApprovalGate.
//
// On Approve it POSTs an approval request to the coordinator and blocks,
// polling for the decision. The request carries the reference, a purpose, and
// the requesting agent/task — NEVER a resolved value (the value is read only
// after approval, by the resolver). Any denial, timeout, or transport error
// returns a non-nil error so secretRead fails closed (E_SECRET_DENIED).
type CloudSecretApprover struct {
	baseURL   string // coordinator base URL (no trailing slash)
	agentID   string
	taskID    string
	authToken string
	idToken   func(ctx context.Context) (string, error) // mints a bearer when authToken is empty
	http      *http.Client
	deadline  time.Duration // total wait for a decision before failing closed
	poll      time.Duration // interval between status polls
}

// Wire payloads. None of these carry a resolved secret value.
type approvalCreateRequest struct {
	Ref     string `json:"ref"`
	Purpose string `json:"purpose"`
	Agent   string `json:"agent,omitempty"`
	Task    string `json:"task,omitempty"`
}

type approvalCreateResponse struct {
	ID string `json:"id"`
}

type approvalStatusResponse struct {
	Status string `json:"status"` // "pending" | "approved" | "denied"
	Reason string `json:"reason,omitempty"`
}

// CloudApproverOption configures a CloudSecretApprover.
type CloudApproverOption func(*CloudSecretApprover)

// WithApproverIdentity records the requesting agent and task on each request.
func WithApproverIdentity(agentID, taskID string) CloudApproverOption {
	return func(a *CloudSecretApprover) { a.agentID, a.taskID = agentID, taskID }
}

// WithApproverDeadline sets the total time to wait for a decision (default 5m).
func WithApproverDeadline(d time.Duration) CloudApproverOption {
	return func(a *CloudSecretApprover) {
		if d > 0 {
			a.deadline = d
		}
	}
}

// WithApproverPollInterval sets the status poll interval (default 2s).
func WithApproverPollInterval(d time.Duration) CloudApproverOption {
	return func(a *CloudSecretApprover) {
		if d > 0 {
			a.poll = d
		}
	}
}

// WithApproverAuthToken attaches a static bearer token (AILANG_APPROVAL_TOKEN,
// the shared secret the dashboard also holds) to every request. When set it
// wins over WithApproverIDTokenSource.
func WithApproverAuthToken(tok string) CloudApproverOption {
	return func(a *CloudSecretApprover) { a.authToken = tok }
}

// WithApproverIDTokenSource sets the function that mints a bearer (a Google
// ID token whose audience is the dashboard URL) per request when no static
// token is configured. This package only takes the function: the minting
// lives in internal/runner, so the js/wasm build that includes this package
// does not pull in the Google credential stack.
func WithApproverIDTokenSource(mint func(ctx context.Context) (string, error)) CloudApproverOption {
	return func(a *CloudSecretApprover) { a.idToken = mint }
}

// WithApproverHTTPClient overrides the HTTP client (used by tests).
func WithApproverHTTPClient(c *http.Client) CloudApproverOption {
	return func(a *CloudSecretApprover) {
		if c != nil {
			a.http = c
		}
	}
}

// NewCloudSecretApprover builds a networked approver targeting the coordinator
// at baseURL (e.g. https://ailang-dev-coordinator-xxx.run.app).
func NewCloudSecretApprover(baseURL string, opts ...CloudApproverOption) *CloudSecretApprover {
	a := &CloudSecretApprover{
		baseURL:  strings.TrimRight(baseURL, "/"),
		http:     &http.Client{Timeout: 30 * time.Second},
		deadline: 5 * time.Minute,
		poll:     2 * time.Second,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

// Approve requests a human approval for ref and blocks until the operator
// approves (nil), denies (error), the deadline elapses (error), or ctx is
// canceled (error). It never reads or logs the resolved value.
func (a *CloudSecretApprover) Approve(ctx context.Context, ref, purpose string) error {
	id, err := a.createRequest(ctx, ref, purpose)
	if err != nil {
		// Fail closed: if we cannot even register the request, deny.
		return fmt.Errorf("could not request approval for %s: %w", ref, err)
	}

	deadline := time.NewTimer(a.deadline)
	defer deadline.Stop()
	ticker := time.NewTicker(a.poll)
	defer ticker.Stop()

	for {
		// Poll once immediately, then on each tick.
		status, reason, perr := a.pollStatus(ctx, id)
		if perr == nil {
			switch status {
			case "approved":
				return nil
			case "denied", "rejected", "timeout":
				// The coordinator store resolves to "rejected"/"timeout"; accept
				// "denied" too for forward-compatibility. All are fail-closed.
				if reason == "" {
					reason = "denied by operator"
				}
				return fmt.Errorf("secret approval %s for %s: %s", status, ref, reason)
				// any other status (e.g. "pending") falls through to wait
			}
		}
		// Transient poll errors are tolerated until the deadline.

		select {
		case <-ctx.Done():
			return fmt.Errorf("secret approval canceled for %s: %w", ref, ctx.Err())
		case <-deadline.C:
			return fmt.Errorf("secret approval timed out after %s for %s", a.deadline, ref)
		case <-ticker.C:
		}
	}
}

func (a *CloudSecretApprover) createRequest(ctx context.Context, ref, purpose string) (string, error) {
	body, err := json.Marshal(approvalCreateRequest{
		Ref: ref, Purpose: purpose, Agent: a.agentID, Task: a.taskID,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/api/approvals", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	mintErr := a.authorize(ctx, req)

	resp, err := a.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("approval service returned %s%s", resp.Status, authHint(resp.StatusCode, mintErr))
	}
	var created approvalCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", err
	}
	if created.ID == "" {
		return "", fmt.Errorf("coordinator returned an empty approval id")
	}
	return created.ID, nil
}

func (a *CloudSecretApprover) pollStatus(ctx context.Context, id string) (status, reason string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+"/api/approvals/"+url.PathEscape(id), nil)
	if err != nil {
		return "", "", err
	}
	mintErr := a.authorize(ctx, req)

	resp, err := a.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("approval service returned %s%s", resp.Status, authHint(resp.StatusCode, mintErr))
	}
	var s approvalStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return "", "", err
	}
	return s.Status, s.Reason, nil
}

// authorize sets the bearer: the static token if configured, else a minted ID
// token. A mint failure is not fatal here — the request goes out without a
// credential (a dashboard with intake auth off still answers) — but it is
// returned so a 401 can say why there was no credential.
func (a *CloudSecretApprover) authorize(ctx context.Context, req *http.Request) error {
	if a.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.authToken)
		return nil
	}
	if a.idToken == nil {
		return nil
	}
	tok, err := a.idToken(ctx)
	if err != nil {
		return err
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	return nil
}

// authHint explains a 401/403 from the approval service, which otherwise
// reads as an unexplained denial of the secret.
func authHint(code int, mintErr error) string {
	switch code {
	case http.StatusUnauthorized:
		if mintErr != nil {
			return fmt.Sprintf(" (no credential: could not mint an ID token: %v; set AILANG_APPROVAL_TOKEN to the dashboard's shared token)", mintErr)
		}
		return " (credential refused: the dashboard needs this caller's ID token audience in AILANG_APPROVAL_AUDIENCE, or AILANG_APPROVAL_TOKEN to match)"
	case http.StatusForbidden:
		return " (this identity is not on the dashboard's AILANG_APPROVAL_ALLOWED_CALLERS)"
	}
	return ""
}
