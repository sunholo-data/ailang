package coordinator

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func pkgTemplate() *AgentConfig {
	return &AgentConfig{
		ID: "pkg-registry-template", Label: "Package template", Inbox: PackageAgentTemplateInbox,
		Workspace: "sunholo-data/ailang-packages", MergeBranch: "main",
		Capabilities: []string{"code", "test"}, ArtifactPatterns: []string{"packages/**/*"},
		Provider: "pi", ToolPolicy: "ailang_only",
	}
}

func sampleIndex() *pkg.RegistryIndex {
	return &pkg.RegistryIndex{Packages: []pkg.IndexEntry{
		{Name: "sunholo/gcp_auth", Repository: "https://github.com/sunholo-data/ailang-packages/tree/main/packages/gcp-auth"},
		{Name: "sunholo/daneel_ext_help", Repository: "https://github.com/sunholo-data/daneel/tree/main/ext/help"},
		{Name: "sunholo/logging"}, // no repository → no derivation
		{Name: "sunholo/email", Repository: "https://github.com/sunholo-data/email-parse/tree/main/packages/email"},
	}}
}

// The whole point: a package that only exists in the registry index gets an
// agent whose inbox, workspace and subdirectory follow its repository URL.
func TestMaterializePackageAgents_DerivesFromIndex(t *testing.T) {
	reg := NewAgentRegistry()
	if err := reg.Register(pkgTemplate()); err != nil {
		t.Fatal(err)
	}
	// Hand-written entry for sunholo/email must win over derivation.
	hand := &AgentConfig{ID: "pkg-sunholo-email", Inbox: "pkg:sunholo/email", Workspace: "sunholo-data/email-parse", Subdirectory: "packages/email", Label: "hand"}
	if err := reg.Register(hand); err != nil {
		t.Fatal(err)
	}

	added := reg.MaterializePackageAgents(sampleIndex())
	if len(added) != 2 || added[0] != "pkg-sunholo-daneel-ext-help" || added[1] != "pkg-sunholo-gcp-auth" {
		t.Fatalf("added = %v", added)
	}

	a := reg.GetAgentForInbox("pkg:sunholo/daneel_ext_help")
	if a == nil || a.ID != "pkg-sunholo-daneel-ext-help" || a.Workspace != "sunholo-data/daneel" || a.Subdirectory != "ext/help" || a.MergeBranch != "main" {
		t.Errorf("derived daneel agent = %+v", a)
	}
	if a.Provider != "pi" || a.ToolPolicy != "ailang_only" || len(a.ArtifactPatterns) != 1 || a.ArtifactPatterns[0] != "ext/help/**/*" {
		t.Errorf("template fields not carried: %+v", a)
	}
	if got := reg.GetAgentForInbox("pkg:sunholo/email"); got == nil || got.Label != "hand" {
		t.Errorf("hand-written agent must win: %+v", got)
	}
	// No repository: falls through to the template pattern, not a wrong repo.
	if got := reg.GetAgentForInbox("pkg:sunholo/logging"); got == nil || got.ID != "pkg-registry-template" {
		t.Errorf("repository-less package should be served by the template: %+v", got)
	}
	// Idempotent.
	if again := reg.MaterializePackageAgents(sampleIndex()); len(again) != 0 {
		t.Errorf("second materialize added %v", again)
	}
	// Mutating the derived agent's slices must not touch the template.
	a.Capabilities[0] = "mutated"
	if reg.PackageAgentTemplate().Capabilities[0] == "mutated" {
		t.Error("derived agent shares the template's Capabilities slice")
	}
}

func TestMaterializePackageAgents_NoTemplateIsNoop(t *testing.T) {
	reg := NewAgentRegistry()
	_ = reg.Register(&AgentConfig{ID: "x", Inbox: "pkg:sunholo/motoko_ext_*", Workspace: "w"}) // a family pattern is not the template
	if added := reg.MaterializePackageAgents(sampleIndex()); len(added) != 0 {
		t.Fatalf("no pkg:* template but derived %v", added)
	}
	if reg.PackageAgentTemplate() != nil {
		t.Error("a family pattern must not be mistaken for the pkg:* template")
	}
}

func TestDerivePackageAgent_RefusesNonGitHub(t *testing.T) {
	if _, ok := DerivePackageAgent(pkgTemplate(), pkg.IndexEntry{Name: "v/n", Repository: "https://gitlab.com/v/n"}); ok {
		t.Error("non-GitHub repository must not derive an agent")
	}
	if _, ok := DerivePackageAgent(nil, pkg.IndexEntry{Name: "v/n", Repository: "https://github.com/v/n"}); ok {
		t.Error("nil template must not derive")
	}
}
