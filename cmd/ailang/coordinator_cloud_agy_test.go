package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/executor"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// The agy cloud credential uses the same rotating-credential machinery as codex
// (M-AGY-CLOUD). These pin the agy-specific seams; the shared lifecycle is pinned by
// the codex tests, which build their credentials with codexCredentialSpec.

func TestInstallRotatingCredential_AgyNoSecretFailsLoudly(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_AGY_AUTH_SECRET", "")
	t.Setenv("GEMINI_AGENTS_API_KEY", "must-not-be-used")
	_, err := installRotatingCredential(t.Context(), "agy", "proj")
	if err == nil || !strings.Contains(err.Error(), "AILANG_AGY_AUTH_SECRET") {
		t.Fatalf("want a loud error naming AILANG_AGY_AUTH_SECRET, got %v", err)
	}
}

func TestInstallRotatingCredential_AgyNeedsExplicitHomeRoot(t *testing.T) {
	testutil.SetHomeDir(t, t.TempDir())
	t.Setenv("AILANG_AGY_AUTH_SECRET", "ailang-dev-agy-oauth-token")
	t.Setenv("AILANG_AGY_HOME_ROOT", "")
	_, err := installRotatingCredential(t.Context(), "agy", "proj")
	if err == nil || !strings.Contains(err.Error(), "AILANG_AGY_HOME_ROOT") {
		t.Fatalf("want a refusal naming AILANG_AGY_HOME_ROOT before any Firestore/secret access, got %v", err)
	}
}

func TestInstallRotatingCredential_OtherProvidersAreNoOps(t *testing.T) {
	for _, p := range []string{"claude", "pi", "opencode", "motoko", "gemini_agents"} {
		if c, err := installRotatingCredential(t.Context(), p, "proj"); c != nil || err != nil {
			t.Errorf("%s: got (%v, %v), want (nil, nil)", p, c, err)
		}
	}
}

const agyTokenOld = `{"token":{"access_token":"a1","token_type":"Bearer","refresh_token":"r","expiry":"2026-11-24T12:00:00Z","expires_in":3599},"auth_method":"oauth","id_token_email":"x@example.com"}`
const agyTokenNew = `{"token":{"access_token":"a2","token_type":"Bearer","refresh_token":"r","expiry":"2026-11-24T13:00:00Z","expires_in":3599},"auth_method":"oauth","id_token_email":"x@example.com"}`

// A refreshed agy token file (new access token, same refresh token, the measured
// shape) is written back before release.
func TestAgyCredentialFinishPersistsRefreshBeforeRelease(t *testing.T) {
	f := &fakeCodexLease{owner: "test"}
	written := false
	c := &rotatingCredential{spec: agyCredentialSpec, baseline: []byte(agyTokenOld), lease: f, owner: "test", key: "projects/p/secrets/s",
		read: func() ([]byte, error) { return []byte(agyTokenNew), nil },
		write: func(context.Context, []byte) error {
			written = true
			if f.owner == "" {
				t.Fatal("release before write")
			}
			return nil
		}}
	if err := c.finish(); err != nil {
		t.Fatal(err)
	}
	if !written || f.owner != "" {
		t.Fatal("refreshed agy token not persisted and released")
	}
}

// A token file that no longer validates (agy signed out, or an API-key settings
// file landed there) quarantines rather than overwriting the good secret.
func TestAgyCredentialInvalidCurrentQuarantines(t *testing.T) {
	f := &fakeCodexLease{owner: "test"}
	c := &rotatingCredential{spec: agyCredentialSpec, baseline: []byte(agyTokenOld), lease: f, owner: "test", key: "projects/p/secrets/s",
		read:  func() ([]byte, error) { return []byte(`{"modelProvider":"gemini"}`), nil },
		write: func(context.Context, []byte) error { t.Fatal("invalid credential written back"); return nil }}
	if err := c.finish(); err == nil {
		t.Fatal("invalid current credential hidden")
	}
	if !f.quarantined || f.owner == "" {
		t.Fatal("invalid credential released the lease")
	}
}

// quotaOut is an `agy -p /quota` reading whose weekly window started one day ago.
func quotaOut(weekRemaining string, now time.Time) []byte {
	reset := now.Add(6 * 24 * time.Hour).UTC().Format(time.RFC3339)
	five := now.Add(2 * time.Hour).UTC().Format(time.RFC3339)
	return []byte("Gemini Models\tWeekly Limit Remaining\t" + weekRemaining + "\t" + reset + "\n" +
		"Gemini Models\tFive Hour Limit Remaining\t90%\t" + five + "\n")
}

func TestCheckAgyRation(t *testing.T) {
	t.Setenv("AILANG_AGY_HOME_ROOT", t.TempDir())
	t.Setenv("AILANG_GOOGLE_DAILY_PACE", "15")
	t.Setenv("AILANG_QUOTA_MARGIN_GOOGLE", "")
	old := agyQuotaRun
	t.Cleanup(func() { agyQuotaRun = old })

	// Day 1 of the window, line 15%: 1% used is within it.
	agyQuotaRun = func(context.Context, string) ([]byte, error) { return quotaOut("99%", time.Now()), nil }
	if err := checkAgyRation(t.Context()); err != nil {
		t.Fatalf("within pace: %v", err)
	}
	// 50% used one day in: over the line, refuse before cloning.
	agyQuotaRun = func(context.Context, string) ([]byte, error) { return quotaOut("50%", time.Now()), nil }
	if err := checkAgyRation(t.Context()); err == nil || !strings.Contains(err.Error(), "not cloning") {
		t.Fatalf("over pace: %v", err)
	}
	// Unreadable quota fails closed.
	agyQuotaRun = func(context.Context, string) ([]byte, error) { return []byte("boom"), fmt.Errorf("exit status 2") }
	if err := checkAgyRation(t.Context()); err == nil || !strings.Contains(err.Error(), "not cloning") {
		t.Fatalf("unreadable: %v", err)
	}
	// An unproven termination is passed through untouched, so the caller quarantines.
	agyQuotaRun = func(context.Context, string) ([]byte, error) {
		return nil, fmt.Errorf("%w: test", executor.ErrProcessTerminationUnconfirmed)
	}
	if err := checkAgyRation(t.Context()); !errors.Is(err, executor.ErrProcessTerminationUnconfirmed) {
		t.Fatalf("termination unconfirmed hidden: %v", err)
	}
}
