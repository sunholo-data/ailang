package eval_harness

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestAbsentValidityMeansValid is THE back-compat guarantee of M2.
//
// Every eval row banked before v0.31.0 has no `validity` field. If absent
// decoded to the zero value (Valid:false) every historical datapoint would
// silently become "invalid" and vanish from every aggregate — a far worse data
// loss than the bug this sprint fixes. Absent MUST mean valid.
func TestAbsentValidityMeansValid(t *testing.T) {
	legacy := `{"id":"fizzbuzz","lang":"ailang","model":"m","stdout_ok":true}`

	var m RunMetrics
	if err := json.Unmarshal([]byte(legacy), &m); err != nil {
		t.Fatalf("unmarshal legacy row: %v", err)
	}
	if !m.IsValid() {
		t.Fatal("a legacy row with no validity field MUST be treated as valid — otherwise all pre-v0.31.0 history is erased")
	}
	if m.InvalidReason() != "" {
		t.Errorf("legacy row should have no invalid reason, got %q", m.InvalidReason())
	}
}

func TestValidityRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		validity   *Validity
		wantValid  bool
		wantReason string
	}{
		{name: "nil (legacy)", validity: nil, wantValid: true},
		{name: "explicitly valid", validity: MarkValid(), wantValid: true},
		{
			name:       "invalid with reason",
			validity:   MarkInvalid(ReasonZeroPassAll),
			wantValid:  false,
			wantReason: ReasonZeroPassAll,
		},
		{
			name:       "canary failure",
			validity:   MarkInvalid(ReasonCanaryFailed),
			wantValid:  false,
			wantReason: ReasonCanaryFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := RunMetrics{ID: "b", Validity: tt.validity}

			data, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			var decoded RunMetrics
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if decoded.IsValid() != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v (json: %s)", decoded.IsValid(), tt.wantValid, data)
			}
			if decoded.InvalidReason() != tt.wantReason {
				t.Errorf("InvalidReason() = %q, want %q", decoded.InvalidReason(), tt.wantReason)
			}
		})
	}
}

// TestValidRowOmitsNothingSurprising keeps the on-disk shape predictable: a
// valid row should not bloat every result file with redundant nesting.
func TestNilValidityIsOmittedFromJSON(t *testing.T) {
	data, err := json.Marshal(RunMetrics{ID: "b"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "validity") {
		t.Errorf("a row with no validity should omit the field entirely, got: %s", data)
	}
}

// TestMarkInvalidRequiresReason: an invalid row with no reason is useless —
// it tells you a number is wrong but not why, which is how the 2026-07-20
// artefact survived unexamined for a week.
func TestMarkInvalidRequiresReason(t *testing.T) {
	v := MarkInvalid("")
	if v.Reason != ReasonHarnessError {
		t.Errorf("MarkInvalid(\"\") should fall back to %q, got %q", ReasonHarnessError, v.Reason)
	}
}

func TestFilterValid(t *testing.T) {
	rows := []RunMetrics{
		{ID: "a"},                        // legacy, valid
		{ID: "b", Validity: MarkValid()}, // explicitly valid
		{ID: "c", Validity: MarkInvalid(ReasonZeroPassAll)},  // invalid
		{ID: "d", Validity: MarkInvalid(ReasonCanaryFailed)}, // invalid
	}

	valid := FilterValid(rows)
	if len(valid) != 2 {
		t.Fatalf("FilterValid returned %d rows, want 2 (legacy + explicitly valid)", len(valid))
	}
	if valid[0].ID != "a" || valid[1].ID != "b" {
		t.Errorf("FilterValid kept %q,%q; want a,b", valid[0].ID, valid[1].ID)
	}
}
