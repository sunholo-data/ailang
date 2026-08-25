package auth

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func auditProfile() SafeProfile {
	return SafeProfile{
		Alias:       "crm-readonly-eu",
		Version:     "v7",
		ProfileHash: "sha256:0f1e2d3c4b5a6978",
		Provider:    "browserbase",
		Policy: AuthProfilePolicy{
			AllowedOrigins: []string{"https://crm.example.com"},
			AccountClass:   AccountReadonly,
			MaxConcurrent:  1,
			AllowArtifacts: []string{},
			EgressBoundary: EgressOperatorAcknowledged,
		},
	}
}

func TestAuditEventCarriesEveryRequiredField(t *testing.T) {
	profile := auditProfile()
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	lease := ProfileLease{SafeID: "lease-abc", Alias: profile.Alias, Version: profile.Version, Mode: LeaseRead}
	identity := RunIdentity{RunID: "run-42", ChainID: "chain-9", StageID: "stage-2", Principal: "eval-harness"}

	event := NewAuditEvent("acquire", profile, lease, identity, at, nil)

	checks := map[string]string{
		"alias":        event.Alias,
		"version":      event.Version,
		"profile hash": event.ProfileHash,
		"lease id":     event.LeaseID,
		"mode":         string(event.Mode),
		"run id":       event.Run.RunID,
		"chain id":     event.Run.ChainID,
		"stage id":     event.Run.StageID,
		"principal":    event.Run.Principal,
		"provider":     event.Provider,
		"decision":     string(event.Decision),
	}
	for name, value := range checks {
		if value == "" {
			t.Fatalf("audit event is missing %s", name)
		}
	}
	if len(event.AllowedOrigins) != 1 {
		t.Fatalf("audit event dropped the allowed origins")
	}
	if event.At.IsZero() {
		t.Fatalf("audit event has no timestamp")
	}
	if event.Decision != DecisionAllowed {
		t.Fatalf("decision = %q, want %q", event.Decision, DecisionAllowed)
	}
}

func TestAuditEventRecordsTheDenialCategoryAndReason(t *testing.T) {
	profile := auditProfile()
	profile.Policy.EgressBoundary = EgressAbsent
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	err := Preflight(PreflightInput{Profile: profile, Mode: LeaseRead, Now: at})
	if err == nil {
		t.Fatalf("expected the preflight to deny this profile")
	}

	event := NewAuditEvent("preflight", profile, ProfileLease{}, RunIdentity{RunID: "run-42"}, at, err)
	if event.Decision != DecisionDenied {
		t.Fatalf("decision = %q, want %q", event.Decision, DecisionDenied)
	}
	if event.FailureCategory != FailureScopeDenied {
		t.Fatalf("failure category = %q, want %q", event.FailureCategory, FailureScopeDenied)
	}
	if !strings.Contains(event.Reason, "egress") {
		t.Fatalf("reason %q does not name the absent policy", event.Reason)
	}
}

// The safety property here is structural: AuditEvent has no field capable of
// holding material, so no filter has to be maintained to keep secrets out. This
// test fails if someone later adds one.
func TestAuditEventHasNoFieldThatCouldHoldASecret(t *testing.T) {
	forbidden := []string{
		"material", "state", "cookie", "token", "secret", "password",
		"context", "sealed", "ciphertext", "key", "credential", "blob",
	}

	for field := range reflect.TypeFor[AuditEvent]().Fields() {
		name := strings.ToLower(field.Name)
		for _, marker := range forbidden {
			if strings.Contains(name, marker) {
				t.Fatalf("AuditEvent.%s could hold credential-grade data; audit records must stay structurally safe", field.Name)
			}
		}
		// Raw byte slices are the shape material arrives in.
		if field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Uint8 {
			t.Fatalf("AuditEvent.%s is a []byte; audit records must not carry raw bytes", field.Name)
		}
	}
}

func TestAuditEventSerializesWithoutSecrets(t *testing.T) {
	profile := auditProfile()
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	lease := ProfileLease{SafeID: "lease-abc", Alias: profile.Alias, Version: profile.Version, Mode: LeaseRead}

	event := NewAuditEvent("acquire", profile, lease, RunIdentity{RunID: "run-42", Principal: "eval-harness"}, at, nil)
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal audit event: %v", err)
	}
	rendered := string(encoded)

	for _, marker := range leakMarkers {
		if strings.Contains(rendered, marker) {
			t.Fatalf("audit JSON leaked %q: %s", marker, rendered)
		}
	}
	for _, want := range []string{"crm-readonly-eu", "v7", "lease-abc", "run-42", "eval-harness"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("audit JSON dropped %q: %s", want, rendered)
		}
	}
}

// Mutating the profile after the event was built must not rewrite history.
func TestAuditEventCopiesAllowedOrigins(t *testing.T) {
	profile := auditProfile()
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	event := NewAuditEvent("acquire", profile, ProfileLease{}, RunIdentity{RunID: "run-42"}, at, nil)
	profile.Policy.AllowedOrigins[0] = "https://evil.example.com"

	if event.AllowedOrigins[0] != "https://crm.example.com" {
		t.Fatalf("audit event aliased the profile's origin slice")
	}
}

func TestMemoryAuditSinkRecordsInOrder(t *testing.T) {
	sink := NewMemoryAuditSink()
	ctx := context.Background()
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

	for _, op := range []string{"resolve", "acquire", "release"} {
		if err := sink.Record(ctx, NewAuditEvent(op, auditProfile(), ProfileLease{}, RunIdentity{RunID: "run-42"}, at, nil)); err != nil {
			t.Fatalf("record %s: %v", op, err)
		}
	}

	events := sink.Events()
	if len(events) != 3 {
		t.Fatalf("sink holds %d events, want 3", len(events))
	}
	for i, want := range []string{"resolve", "acquire", "release"} {
		if events[i].Op != want {
			t.Fatalf("event %d op = %q, want %q", i, events[i].Op, want)
		}
	}

	// Events() must hand back a copy, not the live slice.
	events[0].Op = "tampered"
	if sink.Events()[0].Op != "resolve" {
		t.Fatalf("Events() exposed the sink's backing slice")
	}
}
