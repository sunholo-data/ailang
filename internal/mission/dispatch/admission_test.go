package dispatch

import (
	"context"
	"testing"
	"time"
)

func TestAdmissionRequiredAndRechecked(t *testing.T) {
	for _, phase := range []string{"missing", "preflight", "dispatch"} {
		t.Run(phase, func(t *testing.T) {
			req, r, f, _ := fixture(t)
			req.Models = []string{"judge"}
			calls := 0
			r.Admit = nil
			if phase != "missing" {
				r.Admit = func(context.Context, Candidate) (Admission, error) {
					calls++
					return Admission{Allowed: phase == "dispatch" && calls == 1, Reason: "quota blocked", ObservedAt: time.Now(), Policy: "fixture"}, nil
				}
			}
			report, err := r.Run(context.Background(), req)
			if err == nil || f["pi"].calls != 0 {
				t.Fatalf("admission leaked execution: %+v %v", report, err)
			}
			if phase == "dispatch" && calls != 2 {
				t.Fatalf("checks=%d", calls)
			}
		})
	}
}
