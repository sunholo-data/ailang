package notify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DefaultNtfyEventTypes restricts the ntfy push channel to approval requests
// only — this is a dedicated, actionable "approve from your phone" channel, not
// a general feed. Override with SetEventTypes (nil/empty = accept everything).
var DefaultNtfyEventTypes = []string{"pending_approval"}

// NtfyChannel delivers actionable notifications to an ntfy server (self-hosted
// on Cloud Run). It renders Notification.Actions as ntfy action buttons so the
// operator can Approve/Deny from the lock screen. Outbound only: a single POST
// to <serverURL>/<topic> per notification.
//
// It is a remote/authoritative channel (it does NOT implement LocalChannel), so
// the fan-out treats a failed push as a hard error and redelivers.
type NtfyChannel struct {
	serverURL  string // e.g. https://ailang-ntfy.example.run.app
	topic      string // ntfy topic, e.g. "ailang-approvals"
	authToken  string // optional bearer token for the ntfy server ("" = none)
	http       httpDoer
	eventTypes []string
}

// NewNtfyChannel builds an ntfy channel for serverURL/topic, restricted to
// approval events by default.
func NewNtfyChannel(serverURL, topic, authToken string) *NtfyChannel {
	return &NtfyChannel{
		serverURL:  strings.TrimRight(serverURL, "/"),
		topic:      topic,
		authToken:  authToken,
		http:       http.DefaultClient,
		eventTypes: append([]string{}, DefaultNtfyEventTypes...),
	}
}

// Name implements Channel.
func (c *NtfyChannel) Name() string { return "ntfy" }

// SetEventTypes overrides the allow-list. Pass nil/empty to accept every event.
func (c *NtfyChannel) SetEventTypes(types []string) { c.eventTypes = types }

// Accepts implements EventFilter — empty allow-list accepts everything.
func (c *NtfyChannel) Accepts(eventType string) bool {
	if len(c.eventTypes) == 0 {
		return true
	}
	for _, t := range c.eventTypes {
		if t == eventType {
			return true
		}
	}
	return false
}

// Send renders n and POSTs it to <serverURL>/<topic>. Title and action buttons
// are conveyed via ntfy headers; the body is the request payload.
func (c *NtfyChannel) Send(ctx context.Context, n Notification) error {
	url := c.serverURL + "/" + c.topic
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(n.Body))
	if err != nil {
		return fmt.Errorf("ntfy send: build request: %w", err)
	}
	if n.Title != "" {
		req.Header.Set("X-Title", n.Title)
	}
	// Approvals are high-priority and tagged for at-a-glance recognition.
	req.Header.Set("X-Priority", "high")
	req.Header.Set("X-Tags", "lock,key")
	if actions := buildNtfyActions(n.Actions); actions != "" {
		req.Header.Set("X-Actions", actions)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy send: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return fmt.Errorf("ntfy send: status %d: %s", resp.StatusCode, strings.TrimSpace(string(snippet)))
	}
	return nil
}

// buildNtfyActions encodes actions as an ntfy X-Actions header value:
//
//	action=http, label=Approve, url=..., method=POST, clear=true; action=http, ...
//
// Returns "" when there are no actions.
func buildNtfyActions(actions []NotificationAction) string {
	if len(actions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(actions))
	for _, a := range actions {
		method := a.Method
		if method == "" {
			method = "POST"
		}
		parts = append(parts, fmt.Sprintf("action=http, label=%s, url=%s, method=%s, clear=true", a.Label, a.URL, method))
	}
	return strings.Join(parts, "; ")
}
