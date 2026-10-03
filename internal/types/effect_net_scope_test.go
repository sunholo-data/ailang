package types

import (
	"strings"
	"testing"
)

// M-NET-SCOPE-PUBLIC (#1522): Net[scope=public] in the effect-param schema
// and the narrowing-param subsumption rule (design D-A).

func TestNetScopePublic_SchemaLegal(t *testing.T) {
	row, err := ElaborateEffectRowWithBudgets(ann("Net", "scope", "public"))
	if err != nil {
		t.Fatalf("Net[scope=public] must elaborate clean, got %v", err)
	}
	if got := row.Params["Net"]["scope"]; got != "public" {
		t.Fatalf("scope param not carried on the row: %#v", row.Params)
	}
	// Bare Net stays bare: no default is registered, so rows print unchanged.
	bare, err := ElaborateEffectRow([]string{"Net"})
	if err != nil {
		t.Fatal(err)
	}
	if len(bare.Params["Net"]) != 0 {
		t.Fatalf("bare Net gained params: %#v", bare.Params)
	}
	if FormatEffectRow(bare) != "! {Net}" {
		t.Fatalf("bare Net renders as %q", FormatEffectRow(bare))
	}
}

func TestNetScope_UnknownValueRejected(t *testing.T) {
	_, err := ElaborateEffectRowWithBudgets(ann("Net", "scope", "internal"))
	if err == nil || !strings.Contains(err.Error(), "EFF_UNKNOWN_MODE") || !strings.Contains(err.Error(), "public") {
		t.Fatalf("Net[scope=internal] must be EFF_UNKNOWN_MODE listing public, got %v", err)
	}
	_, err = ElaborateEffectRowWithBudgets(ann("Net", "mode", "live"))
	if err == nil || !strings.Contains(err.Error(), "EFF_UNKNOWN_PARAM_KEY") {
		t.Fatalf("Net[mode=live] must be EFF_UNKNOWN_PARAM_KEY, got %v", err)
	}
}

func TestNetScopePublic_SubsumesBareNet(t *testing.T) {
	bare := effectTestRow("Net", nil)
	public := effectTestRow("Net", map[string]string{"scope": "public"})
	// A public function may call httpGet ! {Net} (crypto-satisfies-os direction).
	if !SubsumeEffectRows(bare, public) {
		t.Fatalf("Net[scope=public] declaration must cover a bare Net requirement: %#v", DiffEffectRows(bare, public))
	}
	// A bare-Net handler may call a public-scoped function: the scope narrows
	// the callee's own frame, it is not a capability the caller must hold.
	if !SubsumeEffectRows(public, bare) {
		t.Fatalf("bare Net declaration must cover a Net[scope=public] requirement: %#v", DiffEffectRows(public, bare))
	}
	if !SubsumeEffectRows(public, public) {
		t.Fatal("Net[scope=public] must cover itself")
	}
}

func TestNetScope_DoesNotLeakToOtherEffects(t *testing.T) {
	// The narrowing rule is keyed by (effect, key, value): AI's scope stays
	// invariant (TestNonModeParametersRemainInvariant covers byok vs managed;
	// this covers declared-only byok, which has no narrowing entry).
	if SubsumeEffectRows(effectTestRow("AI", nil), effectTestRow("AI", map[string]string{"mode": "fixed", "scope": "byok"})) {
		t.Fatal("AI[scope=byok] wrongly inherited Net's narrowing rule")
	}
	if _, err := ElaborateEffectRowWithBudgets(ann("Rand", "scope", "public")); err == nil {
		t.Fatal("Rand[scope=public] must be rejected: scope=public is a Net-only value")
	}
	if isNarrowingParam("Net", "mode", "public") || isNarrowingParam("AI", "scope", "public") {
		t.Fatal("narrowing lookup must key on effect AND key")
	}
	// Rand's mode edges are unaffected.
	if SubsumeEffectRows(effectTestRow("Rand", map[string]string{"mode": "crypto"}), effectTestRow("Rand", nil)) {
		t.Fatal("bare Rand must still not cover crypto")
	}
}
