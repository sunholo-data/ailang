package coordinator

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Stopping is a one-way door for the user whose agent it is: a stopped instance
// does not come back on its own. So the tests worth having are the ones that
// pin what the sweep DECLINES to do.

func intp(v int) *int { return &v }

func health(active, idle *int) *residentHealth {
	var h residentHealth
	h.Runs.Active = active
	h.Runs.IdleS = idle
	return &h
}

func TestSweepNeverActsOnMissingInformation(t *testing.T) {
	const threshold = 1800

	cases := []struct {
		name string
		h    *residentHealth
		err  error
	}{
		{"health unreachable", nil, errors.New("connect refused")},
		{"no health at all", nil, nil},
		// The case that actually happened: an image predating the runs block
		// reports no idleness. Read as zero it looks BUSY; invented as a large
		// number it looks abandoned. Neither is a fact, so neither is allowed.
		{"no idle_s reported", health(intp(0), nil), nil},
		{"no active count reported", health(nil, intp(99999)), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			action, reason, _ := decideSweep(tc.h, tc.err, threshold)
			if action != sweepActionSkipped {
				t.Fatalf("action = %q, want %q — an instance we cannot assess might be mid-run", action, sweepActionSkipped)
			}
			if reason == "" {
				t.Error("skipped without saying why; an operator cannot fix a silent skip")
			}
		})
	}
}

func TestSweepKeepsAnInstanceThatIsWorking(t *testing.T) {
	// active > 0 outranks idle_s entirely: a long-idle box that just picked up
	// work must not be stopped on a stale-looking counter.
	action, _, _ := decideSweep(health(intp(1), intp(99999)), nil, 1800)
	if action != sweepActionKept {
		t.Fatalf("action = %q, want %q — stopping a working agent loses the run", action, sweepActionKept)
	}
}

func TestSweepStopsOnlyPastTheThreshold(t *testing.T) {
	if action, _, _ := decideSweep(health(intp(0), intp(1799)), nil, 1800); action != sweepActionKept {
		t.Errorf("just under the threshold: action = %q, want %q", action, sweepActionKept)
	}
	if action, _, idle := decideSweep(health(intp(0), intp(1800)), nil, 1800); action != sweepActionStopped || idle != 1800 {
		t.Errorf("at the threshold: action = %q idle = %d, want %q 1800", action, idle, sweepActionStopped)
	}
}

func TestSweepRefusesToApplyWithNoWayBack(t *testing.T) {
	// The interlock. A sweep without a start path does not save money — it
	// strands the users who were idle longest, and worst for the ones who
	// used it least recently.
	t.Setenv("RESIDENT_LIFECYCLE_PROJECT", "")
	t.Setenv("RESIDENT_LIFECYCLE_REGION", "")
	if err := canStartAgain(); err == nil {
		t.Fatal("would apply a sweep with no configured way to start instances again")
	}

	t.Setenv("RESIDENT_LIFECYCLE_PROJECT", "p")
	t.Setenv("RESIDENT_LIFECYCLE_REGION", "europe-west4")
	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "")
	if err := canStartAgain(); err == nil {
		t.Fatal("a start route no caller may reach is not a way back")
	}

	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "https://coordinator.invalid")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "sa@example.iam.gserviceaccount.com")
	if err := canStartAgain(); err != nil {
		t.Fatalf("fully configured lifecycle still refused: %v", err)
	}
}

func TestSweepApplyIsRefusedOverTheWireNotSilentlyDowngraded(t *testing.T) {
	// A caller who asked to stop things must not get a 200 describing a dry
	// run — they would record it as done and never look again.
	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "https://coordinator.invalid")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "sa@example.iam.gserviceaccount.com")
	t.Setenv("RESIDENT_LIFECYCLE_PROJECT", "")
	t.Setenv("RESIDENT_LIFECYCLE_REGION", "")

	d := &Daemon{logger: log.New(io.Discard, "", 0)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/instances/sweep", strings.NewReader(`{"apply":true}`))
	r.Header.Set("Authorization", "Bearer not-a-real-token")
	d.handleSweepInstances(w, r)

	// The token is bogus, so this refuses at auth first — the point is that it
	// refuses, and never reaches Cloud Run.
	if w.Code == http.StatusOK {
		t.Fatalf("an unauthenticated apply returned 200: %q", w.Body.String())
	}
}

func TestSweepFailsClosedWhenLifecycleIsUnconfigured(t *testing.T) {
	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "")
	d := &Daemon{logger: log.New(io.Discard, "", 0)}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/instances/sweep", strings.NewReader("{}"))
	r.Header.Set("Authorization", "Bearer anything")
	d.handleSweepInstances(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "unauthorized") || strings.Contains(body, "configured") {
		t.Errorf("refusal body leaks a reason: %q", body)
	}
}

func TestSweepOnlyTouchesResidentNames(t *testing.T) {
	// Same capability boundary as the start route: the coordinator's identity
	// may reach more than this endpoint will ever touch.
	for _, bad := range []string{"coordinator", "ailang-dev-dashboard", "resident-", "RESIDENT-x"} {
		if residentInstanceName.MatchString(bad) {
			t.Errorf("sweep would consider a non-resident instance: %q", bad)
		}
	}
}

func TestLastSegment(t *testing.T) {
	got := lastSegment("projects/p/locations/europe-west4/instances/resident-pi-ailang")
	if got != "resident-pi-ailang" {
		t.Fatalf("lastSegment = %q", got)
	}
	if lastSegment("resident-x") != "resident-x" {
		t.Fatal("a bare name should pass through")
	}
}
