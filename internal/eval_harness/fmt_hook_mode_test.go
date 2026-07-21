package eval_harness

import (
	"encoding/json"
	"os"
	"os/exec"
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

// TestDetectFmtHookEvent covers the classifier that maps format_ail.sh's own
// status markers to a FmtHookEvent (M-EVAL-FMT-WEAKMODEL-AB hook-reality metric).
// It must classify each of the 4 landed markers and, critically, emit NO event
// for an unrelated line (fail-closed: an absent hook marker is never counted as
// a successful format).
func TestDetectFmtHookEvent(t *testing.T) {
	const turn = 7
	cases := []struct {
		name       string
		line       string
		wantOK     bool
		wantStatus string
		wantDetail string // "" = don't assert detail
		wantFile   string // "" = don't assert file
	}{
		{
			name:       "formatted marker → formatted",
			line:       `{"type":"user","message":"✓ Formatted src/main.ail"}`,
			wantOK:     true,
			wantStatus: "formatted",
			wantFile:   "src/main.ail",
		},
		{
			name:       "fmt failed marker → error",
			line:       `hookSpecificOutput: ailang fmt failed on foo.ail`,
			wantOK:     true,
			wantStatus: "error",
			wantDetail: "ailang fmt failed",
		},
		{
			name:       "fmt timed out marker → error/timeout",
			line:       `additionalContext: ailang fmt timed out after 10s`,
			wantOK:     true,
			wantStatus: "error",
			wantDetail: "fmt timed out",
		},
		{
			name:       "jq missing marker → error",
			line:       `format_ail hook: jq not found, skipping`,
			wantOK:     true,
			wantStatus: "error",
			wantDetail: "jq missing — fmt skipped",
		},
		{
			name:   "unrelated line → no event (fail-closed)",
			line:   `{"type":"assistant","text":"Let me edit the file"}`,
			wantOK: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ev, ok := detectFmtHookEvent(tc.line, turn)
			if ok != tc.wantOK {
				t.Fatalf("detectFmtHookEvent(%q) ok = %v, want %v", tc.line, ok, tc.wantOK)
			}
			if !tc.wantOK {
				if ev != (FmtHookEvent{}) {
					t.Errorf("no-match must return zero FmtHookEvent, got %+v", ev)
				}
				return
			}
			if ev.Turn != turn {
				t.Errorf("Turn = %d, want %d", ev.Turn, turn)
			}
			if ev.Status != tc.wantStatus {
				t.Errorf("Status = %q, want %q", ev.Status, tc.wantStatus)
			}
			if tc.wantDetail != "" && ev.Detail != tc.wantDetail {
				t.Errorf("Detail = %q, want %q", ev.Detail, tc.wantDetail)
			}
			if tc.wantFile != "" && ev.File != tc.wantFile {
				t.Errorf("File = %q, want %q", ev.File, tc.wantFile)
			}
		})
	}
}

// TestReadFmtHookSink is the M2b regression test: the out-of-band JSONL sink the
// LANDED format_ail.sh writes must parse back into populated FmtHookEvents with
// the right status/file. This is the reliable capture path (Claude Code swallows
// the hook's exit-0 stderr, so the old stream-scan was structurally empty).
func TestReadFmtHookSink(t *testing.T) {
	t.Run("missing sink → no events, no error", func(t *testing.T) {
		ws := t.TempDir()
		evts, err := ReadFmtHookSink(ws)
		if err != nil {
			t.Fatalf("missing sink must not error, got %v", err)
		}
		if evts != nil {
			t.Fatalf("missing sink must yield nil events, got %+v", evts)
		}
	})

	t.Run("formatted + error events parse", func(t *testing.T) {
		ws := t.TempDir()
		if err := os.MkdirAll(filepath.Join(ws, ".claude"), 0755); err != nil {
			t.Fatal(err)
		}
		sink := FmtHookSinkPath(ws)
		lines := strings.Join([]string{
			`{"status":"formatted","file":"benchmark/solution.ail","exit_code":0,"id":"tool_1","detail":""}`,
			`{"status":"error","file":"benchmark/solution.ail","exit_code":2,"id":"tool_2","detail":"ailang fmt failed: boom"}`,
			``,                // blank line must be skipped
			`{not valid json`, // corrupt line must be skipped, not lose the rest
			`{"status":"formatted","file":"other.ail","exit_code":0,"id":"tool_3","detail":""}`,
		}, "\n")
		if err := os.WriteFile(sink, []byte(lines+"\n"), 0644); err != nil {
			t.Fatal(err)
		}

		evts, err := ReadFmtHookSink(ws)
		if err != nil {
			t.Fatalf("ReadFmtHookSink error: %v", err)
		}
		if len(evts) != 3 {
			t.Fatalf("want 3 events (2 formatted + 1 error, corrupt/blank skipped), got %d: %+v", len(evts), evts)
		}
		if evts[0].Status != "formatted" || evts[0].File != "benchmark/solution.ail" {
			t.Errorf("event 0 = %+v, want formatted benchmark/solution.ail", evts[0])
		}
		if evts[0].Detail != "" {
			t.Errorf("formatted event must carry no detail, got %q", evts[0].Detail)
		}
		if evts[1].Status != "error" || !strings.Contains(evts[1].Detail, "ailang fmt failed") {
			t.Errorf("event 1 = %+v, want error with fmt-failed detail", evts[1])
		}
		if evts[2].Status != "formatted" || evts[2].File != "other.ail" {
			t.Errorf("event 2 = %+v, want formatted other.ail", evts[2])
		}
	})
}

