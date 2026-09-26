package observatory

import "testing"

// M-V1-SIMPLIFY-S4 M1: a span whose model the registry cannot price is stored
// at cost_usd=0 (the float column has no other honest value) but STAMPED
// ailang.cost.unpriced=true, so a reader can tell that zero from a free or
// unbilled call. The priced span is the positive control: no stamp.
func TestPriceSpanTokens_StampsUnpricedOnTheSpan(t *testing.T) {
	loadShippedRegistry(t)

	priced := map[string]any{}
	if cost := priceSpanTokens(priced, "claude-sonnet-4-5", 1000, 1000, 0, 0); cost <= 0 {
		t.Fatalf("control: claude-sonnet-4-5 priced at %v; the registry has this row", cost)
	}
	if _, stamped := priced[AttrCostUnpriced]; stamped {
		t.Fatalf("control: a priced span must not carry %s: %v", AttrCostUnpriced, priced)
	}

	unpriced := map[string]any{"gen_ai.request.model": "unknown-nonexistent-model"}
	if cost := priceSpanTokens(unpriced, "unknown-nonexistent-model", 1000, 1000, 0, 0); cost != 0 {
		t.Fatalf("unpriced: cost = %v, want 0", cost)
	}
	if v, ok := unpriced[AttrCostUnpriced].(bool); !ok || !v {
		t.Fatalf("unpriced: %s = %v, want true", AttrCostUnpriced, unpriced[AttrCostUnpriced])
	}
	if reason, _ := unpriced[AttrCostUnpriced+"_reason"].(string); reason == "" {
		t.Fatalf("unpriced: the reason attribute must name why; got %v", unpriced)
	}
	// The span's own attributes are untouched.
	if unpriced["gen_ai.request.model"] != "unknown-nonexistent-model" {
		t.Fatalf("existing attributes clobbered: %v", unpriced)
	}
}
