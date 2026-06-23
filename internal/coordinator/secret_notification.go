package coordinator

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/approvaltoken"
	"github.com/sunholo-data/ailang/internal/notify"
)

// BuildSecretApprovalNotification renders a pending secret-approval request as
// an actionable notification: Approve/Deny buttons whose URLs target the
// coordinator's approval endpoint and carry single-use, signed, short-TTL
// tokens. Only the op:// reference and purpose appear — never the value.
//
//   - baseURL: the public coordinator base, e.g. "https://ailang-coordinator.run.app"
//   - signer:  mints the action tokens
//   - ttl:     token lifetime (also the practical approval window for the buttons)
func BuildSecretApprovalNotification(req *ApprovalRequest, baseURL string, signer *approvaltoken.Signer, ttl time.Duration) (notify.Notification, error) {
	if req == nil || req.Type != ApprovalTypeSecret {
		return notify.Notification{}, fmt.Errorf("BuildSecretApprovalNotification: not a secret approval request")
	}
	approveTok, err := signer.Mint(req.ID, "approve", ttl)
	if err != nil {
		return notify.Notification{}, fmt.Errorf("mint approve token: %w", err)
	}
	rejectTok, err := signer.Mint(req.ID, "reject", ttl)
	if err != nil {
		return notify.Notification{}, fmt.Errorf("mint reject token: %w", err)
	}

	base := fmt.Sprintf("%s/api/approvals/%s", baseURL, url.PathEscape(req.ID))

	// Multi-line body so the operator sees who/what/why/which-task at a glance.
	agent := req.AgentID
	if agent == "" {
		agent = "An agent"
	}
	lines := []string{
		fmt.Sprintf("Requested by: %s", agent),
		fmt.Sprintf("Secret: %s", req.SecretRef),
	}
	// Only show a purpose when it adds information (it defaults to the ref).
	if req.SecretPurpose != "" && req.SecretPurpose != req.SecretRef {
		lines = append(lines, fmt.Sprintf("Purpose: %s", req.SecretPurpose))
	}
	if req.TaskID != "" {
		lines = append(lines, fmt.Sprintf("Task: %s", req.TaskID))
	}
	lines = append(lines, fmt.Sprintf("Decide within %s.", ttl.Round(time.Minute)))

	return notify.Notification{
		Title:     fmt.Sprintf("Approve secret: %s", req.SecretRef),
		Body:      strings.Join(lines, "\n"),
		EventType: "pending_approval",
		Actions: []notify.NotificationAction{
			{Label: "Approve", URL: base + "/approve?token=" + url.QueryEscape(approveTok), Method: "POST"},
			{Label: "Deny", URL: base + "/reject?token=" + url.QueryEscape(rejectTok), Method: "POST"},
		},
	}, nil
}

// BuildSecretApprovalResolvedNotification renders the value-free confirmation
// push sent AFTER a secret approval is decided. ntfy action buttons can't show
// their own outcome, so this follow-up tells the operator the decision landed.
// It carries no action buttons. decision is "approved" or "rejected".
func BuildSecretApprovalResolvedNotification(ref, decision string) notify.Notification {
	emoji, word := "✅", "Approved"
	if decision == "rejected" {
		emoji, word = "❌", "Denied"
	}
	return notify.Notification{
		Title:     fmt.Sprintf("%s %s: %s", emoji, word, ref),
		Body:      fmt.Sprintf("Secret request %s.", word),
		EventType: "pending_approval", // accepted by NtfyChannel's event filter
	}
}
