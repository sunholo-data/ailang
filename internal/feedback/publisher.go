// Package feedback wires the public submit_feedback MCP tool to the existing
// ailang-messages topology — Firestore-backed inbox + Pub/Sub notification —
// so submissions appear in `ailang messages list --inbox public-feedback`
// alongside everything else the team triages.
//
// The MCP server runs in Cloud Run with AILANG_STORAGE=gcp,
// AILANG_CLOUD_PROJECT, and roles/pubsub.publisher on ailang-messages
// (granted in ailang-multivac/terraform/cloud_run_mcp.tf). Locally, set
// AILANG_STORAGE=gcp + AILANG_CLOUD_PROJECT to test against a real Firestore.
package feedback

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/messaging"
	"github.com/sunholo-data/ailang/internal/pubsub"
	"github.com/sunholo-data/ailang/internal/storage/firestore"
)

// Categories accepted by submit_feedback. Keep in sync with the AILANG-side
// validator in mcp_tools/feedback.ail.
var allowedCategories = map[string]bool{
	"bug":        true,
	"feature":    true,
	"docs":       true,
	"limitation": true,
}

// Limits — keep in sync with mcp_tools/feedback.ail.
const (
	maxBodyBytes    = 10 * 1024
	maxSnippetBytes = 4 * 1024
	maxTitleBytes   = 256
	maxContactBytes = 512
	maxVersionBytes = 64
	maxPackageBytes = 128
	defaultInbox    = "public-feedback"
	fromAgent       = "mcp-public"
	messageType     = "feedback"
)

// packageRe validates a vendor/name package coordinate. Vendor and name are
// alphanumeric with hyphens or underscores; both are required.
var packageRe = regexp.MustCompile(`^[a-zA-Z0-9_-]+/[a-zA-Z0-9_-]+$`)

// Request is the public-facing input shape for submit_feedback.
type Request struct {
	Title         string
	Body          string
	Category      string
	AILangVersion string
	Snippet       string // optional
	Contact       string // optional

	// Package routes feedback to a specific package's inbox (pkg:<vendor>/<name>)
	// instead of the default public-feedback inbox. Empty = general AILANG feedback.
	// Format: vendor/name (e.g. "sunholo/auth"). Validated against packageRe.
	Package string

	// AutoDispatch hints to the cloud coordinator that the user explicitly
	// authorizes the package's autonomous agent to act on this submission.
	// Default false — feedback files in the inbox for human/agent triage.
	// Today, package agents use pkg-update.md (release-sync template); a
	// dedicated pkg-feedback.md template is planned but not yet wired, so
	// auto-dispatch=true on a pkg:* inbox today would trigger a release-sync
	// workflow on a bug report (wrong). Surface as an attribute on the
	// Pub/Sub notification so the coordinator can filter when ready.
	AutoDispatch bool
}

// Result is what we hand back to the agent.
type Result struct {
	TicketID string `json:"ticket_id"`
	QueuedAt string `json:"queued_at"`
	Status   string `json:"status"`
}

// FieldError is a structured validation error so the MCP tool can return
// {error, field} envelopes the AILANG-side schema documents.
type FieldError struct {
	Code   string
	Field  string
	Detail string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s (field=%s): %s", e.Code, e.Field, e.Detail)
}

// Publisher holds the lazily-initialized clients.
type Publisher struct {
	store    messaging.MessageStore
	notifier *pubsub.Publisher
	pubsub   *pubsub.Client
}

var (
	singletonOnce sync.Once
	singleton     *Publisher
	singletonErr  error
)

// Get returns the process-wide Publisher, initializing it on first use.
// Returns the same error on every subsequent call if init failed — caller
// should surface it as a 5xx-equivalent error envelope, not retry forever.
func Get(ctx context.Context) (*Publisher, error) {
	singletonOnce.Do(func() {
		singleton, singletonErr = newPublisher(ctx)
	})
	return singleton, singletonErr
}

func newPublisher(ctx context.Context) (*Publisher, error) {
	storage := os.Getenv("AILANG_STORAGE")
	if storage != "gcp" {
		return nil, fmt.Errorf("feedback publisher requires AILANG_STORAGE=gcp (got %q); local SQLite mode is not supported for the public feedback channel", storage)
	}
	projectID := os.Getenv("AILANG_CLOUD_PROJECT")
	if projectID == "" {
		return nil, errors.New("AILANG_CLOUD_PROJECT must be set for feedback publisher")
	}

	fsClient, err := firestore.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("firestore client: %w", err)
	}
	store := firestore.NewMessagingStore(fsClient)

	prefix := os.Getenv("AILANG_TOPIC_PREFIX")
	if prefix == "" {
		prefix = pubsub.DefaultTopicPrefix
	}
	psClient, err := pubsub.NewClient(ctx, projectID, prefix)
	if err != nil {
		return nil, fmt.Errorf("pubsub client: %w", err)
	}
	notifier := pubsub.NewPublisher(psClient)

	return &Publisher{
		store:    store,
		notifier: notifier,
		pubsub:   psClient,
	}, nil
}

// Validate runs the input checks without touching GCP. Useful for callers
// that want to validate before paying the publisher init cost (e.g. the
// MCP tool handler returns validation errors even when AILANG_STORAGE != gcp).
func Validate(req Request) error {
	return validate(req)
}

