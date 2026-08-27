package eval_harness

import (
	"github.com/sunholo-data/ailang/internal/modelreg"
	"strings"
	"testing"
)

// TestCloudRowsAreIdentifiableFromTheirKey enforces D6.
//
// AC2 requires that the ROUTE a trial took is recoverable from banked data
// alone. Three carriers were considered and two are dead, both by measurement:
//
//   - the `-cloud` suffix on api_name: the local daemon REWRITES the model field
//     to the base name before dispatch, so a `...:0731-cloud` request comes back
//     as `...:0731` (V21). It never reaches the bank.
//   - `remote_model` / `remote_host`: declared on ollama's ChatResponse struct,
//     but ABSENT from the actual response on BOTH /v1 and native /api/chat (V48).
//     The fields exist; the proxy does not populate them.
//
// What the bank DOES store is the models.yml row key (agent.friendlyName, V23).
// So the key is the only available carrier — which makes its naming a load-
// bearing invariant rather than a style preference. A design-quorum reviewer
// correctly refused to count an unenforced convention as a structural property;
// this test is what converts it into one.
//
// If this fails, a cloud trial has banked under a name indistinguishable from an
// on-device one, and no amount of later analysis can separate them.
func TestCloudRowsAreIdentifiableFromTheirKey(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("load models.yml: %v", err)
	}

	var offenders []string
	var cloudRows int
	for key, cfg := range modelreg.GlobalModelsConfig.Models {
		isCloud := IsOllamaCloudRoute(cfg.APIName) ||
			(cfg.AgentModelName != nil && IsOllamaCloudRoute(*cfg.AgentModelName))
		if !isCloud {
			continue
		}
		cloudRows++
		// "cloud" or the "oc-" prefix both mark the route unambiguously in the
		// banked model string. Anything else is invisible after banking.
		if !strings.Contains(key, "cloud") && !strings.HasPrefix(key, "oc-") {
			offenders = append(offenders, key)
		}
	}

	// Guard the guard: if the predicate ever stops matching, this test would
	// pass vacuously while enforcing nothing.
	if cloudRows == 0 {
		t.Fatal("no Ollama Cloud rows matched — the test is vacuous, so either " +
			"the rows were removed or IsOllamaCloudRoute stopped recognising them")
	}
	if len(offenders) > 0 {
		t.Errorf("%d Ollama Cloud row(s) are not identifiable as cloud from the banked "+
			"model string (the row key is the ONLY carrier — see V21/V23/V48): %v",
			len(offenders), offenders)
	}
	t.Logf("%d cloud rows, all route-identifiable from their key", cloudRows)
}

// TestNonCloudRowsAreNotMislabelled is the other direction: a local or metered
// row must NOT carry a cloud marker, or route recovery would report a GPU-bound
// or OpenRouter run as flat-rate cloud — and cost provenance keys off the same
// predicate, so the error would propagate into the KPI.
func TestNonCloudRowsAreNotMislabelled(t *testing.T) {
	if err := InitModelsConfig(); err != nil {
		t.Fatalf("load models.yml: %v", err)
	}
	for key, cfg := range modelreg.GlobalModelsConfig.Models {
		if !strings.Contains(key, "cloud") && !strings.HasPrefix(key, "oc-") {
			continue
		}
		isCloud := IsOllamaCloudRoute(cfg.APIName) ||
			(cfg.AgentModelName != nil && IsOllamaCloudRoute(*cfg.AgentModelName))
		if !isCloud {
			t.Errorf("row %q reads as a cloud row by name but its api_name %q is not a "+
				"cloud route — the name would mislead route recovery and cost provenance",
				key, cfg.APIName)
		}
	}
}
