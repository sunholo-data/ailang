package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSafeToolNamePattern_Accepts pins the rules for valid tool names.
// Bedrock + Vertex AI accept [A-Za-z0-9_]{1,128}.
func TestSafeToolNamePattern_Accepts(t *testing.T) {
	good := []string{
		"ctx_execute",
		"CtxExecute",
		"CTXEXECUTE",
		"a",
		"X",
		"_underscore_start",
		"with9digits1",
		"snake_case_with_lots_of_underscores",
	}
	for _, name := range good {
		if !safeToolNamePattern.MatchString(name) {
			t.Errorf("safeToolNamePattern.MatchString(%q) = false; want true", name)
		}
	}
}

// TestSafeToolNamePattern_Rejects pins the bug-class arniwesth hit at v0.18.1.
func TestSafeToolNamePattern_Rejects(t *testing.T) {
	bad := []struct {
		name string
		why  string
	}{
		{"ctx.execute", "contains '.'"},
		{"ctx-exec", "contains '-'"},
		{"ctx:exec", "contains ':'"},
		{"ctx execute", "contains whitespace"},
		{"ctx@exec", "contains @"},
		{"", "empty string"},
		{"context_mode.execute", "contains '.'"},
	}
	for _, c := range bad {
		if safeToolNamePattern.MatchString(c.name) {
			t.Errorf("safeToolNamePattern.MatchString(%q) = true; want false (%s)", c.name, c.why)
		}
	}
}

// TestValidateToolNames_AcceptsCleanPackage exercises the regex-based extractor
// against a synthetic .ail file with the canonical provided_tools shape.
func TestValidateToolNames_AcceptsCleanPackage(t *testing.T) {
	dir := t.TempDir()
	src := `module sunholo/clean/register

export func make_hooks() -> ExtensionHooks {
  {
    id: "clean",
    provided_tools: ["CtxExecute", "ctx_execute", "CtxDoctor", "ctx_doctor"],
    on_describe_tools: \_ . [
      { name: "CtxExecute", description: "x", parameters: "{}" },
      { name: "ctx_execute", description: "x", parameters: "{}" }
    ]
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "register.ail"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	names, bad, reason := validateToolNames(dir)
	if bad != "" {
		t.Errorf("validateToolNames flagged %q (%s) on a clean package; names=%v", bad, reason, names)
	}
	if len(names) < 4 {
		t.Errorf("validateToolNames extracted only %d names; expected at least 4 (CtxExecute, ctx_execute, CtxDoctor, ctx_doctor)", len(names))
	}
}

// TestValidateToolNames_RejectsDottedAliases is the regression fixture for
// the v0.18.1 Bedrock incident. Pre-M3, dotted aliases shipped silently.
func TestValidateToolNames_RejectsDottedAliases(t *testing.T) {
	dir := t.TempDir()
	src := `module sunholo/dotted/register

export func make_hooks() -> ExtensionHooks {
  {
    id: "dotted",
    provided_tools: ["CtxExecute", "ctx.execute"],
    on_describe_tools: \_ . []
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "register.ail"), []byte(src), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, bad, reason := validateToolNames(dir)
	if bad != "ctx.execute" {
		t.Errorf("validateToolNames did not flag 'ctx.execute'; got bad=%q reason=%q", bad, reason)
	}
}

// TestSuggestSafeName_Underscore covers the rewrite hint we surface in the
// rejection error message.
func TestSuggestSafeName_Underscore(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ctx.execute", "ctx_execute"},
		{"context_mode.execute", "context_mode_execute"},
		{"a.b.c", "a_b_c"},
		{"already_safe", "already_safe"},
	}
	for _, c := range cases {
		got := suggestSafeName(c.in, "_")
		if got != c.want {
			t.Errorf("suggestSafeName(%q, \"_\") = %q; want %q", c.in, got, c.want)
		}
	}
}

// TestSuggestSafeName_PascalCase covers the PascalCase rewrite hint.
func TestSuggestSafeName_PascalCase(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ctx.execute", "ctxExecute"},
		{"context_mode.execute", "context_modeExecute"},
		{"a.b.c", "aBC"},
	}
	for _, c := range cases {
		got := suggestSafeName(c.in, "")
		if got != c.want {
			t.Errorf("suggestSafeName(%q, \"\") = %q; want %q", c.in, got, c.want)
		}
	}
}

// TestDescribeBadName_NamesTheCharacter pins the human-readable reason in
// the rejection error, so the AI-correction loop has a precise signal.
func TestDescribeBadName_NamesTheCharacter(t *testing.T) {
	cases := map[string]string{
		"ctx.execute": "contains '.'",
		"ctx-execute": "contains '-'",
		"ctx:execute": "contains ':'",
		"ctx execute": "contains whitespace",
		"":            "empty string",
		"ctx@execute": "contains invalid character '@'",
	}
	for name, want := range cases {
		got := describeBadName(name)
		if got != want {
			t.Errorf("describeBadName(%q) = %q; want %q", name, got, want)
		}
	}
}
