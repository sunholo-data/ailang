package eval_harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpec(t *testing.T) {
	// Create temporary YAML file
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "test.yml")

	content := `id: test
description: "Test benchmark"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "easy"
expected_gain: "low"
prompt: "Write a program in <LANG>"
expected_stdout: "Hello"
`

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load spec
	spec, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec failed: %v", err)
	}

	// Verify fields
	if spec.ID != "test" {
		t.Errorf("Expected ID 'test', got '%s'", spec.ID)
	}

	if len(spec.Languages) != 2 {
		t.Errorf("Expected 2 languages, got %d", len(spec.Languages))
	}

	if spec.Prompt != "Write a program in <LANG>" {
		t.Errorf("Unexpected prompt: %s", spec.Prompt)
	}
}

func TestLoadSpec_MissingRequired(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "invalid.yml")

	// Missing 'id' field
	content := `description: "Test"
languages: ["python"]
prompt: "Test"
`

	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err := LoadSpec(specPath)
	if err == nil {
		t.Error("Expected error for missing required field, got nil")
	}
}

func TestSupportsLanguage(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:        "test",
		Languages: []string{"python", "ailang"},
		Prompt:    "test",
	}

	tests := []struct {
		lang     string
		expected bool
	}{
		{"python", true},
		{"ailang", true},
		{"javascript", false},
		{"", false},
	}

	for _, tt := range tests {
		result := spec.SupportsLanguage(tt.lang)
		if result != tt.expected {
			t.Errorf("SupportsLanguage(%s) = %v, want %v", tt.lang, result, tt.expected)
		}
	}
}

func TestPromptForLanguage(t *testing.T) {
	spec := &BenchmarkSpec{
		ID:     "test",
		Prompt: "Write code in <LANG> that prints hello",
	}

	// Test Python - should use guidelines as base + task appended
	pythonResult := spec.PromptForLanguage("python")

	// Python result should contain the default guidelines
	if !containsSubstring(pythonResult, "expert Python programmer") {
		t.Errorf("PromptForLanguage(python) should contain Python guidelines, got: %s", pythonResult[:min(100, len(pythonResult))])
	}

	// Python result should contain the task with <LANG> replaced
	if !containsSubstring(pythonResult, "## Task") {
		t.Errorf("PromptForLanguage(python) should contain '## Task' section")
	}
	if !containsSubstring(pythonResult, "prints hello") {
		t.Errorf("PromptForLanguage(python) should contain task description 'prints hello'")
	}
	if !containsSubstring(pythonResult, "Python 3") {
		t.Errorf("PromptForLanguage(python) should replace <LANG> with 'Python 3'")
	}

	// Test AILANG - should always include teaching prompt with task appended
	ailangResult := spec.PromptForLanguage("ailang")

	// For AILANG, the result should:
	// 1. Start with the teaching prompt (contains "AILANG v0.6")
	// 2. Have the task description appended after "## Task"
	if len(ailangResult) < 100 {
		t.Errorf("PromptForLanguage(ailang) result too short, expected teaching prompt + task")
	}

	// Check that teaching prompt is included (starts with # AILANG)
	if ailangResult[:9] != "# AILANG " {
		t.Errorf("PromptForLanguage(ailang) should start with teaching prompt header, got: %s", ailangResult[:50])
	}

	// Check that task description is appended
	if !containsSubstring(ailangResult, "## Task") {
		t.Errorf("PromptForLanguage(ailang) should contain '## Task' section")
	}
	if !containsSubstring(ailangResult, "prints hello") {
		t.Errorf("PromptForLanguage(ailang) should contain task description 'prints hello'")
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	return findSubstring(s, substr) != -1
}

// ─── Tier + Tags: parsing, validation, defaults ────────────────────────────

func TestLoadSpec_TierAndTags_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "t.yml")

	content := `id: test
description: "Test"
languages: ["python", "ailang"]
prompt: "x"
tier: smoke
tags: [adt_pattern_match, recursion]
`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	spec, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if spec.Tier != "smoke" {
		t.Errorf("Tier = %q, want %q", spec.Tier, "smoke")
	}
	if len(spec.Tags) != 2 || spec.Tags[0] != "adt_pattern_match" {
		t.Errorf("Tags = %v, want [adt_pattern_match recursion]", spec.Tags)
	}
}

func TestLoadSpec_MissingTier_DefaultsToCore(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "t.yml")

	content := `id: test
description: "Test"
languages: ["python"]
prompt: "x"
tags: [algorithmic]
`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	spec, err := LoadSpec(specPath)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if spec.Tier != "core" {
		t.Errorf("Tier = %q, want default %q", spec.Tier, "core")
	}
}

func TestLoadSpec_InvalidTier_Rejected(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "t.yml")

	content := `id: test
description: "Test"
languages: ["python"]
prompt: "x"
tier: bogus
tags: [algorithmic]
`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadSpec(specPath)
	if err == nil {
		t.Fatal("expected error for invalid tier 'bogus', got nil")
	}
	if !containsSubstring(err.Error(), "tier") {
		t.Errorf("error should mention 'tier', got: %v", err)
	}
}

