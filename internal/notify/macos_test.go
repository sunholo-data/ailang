package notify

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// fakeRunner records invocations for assertions in tests.
type fakeRunner struct {
	commands [][]string
	failOn   map[string]error // map binary name -> error to return
	lookErrs map[string]error // map binary name -> LookPath error
}

func (f *fakeRunner) lookPath(name string) (string, error) {
	if err, ok := f.lookErrs[name]; ok {
		return "", err
	}
	return "/fake/" + name, nil
}

func (f *fakeRunner) run(name string, args ...string) error {
	f.commands = append(f.commands, append([]string{name}, args...))
	if err, ok := f.failOn[name]; ok {
		return err
	}
	return nil
}

func withRunner(t *testing.T, r runner) func() {
	t.Helper()
	prev := defaultRunner
	defaultRunner = r
	return func() { defaultRunner = prev }
}

func TestNotify_TerminalNotifierPath(t *testing.T) {
	r := &fakeRunner{}
	defer withRunner(t, r)()

	err := Notify(Notification{
		Title:    "T",
		Subtitle: "S",
		Body:     "B",
		Sound:    "Glass",
		Group:    "g1",
		URL:      "https://example.com",
	})
	if err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if len(r.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(r.commands))
	}
	cmd := r.commands[0]
	if cmd[0] != "terminal-notifier" {
		t.Errorf("expected terminal-notifier, got %s", cmd[0])
	}
	joined := strings.Join(cmd, " ")
	for _, want := range []string{"-title", "T", "-subtitle", "S", "-message", "B", "-sound", "Glass", "-group", "g1", "-execute"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected command to contain %q, got: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "https://example.com") {
		t.Errorf("expected click-action URL in command, got: %s", joined)
	}
}

func TestNotify_OsascriptFallback(t *testing.T) {
	r := &fakeRunner{lookErrs: map[string]error{"terminal-notifier": exec.ErrNotFound}}
	defer withRunner(t, r)()

	err := Notify(Notification{Title: "Hi", Body: "There"})
	if err != nil {
		t.Fatalf("Notify returned error: %v", err)
	}
	if len(r.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(r.commands))
	}
	cmd := r.commands[0]
	if cmd[0] != "osascript" {
		t.Errorf("expected osascript, got %s", cmd[0])
	}
	// The osascript invocation runs `display notification "Body" with title "Hi"`.
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, `display notification`) {
		t.Errorf("expected display-notification AppleScript, got: %s", joined)
	}
	if !strings.Contains(joined, "Hi") || !strings.Contains(joined, "There") {
		t.Errorf("expected title and body in AppleScript, got: %s", joined)
	}
}

func TestNotify_BothUnavailable(t *testing.T) {
	r := &fakeRunner{lookErrs: map[string]error{
		"terminal-notifier": exec.ErrNotFound,
		"osascript":         exec.ErrNotFound,
	}}
	defer withRunner(t, r)()

	err := Notify(Notification{Title: "x", Body: "y"})
	if !errors.Is(err, ErrNotifierUnavailable) {
		t.Fatalf("expected ErrNotifierUnavailable, got %v", err)
	}
	if len(r.commands) != 0 {
		t.Errorf("expected no command runs when both unavailable, got %d", len(r.commands))
	}
}

func TestNotify_RunErrorPropagates(t *testing.T) {
	wantErr := errors.New("boom")
	r := &fakeRunner{failOn: map[string]error{"terminal-notifier": wantErr}}
	defer withRunner(t, r)()

	err := Notify(Notification{Title: "x", Body: "y"})
	if !errors.Is(err, wantErr) {
		t.Errorf("expected wrapped error %v, got %v", wantErr, err)
	}
}

func TestNotify_DefaultSound(t *testing.T) {
	r := &fakeRunner{}
	defer withRunner(t, r)()

	if err := Notify(Notification{Title: "x", Body: "y"}); err != nil {
		t.Fatal(err)
	}
	cmd := r.commands[0]
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "-sound Glass") {
		t.Errorf("expected default sound Glass, got: %s", joined)
	}
}

func TestNotify_NoURLOmitsExecuteFlag(t *testing.T) {
	r := &fakeRunner{}
	defer withRunner(t, r)()

	if err := Notify(Notification{Title: "x", Body: "y"}); err != nil {
		t.Fatal(err)
	}
	cmd := r.commands[0]
	joined := strings.Join(cmd, " ")
	if strings.Contains(joined, "-execute") {
		t.Errorf("expected no -execute flag when URL empty, got: %s", joined)
	}
}

func TestNotify_OsascriptEscapesQuotes(t *testing.T) {
	r := &fakeRunner{lookErrs: map[string]error{"terminal-notifier": exec.ErrNotFound}}
	defer withRunner(t, r)()

	err := Notify(Notification{Title: `with "quotes"`, Body: `body with "quotes"`})
	if err != nil {
		t.Fatal(err)
	}
	cmd := r.commands[0]
	joined := strings.Join(cmd, " ")
	// AppleScript escapes double quotes with backslash. The fake runner stores
	// raw arg strings; the AppleScript payload should contain \" sequences.
	if !strings.Contains(joined, `\"quotes\"`) {
		t.Errorf("expected escaped quotes in AppleScript, got: %s", joined)
	}
}
