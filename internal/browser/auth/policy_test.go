package auth

import (
	"strings"
	"testing"
	"time"
)

func preflightNow() time.Time { return time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC) }

// goodInput is a request that must pass, so every failing case below differs
// from a passing one in exactly one respect.
func goodInput() PreflightInput {
	policy := AuthProfilePolicy{
		AllowedOrigins: []string{"https://crm.example.com"},
		AccountClass:   AccountReadonly,
		MaxConcurrent:  1,
		AllowArtifacts: []string{},
		EgressBoundary: EgressOperatorAcknowledged,
	}
	return PreflightInput{
		Profile: SafeProfile{Alias: "crm", Version: "v7", ProfileHash: "sha256:abc", Policy: policy},
		Mode:    LeaseRead,
		Now:     preflightNow(),
	}
}

func TestPreflightAllowsAWellFormedReadRequest(t *testing.T) {
	if err := Preflight(goodInput()); err != nil {
		t.Fatalf("a well-formed read request was denied: %v", err)
	}
}

func TestPreflightRejectsRevokedBeforeAnythingElse(t *testing.T) {
	in := goodInput()
	in.Profile.RevokedAt = preflightNow().Add(-time.Hour)
	// Also break the egress policy: revocation must still be the reported cause.
	in.Profile.Policy.EgressBoundary = EgressAbsent

	err := Preflight(in)
	if !IsFailure(err, FailureProfileRevoked) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureProfileRevoked)
	}
}

func TestPreflightRejectsExpired(t *testing.T) {
	in := goodInput()
	in.Profile.ExpiresAtOrZero = preflightNow().Add(-time.Second)

	err := Preflight(in)
	if !IsFailure(err, FailureProfileExpired) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureProfileExpired)
	}
}

// An ordinary run asking to persist is the exact failure this design exists to
// prevent, so it is refused rather than quietly ignored.
func TestPreflightRefusesPersistenceUnderAReadLease(t *testing.T) {
	in := goodInput()
	in.RequestPersistence = true

	err := Preflight(in)
	if !IsFailure(err, FailureWritebackDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureWritebackDenied)
	}

	in.Mode = LeaseRefresh
	if err := Preflight(in); err != nil {
		t.Fatalf("persistence under a refresh lease was denied: %v", err)
	}
}

func TestPreflightFailsClosedWhenEgressBoundaryIsAbsentAndNamesIt(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.EgressBoundary = EgressAbsent

	err := Preflight(in)
	if !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureScopeDenied)
	}
	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("error is not a *Failure: %v", err)
	}
	if !strings.Contains(failure.Reason, "egress") {
		t.Fatalf("denial reason %q does not name the absent policy", failure.Reason)
	}
}

func TestPreflightFailsClosedWhenArtifactPolicyIsAbsentAndNamesIt(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.AllowArtifacts = nil // no decision made

	err := Preflight(in)
	if !IsFailure(err, FailureArtifactPolicyDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureArtifactPolicyDenied)
	}
	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("error is not a *Failure: %v", err)
	}
	if !strings.Contains(failure.Reason, "artifact") {
		t.Fatalf("denial reason %q does not name the absent policy", failure.Reason)
	}
}

// An explicitly empty list is a decision (allow nothing) and must not be
// confused with no decision at all.
func TestPreflightDistinguishesAllowNothingFromNoDecision(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.AllowArtifacts = []string{}
	if err := Preflight(in); err != nil {
		t.Fatalf("an explicit allow-nothing artifact policy was treated as absent: %v", err)
	}
}

func TestPreflightRejectsAnUnlistedArtifactClass(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.AllowArtifacts = []string{"screenshot"}
	in.RequestedArtifacts = []string{"screenshot", "video"}

	err := Preflight(in)
	if !IsFailure(err, FailureArtifactPolicyDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureArtifactPolicyDenied)
	}
	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("error is not a *Failure: %v", err)
	}
	if !strings.Contains(failure.Reason, "video") {
		t.Fatalf("denial reason %q does not name the denied class", failure.Reason)
	}
	if strings.Contains(failure.Reason, "screenshot") {
		t.Fatalf("denial reason %q named the allowed class too", failure.Reason)
	}
}

func TestPreflightRejectsAProfileWithNoAllowedOrigins(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.AllowedOrigins = nil

	if err := Preflight(in); !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureScopeDenied)
	}
}

func TestPreflightRejectsARequestedOriginOutsideThePolicy(t *testing.T) {
	in := goodInput()
	in.RequestedOrigins = []string{"https://crm.example.com", "https://evil.example.com"}

	err := Preflight(in)
	if !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureScopeDenied)
	}
	failure, ok := AsFailure(err)
	if !ok {
		t.Fatalf("error is not a *Failure: %v", err)
	}
	if !strings.Contains(failure.Reason, "evil.example.com") {
		t.Fatalf("denial reason %q does not name the denied origin", failure.Reason)
	}
}

func TestPreflightRejectsHumanTakeoverWhenPolicyForbidsIt(t *testing.T) {
	in := goodInput()
	in.RequestHumanTakeover = true

	if err := Preflight(in); !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("Preflight returned %v, want %s", err, FailureScopeDenied)
	}

	in.Profile.Policy.AllowHumanTakeover = true
	if err := Preflight(in); err != nil {
		t.Fatalf("human takeover was denied despite policy allowing it: %v", err)
	}
}

func TestPreflightRejectsAnInvalidMode(t *testing.T) {
	in := goodInput()
	in.Mode = LeaseMode("write")
	if err := Preflight(in); err == nil {
		t.Fatalf("Preflight accepted an unknown lease mode")
	}
}

// The reported cause must not depend on which check happens to run first for a
// given input: the order is fixed and documented.
func TestPreflightCheckOrderIsDeterministic(t *testing.T) {
	in := goodInput()
	in.Profile.Policy.EgressBoundary = EgressAbsent
	in.Profile.Policy.AllowArtifacts = nil
	in.RequestedOrigins = []string{"https://evil.example.com"}
	in.RequestHumanTakeover = true

	// Egress is checked before artifacts, which are checked before origins.
	if err := Preflight(in); !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("with every policy broken, Preflight returned %v, want the egress denial", err)
	}

	in.Profile.Policy.EgressBoundary = EgressOperatorAcknowledged
	if err := Preflight(in); !IsFailure(err, FailureArtifactPolicyDenied) {
		t.Fatalf("after fixing egress, Preflight returned %v, want the artifact denial", err)
	}

	in.Profile.Policy.AllowArtifacts = []string{}
	if err := Preflight(in); !IsFailure(err, FailureScopeDenied) {
		t.Fatalf("after fixing artifacts, Preflight returned %v, want the origin denial", err)
	}
}
