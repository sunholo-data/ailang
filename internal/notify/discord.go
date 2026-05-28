package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// discordContentLimit is Discord's per-message character cap for webhook content.
const discordContentLimit = 2000

// httpDoer is the slice of *http.Client the Discord channel needs; tests
// substitute a fake so no network is required.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// DiscordChannel delivers notifications to a Discord incoming webhook. Outbound
// only needs the webhook URL (no bot, no signature) — a single authenticated
// POST per chunk. Mirrors Aitana's channels/discord.py::send.
type DiscordChannel struct {
	webhookURL string
	http       httpDoer
}

// NewDiscordChannel builds a Discord channel for the given incoming-webhook URL.
func NewDiscordChannel(webhookURL string) *DiscordChannel {
	return &DiscordChannel{webhookURL: webhookURL, http: http.DefaultClient}
}

// Name implements Channel.
func (c *DiscordChannel) Name() string { return "discord" }

// Send renders n and POSTs it to the webhook, chunked at Discord's 2000-char
// limit. A transport error or any non-2xx response yields a typed error so the
// caller can dead-letter the event.
func (c *DiscordChannel) Send(ctx context.Context, n Notification) error {
	for _, chunk := range chunkMessage(renderDiscord(n), discordContentLimit) {
		if err := c.post(ctx, chunk); err != nil {
			return err
		}
	}
	return nil
}

func (c *DiscordChannel) post(ctx context.Context, content string) error {
	body, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return fmt.Errorf("discord send: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("discord send: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("discord send: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord send: unexpected status %d", resp.StatusCode)
	}
	return nil
}

// renderDiscord turns a Notification into Discord message content. Title is bold;
// Subtitle, Body, and URL are appended when present. macOS-only fields (Sound,
// Group) are ignored — the "not every channel honours every field" rule.
func renderDiscord(n Notification) string {
	var b strings.Builder
	if n.Title != "" {
		b.WriteString("**" + n.Title + "**")
	}
	if n.Subtitle != "" {
		b.WriteString("\n" + n.Subtitle)
	}
	if n.Body != "" {
		b.WriteString("\n" + n.Body)
	}
	if n.URL != "" {
		b.WriteString("\n" + n.URL)
	}
	s := strings.TrimPrefix(b.String(), "\n")
	if s == "" {
		s = "(no content)"
	}
	return s
}
