package coordinator

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// handleGitHubWebhook handles incoming GitHub webhook events.
// Replaces polling-based ApprovalWatcher and GitHub sync in cloud mode.
//
// Supported events:
//   - issues + labeled → detects approval labels, calls handleEvent (same as polling)
//   - issues + opened  → imports issue as message (replaces runGitHubSync)
//   - ping             → responds 200 (GitHub connectivity check)
//
// Authentication: HMAC-SHA256 via GITHUB_WEBHOOK_SECRET env var.
// Always returns 200 to GitHub (except signature failures) to prevent retries.
func (d *Daemon) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Read body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		d.logger.Printf("Webhook: failed to read body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate HMAC-SHA256 signature
	secret := os.Getenv("GITHUB_WEBHOOK_SECRET")
	if secret != "" {
		sig := r.Header.Get("X-Hub-Signature-256")
		if !verifyWebhookSignature(body, sig, secret) {
			d.logger.Printf("Webhook: invalid signature")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	event := r.Header.Get("X-GitHub-Event")
	switch event {
	case "ping":
		d.logger.Println("Webhook: ping received")
		w.WriteHeader(http.StatusOK)
		return

	case "issues":
		d.handleWebhookIssueEvent(w, body)
		return

	default:
		d.logger.Printf("Webhook: ignoring event type %q", event)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// webhookIssuePayload is the subset of GitHub's issues webhook payload we need.
type webhookIssuePayload struct {
	Action string `json:"action"`
	Label  struct {
		Name string `json:"name"`
	} `json:"label"`
	Issue struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Body   string `json:"body"`
	} `json:"issue"`
	Sender struct {
		Login string `json:"login"`
		Type  string `json:"type"` // "User" or "Bot"
	} `json:"sender"`
	Repository struct {
		FullName string `json:"full_name"` // e.g., "sunholo-data/ailang"
	} `json:"repository"`
}

// handleWebhookIssueEvent processes issues webhook events (labeled, opened, etc.).
func (d *Daemon) handleWebhookIssueEvent(w http.ResponseWriter, body []byte) {
	var payload webhookIssuePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		d.logger.Printf("Webhook: failed to parse issues payload: %v", err)
		w.WriteHeader(http.StatusOK) // Ack to prevent retry
		return
	}

	// Skip bot-generated events to avoid infinite loops
	// (coordinator adds its own labels, which would trigger another webhook)
	if payload.Sender.Type == "Bot" {
		d.logger.Printf("Webhook: skipping bot event from %s", payload.Sender.Login)
		w.WriteHeader(http.StatusOK)
		return
	}

	switch payload.Action {
	case "labeled":
		d.handleWebhookLabeled(payload)
	case "opened":
		d.handleWebhookOpened(payload)
	default:
		d.logger.Printf("Webhook: ignoring issues action %q for #%d", payload.Action, payload.Issue.Number)
	}

	w.WriteHeader(http.StatusOK)
}

// handleWebhookLabeled processes a "labeled" action on an issue.
// This replaces the ApprovalWatcher polling loop for cloud mode.
func (d *Daemon) handleWebhookLabeled(payload webhookIssuePayload) {
	label := payload.Label.Name
	issueNum := payload.Issue.Number
	d.logger.Printf("Webhook: issue #%d labeled %q by %s", issueNum, label, payload.Sender.Login)

	if d.approvalWatcher == nil {
		d.logger.Printf("Webhook: no approval watcher configured, ignoring label event")
		return
	}

	// Look up task by issue number from the approval watcher's in-memory map
	d.approvalWatcher.mu.Lock()
	taskID, found := d.approvalWatcher.watchedIssues[issueNum]
	d.approvalWatcher.mu.Unlock()

	if !found {
		d.logger.Printf("Webhook: no task found for issue #%d (may not be tracked)", issueNum)
		return
	}

	// Determine event type from label
	eventType := labelToEventType(label, d.approvalWatcher)
	if eventType == "" {
		d.logger.Printf("Webhook: label %q on issue #%d is not an approval label", label, issueNum)
		return
	}

	event := &ApprovalEvent{
		TaskID:      taskID,
		IssueNumber: issueNum,
		Label:       label,
		EventType:   eventType,
		Channel:     "github-webhook",
	}

	d.logger.Printf("Webhook: processing approval %s for task %s (issue #%d)", eventType, taskID, issueNum)
	d.approvalWatcher.handleEvent(d.ctx, event)
}

// handleWebhookOpened processes a new issue being opened.
// This replaces the periodic GitHub sync for cloud mode.
func (d *Daemon) handleWebhookOpened(payload webhookIssuePayload) {
	d.logger.Printf("Webhook: new issue #%d opened: %s", payload.Issue.Number, payload.Issue.Title)

	if d.msgStore == nil {
		d.logger.Printf("Webhook: no message store, skipping import")
		return
	}

	// Route the issue to the appropriate inbox based on labels
	// For opened events, the issue doesn't have the label in the event payload,
	// so route to the configured default target inbox
	targetInbox := "design-doc-creator"
	if d.coordConfig != nil && d.coordConfig.GitHubSync != nil {
		repo := payload.Repository.FullName
		repoConfig := d.findRepoConfig(repo)
		if repoConfig != nil && repoConfig.TargetInbox != "" {
			targetInbox = repoConfig.TargetInbox
		}
	}

	// Import the issue as a message using the sync mechanism
	// We reuse syncRepoIssues which calls `ailang messages import-github`
	// This is simpler and more robust than duplicating the import logic.
	repo := payload.Repository.FullName
	if repo == "" {
		d.logger.Printf("Webhook: no repository in payload, skipping import")
		return
	}

	repoConfig := d.findRepoConfig(repo)
	if repoConfig == nil {
		// Create minimal config for ad-hoc import
		repoConfig = &RepoSyncConfig{
			Repo:        repo,
			Enabled:     true,
			TargetInbox: targetInbox,
		}
	}

	d.syncRepoIssues(*repoConfig)

	// Immediately process any new messages into tasks
	if d.msgAdapter != nil {
		if err := d.pollAndProcessTasks(); err != nil {
			d.logger.Printf("Webhook: poll error after import: %v", err)
		}
	}
	if err := d.executeTaskQueue(); err != nil {
		d.logger.Printf("Webhook: dispatch error after import: %v", err)
	}
}

// labelToEventType maps a GitHub label to an ApprovalEventType.
// Checks both legacy hardcoded labels and config-driven labels.
func labelToEventType(label string, watcher *ApprovalWatcher) ApprovalEventType {
	// Check legacy hardcoded labels
	switch label {
	case "design-approved":
		return ApprovalEventDesign
	case "sprint-approved":
		return ApprovalEventSprint
	case "merge-approved":
		return ApprovalEventMerge
	case LabelNeedsRevision:
		return ApprovalEventRevision
	}

	// Check config-driven labels
	if agent := watcher.GetAgentByLabel(label); agent != nil {
		// Config-driven labels are treated as their matching event type.
		// The approval watcher's handleEvent will route via customHandlers.
		return ApprovalEventType(label)
	}

	return ""
}

// verifyWebhookSignature validates the HMAC-SHA256 signature from GitHub.
// sig format: "sha256=<hex-encoded HMAC>"
func verifyWebhookSignature(body []byte, sig, secret string) bool {
	if sig == "" || !strings.HasPrefix(sig, "sha256=") {
		return false
	}

	expected := computeHMACSHA256(body, []byte(secret))
	actual, err := hex.DecodeString(sig[len("sha256="):])
	if err != nil {
		return false
	}

	return hmac.Equal(expected, actual)
}

// computeHMACSHA256 computes the HMAC-SHA256 of a message with the given key.
func computeHMACSHA256(message, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	return mac.Sum(nil)
}

// FormatWebhookSetupCommand returns the gh CLI command to configure a webhook
// for the given repo and coordinator URL.
func FormatWebhookSetupCommand(repo, coordinatorURL, secret string) string {
	return fmt.Sprintf(
		"gh webhook create --repo %s --events issues --url %q --secret %q",
		repo, coordinatorURL+"/github/webhook", secret,
	)
}
