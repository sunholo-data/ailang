package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// utf8BOM is the byte sequence Windows tools (e.g. PowerShell Set-Content)
// often prepend to UTF-8 files. JSON decoders reject it, so we strip it.
const utf8BOM = "\ufeff"

// resolveArgsJSON returns the effective JSON argument string for `ailang run`
// based on the values of --args-json and --args-file.
//
// Resolution rules:
//
//	-args-json default ("null"), -args-file empty   → "null" (current behavior)
//	-args-json "-",              -args-file empty   → read all of stdin
//	-args-json <other>,          -args-file empty   → use literal value (current behavior)
//	-args-json default,          -args-file <path>  → read file
//	-args-json non-default,      -args-file <path>  → error
//
// File and stdin contents are stripped of a leading UTF-8 BOM and have
// trailing whitespace trimmed. Empty input from either source is an error
// (instead of letting the JSON decoder produce a generic "unexpected end of
// input" message).
//
// The "null" default originates from the flag declaration in runCommand.
func resolveArgsJSON(argsJSON, argsFile string, stdin io.Reader) (string, error) {
	const defaultArgs = "null"

	if argsFile != "" && argsJSON != defaultArgs {
		return "", fmt.Errorf("specify exactly one of -args-json or -args-file")
	}

	if argsFile != "" {
		data, err := os.ReadFile(argsFile)
		if err != nil {
			return "", fmt.Errorf("failed to read -args-file %q: %w", argsFile, err)
		}
		s := normalizeJSONInput(string(data))
		if s == "" {
			return "", fmt.Errorf("empty -args-file %q", argsFile)
		}
		return s, nil
	}

	if argsJSON == "-" {
		data, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("failed to read -args-json from stdin: %w", err)
		}
		s := normalizeJSONInput(string(data))
		if s == "" {
			return "", fmt.Errorf("empty stdin for -args-json -")
		}
		return s, nil
	}

	return argsJSON, nil
}

func normalizeJSONInput(s string) string {
	s = strings.TrimPrefix(s, utf8BOM)
	return strings.TrimSpace(s)
}
