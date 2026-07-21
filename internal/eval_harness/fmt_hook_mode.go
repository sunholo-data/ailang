package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FmtHookMode is the eval-suite -fmt-hook flag value (M-EVAL-FMT-WEAKMODEL-AB).
//
// It is the ONLY per-arm difference in the fmt-hook A/B experiment: whether the
// LANDED scripts/hooks/format_ail.sh PostToolUse hook is wired into the agent
// workspace so `ailang fmt --write` canonically formats every edited .ail file.
//
// Modeled on MicroragMode (type + Parse* + apply + ResolvedState), but the
// treatment is a workspace .claude/settings.json + --settings flag rather than
// a subprocess env var.
type FmtHookMode string

const (
	// FmtHookModeOff omits the fmt PostToolUse hook entirely — byte-identical to
	// the harness path that ships today (no .claude/settings.json, no --settings
	// flag). This is the DEFAULT so existing eval runs are unchanged, and it is
	// the baseline arm of the A/B.
	FmtHookModeOff FmtHookMode = "off"
	// FmtHookModeOn writes a workspace .claude/settings.json registering
	// scripts/hooks/format_ail.sh as a PostToolUse hook for Edit|Write, and
	// passes --settings <path> to the claude command. This is the treatment arm.
	FmtHookModeOn FmtHookMode = "on"
)

// ParseFmtHookMode normalises CLI input. Empty / unknown → off (default,
// preserves today's behaviour). Mirrors ParseMicroragMode.
func ParseFmtHookMode(s string) FmtHookMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "1", "true", "enabled":
		return FmtHookModeOn
	default:
		return FmtHookModeOff
	}
}

// ResolvedState returns what should be recorded in RunMetrics.FmtHookState /
// AgentBenchmarkResult.FmtHook so the A/B arm is unambiguous in analysis.
func (m FmtHookMode) ResolvedState() string {
	if m == FmtHookModeOn {
		return "on"
	}
	return "off"
}

// SettingsJSON is the exact bytes written to the workspace .claude/settings.json
// for the ON arm. It registers the LANDED format_ail.sh as a PostToolUse hook
// matching Edit|Write. hookScriptPath must be an ABSOLUTE path (the agent's CWD
// is the isolated workspace, not the repo root). Returns the marshalled JSON so
// callers can log the resolved settings for config-diff review.
func (m FmtHookMode) SettingsJSON(hookScriptPath string) ([]byte, error) {
	settings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PostToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Edit|Write",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": hookScriptPath,
						},
					},
				},
			},
		},
	}
	return json.MarshalIndent(settings, "", "  ")
}

// Apply wires the fmt hook into the given agent workspace when the mode is ON.
//
// For ON: it locates scripts/hooks/format_ail.sh (relative to the repo root =
// the process CWD during eval), writes workspace/.claude/settings.json
// registering it as a PostToolUse Edit|Write hook, and returns the absolute path
// to that settings file so the caller can pass `--settings <path>` to claude.
// It also returns the resolved settings bytes for config-diff logging.
//
// For OFF: it writes nothing and returns ("", nil, nil) — byte-identical to the
// path that ships today. This is the ONLY per-arm difference in the experiment.
//
// A missing hook script in ON mode is a HARD error (fail loudly, per the repo's
// no-silent-fallback rule): silently degrading to OFF would corrupt the A/B by
// classifying an untreated run as treated.
func (m FmtHookMode) Apply(workspace string) (settingsPath string, settingsJSON []byte, err error) {
	if m != FmtHookModeOn {
		return "", nil, nil
	}

	hookScript, err := resolveFmtHookScript()
	if err != nil {
		return "", nil, err
	}

	jsonBytes, err := m.SettingsJSON(hookScript)
	if err != nil {
		return "", nil, fmt.Errorf("fmt-hook: marshal settings: %w", err)
	}

	claudeDir := filepath.Join(workspace, ".claude")
	if err := os.MkdirAll(claudeDir, 0755); err != nil {
		return "", nil, fmt.Errorf("fmt-hook: create %s: %w", claudeDir, err)
	}
	settingsPath = filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, jsonBytes, 0644); err != nil {
		return "", nil, fmt.Errorf("fmt-hook: write %s: %w", settingsPath, err)
	}
	return settingsPath, jsonBytes, nil
}