// TestFormatAilHookSinkRoundTrip drives the LANDED scripts/hooks/format_ail.sh
// with a crafted PostToolUse stdin JSON and asserts it appends a "formatted"
// event to the cwd-derived sink — the true end-to-end capture path. Skips when
// bash/jq/ailang are unavailable (CI has them; keep the pure-Go test above as the
// portable minimum).
func TestFormatAilHookSinkRoundTrip(t *testing.T) {
	for _, bin := range []string{"bash", "jq", "ailang"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s not on PATH — skipping hook round-trip", bin)
		}
	}
	// Locate the LANDED hook relative to the repo root (two dirs up from this
	// package): internal/eval_harness → repo root.
	hook, err := filepath.Abs(filepath.Join("..", "..", "scripts", "hooks", "format_ail.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(hook); err != nil {
		t.Skipf("LANDED hook not found at %s: %v", hook, err)
	}

	ws := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ws, ".claude"), 0755); err != nil {
		t.Fatal(err)
	}
	// A well-formed .ail file so `ailang fmt --write` exits 0 (formatted).
	ailFile := filepath.Join(ws, "solution.ail")
	if err := os.WriteFile(ailFile, []byte("module solution\n\nexport func main() -> () { () }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Build the PostToolUse stdin with json.Marshal, NOT string concatenation:
	// on Windows t.TempDir() is C:\Users\RUNNER~1\AppData\Local\Temp\..., and
	// pasting that raw into a JSON string literal produces invalid escapes
	// (\U, \A, \T) that jq rejects outright ("Invalid escape at line 1").
	// ToSlash additionally hands the hook C:/... paths, which git-bash treats
	// unambiguously in `[ -d ]` and in the `>>"$FMT_SINK"` redirect, where a
	// backslash path is separator-vs-escape ambiguous.
	payload, err := json.Marshal(map[string]any{
		"cwd":         filepath.ToSlash(ws),
		"tool_use_id": "tool_rt_1",
		"tool_input":  map[string]any{"file_path": filepath.ToSlash(ailFile)},
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", hook)
	cmd.Stdin = strings.NewReader(string(payload))
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		t.Fatalf("hook run failed: %v\nstderr: %s", err, errb.String())
	}

	evts, err := ReadFmtHookSink(ws)
	if err != nil {
		t.Fatalf("ReadFmtHookSink after hook: %v", err)
	}
	if len(evts) == 0 {
		t.Fatalf("hook wrote no sink event (stderr: %s)", errb.String())
	}
	last := evts[len(evts)-1]
	if last.Status != "formatted" {
		t.Errorf("round-trip status = %q, want formatted (hook stderr: %s)", last.Status, errb.String())
	}
	if !strings.HasSuffix(last.File, "solution.ail") {
		t.Errorf("round-trip file = %q, want *solution.ail", last.File)
	}
}

func jsonContains(t *testing.T, raw []byte, want string) bool {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	// Search for the JSON-encoded form of want, not the raw string: json.Marshal
	// escapes backslashes, so a Windows path (C:\...\format_ail.sh) appears in the
	// settings as C:\\...\\format_ail.sh. Encoding want the same way keeps this
	// assertion correct on every OS (no-op on Unix forward-slash paths).
	enc, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	needle := string(enc[1 : len(enc)-1]) // strip the surrounding quotes
	return strings.Contains(string(raw), needle)
}
