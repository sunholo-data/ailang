package eval_analysis

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func loadFixtureArm(t *testing.T, name string) []*BenchmarkResult {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "paired", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var rows []*BenchmarkResult
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("parse fixture %s: %v", name, err)
	}
	return rows
}

// TestPairArms_Reproduces20260727 is the regression that anchors M3 to REAL
// banked data, not a hand-made fixture: these 168 rows are the actual
// 2026-07-27 weekly microRAG A/B, distilled to the pairing-relevant fields.
//
// The aggregate delta MUST come out at the historically-reported +13.1, or the
// schema change has silently altered a number the trend line depends on.
//
// It also demonstrates why M3 exists. The aggregate comparison could say
// nothing about this week: at n=84 the unpaired difference has a ~6.8pp
// standard error, so +13.1 sits right at the detection threshold and reads as
// noise. Paired, the same runs yield 25 discordant pairs split 18/7 — a real
// statistic from data we already had.
func TestPairArms_Reproduces20260727(t *testing.T) {
	on := loadFixtureArm(t, "microrag_20260727_on.json")
	off := loadFixtureArm(t, "microrag_20260727_off.json")

	p := PairArms(on, off)

	// Every row must pair; a non-zero Unpaired here would mean the join key is
	// wrong (e.g. Trial dropped) and the comparison is invalid.
	if len(p.Pairs) != 84 || p.Unpaired != 0 {
		t.Fatalf("pairs=%d unpaired=%d, want 84 and 0 — the arms must line up exactly", len(p.Pairs), p.Unpaired)
	}

	if p.OnPass != 65 || p.OnTotal != 84 || p.OffPass != 54 || p.OffTotal != 84 {
		t.Errorf("aggregates on=%d/%d off=%d/%d, want 65/84 and 54/84",
			p.OnPass, p.OnTotal, p.OffPass, p.OffTotal)
	}

	// The headline back-compat assertion.
	if math.Abs(p.DeltaPP-13.1) > 0.05 {
		t.Errorf("DeltaPP = %.4f, want the historically-banked +13.1", p.DeltaPP)
	}

	if p.OnlyOnPassed != 18 || p.OnlyOffPassed != 7 {
		t.Errorf("discordant b=%d c=%d, want 18 and 7", p.OnlyOnPassed, p.OnlyOffPassed)
	}

	// 25 discordant pairs is above both the reporting floor and the exact-test
	// cutoff, so this must use chi-square and produce a p-value.
	if !p.McNemar.Reportable {
		t.Fatalf("25 discordant pairs must be reportable, got: %s", p.McNemar.Note)
	}
	if p.McNemar.Method != "chi_square_continuity" {
		t.Errorf("method = %q, want chi_square_continuity (b+c=25)", p.McNemar.Method)
	}
	if math.Abs(p.McNemar.PValue-0.0455) > 0.001 {
		t.Errorf("p = %.4f, want ~0.0455", p.McNemar.PValue)
	}
}
