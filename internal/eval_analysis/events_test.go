package eval_analysis

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadSuiteEventsMissingFile: missing events.yml returns (nil, nil) —
// events are optional, so the caller in export_json.go can continue.
func TestLoadSuiteEventsMissingFile(t *testing.T) {
	events, err := LoadSuiteEvents(filepath.Join(t.TempDir(), "nonexistent.yml"))
	if err != nil {
		t.Fatalf("missing file should not error, got: %v", err)
	}
	if events != nil {
		t.Errorf("want nil slice, got %+v", events)
	}
}

// TestLoadSuiteEventsValid parses a minimal YAML doc with both simple and
// tier-scoped entries.
func TestLoadSuiteEventsValid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.yml")
	yaml := `
- version: v0.9.1.1
  label: "+5 contract benchmarks"
  kind: benchmark_add
  color: "#888"
- version: v0.14.0
  label: "Tier + tag taxonomy"
  kind: taxonomy
  affects_tiers: [stretch, vision]
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	events, err := LoadSuiteEvents(path)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("want 2 events, got %d", len(events))
	}
	if events[0].Kind != "benchmark_add" || events[0].Color != "#888" {
		t.Errorf("event[0] kind/color mismatch: %+v", events[0])
	}
	if len(events[1].AffectsTiers) != 2 || events[1].AffectsTiers[0] != "stretch" {
		t.Errorf("event[1] affects_tiers mismatch: %+v", events[1].AffectsTiers)
	}
}

// TestLoadSuiteEventsMissingRequired fails validation when version/label is
// missing. The loader surfaces the row index so a mistaken entry in
// benchmarks/events.yml is easy to pinpoint.
func TestLoadSuiteEventsMissingRequired(t *testing.T) {
	path := filepath.Join(t.TempDir(), "events.yml")
	yaml := `
- version: v0.9.1.1
  label: "ok"
  kind: benchmark_add
- kind: taxonomy
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSuiteEvents(path); err == nil {
		t.Fatal("want validation error for missing version/label, got nil")
	}
}
