package observatory

import "testing"

// The Firestore repair (internal/storage/firestore/observatory_repair_ids.go)
// and migrate_v18 MUST agree, because they repair the same corruption in two
// stores. They agree by construction — both call RecoverCorruptedID — and these
// tests pin the exported contract that makes sharing safe, so a future edit to
// the SQLite path cannot silently change what the cloud path does.

// TestRecoverCorruptedID_ExportedContract locks the behaviour both callers rely on.
func TestRecoverCorruptedID_ExportedContract(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		correctLen   int
		corruptedLen int
		want         string
		wantChanged  bool
	}{
		{
			// The production pair. If this ever changes, both stores are wrong.
			name: "production trace id", value: prodCorruptedTraceID,
			correctLen: CorrectTraceIDHexLen, corruptedLen: CorruptedTraceIDHexLen,
			want: prodRecoveredTraceID, wantChanged: true,
		},
		{
			name: "already-correct trace id is untouched", value: knownTraceIDHex,
			correctLen: CorrectTraceIDHexLen, corruptedLen: CorruptedTraceIDHexLen,
			want: knownTraceIDHex, wantChanged: false,
		},
		{
			name: "already-correct span id is untouched", value: knownSpanIDHex,
			correctLen: CorrectSpanIDHexLen, corruptedLen: CorruptedSpanIDHexLen,
			want: knownSpanIDHex, wantChanged: false,
		},
		{
			name: "empty is untouched", value: "",
			correctLen: CorrectSpanIDHexLen, corruptedLen: CorruptedSpanIDHexLen,
			want: "", wantChanged: false,
		},
		{
			// Right length, not hex: left alone rather than guessed at. Guessing
			// is the silent repair this whole milestone exists to remove.
			name: "corrupt-length non-hex is left alone", value: "zzzzzzzzzzzzzzzzzzzzzzzz",
			correctLen: CorrectSpanIDHexLen, corruptedLen: CorruptedSpanIDHexLen,
			want: "zzzzzzzzzzzzzzzzzzzzzzzz", wantChanged: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := RecoverCorruptedID(tt.value, tt.correctLen, tt.corruptedLen)
			if err != nil {
				t.Fatalf("RecoverCorruptedID: %v", err)
			}
			if got != tt.want {
				t.Errorf("value = %q, want %q", got, tt.want)
			}
			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}
		})
	}
}

// TestRecoverCorruptedID_Idempotent is the property the chunked Firestore repair
// depends on for safety: a batch failing part-way is recoverable by re-running,
// which is only true if applying the transform twice is a no-op.
func TestRecoverCorruptedID_Idempotent(t *testing.T) {
	once, changed, err := RecoverCorruptedID(prodCorruptedTraceID, CorrectTraceIDHexLen, CorruptedTraceIDHexLen)
	if err != nil || !changed {
		t.Fatalf("first pass: changed=%v err=%v", changed, err)
	}
	twice, changedAgain, err := RecoverCorruptedID(once, CorrectTraceIDHexLen, CorruptedTraceIDHexLen)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if changedAgain {
		t.Error("second pass reported a change; the transform is not idempotent")
	}
	if twice != once {
		t.Errorf("second pass altered the value: %q -> %q", once, twice)
	}
}

// TestRepairLengthConstants_AreSelfConsistent guards the 1.5x relationship the
// whole repair rests on: base64-decoding a hex string of length N yields 3N/4
// bytes, rendering as 3N/2 hex characters.
func TestRepairLengthConstants_AreSelfConsistent(t *testing.T) {
	cases := []struct{ correct, corrupted int }{
		{CorrectTraceIDHexLen, CorruptedTraceIDHexLen},
		{CorrectSpanIDHexLen, CorruptedSpanIDHexLen},
	}
	for _, c := range cases {
		if want := c.correct * 3 / 2; c.corrupted != want {
			t.Errorf("corrupted length %d for correct %d; want %d (1.5x)", c.corrupted, c.correct, want)
		}
	}
}