// Marker strings emitted by the LANDED scripts/hooks/format_ail.sh. These are
// the ground-truth signals of what the hook actually did; we key hook-reality
// classification off them so we never fabricate a "treated" event.
const (
	fmtHookMarkerFormatted = "✓ Formatted "            // exit 0: canonical format applied
	fmtHookMarkerFailed    = "ailang fmt failed"       // non-0/non-3 exit surfaced via additionalContext
	fmtHookMarkerTimedOut  = "ailang fmt timed out"    // synthetic timeout branch
	fmtHookMarkerSkippedJq = "format_ail hook: jq not" // jq missing → skipped
)

// detectFmtHookEvent inspects one stream-json line for a format_ail.sh status
// marker and, if present, returns a classified FmtHookEvent tagged with the
// current turn. Returns ok=false when the line carries no hook marker.
//
// Classification (M-EVAL-FMT-WEAKMODEL-AB hook-reality metric):
//   - "formatted" — the hook ran ailang fmt and it exited 0 (treatment delivered)
//   - "error"     — the hook surfaced a failure/timeout/skip via additionalContext
//     or stderr (refusal/no-op class; counts against the treatment-delivery rate)
//
// The exit-3 "deferred" (unparseable-mid-edit) case is intentionally SILENT in
// the LANDED hook (contract clause 5), so it emits no marker and is not counted
// as either treated or refused — which is the correct treatment-integrity
// accounting.
func detectFmtHookEvent(line string, turn int) (FmtHookEvent, bool) {
	switch {
	case strings.Contains(line, fmtHookMarkerFormatted):
		return FmtHookEvent{Turn: turn, Status: "formatted", File: extractFmtFile(line, fmtHookMarkerFormatted)}, true
	case strings.Contains(line, fmtHookMarkerTimedOut):
		return FmtHookEvent{Turn: turn, Status: "error", Detail: "fmt timed out"}, true
	case strings.Contains(line, fmtHookMarkerFailed):
		return FmtHookEvent{Turn: turn, Status: "error", Detail: "ailang fmt failed"}, true
	case strings.Contains(line, fmtHookMarkerSkippedJq):
		return FmtHookEvent{Turn: turn, Status: "error", Detail: "jq missing — fmt skipped"}, true
	default:
		return FmtHookEvent{}, false
	}
}

// extractFmtFile pulls the .ail filename out of a "✓ Formatted <file>" marker.
// Best-effort: the marker may be embedded in a JSON string, so we take the token
// after the marker up to the next quote/whitespace/escape.
func extractFmtFile(line, marker string) string {
	i := strings.Index(line, marker)
	if i < 0 {
		return ""
	}
	rest := line[i+len(marker):]
	end := strings.IndexAny(rest, "\"\\ \t\n")
	if end < 0 {
		return strings.TrimSpace(rest)
	}
	return rest[:end]
}

// resolveFmtHookScript returns the absolute path to the LANDED fmt PostToolUse
// hook. Eval runs execute from the repo root, so scripts/hooks/format_ail.sh is
// CWD-relative. Fails loudly if the LANDED hook is absent (do not silently ship
// an untreated ON arm).
func resolveFmtHookScript() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("fmt-hook: getwd: %w", err)
	}
	script := filepath.Join(cwd, "scripts", "hooks", "format_ail.sh")
	if _, statErr := os.Stat(script); statErr != nil {
		return "", fmt.Errorf("fmt-hook: LANDED hook not found at %s (run eval from repo root): %w", script, statErr)
	}
	return script, nil
}
