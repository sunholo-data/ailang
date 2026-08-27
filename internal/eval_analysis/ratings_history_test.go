package eval_analysis

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestHistoryEntry_PreExistingEntriesRoundTrip pins the additive contract: the
// 47 history entries already published carry no `ratings` key, and adding the
// field must not change how they serialize. If this fails, a release would
// silently rewrite its own history (M-EVAL-ROLLING-ELO M4).
func TestHistoryEntry_PreExistingEntriesRoundTrip(t *testing.T) {
	legacy := `{"version":"v0.32.0","timestamp":"2026-08-04T00:00:00Z","successRate":0.7,` +
		`"totalRuns":100,"successCount":70,"languages":"ailang,python"}`

	var entry HistoryEntry
	if err := json.Unmarshal([]byte(legacy), &entry); err != nil {
		t.Fatalf("unmarshal legacy entry: %v", err)
	}
	if entry.Ratings != nil {
		t.Errorf("legacy entry gained a non-nil Ratings: %+v", entry.Ratings)
	}
	out, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(legacy), &want); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if len(got) != len(want) {
		t.Errorf("key count changed: got %d keys %v, want %d keys %v", len(got), keysOf(got), len(want), keysOf(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("field %q changed: got %v, want %v", k, got[k], v)
		}
	}
}

// TestAttachRatingsHistory_ReadsStampedIndex proves the index is READ from the
// release's artifact, not recomputed — including the bridge strengths that make
// a historical index replayable.
func TestAttachRatingsHistory_ReadsStampedIndex(t *testing.T) {
	dir := t.TempDir()
	idx := filepath.Join(dir, "direction_index.json")
	doc := `{"version":"v0.35.0","panel_version":"direction_panel_v1","index_overall":1551.1,` +
		`"index_by_tier":{"core":1328.6},"bridge_strengths_used":{"gemini-3-flash":1922.8},"trials":86}`
	if err := os.WriteFile(idx, []byte(doc), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	entry := HistoryEntry{Version: "v0.35.0"}
	AttachRatingsHistory(&entry, map[string]float64{"model-a": 1994.84}, "anchor_v1", idx)

	if entry.Ratings == nil {
		t.Fatal("expected Ratings to be attached")
	}
	if entry.Ratings.DirectionIndex != 1551.1 {
		t.Errorf("direction index: got %v, want 1551.1", entry.Ratings.DirectionIndex)
	}
	if entry.Ratings.AnchorVersion != "anchor_v1" || entry.Ratings.PanelVersion != "direction_panel_v1" {
		t.Errorf("provenance missing: anchor=%q panel=%q", entry.Ratings.AnchorVersion, entry.Ratings.PanelVersion)
	}
	if got := entry.Ratings.BridgeStrengths["gemini-3-flash"]; got != 1922.8 {
		t.Errorf("bridge strengths not carried through: got %v", got)
	}
	if got := entry.Ratings.Models["model-a"]; got != 1994.8 {
		t.Errorf("model rating rounding: got %v, want 1994.8", got)
	}
}

// TestAttachRatingsHistory_NoIndexIsNotAnError: releases measured before the
// linking-run protocol carry the model half only, never a fabricated index.
func TestAttachRatingsHistory_NoIndexIsNotAnError(t *testing.T) {
	entry := HistoryEntry{Version: "v0.30.0"}
	AttachRatingsHistory(&entry, map[string]float64{"model-a": 1800}, "anchor_v1", "")
	if entry.Ratings == nil {
		t.Fatal("expected model half to attach")
	}
	if entry.Ratings.DirectionIndex != 0 {
		t.Errorf("index must stay zero when no artifact exists, got %v", entry.Ratings.DirectionIndex)
	}
}

func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