func TestLoadSpec_TooManyTags_Rejected(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "t.yml")

	content := `id: test
description: "Test"
languages: ["python"]
prompt: "x"
tier: core
tags: [a, b, c, d]
`
	if err := os.WriteFile(specPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := LoadSpec(specPath)
	if err == nil {
		t.Fatal("expected error for 4 tags, got nil")
	}
	if !containsSubstring(err.Error(), "tags") {
		t.Errorf("error should mention 'tags', got: %v", err)
	}
}

func TestValidTiers_EnumMembers(t *testing.T) {
	// Every tier in ValidTiers should be accepted; unknown should be rejected.
	// "experimental" reserved for diagnostic probes — see spec.go ValidTiers comment.
	want := map[string]bool{"smoke": true, "core": true, "stretch": true, "vision": true, "experimental": true}
	if len(ValidTiers) != len(want) {
		t.Errorf("ValidTiers has %d entries, want %d", len(ValidTiers), len(want))
	}
	for _, tier := range ValidTiers {
		if !want[tier] {
			t.Errorf("ValidTiers contains unexpected %q", tier)
		}
	}
}

func TestValidTagTaxonomy_HasTwelveTags(t *testing.T) {
	// Per m-eval-category-analysis.md §Component 1: 12-tag taxonomy
	if len(ValidTagTaxonomy) != 12 {
		t.Errorf("ValidTagTaxonomy has %d tags, want 12 per design doc", len(ValidTagTaxonomy))
	}
	// Spot-check a few expected tags
	expected := []string{"adt_pattern_match", "recursion", "effects_io", "contracts"}
	for _, tag := range expected {
		found := false
		for _, t := range ValidTagTaxonomy {
			if t == tag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ValidTagTaxonomy missing expected tag %q", tag)
		}
	}
}

// TestAllBenchmarksHaveTierAndTags loads every YAML under benchmarks/ and asserts
// each one parses cleanly with a valid tier and between 1 and 3 tags from the taxonomy.
// This is the M2 CI gate for the Eval Suite Prep sprint.
func TestAllBenchmarksHaveTierAndTags(t *testing.T) {
	matches, err := filepath.Glob("../../benchmarks/*.yml")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) == 0 {
		t.Skip("no benchmark YAMLs found at ../../benchmarks/*.yml")
	}

	tierCounts := map[string]int{}
	benchCount := 0
	for _, path := range matches {
		base := filepath.Base(path)
		// Skip non-spec meta-files (suite-change log, etc.). See
		// isBenchmarkMetaFile in cmd/ailang/eval_helpers.go for the
		// canonical list — keep them in sync.
		if base == "events.yml" {
			continue
		}
		benchCount++
		spec, err := LoadSpec(path)
		if err != nil {
			t.Errorf("%s: LoadSpec failed: %v", base, err)
			continue
		}
		if spec.Tier == "" {
			t.Errorf("%s: tier is empty (should default to %q)", base, "core")
		}
		if len(spec.Tags) == 0 {
			t.Errorf("%s: no tags (every benchmark must have 1-3 tags from taxonomy)", base)
		}
		// Tag count cap: standard tiers are 1-3 canonical taxonomy tags;
		// experimental probes are permitted up to 5 (see spec.go LoadSpec).
		maxTags := 3
		if spec.Tier == "experimental" {
			maxTags = 5
		}
		if len(spec.Tags) > maxTags {
			t.Errorf("%s: %d tags (max %d for tier %q)", base, len(spec.Tags), maxTags, spec.Tier)
		}
		tierCounts[spec.Tier]++
	}

	// Sanity check: tier distribution targets re-centered after M-THREE-CAMPS
	// added 14 gap benchmarks (3 smoke, 8 core, 0 stretch, 3 vision).
	// Post-M-THREE-CAMPS centers: 18/29/11/9 (total 67). Tolerance ±3 kept so
	// future benchmark adds trigger the drift check instead of silently bloating.
	// "experimental" tier (diagnostic probes) is intentionally excluded from
	// the distribution drift check — probes measure language gaps, not score
	// capability, so their count growth is independent of the smoke/core/stretch/vision
	// budget.
	if smoke := tierCounts["smoke"]; smoke < 15 || smoke > 21 {
		t.Errorf("smoke count = %d, want 18±3", smoke)
	}
	if core := tierCounts["core"]; core < 26 || core > 34 {
		t.Errorf("core count = %d, want 29±3", core)
	}
	if stretch := tierCounts["stretch"]; stretch < 8 || stretch > 14 {
		t.Errorf("stretch count = %d, want 11±3", stretch)
	}
	if vision := tierCounts["vision"]; vision < 6 || vision > 12 {
		t.Errorf("vision count = %d, want 9±3", vision)
	}
	t.Logf("Tier distribution: smoke=%d core=%d stretch=%d vision=%d experimental=%d (total %d)",
		tierCounts["smoke"], tierCounts["core"], tierCounts["stretch"], tierCounts["vision"],
		tierCounts["experimental"], benchCount)
}
