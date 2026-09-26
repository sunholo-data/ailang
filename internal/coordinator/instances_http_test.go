package coordinator

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// The assertions worth having are the refusals. This route starts and stops
// infrastructure on behalf of another estate, so what it declines to do is the
// specification.

func TestInstanceNameFromURL(t *testing.T) {
	cases := map[string]string{
		"https://resident-abc123-812435936917.europe-west4.run.app":        "resident-abc123",
		"https://resident-pi-ailang-812435936917.europe-west4.run.app/a2a": "resident-pi-ailang",
		"resident-x-1.europe-west4.run.app":                                "resident-x",
	}
	for in, want := range cases {
		if got := instanceNameFromURL(in); got != want {
			t.Errorf("instanceNameFromURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOnlyResidentInstanceNamesAreAccepted(t *testing.T) {
	// The coordinator's identity may be able to touch more than this; the
	// endpoint may not. A capability is defined by what it refuses.
	for _, bad := range []string{
		"", "coordinator", "ailang-dev-dashboard",
		"../../../services/prod", "resident-", "RESIDENT-abc",
		"resident-abc/../other",
	} {
		if residentInstanceName.MatchString(bad) {
			t.Errorf("accepted a name it should refuse: %q", bad)
		}
	}
	for _, good := range []string{"resident-abc123", "resident-pi-ailang", "resident-5650d6cf11a7"} {
		if !residentInstanceName.MatchString(good) {
			t.Errorf("refused a legitimate name: %q", good)
		}
	}
}

func TestLifecycleFailsClosedWhenUnconfigured(t *testing.T) {
	// requireAPIKey passes everything when its key is unset. For a route that
	// operates infrastructure the opposite default is the only safe one:
	// unconfigured means nobody, not everybody.
	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "")
	r := httptest.NewRequest(http.MethodPost, "/instances/start", strings.NewReader(`{"instance":"resident-x"}`))
	r.Header.Set("Authorization", "Bearer anything")
	if err := verifyLifecycleCaller(r.Context(), r); err == nil {
		t.Fatal("unconfigured lifecycle accepted a caller")
	}
}

func TestLifecycleRefusesAMissingToken(t *testing.T) {
	t.Setenv("RESIDENT_LIFECYCLE_AUDIENCE", "https://coordinator.invalid")
	t.Setenv("RESIDENT_LIFECYCLE_ALLOWED_CALLERS", "sa@example.iam.gserviceaccount.com")
	r := httptest.NewRequest(http.MethodPost, "/instances/start", strings.NewReader("{}"))
	if err := verifyLifecycleCaller(r.Context(), r); err == nil {
		t.Fatal("accepted a request with no bearer token")
	}
}

func TestRefusalsAreIndistinguishable(t *testing.T) {
	// A probe must not be able to tell "not configured" from "not allowed" and
	// map the allowlist from the difference.
	d := &Daemon{logger: log.New(io.Discard, "", 0)}
	os.Unsetenv("RESIDENT_LIFECYCLE_AUDIENCE")
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/instances/start", strings.NewReader("{}"))
	d.handleStartInstance(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "unauthorized") || strings.Contains(body, "configured") {
		t.Errorf("refusal body leaks a reason: %q", body)
	}
}
