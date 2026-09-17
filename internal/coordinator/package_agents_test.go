package coordinator

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

func pkgTemplate() *AgentConfig {
	return &AgentConfig{
		Label:     "Package template",
		Workspace: "sunholo-data/ailang-packages", MergeBranch: "main",
		Capabilities: []string{"code", "test"}, ArtifactPatterns: []string{"packages/**/*"},
		Provider: "pi", ToolPolicy: "ailang_only", PolicyPath: "/etc/ailang-config/policies/pkg-ailang-only.toml",
	}
}

// A template on the ailang_only lane without a policy_path would derive agents
// that can execute nothing (dispatch refuses them): derive nothing, loudly.
func TestMaterializePackageAgents_RefusesLaneWithoutPolicy(t *testing.T) {
	reg := NewAgentRegistry()
	tmpl := pkgTemplate()
	tmpl.PolicyPath = ""
	tmpl.ToolPolicy = "" // defaults to ailang_only for a pkg: inbox
	reg.SetPackageAgentTemplate(tmpl)
	if added := reg.MaterializePackageAgents(sampleIndex()); len(added) != 0 {
		t.Fatalf("derived %v from a template with no policy_path", added)
	}
	tmpl.ToolPolicy = "full" // a shell lane needs no policy file
	if added := reg.MaterializePackageAgents(sampleIndex()); len(added) == 0 {
		t.Fatal("full-lane template without policy_path should derive")
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
	reg.SetPackageAgentTemplate(pkgTemplate())
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
	if a.Provider != "pi" || a.ToolPolicy != "ailang_only" || a.PolicyPath == "" || len(a.ArtifactPatterns) != 1 || a.ArtifactPatterns[0] != "ext/help/**/*" {
		t.Errorf("template fields not carried: %+v", a)
	}
	// Repo-is-the-package: `**/*` stated explicitly, not left to default.
	if root, ok := DerivePackageAgent(pkgTemplate(), pkg.IndexEntry{Name: "sunholo/ailang_parse", Repository: "https://github.com/sunholo-data/ailang-parse"}); !ok || root.Subdirectory != "" || len(root.ArtifactPatterns) != 1 || root.ArtifactPatterns[0] != "**/*" || root.MergeBranch != "main" {
		t.Errorf("root-package derivation = %+v", root)
	}
	if got := reg.GetAgentForInbox("pkg:sunholo/email"); got == nil || got.Label != "hand" {
		t.Errorf("hand-written agent must win: %+v", got)
	}
	// No repository: the repo cannot be guessed from the name, so the inbox
	// stays visibly unserved (PUB021 on publish) rather than dispatching to
	// a clone that finds nothing.
	if got := reg.GetAgentForInbox("pkg:sunholo/logging"); got != nil {
		t.Errorf("repository-less package must not get a guessed agent: %+v", got)
	}
	if !strings.Contains(PackageInboxStatus(pkg.IndexEntry{Name: "sunholo/logging"}), "no [metadata] repository") {
		t.Error("inbox status must say why no agent was derived")
	}
	// A package that is NOT in the registry is a sender typo: the template
	// must not swallow it into a task on the wrong repo (the config's
	// deliberate "typos bounce" control).
	if got := reg.GetAgentForInbox("pkg:sunholo/emial"); got != nil {
		t.Errorf("typo inbox must stay unserved, got %+v", got)
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
		t.Fatalf("no template but derived %v", added)
	}
	if reg.PackageAgentTemplate() != nil {
		t.Error("a family pattern must not be mistaken for the template")
	}
}

// The template is a config SECTION, not an agent: loading a config that
// declares it must not register an agent or serve any inbox by itself.
func TestBuildRegistryFromConfig_TemplateIsNotAnAgent(t *testing.T) {
	cfg := &CoordinatorConfig{
		Agents:               []*AgentConfig{{ID: "a", Inbox: "sprint-executor", Workspace: "w"}},
		PackageAgentTemplate: pkgTemplate(),
	}
	// Point the registry client at nothing reachable so materialization is a
	// logged no-op rather than a network call in a unit test.
	t.Setenv("AILANG_REGISTRY", "http://127.0.0.1:9/nope")
	reg, err := buildRegistryFromConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if reg.Count() != 1 {
		t.Errorf("template must not be registered as an agent: %d agents", reg.Count())
	}
	if reg.GetAgentForInbox("pkg:sunholo/typo") != nil {
		t.Error("an undeclared pkg: inbox must stay unserved (typos bounce)")
	}
	if reg.PackageAgentTemplate() == nil {
		t.Error("template not installed on the registry")
	}
}

func TestDerivePackageAgent_RefusesUnparseableRepository(t *testing.T) {
	if _, ok := DerivePackageAgent(pkgTemplate(), pkg.IndexEntry{Name: "v/n", Repository: "https://gitlab.com/v/n"}); ok {
		t.Error("non-GitHub repository must not derive a guessed agent")
	}
	if _, ok := DerivePackageAgent(nil, pkg.IndexEntry{Name: "v/n", Repository: "https://github.com/v/n"}); ok {
		t.Error("nil template must not derive")
	}
	if _, ok := DerivePackageAgent(pkgTemplate(), pkg.IndexEntry{}); ok {
		t.Error("nameless entry must not derive")
	}
}