// Submit validates the request, writes the message to Firestore, and
// publishes a notification on the ailang-messages topic. Returns ticket_id
// (the Firestore message_id) on success.
//
// Validation errors return *FieldError so the caller can convert to a
// structured {error, field} JSON envelope. Publish/store errors are wrapped
// generically.
func (p *Publisher) Submit(ctx context.Context, req Request) (*Result, error) {
	if err := validate(req); err != nil {
		return nil, err
	}

	// message_id = "fb_" + 16 random hex chars (8 bytes). Doubles as the
	// human-facing ticket_id we return to the caller.
	messageID, err := newID("fb_")
	if err != nil {
		return nil, fmt.Errorf("generate message id: %w", err)
	}

	// Resolve target inbox. Default = public-feedback (general AILANG feedback,
	// triaged manually). When package is set, route to pkg:<vendor>/<name>
	// where the autonomous package agent is watching (per
	// ailang-multivac/config/config.cloud.yaml).
	targetInbox := defaultInbox
	if req.Package != "" {
		targetInbox = "pkg:" + req.Package
	}

	now := time.Now().UTC()
	msg := &messaging.InboxMessage{
		MessageID:   messageID,
		ToInbox:     targetInbox,
		FromAgent:   fromAgent,
		Title:       req.Title,
		Payload:     formatBody(req),
		Status:      messaging.InboxStatusUnread,
		MessageType: messageType,
		Category:    req.Category,
		CreatedAt:   now,
	}

	if err := p.store.InsertInboxMessageWithContext(ctx, msg); err != nil {
		return nil, fmt.Errorf("store insert: %w", err)
	}

	// Pub/Sub notification — coordinator push subscription picks this up and
	// fans it out to the dashboard/laptop subscribers. AutoDispatch is
	// surfaced as a category prefix so the existing attribute schema (which
	// has no dispatch field) doesn't need extending; coordinator can grep
	// for "auto:" prefix when the pkg-feedback template lands.
	dispatchedCategory := req.Category
	if req.AutoDispatch {
		dispatchedCategory = "auto:" + req.Category
	}
	notifyErr := p.notifier.PublishMessage(ctx, messageID, pubsub.MessageAttributes{
		Inbox:       targetInbox,
		FromAgent:   fromAgent,
		Category:    dispatchedCategory,
		MessageType: messageType,
	})
	if notifyErr != nil {
		// The store write succeeded — submission IS visible to anyone reading
		// the inbox directly. Notification failure just means the dashboard
		// won't auto-refresh. Log and continue rather than fail the whole call.
		fmt.Fprintf(os.Stderr, "feedback: pubsub notify failed for %s (store insert succeeded): %v\n", messageID, notifyErr)
	}

	return &Result{
		TicketID: messageID,
		QueuedAt: now.Format(time.RFC3339),
		Status:   "queued",
	}, nil
}

// formatBody renders the user-facing message body, including the optional
// snippet and contact fields plus the reporter's CLI version for triage.
func formatBody(req Request) string {
	var b strings.Builder
	b.WriteString(req.Body)
	b.WriteString("\n\n---\n")
	fmt.Fprintf(&b, "ailang_version: %s\n", req.AILangVersion)
	if req.Contact != "" {
		fmt.Fprintf(&b, "contact: %s\n", req.Contact)
	}
	if req.Snippet != "" {
		b.WriteString("\nSnippet:\n```\n")
		b.WriteString(req.Snippet)
		b.WriteString("\n```\n")
	}
	return b.String()
}

func validate(req Request) error {
	switch {
	case strings.TrimSpace(req.Title) == "":
		return &FieldError{Code: "missing_field", Field: "title", Detail: "title is required"}
	case len(req.Title) > maxTitleBytes:
		return &FieldError{Code: "title_too_large", Field: "title", Detail: fmt.Sprintf("title must be <= %d bytes", maxTitleBytes)}
	case strings.TrimSpace(req.Body) == "":
		return &FieldError{Code: "missing_field", Field: "body", Detail: "body is required"}
	case len(req.Body) > maxBodyBytes:
		return &FieldError{Code: "body_too_large", Field: "body", Detail: fmt.Sprintf("body must be <= %d bytes", maxBodyBytes)}
	case len(req.Snippet) > maxSnippetBytes:
		return &FieldError{Code: "snippet_too_large", Field: "snippet", Detail: fmt.Sprintf("snippet must be <= %d bytes", maxSnippetBytes)}
	case len(req.Contact) > maxContactBytes:
		return &FieldError{Code: "contact_too_large", Field: "contact", Detail: fmt.Sprintf("contact must be <= %d bytes", maxContactBytes)}
	case len(req.AILangVersion) > maxVersionBytes:
		return &FieldError{Code: "ailang_version_too_large", Field: "ailang_version", Detail: fmt.Sprintf("ailang_version must be <= %d bytes", maxVersionBytes)}
	case !allowedCategories[req.Category]:
		return &FieldError{Code: "invalid_category", Field: "category", Detail: "category must be one of: bug, feature, docs, limitation"}
	case len(req.Package) > maxPackageBytes:
		return &FieldError{Code: "package_too_large", Field: "package", Detail: fmt.Sprintf("package must be <= %d bytes", maxPackageBytes)}
	case req.Package != "" && !packageRe.MatchString(req.Package):
		return &FieldError{Code: "invalid_package", Field: "package", Detail: "package must look like vendor/name (e.g. sunholo/auth) — alphanumerics, hyphens, underscores only"}
	}
	return nil
}

func newID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(buf), nil
}
