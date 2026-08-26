package auth

import (
	"fmt"
	"strings"
	"time"
)

// PreflightInput is everything the policy decision needs. It carries no secrets:
// a denial can therefore be logged verbatim.
type PreflightInput struct {
	Profile SafeProfile
	Mode    LeaseMode
	Now     time.Time

	// RequestedOrigins are the origins the task declares it will visit. Empty
	// means "no specific declaration"; the profile's own allowlist still applies
	// at navigation time.
	RequestedOrigins []string

	// RequestedArtifacts are the artifact classes the run wants exported.
	RequestedArtifacts []string

	RequestHumanTakeover bool

	// RequestPersistence is true when the caller wants canonical write-back.
	// Only a refresh lease may ask.
	RequestPersistence bool
}

// Preflight decides whether an authenticated session may be provisioned. It runs
// BEFORE any browser is created, so a denial costs nothing.
//
// The check order is fixed and tested, so the reported cause never depends on
// input ordering:
//
//  1. lease mode is known
//  2. profile revoked
//  3. profile expired
//  4. write-back requested without a refresh lease
//  5. egress boundary absent
//  6. artifact policy absent
//  7. requested artifact class not allowed
//  8. profile has no allowed origins, or a requested origin is outside them
//  9. human takeover requested but not permitted
//
// Steps 5 and 6 are the fail-closed hooks for the two P0 follow-up designs
// (M-BROWSER-EGRESS-BOUNDARY and M-BROWSER-ARTIFACT-DATA-POLICY). Until those
// ship, a profile must carry an explicit operator acknowledgement to run at all.
func Preflight(in PreflightInput) error {
	const op = "preflight"

	if !in.Mode.Valid() {
		return fmt.Errorf("unknown lease mode %q", in.Mode)
	}

	if in.Profile.Revoked() {
		return NewFailureReason(FailureProfileRevoked, op, "profile "+in.Profile.Ref().String()+" is revoked")
	}
	if in.Profile.Expired(in.Now) {
		return NewFailureReason(FailureProfileExpired, op, "profile "+in.Profile.Ref().String()+" is expired")
	}

	if in.RequestPersistence && !in.Mode.Writes() {
		return NewFailureReason(FailureWritebackDenied, op,
			"persistence requested under a "+string(in.Mode)+" lease")
	}

	policy := in.Profile.Policy

	if !policy.EgressBoundaryPresent() {
		return NewFailureReason(FailureScopeDenied, op, "egress_boundary_absent")
	}

	if !policy.ArtifactPolicyPresent() {
		return NewFailureReason(FailureArtifactPolicyDenied, op, "artifact_policy_absent")
	}
	var deniedArtifacts []string
	for _, class := range in.RequestedArtifacts {
		if !policy.AllowsArtifact(class) {
			deniedArtifacts = append(deniedArtifacts, class)
		}
	}
	if len(deniedArtifacts) > 0 {
		return NewFailureReason(FailureArtifactPolicyDenied, op,
			"artifact_class_denied: "+strings.Join(deniedArtifacts, ", "))
	}

	if len(policy.AllowedOrigins) == 0 {
		return NewFailureReason(FailureScopeDenied, op, "no_allowed_origins")
	}
	var deniedOrigins []string
	for _, origin := range in.RequestedOrigins {
		if !policy.AllowsOrigin(origin) {
			deniedOrigins = append(deniedOrigins, origin)
		}
	}
	if len(deniedOrigins) > 0 {
		return NewFailureReason(FailureScopeDenied, op,
			"origin_denied: "+strings.Join(deniedOrigins, ", "))
	}

	if in.RequestHumanTakeover && !policy.AllowHumanTakeover {
		return NewFailureReason(FailureScopeDenied, op, "human_takeover_denied")
	}

	return nil
}
