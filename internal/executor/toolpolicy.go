package executor

import (
	"fmt"
	"sort"
	"strings"
)

// M-AGENT-AILANG-ONLY-EXECUTION M4 — tool-policy profiles, executor-neutral.
//
// A registry agent declares `tool_policy: full | ailang_only | <canonical list>`.
// This file turns that into a CANONICAL Task.AllowedTools; each executor maps
// canonical names to its CLI's vocabulary (pi: internal/executor/pi/toolnames.go,
// which errors on a name it cannot map — D7).

const (
	ToolProfileFull       = "full"
	ToolProfileAILANGOnly = "ailang_only"
)

// CanonicalTools is every tool name a profile or a registry list may use.
// Executors decide which of these they can honour; a name here with no
// counterpart on a given CLI is an ERROR at that executor, never silently
// dropped.
var CanonicalTools = map[string]bool{
	"Read": true, "Write": true, "Edit": true, "Bash": true,
	"Grep": true, "Glob": true, "WebFetch": true, "WebSearch": true,
	"AilangCheck": true, "AilangRun": true,
	"QuotaReport": true, "BuiltinsSearch": true, "MicroragSearch": true,
}

// ProfileTools expands a tool_policy value into canonical names.
//
//	"" / "full"    → nil   (the CLI's own defaults apply)
//	"ailang_only"  → Read, Edit, Write, AilangCheck, AilangRun, BuiltinsSearch — no Bash
//	                 (BuiltinsSearch is read-only API discovery: without it the
//	                 model reaches for `ailang docs` in a shell it does not have)
//	"A,B,C"        → that list, each name validated
func ProfileTools(profile string) ([]string, error) {
	switch strings.TrimSpace(profile) {
	case "", ToolProfileFull:
		return nil, nil
	case ToolProfileAILANGOnly:
		return []string{"Read", "Edit", "Write", "AilangCheck", "AilangRun", "BuiltinsSearch"}, nil
	}
	var out []string
	for _, t := range strings.Split(profile, ",") {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if !CanonicalTools[t] {
			return nil, fmt.Errorf("tool_policy names unknown tool %q (known: %s)", t, knownCanonicalTools())
		}
		out = append(out, t)
	}
	return out, nil
}

// IntersectTools keeps the names in a that are also in b, preserving a's
// order. Used when a task-kind restriction (question → read-only) meets an
// agent's declared profile: the run gets what BOTH allow.
func IntersectTools(a, b []string) []string {
	in := make(map[string]bool, len(b))
	for _, t := range b {
		in[t] = true
	}
	out := make([]string, 0, len(a))
	for _, t := range a {
		if in[t] {
			out = append(out, t)
		}
	}
	return out
}

func knownCanonicalTools() string {
	names := make([]string, 0, len(CanonicalTools))
	for k := range CanonicalTools {
		names = append(names, k)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
