package trace

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEffectEvent_OldFormatParses proves an EffectEvent serialised BEFORE the
// mode/contract fields existed still unmarshals cleanly, with Mode/Contract
// defaulting to "". This is the additive-schema back-compat guarantee
// (M-EFFECT-REPLAY-CONTRACTS).
func TestEffectEvent_OldFormatParses(t *testing.T) {
	oldJSON := `{"effect_name":"Rand","op_name":"rand_int","args":["1","6"],"result":"4"}`
	var evt EffectEvent
	if err := json.Unmarshal([]byte(oldJSON), &evt); err != nil {
		t.Fatalf("old-format EffectEvent failed to parse: %v", err)
	}
	if evt.Mode != "" {
		t.Errorf("Mode = %q, want empty for old-format events", evt.Mode)
	}
	if evt.Contract != "" {
		t.Errorf("Contract = %q, want empty for old-format events", evt.Contract)
	}
	if evt.EffectName != "Rand" || evt.OpName != "rand_int" {
		t.Errorf("core fields corrupted: %+v", evt)
	}
}

// TestEffectEvent_ModeContractRoundTrip verifies a moded event round-trips.
func TestEffectEvent_ModeContractRoundTrip(t *testing.T) {
	original := EffectEvent{
		EffectName: "Rand",
		OpName:     "rand_int",
		Args:       []string{"1", "100"},
		Result:     "57",
		Mode:       "seeded",
		Contract:   "deterministic",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got EffectEvent
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Mode != "seeded" {
		t.Errorf("Mode = %q, want seeded", got.Mode)
	}
	if got.Contract != "deterministic" {
		t.Errorf("Contract = %q, want deterministic", got.Contract)
	}
}

// TestEffectEvent_EmptyModeContractOmitted proves empty mode/contract are
// omitted from the wire (so old consumers see no new keys) — what makes the
// change additive.
func TestEffectEvent_EmptyModeContractOmitted(t *testing.T) {
	evt := EffectEvent{EffectName: "IO", OpName: "println", Result: "hi"}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	if strings.Contains(s, `"mode"`) {
		t.Errorf("empty Mode should be omitted, got: %s", s)
	}
	if strings.Contains(s, `"contract"`) {
		t.Errorf("empty Contract should be omitted, got: %s", s)
	}
}

// TestCollector_RecordModedEffect verifies the collector method attaches
// mode + contract to the emitted event.
func TestCollector_RecordModedEffect(t *testing.T) {
	c := NewCollector()
	c.RecordModedEffect("Rand", "rand_int", []string{"1", "100"}, "57", "seeded", "deterministic")

	events := c.Events()
	var found *EffectEvent
	for i := range events {
		if events[i].Effect != nil && events[i].Effect.EffectName == "Rand" {
			found = events[i].Effect
			break
		}
	}
	if found == nil {
		t.Fatal("no Rand effect event recorded")
	}
	if found.Mode != "seeded" || found.Contract != "deterministic" {
		t.Errorf("mode/contract = %q/%q, want seeded/deterministic", found.Mode, found.Contract)
	}
}
