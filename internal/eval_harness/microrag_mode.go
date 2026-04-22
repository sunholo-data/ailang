package eval_harness

import (
	"os"
	"strings"
)

// MicroragMode is the eval-suite --microrag flag value.
type MicroragMode string

const (
	// MicroragModeAuto respects the inherited environment. Default.
	// Use when running outside an A/B comparison.
	MicroragModeAuto MicroragMode = "auto"
	// MicroragModeOn forces AILANG_MICRORAG_ENABLED=1 in subprocesses.
	MicroragModeOn MicroragMode = "on"
	// MicroragModeOff forces AILANG_MICRORAG_ENABLED=0 in subprocesses.
	// Use this for the baseline arm of an A/B run.
	MicroragModeOff MicroragMode = "off"
)

// ParseMicroragMode normalises CLI input. Empty / unknown → auto.
func ParseMicroragMode(s string) MicroragMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "1", "true", "enabled":
		return MicroragModeOn
	case "off", "0", "false", "disabled":
		return MicroragModeOff
	default:
		return MicroragModeAuto
	}
}

// ApplyToEnv strips any existing AILANG_MICRORAG_ENABLED entry and appends a
// fresh one matching the mode. Auto leaves the inherited value untouched.
//
// Returns the modified slice. Caller passes the result to cmd.Env. We don't
// mutate os.Environ() — every subprocess gets its own independent copy.
func (m MicroragMode) ApplyToEnv(env []string) []string {
	if m == MicroragModeAuto {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "AILANG_MICRORAG_ENABLED=") {
			continue
		}
		out = append(out, e)
	}
	switch m {
	case MicroragModeOn:
		out = append(out, "AILANG_MICRORAG_ENABLED=1")
	case MicroragModeOff:
		out = append(out, "AILANG_MICRORAG_ENABLED=0")
	}
	return out
}

// ResolvedState returns what should be recorded in RunMetrics.MicroragState.
// For auto, peeks at the actual env so the result file shows the effective
// value rather than "auto" (which would obscure the comparison).
func (m MicroragMode) ResolvedState() string {
	switch m {
	case MicroragModeOn:
		return "on"
	case MicroragModeOff:
		return "off"
	default:
		// Auto: report what the environment actually said.
		v := os.Getenv("AILANG_MICRORAG_ENABLED")
		switch v {
		case "0", "false", "disabled":
			return "off"
		case "", "1", "true", "enabled":
			return "on"
		default:
			return v
		}
	}
}
