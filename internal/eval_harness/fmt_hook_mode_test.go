package eval_harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFmtHookMode(t *testing.T) {
	cases := map[string]FmtHookMode{
		"":         FmtHookModeOff,
		"off":      FmtHookModeOff,
		"OFF":      FmtHookModeOff,
		"disabled": FmtHookModeOff,
		"0":        FmtHookModeOff,
		"unknown":  FmtHookModeOff, // unknown → off (default preserves today's behaviour)
		"on":       FmtHookModeOn,
		"ON":       FmtHookModeOn,
		"true":     FmtHookModeOn,
		"enabled":  FmtHookModeOn,
		"1":        FmtHookModeOn,
	}
	for in, want := range cases {
		if got := ParseFmtHookMode(in); got != want {
			t.Errorf("ParseFmtHookMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFmtHookResolvedState(t *testing.T) {
	if got := FmtHookModeOn.ResolvedState(); got != "on" {
		t.Errorf("on.ResolvedState() = %q, want on", got)
	}
	if got := FmtHookModeOff.ResolvedState(); got != "off" {
		t.Errorf("off.ResolvedState() = %q, want off", got)
	}
}

func TestSettingsJSON_RegistersHook(t *testing.T) {
	raw, err := FmtHookModeOn.SettingsJSON("/abs/path/format_ail.sh")
	if err != nil {
		t.Fatalf("SettingsJSON: %v", err)
	}
	var parsed struct {
		Hooks struct {
			PostToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PostToolUse"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("settings JSON did not parse: %v\n%s", err, raw)
	}
	if len(parsed.Hooks.PostToolUse) != 1 {
		t.Fatalf("expected 1 PostToolUse matcher, got %d", len(parsed.Hooks.PostToolUse))
	}
	pt := parsed.Hooks.PostToolUse[0]
	if pt.Matcher != "Edit|Write" {
		t.Errorf("matcher = %q, want Edit|Write", pt.Matcher)
	}
	if len(pt.Hooks) != 1 || pt.Hooks[0].Type != "command" || pt.Hooks[0].Command != "/abs/path/format_ail.sh" {
		t.Errorf("hook entry wrong: %+v", pt.Hooks)
	}
}

// Off is byte-identical to today's path: no settings file, no path.
func TestApply_OffWritesNothing(t *testing.T) {
	ws := t.TempDir()
	path, jsonBytes, err := FmtHookModeOff.Apply(ws)
	if err != nil {
		t.Fatalf("off Apply errored: %v", err)
	}
	if path != "" || jsonBytes != nil {
		t.Fatalf("off must return empty path/json; got path=%q json=%v", path, jsonBytes)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".claude", "settings.json")); !os.IsNotExist(statErr) {
		t.Fatalf("off must not create .claude/settings.json (stat err=%v)", statErr)
	}
}

// On writes .claude/settings.json referencing the LANDED hook and returns its path.
func TestApply_OnWritesSettings(t *testing.T) {
	// Apply resolves scripts/hooks/format_ail.sh relative to CWD. Build a fake
	// repo root with that script present, and run from there.
	repoRoot := t.TempDir()
	hookDir := filepath.Join(repoRoot, "scripts", "hooks")
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		t.Fatal(err)
	}
	hookPath := filepath.Join(hookDir, "format_ail.sh")
	if err := os.WriteFile(hookPath, []byte("#!/bin/bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}

	ws := filepath.Join(repoRoot, "workspace")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatal(err)
	}
	path, jsonBytes, err := FmtHookModeOn.Apply(ws)
	if err != nil {
		t.Fatalf("on Apply errored: %v", err)
	}
	wantPath := filepath.Join(ws, ".claude", "settings.json")
	if path != wantPath {
		t.Errorf("settings path = %q, want %q", path, wantPath)
	}
	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("settings not written: %v", err)
	}
	if string(onDisk) != string(jsonBytes) {
		t.Errorf("returned json != on-disk json")
	}
	// The resolved (absolute) hook path must appear in the settings.
	if !jsonContains(t, onDisk, hookPath) {
		t.Errorf("settings do not reference the hook script %q:\n%s", hookPath, onDisk)
	}
}

// On with a missing LANDED hook must FAIL LOUDLY, not silently degrade to off
// (silently shipping an untreated ON arm would corrupt the A/B).
func TestApply_OnMissingHookFailsLoud(t *testing.T) {
	repoRoot := t.TempDir() // no scripts/hooks/format_ail.sh
	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	ws := filepath.Join(repoRoot, "workspace")
	if err := os.MkdirAll(ws, 0755); err != nil {
		t.Fatal(err)
	}
	if _, _, err := FmtHookModeOn.Apply(ws); err == nil {
		t.Fatal("expected Apply to error when the LANDED hook is absent")
	}
}

func jsonContains(t *testing.T, raw []byte, want string) bool {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	return strings.Contains(string(raw), want)
}
