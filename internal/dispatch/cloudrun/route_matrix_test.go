package cloudrun

import (
	"sort"
	"testing"
	_ "unsafe"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// coordinatorCloudProviderVariants links this test-only probe to the real M1
// provider matrix without copying its keys into a second table.
//
//go:linkname coordinatorCloudProviderVariants github.com/sunholo-data/ailang/internal/coordinator.cloudProviderVariants
var coordinatorCloudProviderVariants map[string]map[string]struct{}

func TestRouteMatrixParity(t *testing.T) {
	variants := make([]string, 0, len(knownVariants))
	for variant := range knownVariants {
		variants = append(variants, variant)
	}
	sort.Strings(variants)

	providers := make([]string, 0, len(coordinatorCloudProviderVariants)+2)
	for provider := range coordinatorCloudProviderVariants {
		providers = append(providers, provider)
	}
	providers = append(providers, "managed_agents", "")
	sort.Strings(providers)

	wantCells := len(variants) * len(providers)
	if wantCells == 0 {
		t.Fatalf("route matrix enumeration is vacuous: variants=%d providers=%d", len(variants), len(providers))
	}

	cells := 0
	overRejects := 0
	underRejects := 0
	positiveControl := false
	t.Log("variant\tprovider\tdev_accepts\tm1_accepts\tdisagrees")
	for _, variant := range variants {
		for _, provider := range providers {
			devAccepts := checkVariantProviderAgreement(variant, provider) == nil
			normalized := coordinator.NormalizeExecutionVariant(provider, variant)
			m1Accepts := coordinator.ValidateExecutionRoute("route-matrix-probe", provider, normalized) == nil
			disagrees := devAccepts != m1Accepts
			t.Logf("%q\t%q\t%t\t%t\t%t", variant, provider, devAccepts, m1Accepts, disagrees)

			cells++
			if devAccepts && !m1Accepts {
				overRejects++
			}
			if !devAccepts && m1Accepts {
				underRejects++
			}
			if variant == "pi-go" && provider == "pi" {
				positiveControl = disagrees
			}
		}
	}

	if cells != wantCells {
		t.Fatalf("route matrix enumeration incomplete: got %d cells, want %d (%d variants x %d providers)",
			cells, wantCells, len(variants), len(providers))
	}
	if !positiveControl {
		t.Fatal(`positive control failed: (variant "pi-go", provider "pi") is not a disagreement`)
	}
	t.Logf("TOTALS cells=%d m1_over_rejects=%d m1_under_rejects=%d positive_control=%t",
		cells, overRejects, underRejects, positiveControl)
}
