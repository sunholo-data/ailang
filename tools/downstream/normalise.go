package main

import (
	"regexp"
	"strings"
)

// Error-code normalisation (design doc §6.6). `check --json` reports "code":
// "ERROR" for every diagnostic that is not an ailerrors.Report, so the real code,
// when there is one, sits inside the message. The normaliser tries, in order: the
// structured code; a code embedded in the message; a family keyed on stable
// message phrases; else UNCODED:<fingerprint> with module paths stripped, so one
// cause in many modules stays one bucket. The UNCODED share is reported as an
// instrument-quality number.

const uncodedPrefix = "UNCODED:"

// embeddedCodeRE matches the registered code families (internal/errors/codes.go)
// in either shape: PAR001, IMP012_UNSUPPORTED_NAMESPACE, TC_ARITY_001.
var embeddedCodeRE = regexp.MustCompile(`\b((?:PAR|MOD|TC|RT|IMP|LDR|EVA|ELB|LNK|DSG|TST|BC|EFF|IFC|TYP|PKG|VER)(?:\d{3}(?:_[A-Z][A-Z0-9_]*[A-Z0-9])?|_[A-Z][A-Z0-9_]*[A-Z0-9]))\b`)

// phraseFamilies is checked in order; the first phrase found names the bucket.
var phraseFamilies = []struct{ phrase, code string }{
	{"pure keyword contradicts", "EFFECT/pure-with-row"},
	{"information-flow violation", "IFC/flow-violation"},
	{"effect ceiling violation", "EFFECT/ceiling"},
	{"Missing effects", "EFFECT/undeclared"},
	{"uses effects not declared", "EFFECT/undeclared"},
	// The loader stops at these two before any type checking (measured
	// 2026-11-10: a type error planted in the file is not reported), so the
	// entry is NOT_MEASURED with its own reason, not a verdict and not a
	// missing dependency: an import of a module a package does not export, and
	// the entry's own manifest rejected (a dependency's is DEP/setup: runOnce).
	{"is not exported by package", codeUncheckedExport},
	{"invalid manifest", codeUncheckedManifest},
	{"failed to resolve package import", "DEP/unresolved"},
	{"not found in ailang.lock", "DEP/unresolved"},
	{"package resolution setup failed", "DEP/setup"},
	{"non-exhaustive", "MATCH/non-exhaustive"},
}

// Module-ish tokens (a/b/c, file.ail, :12:3 positions, quoted names) are stripped
// from fingerprints so the same cause in different files lands in one bucket.
var (
	fpPathRE   = regexp.MustCompile(`[A-Za-z0-9_.\-]+(?:/[A-Za-z0-9_.\-]+)+`)
	fpQuoteRE  = regexp.MustCompile("['\"`][^'\"`]*['\"`]")
	fpNonWord  = regexp.MustCompile(`[^a-z ]+`)
	fpSpacesRE = regexp.MustCompile(` +`)
)

// Codes for entries the loader rejected before type checking anything.
const (
	uncheckedPrefix       = "UNCHECKED/"
	codeUncheckedManifest = "UNCHECKED/invalid-manifest"
	codeUncheckedExport   = "UNCHECKED/not-exported"
)

// normaliseCode maps one diagnostic to its bucket code.
func normaliseCode(code, msg string) string {
	if code == "PKG_MANIFEST" { // `check --package`: the manifest did not load
		return codeUncheckedManifest
	}
	if code != "" && code != "ERROR" {
		return code
	}
	if m := embeddedCodeRE.FindStringSubmatch(msg); m != nil {
		return m[1]
	}
	for _, f := range phraseFamilies {
		if strings.Contains(msg, f.phrase) {
			return f.code
		}
	}
	if strings.HasPrefix(msg, "effect checking failed") {
		return "EFFECT/other"
	}
	if strings.HasPrefix(msg, "type error") {
		switch {
		case strings.Contains(msg, "cannot unify"), strings.Contains(msg, "unification failed"):
			return "TYPE_ERROR/unify"
		case strings.Contains(msg, "undefined"), strings.Contains(msg, "unbound"):
			return "TYPE_ERROR/unbound"
		default:
			return "TYPE_ERROR/other"
		}
	}
	return uncodedPrefix + fingerprint(msg)
}

// fingerprint reduces a message's first line to a short, file-independent key.
func fingerprint(msg string) string {
	line := msg
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = fpPathRE.ReplaceAllString(line, " ")
	line = fpQuoteRE.ReplaceAllString(line, " ")
	line = fpNonWord.ReplaceAllString(strings.ToLower(line), " ")
	line = strings.TrimSpace(fpSpacesRE.ReplaceAllString(line, " "))
	if len(line) > 48 {
		line = strings.TrimSpace(line[:48])
	}
	if line == "" {
		return "empty"
	}
	return strings.ReplaceAll(line, " ", "-")
}

// isDepCode reports whether a code means "a dependency could not be resolved":
// such files were not really checked (coverage, not a pass or a break).
func isDepCode(code string) bool { return strings.HasPrefix(code, "DEP/") }
