package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
)

// TestRegisterRoutingFlags_Parsing verifies that the shared routing flag
// registration accepts the same flags `ailang exec --api-only` already
// supports. This is the M-AI-OPENROUTER follow-up: `ailang run` must accept
// these flags rather than rejecting them with "flag not defined".
func TestRegisterRoutingFlags_Parsing(t *testing.T) {
	fs := flag.NewFlagSet("test-run", flag.ContinueOnError)
	fs.SetOutput(io_discard{})
	r := registerRoutingFlags(fs)

	args := []string{
		"--routing-fallback", "anthropic,openai,google",
		"--routing-require", "structured_outputs,tool_calling",
		"--routing-prefer", "cheapest",
		"--routing-max-price", "5.00",
		"--allow-routing",
	}
	if err := fs.Parse(args); err != nil {
		t.Fatalf("Parse() error = %v (the run command should accept routing flags)", err)
	}
	if *r.fallback != "anthropic,openai,google" {
		t.Errorf("fallback = %q", *r.fallback)
	}
	if *r.require != "structured_outputs,tool_calling" {
		t.Errorf("require = %q", *r.require)
	}
	if *r.prefer != "cheapest" {
		t.Errorf("prefer = %q", *r.prefer)
	}
	if *r.maxPrice != "5.00" {
		t.Errorf("maxPrice = %q", *r.maxPrice)
	}
	if !*r.allowRouting {
		t.Errorf("allowRouting = false, want true")
	}
}

// TestBuildRoutingPolicy_SafetyGate verifies the design-doc safety gate:
// any --routing-* flag set without --allow-routing must fail loudly. This
// is the runtime equivalent of the AI[Routeable] type-level marker.
func TestBuildRoutingPolicy_SafetyGate(t *testing.T) {
	// fallback set, allowRouting=false → error
	_, err := buildRoutingPolicy("", "anthropic,openai", "", "", "", false)
	if err == nil {
		t.Fatal("buildRoutingPolicy without --allow-routing should fail")
	}
	if !strings.Contains(err.Error(), "--allow-routing") {
		t.Errorf("error = %v, want it to mention --allow-routing", err)
	}

	// fallback set, allowRouting=true → success
	p, err := buildRoutingPolicy("", "anthropic,openai", "", "", "", true)
	if err != nil {
		t.Fatalf("with --allow-routing: error = %v", err)
	}
	if p == nil {
		t.Fatal("policy = nil, want populated")
	}
	if !p.AllowFallback {
		t.Error("AllowFallback = false, want true (set automatically when fallback list provided)")
	}
	if len(p.Order) != 2 || p.Order[0] != "anthropic" || p.Order[1] != "openai" {
		t.Errorf("Order = %v, want [anthropic openai]", p.Order)
	}
}

// TestBuildRoutingPolicy_NoFlagsReturnsNil checks that the empty case
// returns (nil, nil) — no policy, no error. This matters because the run
// command passes the resulting policy to setupAIHandler which checks for nil.
func TestBuildRoutingPolicy_NoFlagsReturnsNil(t *testing.T) {
	p, err := buildRoutingPolicy("", "", "", "", "", false)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if p != nil {
		t.Errorf("policy = %+v, want nil", p)
	}
}

// TestBuildRoutingPolicy_PreferValidation verifies that --routing-prefer is
// validated against the closed set defined in internal/ai/routing.go.
func TestBuildRoutingPolicy_PreferValidation(t *testing.T) {
	cases := []struct {
		prefer  string
		want    ai.RoutePreference
		wantErr bool
	}{
		{"", ai.PreferUnspecified, false},
		{"cheapest", ai.PreferCheapest, false},
		{"fastest", ai.PreferFastest, false},
		{"most_reliable", ai.PreferMostReliable, false},
		{"bogus", ai.PreferUnspecified, true},
	}
	for _, tc := range cases {
		t.Run(tc.prefer, func(t *testing.T) {
			// Need at least one routing flag set so the function actually
			// builds a policy; pair prefer with a fallback list.
			p, err := buildRoutingPolicy("", "anthropic", "", tc.prefer, "", true)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error for invalid prefer")
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if p.Prefer != tc.want {
				t.Errorf("Prefer = %q, want %q", p.Prefer, tc.want)
			}
		})
	}
}

// TestBuildRoutingPolicy_ProviderHintEnforcement verifies that when the
// caller supplies a non-empty provider that isn't openrouter, we reject
// the routing flags with a friendly diagnostic. `ailang run` passes ""
// here (provider is inferred at handler-setup time), so the check is
// exercised primarily by `ailang exec --api-only <provider>`.
func TestBuildRoutingPolicy_ProviderHintEnforcement(t *testing.T) {
	_, err := buildRoutingPolicy("anthropic", "anthropic,openai", "", "", "", true)
	if err == nil {
		t.Fatal("expected error when provider hint is not openrouter")
	}
	if !strings.Contains(err.Error(), "openrouter") {
		t.Errorf("error = %v, want it to mention openrouter", err)
	}

	// Empty provider hint → no enforcement (run command path)
	p, err := buildRoutingPolicy("", "anthropic,openai", "", "", "", true)
	if err != nil {
		t.Fatalf("empty provider hint should skip enforcement, got error = %v", err)
	}
	if p == nil {
		t.Fatal("policy = nil")
	}

	// Explicit openrouter → no enforcement
	p, err = buildRoutingPolicy("openrouter", "anthropic,openai", "", "", "", true)
	if err != nil {
		t.Fatalf("openrouter provider: error = %v", err)
	}
	if p == nil {
		t.Fatal("policy = nil")
	}
}

// io_discard is a tiny io.Writer used to suppress flag.FlagSet error output
// in tests (avoids polluting test output with usage messages on parse failures).
type io_discard struct{}

func (io_discard) Write(p []byte) (int, error) { return len(p), nil }
