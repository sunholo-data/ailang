package coordinator

import "testing"

// The image decides which CLI runs, because the image is the only thing that
// determines which binaries exist.
//
// Measured 2026-09-03: sprint-executor declared provider "codex" with
// executor_variant "codex-go"; the dispatcher substituted the plane default
// "pi"; the runtime guard correctly refused, and the task retried every five
// minutes for half an hour without running. Two of four pipeline stages were
// undispatchable this way.
//
// The earlier fix made that state detectable. Deriving makes it unrepresentable.

func TestProviderForVariant_EachImageRunsExactlyOneCLI(t *testing.T) {
	cases := map[string]string{
		"":          "claude",
		"default":   "claude",
		"go":        "claude",
		"codex":     "codex",
		"codex-go":  "codex",
		"gemini":    "gemini",
		"gemini-go": "gemini",
		"opencode":  "opencode",
		"pi":        "pi",
		"pi-go":     "pi",
		"motoko":    "motoko",
	}
	for variant, want := range cases {
		got, ok := ProviderForVariant(variant)
		if !ok {
			t.Errorf("variant %q: no provider derived, but its image carries exactly one CLI", variant)
			continue
		}
		if got != want {
			t.Errorf("variant %q: derived %q, want %q", variant, got, want)
		}
	}
}

// TestProviderForVariant_MultiCLIImagesAreAChoice: agent-eval installs all five
// CLIs, so nothing can derive the answer and it must be declared.
func TestProviderForVariant_MultiCLIImagesAreAChoice(t *testing.T) {
	for _, variant := range []string{"eval", "eval-go"} {
		if _, ok := ProviderForVariant(variant); ok {
			t.Errorf("variant %q carries several CLIs; deriving one would pick an executor by accident", variant)
		}
	}
}

// TestResolveDispatchProvider_TheImageWinsOverAnyDeclaration is the core change.
// A contradicting declaration cannot take effect, whatever it says.
func TestResolveDispatchProvider_TheImageWinsOverAnyDeclaration(t *testing.T) {
	agent := &AgentConfig{ID: "sprint-executor", Provider: "pi", ExecutorVariant: "codex-go"}

	if got := ResolveDispatchProvider(agent, "pi"); got != "codex" {
		t.Errorf("provider = %q, want \"codex\" — agent-codex-go has no pi binary, so no declaration can make pi correct", got)
	}
}

// TestResolveDispatchProvider_PlaneDefaultCannotOverrideTheImage is the original
// defect, asserted directly.
func TestResolveDispatchProvider_PlaneDefaultCannotOverrideTheImage(t *testing.T) {
	for _, tc := range []struct{ id, variant, want string }{
		{"sprint-planner", "codex", "codex"},
		{"sprint-executor", "codex-go", "codex"},
		{"motoko", "motoko", "motoko"},
	} {
		agent := &AgentConfig{ID: tc.id, ExecutorVariant: tc.variant}
		if got := ResolveDispatchProvider(agent, "pi"); got != tc.want {
			t.Errorf("%s: provider = %q, want %q — the plane default reached the dispatcher and the job was refused", tc.id, got, tc.want)
		}
	}
}

func TestResolveDispatchProvider_MultiCLIImageUsesTheDeclaration(t *testing.T) {
	agent := &AgentConfig{ID: "evaluator", Provider: "opencode", ExecutorVariant: "eval"}
	if got := ResolveDispatchProvider(agent, "pi"); got != "opencode" {
		t.Errorf("provider = %q, want \"opencode\" — on an all-CLI image the declaration is the only signal", got)
	}
}

func TestResolveDispatchProvider_UnknownAgentKeepsThePlaneDefault(t *testing.T) {
	if got := resolveDispatchProvider(NewAgentRegistry(), "not-registered", &CoordinatorConfig{DefaultProvider: "pi"}); got != "pi" {
		t.Errorf("provider = %q, want \"pi\"", got)
	}
	if got := resolveDispatchProvider(nil, "x", nil); got != "claude" {
		t.Errorf("provider = %q, want \"claude\" when nothing is configured", got)
	}
}

// --- config-load validation ---

func TestValidateAgentProviders_RejectsADeclarationTheImageCannotHonour(t *testing.T) {
	errs := ValidateAgentProviders([]*AgentConfig{
		{ID: "broken", Provider: "pi", ExecutorVariant: "codex-go"},
	})
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	// Previously this shape dispatched, was refused at runtime, and retried forever.
	t.Logf("reported as: %v", errs[0])
}

func TestValidateAgentProviders_RequiresADeclarationOnMultiCLIImages(t *testing.T) {
	errs := ValidateAgentProviders([]*AgentConfig{
		{ID: "evaluator", ExecutorVariant: "eval"},
	})
	if len(errs) != 1 {
		t.Fatalf("an all-CLI image with no provider must be an error, got %v", errs)
	}
}

// TestValidateAgentProviders_AllowsARedundantDeclaration: all 34 live agents
// carry one. Redundant is not wrong, and failing on it would break every config
// in the fleet on upgrade.
func TestValidateAgentProviders_AllowsARedundantDeclaration(t *testing.T) {
	errs := ValidateAgentProviders([]*AgentConfig{
		{ID: "design-doc-creator", Provider: "pi", ExecutorVariant: "pi"},
		{ID: "sprint-executor", Provider: "codex", ExecutorVariant: "codex-go"},
		{ID: "pkg-auth", ExecutorVariant: "pi"}, // declares nothing — fine
	})
	if len(errs) != 0 {
		t.Errorf("correct configs were rejected: %v", errs)
	}
}
