package verification

import (
	"strings"
	"testing"
)

func TestEvidenceAbsentAndReasonOrdering(t *testing.T) {
	if !strings.Contains((Evidence{}).Summary(), "unavailable") {
		t.Fatal("absent model must be unavailable")
	}
	if ReasonSummary(nil) != "reason detail unavailable" {
		t.Fatal("absent reasons must be unavailable")
	}
	if ReasonSummary(map[ReasonCode]int{}) == ReasonSummary(nil) {
		t.Fatal("empty measured reasons conflated with absent data")
	}
	want := "missing_postcondition=5, unsupported_encoding=5"
	for i := 0; i < 20; i++ {
		if got := ReasonSummary(map[ReasonCode]int{UnsupportedEncoding: 5, MissingPostcondition: 5}); got != want {
			t.Fatalf("unstable reasons: %s", got)
		}
	}
}
