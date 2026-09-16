package pi

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/executor"
)

// M-AGENT-AILANG-ONLY-EXECUTION M3/M4 — tool names and profiles for pi.
//
// executor.Task.AllowedTools is CANONICAL (Claude-cased: Read, Write, Edit,
// Bash …) so every caller speaks one vocabulary. pi's builtins are lowercase
// and pi SILENTLY IGNORES names it does not know (agent-session.js: "Unknown
// tool names are ignored"), so a Claude list passed straight through yields a
// run with zero tools and exit 0 — measured on the coordinator's question-kind
// path (design doc V4). Mapping is therefore done HERE, and an unmapped name is
// an error naming it, not a silent no-tool run (D7).

// Profile names are executor-neutral: executor.ToolProfileFull / AILANGOnly.
const (
	ToolProfileFull       = executor.ToolProfileFull
	ToolProfileAILANGOnly = executor.ToolProfileAILANGOnly
)

// canonicalToPi maps executor-canonical tool names to pi's registered names.
// Names with no pi counterpart are deliberately ABSENT so they error.
var canonicalToPi = map[string]string{
	"Read":        "read",
	"Write":       "write",
	"Edit":        "edit",
	"Bash":        "bash",
	"AilangCheck": "ailang_check",
	"AilangRun":   "ailang_run",
	// Extension tools that already exist in the embedded suite, exposed so a
	// profile can name them:
	"QuotaReport":    "quota_report",
	"BuiltinsSearch": "builtins_search",
	"MicroragSearch": "microrag_search",
}

// ProfileTools is executor.ProfileTools — canonical names, executor-neutral.
func ProfileTools(profile string) ([]string, error) { return executor.ProfileTools(profile) }

// ToolArgs is the pi argv fragment for a canonical AllowedTools list:
//
//	nil            → nothing (pi defaults)
//	[]             → --no-tools
//	[A, B]         → [--no-extensions -e …] --no-builtin-tools --tools a,b
//	                 (extension tools stay enabled by --no-builtin-tools;
//	                 --tools then allowlists across builtin + extension +
//	                 custom, pi ≥0.70 #3592; the -e flags appear only when
//	                 the list names AilangRun/AilangCheck — profile_assets.go)
//
// Errors on a canonical name pi has no counterpart for.
func ToolArgs(allowed []string) ([]string, error) {
	switch {
	case allowed == nil:
		return nil, nil
	case len(allowed) == 0:
		return []string{"--no-tools"}, nil
	}
	names := make([]string, 0, len(allowed))
	for _, c := range allowed {
		p, ok := canonicalToPi[c]
		if !ok {
			return nil, fmt.Errorf("pi: AllowedTools names %q, which pi has no tool for (pi would silently ignore it and run with fewer tools than asked); known: %s", c, knownCanonical())
		}
		names = append(names, p)
	}
	// A list that names extension tools CARRIES them (profile_assets.go), so
	// the lane does not depend on what the machine's global dir happens to hold.
	ext, err := extensionArgs(allowed)
	if err != nil {
		return nil, err
	}
	args := append([]string{}, ext...)
	return append(args, "--no-builtin-tools", "--tools", strings.Join(names, ",")), nil
}

// ProfileArgs is the ONE string for a profile's pi flags — printed by
// `ailang pi tool-profile <name>` so the resident's pi.mjs and any shell reads
// the same expansion pi.go uses. Empty for "full".
func ProfileArgs(profile string) (string, error) {
	tools, err := ProfileTools(profile)
	if err != nil {
		return "", err
	}
	args, err := ToolArgs(tools)
	if err != nil {
		return "", err
	}
	return strings.Join(args, " "), nil
}

func knownCanonical() string {
	names := make([]string, 0, len(canonicalToPi))
	for k := range canonicalToPi {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
