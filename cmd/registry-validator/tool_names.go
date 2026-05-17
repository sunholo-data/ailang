// M-EXT-AUTHOR-DX M3 (v0.20.1): provider-safe tool name validation.
//
// Anthropic Bedrock + Google Vertex AI accept only `[A-Za-z0-9_]{1,128}` at
// the tools[].custom.name validator. Dotted aliases like "ctx.execute" pass
// Anthropic direct + OpenAI + AI-Studio + OpenRouter but fail loudly on
// Bedrock and Vertex — the v0.18.1 incident class this gate closes.
//
// validateToolNames is called from handlePublish; describeBadName + the two
// suggestSafeName modes are referenced in the rejection error so the
// AI-correction loop has a precise signal to act on.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// safeToolNamePattern matches strings that Anthropic Bedrock + Vertex AI
// accept at the tools[].custom.name validator. Dots, dashes, colons, and
// anything else outside [A-Za-z0-9_] are rejected. Non-empty + ≤128 chars.
var safeToolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,128}$`)

// providedToolsBlockPattern + nameFieldPattern + stringLiteralPattern find
// `"NAME"` string literals in the position of a tool-name field. Source-
// static: looks for advertised names inside `provided_tools: [...]` blocks
// and `name: "..."` fields nested in `on_describe_tools` ToolSchema records.
// Both arniwesth's PR #22 source and our compaction_ai 0.2.x publish-time
// package fit this shape.
//
// Limitations: misses tools added dynamically at runtime (not source-
// visible). The publish-time smoke can still hit them, but this static
// gate catches the common case. Authors who advertise tools through
// non-literal paths can override with --allow-dotted-tool-names.
var providedToolsBlockPattern = regexp.MustCompile(`provided_tools\s*[:=]\s*\[([^\]]*)\]`)
var nameFieldPattern = regexp.MustCompile(`\bname\s*:\s*"([^"]+)"`)
var stringLiteralPattern = regexp.MustCompile(`"([^"]*)"`)

// validateToolNames walks every .ail file in dir and extracts advertised
// tool names from:
//
//   - `provided_tools: ["A", "B"]` literals (one of the canonical
//     positions for the [string] field of ExtensionHooks)
//   - `name: "X"` fields appearing inside ToolSchema records (the per-tool
//     advertised name in on_describe_tools output)
//
// Returns (allNames, firstBadName, reason). If firstBadName == "" the
// package is provider-safe; otherwise the bad name + a human-readable
// reason are returned for the error message.
func validateToolNames(dir string) (allNames []string, firstBadName string, reason string) {
	seen := make(map[string]bool)
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ail") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		src := string(data)

		// Pattern 1: provided_tools: [ "A", "B", ... ]
		for _, match := range providedToolsBlockPattern.FindAllStringSubmatch(src, -1) {
			inner := match[1]
			for _, str := range stringLiteralPattern.FindAllStringSubmatch(inner, -1) {
				if name := str[1]; name != "" && !seen[name] {
					seen[name] = true
					allNames = append(allNames, name)
				}
			}
		}

		// Pattern 2: name: "X" (inside on_describe_tools ToolSchema records).
		// This is a coarse over-match — it'll also catch `name:` fields from
		// other record literals — but the safe-name validator is permissive
		// enough that false-positive matches on non-tool names won't flag
		// (they almost always conform to [A-Za-z0-9_] anyway).
		for _, match := range nameFieldPattern.FindAllStringSubmatch(src, -1) {
			if name := match[1]; name != "" && !seen[name] {
				seen[name] = true
				allNames = append(allNames, name)
			}
		}
		return nil
	})

	for _, name := range allNames {
		if !safeToolNamePattern.MatchString(name) {
			return allNames, name, describeBadName(name)
		}
	}
	return allNames, "", ""
}

// describeBadName returns a human-readable reason a tool name was rejected.
func describeBadName(name string) string {
	if name == "" {
		return "empty string"
	}
	if len(name) > 128 {
		return fmt.Sprintf("length %d exceeds 128-char limit", len(name))
	}
	for _, r := range name {
		switch {
		case r == '.':
			return "contains '.'"
		case r == '-':
			return "contains '-'"
		case r == ':':
			return "contains ':'"
		case r == ' ':
			return "contains whitespace"
		case (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_':
			return fmt.Sprintf("contains invalid character %q", r)
		}
	}
	return "fails [A-Za-z0-9_]{1,128} pattern"
}

// suggestSafeName converts an unsafe tool name to a suggestion. With
// separator="_" → snake-ish (replace bad chars with _). With separator=""
// → PascalCase-ish (drop bad chars + camel-case at word boundaries).
func suggestSafeName(name, separator string) string {
	var b strings.Builder
	upper := false
	for _, r := range name {
		switch {
		case (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_':
			if upper && r >= 'a' && r <= 'z' {
				b.WriteRune(r - 32)
				upper = false
			} else {
				b.WriteRune(r)
			}
		default:
			if separator != "" {
				b.WriteString(separator)
			} else {
				upper = true
			}
		}
	}
	out := b.String()
	if out == "" {
		out = "tool"
	}
	return out
}
