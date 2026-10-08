package coordinator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAgentConfigAutoMergeCode(t *testing.T) {
	raw := []byte("id: site-agent\nauto_merge: true\nauto_merge_code: true\nauto_merge_required_checks: ['site, build']\nauto_merge_approver_secret: site-token\nauto_merge_approver_identity: reviewer\nartifact_patterns: ['site/**']\n")
	var a AgentConfig
	if err := yaml.Unmarshal(raw, &a); err != nil {
		t.Fatal(err)
	}
	if err := a.ValidateAutoMergeCode(); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var b AgentConfig
	if err = json.Unmarshal(data, &b); err != nil {
		t.Fatal(err)
	}
	if !b.AutoMergeCode || b.AutoMergeApproverIdentity != "reviewer" || b.AutoMergeRequiredChecks[0] != "site, build" {
		t.Fatalf("lost config: %s", data)
	}
	for _, mutate := range []func(*AgentConfig){func(a *AgentConfig) { a.AutoMerge = false }, func(a *AgentConfig) { a.AutoMergeRequiredChecks = nil }, func(a *AgentConfig) { a.AutoMergeRequiredChecks = []string{" "} }, func(a *AgentConfig) { a.AutoMergeRequiredChecks = []string{"a\nb"} }, func(a *AgentConfig) { a.AutoMergeApproverSecret = "" }, func(a *AgentConfig) { a.AutoMergeApproverIdentity = "" }, func(a *AgentConfig) { a.ArtifactPatterns = nil }, func(a *AgentConfig) { a.ArtifactPatterns = []string{" "} }, func(a *AgentConfig) { a.SkipApproval = true }} {
		c := b
		mutate(&c)
		if c.ValidateAutoMergeCode() == nil {
			t.Fatalf("invalid config accepted: %+v", c)
		}
	}
	if err := (&AgentConfig{}).ValidateAutoMergeCode(); err != nil {
		t.Fatal("docs default refused", err)
	}
}

func TestAgentConfigAutoMergeGuideExamples(t *testing.T) {
	guide, err := os.ReadFile(filepath.Join("..", "..", "docs", "docs", "guides", "coordinator.md"))
	if err != nil {
		t.Fatal(err)
	}
	blocks := strings.Split(string(guide), "```yaml\n")
	for _, id := range []string{"docs-writer", "site-agent"} {
		example := ""
		for _, block := range blocks[1:] {
			part := strings.SplitN(block, "```", 2)[0]
			if strings.HasPrefix(part, "- id: "+id+"\n") {
				example = part
				break
			}
		}
		if example == "" {
			t.Fatalf("guide example %s missing", id)
		}
		// Add the required registration fields to the guide's field-only example.
		example += "  inbox: " + id + "\n  workspace: /workspace/site\n  capabilities: [docs]\n"
		var b strings.Builder
		b.WriteString("coordinator:\n  agents:\n")
		for _, line := range strings.Split(strings.TrimSpace(example), "\n") {
			b.WriteString("    " + line + "\n")
		}
		file := filepath.Join(t.TempDir(), "config.yaml")
		if err := os.WriteFile(file, []byte(b.String()), 0600); err != nil {
			t.Fatal(err)
		}
		registry, err := LoadAgentRegistryFrom(file)
		if err != nil {
			t.Fatal(err)
		}
		if issues := registry.Validate(); len(issues) != 0 {
			t.Fatal(issues)
		}
		a := registry.GetAgentByID(id)
		if a == nil || !a.AutoMerge || a.AutoMergeCode != (id == "site-agent") {
			t.Fatalf("bad example %s: %+v", id, a)
		}
	}
}
