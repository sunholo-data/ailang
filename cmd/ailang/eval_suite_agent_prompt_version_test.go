package main

import "testing"

// --prompt-version must reach agent runs: agent mode used to hard-code "", so a
// fine-tune arm asking for a slim or empty teaching prompt silently got the full
// active one (m-eval-finetuning-data-pipeline, 2026-11-05).
func TestBuildAgentSuiteConfig_CarriesPromptVersion(t *testing.T) {
	cfg := buildAgentSuiteConfig(true, agentSuiteConfigParams{promptVersion: "v0.16.10"})
	if cfg == nil || cfg.PromptVersion != "v0.16.10" {
		t.Fatalf("PromptVersion = %q, want v0.16.10", cfg.PromptVersion)
	}
	if def := buildAgentSuiteConfig(true, agentSuiteConfigParams{}); def.PromptVersion != "" {
		t.Fatalf("unset --prompt-version must stay \"\" (active prompt), got %q", def.PromptVersion)
	}
}
