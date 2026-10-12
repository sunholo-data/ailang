package credentiallease

import (
	"testing"
	"time"
)

func TestCredentialLeaseNeverTakesOverExpiredOwner(t *testing.T) {
	now := time.Now()
	r, err := TransitionCredentialLease(CredentialLeaseRecord{}, "codex-exec", "acquire", now)
	if err != nil {
		t.Fatal(err)
	}
	r.Heartbeat = now.Add(-24 * time.Hour)
	if _, err = TransitionCredentialLease(r, "codex-go-exec", "acquire", now); err == nil {
		t.Fatal("stale heartbeat allowed duplicate credential ownership")
	}
}
func TestCredentialLeaseRejectsStaleWriterAndQuarantines(t *testing.T) {
	r, _ := TransitionCredentialLease(CredentialLeaseRecord{}, "owner", "acquire", time.Now())
	for _, action := range []string{"heartbeat", "release", "quarantine"} {
		if _, err := TransitionCredentialLease(r, "stale", action, time.Now()); err == nil {
			t.Fatalf("stale %s accepted", action)
		}
	}
	r, err := TransitionCredentialLease(r, "owner", "quarantine", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TransitionCredentialLease(r, "another", "acquire", time.Now()); err == nil {
		t.Fatal("quarantined credential reused")
	}
}
func TestCredentialLeaseReleaseAllowsNextOwner(t *testing.T) {
	r, _ := TransitionCredentialLease(CredentialLeaseRecord{}, "first", "acquire", time.Now())
	r, err := TransitionCredentialLease(r, "first", "release", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := TransitionCredentialLease(r, "second", "acquire", time.Now()); err != nil {
		t.Fatal(err)
	}
}
