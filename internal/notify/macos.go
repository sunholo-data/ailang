// Package notify fires native macOS notifications via terminal-notifier
// (preferred — supports click actions) or osascript (fallback). On non-Darwin
// systems and when neither binary is on PATH, Notify returns ErrNotifierUnavailable
// so callers can degrade gracefully.
package notify

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed ailang-logo.png
var ailangIconPNG []byte

// ErrNotifierUnavailable is returned when neither terminal-notifier nor
// osascript is on PATH. Callers should treat this as a non-fatal degradation
// signal rather than a hard error — log it and continue.
var ErrNotifierUnavailable = errors.New("notify: no notification binary available (install terminal-notifier or run on macOS with osascript)")

// Notification is a single notification payload. Sound and Group are optional;
// URL is honored only on the terminal-notifier path (osascript has no
// click-action equivalent).
type Notification struct {
	Title    string
	Subtitle string
	Body     string
	Sound    string // default "Glass" if empty
	Group    string // collapses repeat notifications when shared
	URL      string // click-action; opens URL when notification is clicked (terminal-notifier only)
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

// iconPathOnce ensures we materialize the embedded AILANG logo at most once
// per process, even under concurrent Notify calls.
var (
	iconPathOnce sync.Once
	iconPath     string
)

// ensureIconPath writes the embedded AILANG logo to ~/.ailang/cache/
// notify-icon.png if it's not already there, and returns the path. Returns ""
// (with no error) if anything fails — the caller treats an empty path as
// "no icon", which is non-fatal. The icon is per-user; we don't pollute /tmp.
func ensureIconPath() string {
	iconPathOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, ".ailang", "cache")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return
		}
		path := filepath.Join(dir, "notify-icon.png")
		// Idempotent: only write if missing or stale (size mismatch).
		if info, err := os.Stat(path); err == nil && info.Size() == int64(len(ailangIconPNG)) {
			iconPath = path
			return
		}
		if err := os.WriteFile(path, ailangIconPNG, 0o644); err != nil {
			return
		}
		iconPath = path
	})
	return iconPath
}

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
	if path := ensureIconPath(); path != "" {
		args = append(args, "-appIcon", path)
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
