// Package notify fires native macOS notifications via terminal-notifier
// (preferred — supports click actions) or osascript (fallback). On non-Darwin
// systems and when neither binary is on PATH, Notify returns ErrNotifierUnavailable
// so callers can degrade gracefully.
//
// Icon caveat: macOS binds the notification's app identity to the bundle that
// posted it (terminal-notifier itself, or osascript via Script Editor).
// terminal-notifier's `-appIcon` is honored on some macOS versions but
// systematically ignored on recent ones — we don't try to ship a custom icon
// because the only reliable way is to publish a signed AILANG.app bundle that
// registers as a notification source, which is out of scope here.
package notify

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrNotifierUnavailable is returned when neither terminal-notifier nor
// osascript is on PATH. Callers should treat this as a non-fatal degradation
// signal rather than a hard error — log it and continue.
var ErrNotifierUnavailable = errors.New("notify: no notification binary available (install terminal-notifier or run on macOS with osascript)")

// Notification is a single notification payload. Sound and Group are optional;
// URL is honored only on the terminal-notifier path (osascript has no
// click-action equivalent). EventType is opaque to the macOS renderer but is
// read by Registry.SendAll to filter per channel (see EventFilter).
type Notification struct {
	Title     string
	Subtitle  string
	Body      string
	Sound     string // default "Glass" if empty
	Group     string // collapses repeat notifications when shared
	URL       string // click-action; opens URL when notification is clicked (terminal-notifier only)
	EventType string // e.g. "pending_approval", "completed", "failed", "public-feedback", "message"

	// Actions are optional, actionable buttons (e.g. Approve/Deny) rendered by
	// channels that support them (ntfy). Channels that don't (macOS, Discord)
	// ignore this field. Each action's URL should already embed any auth token.
	Actions []NotificationAction
}

// NotificationAction is an actionable button on a notification: a labelled
// HTTP request fired when the user taps it (e.g. POST to an approval endpoint).
type NotificationAction struct {
	Label  string // button text, e.g. "Approve"
	URL    string // request target (must already include any auth token)
	Method string // HTTP method, e.g. "POST" (defaults to "POST" if empty)
}

// runner abstracts exec.LookPath + Cmd.Run so tests can substitute a fake.
// Production code uses execRunner{}. The defaultRunner pointer is swapped in
// tests via withRunner; see macos_test.go.
type runner interface {
	lookPath(name string) (string, error)
	run(name string, args ...string) error
}

type execRunner struct{}

func (execRunner) lookPath(name string) (string, error) { return exec.LookPath(name) }
func (execRunner) run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

var defaultRunner runner = execRunner{}

// Notify fires a single macOS notification. Returns ErrNotifierUnavailable when
// neither backend is on PATH; returns the underlying exec error wrapped if the
// chosen backend fails to run.
func Notify(n Notification) error {
	if _, err := defaultRunner.lookPath("terminal-notifier"); err == nil {
		return notifyTerminalNotifier(n)
	}
	if _, err := defaultRunner.lookPath("osascript"); err == nil {
		return notifyOsascript(n)
	}
	return ErrNotifierUnavailable
}

func notifyTerminalNotifier(n Notification) error {
	sound := n.Sound
	if sound == "" {
		sound = "Glass"
	}
	args := []string{
		"-title", n.Title,
		"-message", n.Body,
		"-sound", sound,
	}
	if n.Subtitle != "" {
		args = append(args, "-subtitle", n.Subtitle)
	}
	if n.Group != "" {
		args = append(args, "-group", n.Group)
	}
	if n.URL != "" {
		args = append(args, "-execute", fmt.Sprintf("open %q", n.URL))
	}
	if err := defaultRunner.run("terminal-notifier", args...); err != nil {
		return fmt.Errorf("notify: terminal-notifier failed: %w", err)
	}
	return nil
}

func notifyOsascript(n Notification) error {
	// AppleScript: display notification "body" with title "title" subtitle "sub" sound name "Glass"
	script := fmt.Sprintf(
		`display notification "%s" with title "%s"`,
		escapeAppleScript(n.Body),
		escapeAppleScript(n.Title),
	)
	if n.Subtitle != "" {
		script += fmt.Sprintf(` subtitle "%s"`, escapeAppleScript(n.Subtitle))
	}
	sound := n.Sound
	if sound == "" {
		sound = "Glass"
	}
	script += fmt.Sprintf(` sound name "%s"`, escapeAppleScript(sound))
	if err := defaultRunner.run("osascript", "-e", script); err != nil {
		return fmt.Errorf("notify: osascript failed: %w", err)
	}
	return nil
}

// escapeAppleScript escapes characters that would break the AppleScript string
// literal — backslash first (so it doesn't double-escape later substitutions),
// then double-quote.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
