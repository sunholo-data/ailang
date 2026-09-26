package cloudrun

import (
	"fmt"
	"sort"
	"testing"
)

// TestVariantProviderAgreement_AcceptSetIsPinned pins the EXACT set of
// (variant, provider) cells that checkVariantProviderAgreement accepts today.
//
// Why this exists. checkVariantProviderAgreement is one of two route
// authorities: Dispatcher.Dispatch consults coordinator.ValidateExecutionRoute
// first and this guard second, so the dispatchable set is their INTERSECTION.
// Nothing in the repo compared the two. Measured 2026-08-31 (mission iteration
// 309): the coordinator-side matrix proposed in commit 3500db0a7 accepts only
// 10 of the 104 cells this guard accepts 49 of, so landing it would have
// silently made 39 cells permanently undispatchable -- managed_agents on every
// variant, the empty provider on every variant, the eval/eval-go wildcard
// images, and pi/pi-go. Every existing test called one guard or the other, so
// no gate could see the intersection shrink.
//
// This test does not know about that other matrix and does not need to. It
// pins THIS guard's behaviour, so any future change to the accept-set -- from
// either direction -- has to be deliberate and has to restate the number.
//
// The enumeration is derived from the live maps rather than hand-listed, so a
// newly added variant or provider moves the count and fails here rather than
// slipping past an allowlist someone forgot to update.
func TestVariantProviderAgreement_AcceptSetIsPinned(t *testing.T) {
	variants := make([]string, 0, len(knownVariants))
	for v := range knownVariants {
		variants = append(variants, v)
	}
	sort.Strings(variants)

	providerSet := map[string]struct{}{"": {}}
	for _, provs := range providersForVariant {
		for _, p := range provs { // nil ("any provider") contributes nothing
			providerSet[p] = struct{}{}
		}
	}
	for p := range binarylessProviders {
		providerSet[p] = struct{}{}
	}
	providers := make([]string, 0, len(providerSet))
	for p := range providerSet {
		providers = append(providers, p)
	}
	sort.Strings(providers)

	// Anti-vacuity floor: an empty enumeration would make every assertion
	// below trivially true, which reads exactly like a clean pass.
	if len(variants) == 0 || len(providers) == 0 {
		t.Fatalf("instrument failure: enumeration is vacuous (variants=%d providers=%d)",
			len(variants), len(providers))
	}

	accepted := make([]string, 0, len(variants)*len(providers))
	for _, v := range variants {
		for _, p := range providers {
			if checkVariantProviderAgreement(v, p) == nil {
				accepted = append(accepted, fmt.Sprintf("%s/%s", v, p))
			}
		}
	}

	cells := len(variants) * len(providers)
	const wantCells = 104
	const wantAccepted = 49

	if cells != wantCells {
		t.Errorf("cells = %d, want %d (variants=%d providers=%d).\n"+
			"A variant or provider was added or removed. That is allowed, but it changes the\n"+
			"dispatchable surface: update these constants deliberately and say why in the commit.",
			cells, wantCells, len(variants), len(providers))
	}
	if len(accepted) != wantAccepted {
		t.Errorf("accepted = %d, want %d.\nAccepted set:\n  %v",
			len(accepted), wantAccepted, accepted)
	}

	// Positive controls: a cell this guard must accept, and one it must refuse.
	// Without these, a guard that accepted or refused EVERYTHING could still
	// land on the right total by coincidence.
	if err := checkVariantProviderAgreement("pi-go", "pi"); err != nil {
		t.Errorf("control failed: (pi-go, pi) must be accepted, got %v", err)
	}
	if err := checkVariantProviderAgreement("codex", "pi"); err == nil {
		t.Error("control failed: (codex, pi) must be refused -- the pi binary is not in the codex image")
	}
}
