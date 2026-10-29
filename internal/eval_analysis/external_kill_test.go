package eval_analysis

import "testing"

// external_kill: the agent CLI was SIGTERMed from outside the harness (a
// sibling eval agent's `pkill -f`). It says nothing about the model, so it is
// excluded from capability stats, counted, and bucketed with the other
// not-the-model failures — never as capability_blocked, which is where an
// unknown category used to fall.
func TestBuildSweetSpot_ExternalKillIsNotACapabilityFailure(t *testing.T) {
	if !ShouldExcludeFromCapability("external_kill") {
		t.Error("ShouldExcludeFromCapability(external_kill) = false — killed runs would drag the pass rate down")
	}
	results := []*BenchmarkResult{
		sweetSpotResult("b1", "m", "", "none", true, 0.05, 30_000),
		sweetSpotResult("b2", "m", "", "external_kill", false, 0, 0),
	}
	r := BuildSweetSpot(results, SweetSpotOpts{})
	if len(r.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(r.Rows))
	}
	row := r.Rows[0]
	if row.PassRate != 1.0 {
		t.Errorf("PassRate = %v, want 1.0 (killed row excluded)", row.PassRate)
	}
	if row.ExternalKillCount != 1 {
		t.Errorf("ExternalKillCount = %d, want 1", row.ExternalKillCount)
	}
	if row.Buckets.CapabilityBlocked != 0 || row.Buckets.ProviderBlocked != 1 {
		t.Errorf("buckets = %+v, want the killed benchmark provider_blocked, not capability_blocked", row.Buckets)
	}
}
