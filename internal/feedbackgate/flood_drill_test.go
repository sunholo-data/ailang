package feedbackgate

import (
	"bytes"
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestRunFloodDrill exercises the drill engine and asserts the gate caps
// dispatch far below the baseline (the whole point of the flood drill).
func TestRunFloodDrill(t *testing.T) {
	cfg := FeedbackGateConfig{}.normalized()
	cfg.Cooldown = newFakeCooldownStore()
	// No classifier: rules + cooldown alone should already gate the flood.

	res, err := RunFloodDrill(context.Background(), cfg, 1000, 10)
	if err != nil {
		t.Fatalf("RunFloodDrill error: %v", err)
	}
	if res.Total != 1000 {
		t.Fatalf("total = %d, want 1000", res.Total)
	}
	// 10 contacts * 3/hour dispatch cap = at most 30 dispatched.
	if res.Dispatched > 30 {
		t.Errorf("dispatched = %d, want <= 30 (10 contacts * 3/hr)", res.Dispatched)
	}
	// Spend with the gate must be a small fraction of the ungated baseline.
	if res.SimulatedSpend >= res.BaselineSpend {
		t.Errorf("gated spend %.4f not below baseline %.4f", res.SimulatedSpend, res.BaselineSpend)
	}
	// Every message accounted for (no silent drops).
	if res.Dispatched+res.Filed+res.Rejected != res.Total {
		t.Errorf("verdict counts %d+%d+%d != total %d",
			res.Dispatched, res.Filed, res.Rejected, res.Total)
	}

	var buf bytes.Buffer
	res.WriteReport(&buf)
	if !strings.Contains(buf.String(), "simulated spend") {
		t.Error("report missing spend line")
	}
}

// TestFloodDrillReportEntrypoint is the entrypoint the offline shell script
// (scripts/security/feedback_flood_drill.sh) drives. It runs ONLY when
// FEEDBACK_FLOOD_DRILL=1 so it doesn't add noise to the normal suite. It reads
// FLOOD_N / FLOOD_CONTACTS from the env and prints the report to stdout.
// Entirely offline: in-memory cooldown, no provider, no network.
func TestFloodDrillReportEntrypoint(t *testing.T) {
	if os.Getenv("FEEDBACK_FLOOD_DRILL") != "1" {
		t.Skip("set FEEDBACK_FLOOD_DRILL=1 to run the flood-drill report")
	}
	n := envInt("FLOOD_N", 1000)
	contacts := envInt("FLOOD_CONTACTS", 10)

	cfg := FeedbackGateConfig{}.normalized()
	cfg.Cooldown = newFakeCooldownStore()

	res, err := RunFloodDrill(context.Background(), cfg, n, contacts)
	if err != nil {
		t.Fatalf("RunFloodDrill error: %v", err)
	}
	res.WriteReport(os.Stdout)
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
