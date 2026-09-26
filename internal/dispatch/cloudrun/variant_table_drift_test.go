package cloudrun

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// The coordinator keeps a copy of providersForVariant for its startup audit,
// because this package already imports the coordinator's types and the reverse
// edge would be a cycle.
//
// A duplicated table that silently diverges is worse than no table at all: the
// audit would clear an agent that dispatch then refuses, which is precisely the
// class of "the control passed and the thing still failed" defect the audit was
// added to remove.
func TestVariantTable_CoordinatorCopyHasNotDrifted(t *testing.T) {
	mine := ProvidersForVariant()
	theirs := coordinator.VariantProviders()

	for variant, allowed := range mine {
		got, ok := theirs[variant]
		if !ok {
			t.Errorf("dispatch knows variant %q; the coordinator's audit does not, so an agent using it is cleared at startup and refused at dispatch", variant)
			continue
		}
		if len(got) != len(allowed) {
			t.Errorf("variant %q: dispatch allows %v, audit allows %v", variant, allowed, got)
			continue
		}
		for i := range allowed {
			if got[i] != allowed[i] {
				t.Errorf("variant %q: dispatch allows %v, audit allows %v", variant, allowed, got)
				break
			}
		}
	}

	for variant := range theirs {
		if _, ok := mine[variant]; !ok {
			t.Errorf("the coordinator's audit knows variant %q and dispatch does not — the audit would judge an agent against a rule dispatch never applies", variant)
		}
	}
}
