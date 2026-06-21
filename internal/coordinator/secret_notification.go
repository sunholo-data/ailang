package coordinator

import (
	"fmt"
	"net/url"
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
	body := fmt.Sprintf("Agent %s requests %s", req.AgentID, req.SecretRef)
	if req.SecretPurpose != "" {
		body += " — " + req.SecretPurpose
	}

	return notify.Notification{
		Title:     fmt.Sprintf("Secret requested: %s", req.SecretRef),
		Body:      body,
		EventType: "pending_approval",
		Actions: []notify.NotificationAction{
			{Label: "Approve", URL: base + "/approve?token=" + url.QueryEscape(approveTok), Method: "POST"},
			{Label: "Deny", URL: base + "/reject?token=" + url.QueryEscape(rejectTok), Method: "POST"},
		},
	}, nil
}
