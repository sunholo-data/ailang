package messaging

import (
	"testing"
	"time"
)

func newValidEnvelope(kind PackageMessageKind) *PackageMessageEnvelope {
	brk := false
	return &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      kind,
		From:      "pkg:sunholo/auth",
		To:        []string{"workspace:docparse"},
		Timestamp: time.Now(),
		Package: PackageRef{
			Name:              "sunholo/auth",
			FromVersion:       "0.1.0",
			ToVersion:         "0.2.0",
			FromInterfaceHash: "sha256:aaa",
			ToInterfaceHash:   "sha256:bbb",
			FromContentHash:   "sha256:ccc",
			ToContentHash:     "sha256:ddd",
			ChangeClass:       "C",
			Breaking:          &brk,
		},
	}
}

func TestAllMessageKindsExist(t *testing.T) {
	if len(AllPackageMessageKinds) != 11 {
		t.Errorf("expected 11 message kinds, got %d", len(AllPackageMessageKinds))
	}
}

func TestValidateUpgradeAvailable(t *testing.T) {
	env := newValidEnvelope(PkgMsgUpgradeAvailable)
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid upgrade-available rejected: %v", err)
	}

	// Missing change_class
	env2 := newValidEnvelope(PkgMsgUpgradeAvailable)
	env2.Package.ChangeClass = ""
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing change_class")
	}

	// Missing from_version
	env3 := newValidEnvelope(PkgMsgUpgradeAvailable)
	env3.Package.FromVersion = ""
	if err := ValidatePackageMessage(env3); err == nil {
		t.Fatal("expected error for missing from_version")
	}
}

func TestValidateInterfaceChange(t *testing.T) {
	env := newValidEnvelope(PkgMsgInterfaceChange)
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid interface-change-notice rejected: %v", err)
	}

	env2 := newValidEnvelope(PkgMsgInterfaceChange)
	env2.Package.FromInterfaceHash = ""
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing from_interface_hash")
	}
}

func TestValidateEffectWidening(t *testing.T) {
	env := newValidEnvelope(PkgMsgEffectWidening)
	env.Package.PrevEffectCeiling = []string{"io", "net"}
	env.Package.NewEffectCeiling = []string{"io", "net", "fs"}
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid effect-widening-warning rejected: %v", err)
	}

	env2 := newValidEnvelope(PkgMsgEffectWidening)
	// Missing ceilings
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing effect ceilings")
	}
}

func TestValidateCompatibilityReport(t *testing.T) {
	env := newValidEnvelope(PkgMsgCompatibilityReport)
	env.Package.TargetWorkspace = "workspace:docparse"
	env.Package.Result = "pass"
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid compatibility-report rejected: %v", err)
	}

	// Missing result
	env2 := newValidEnvelope(PkgMsgCompatibilityReport)
	env2.Package.TargetWorkspace = "workspace:docparse"
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing result")
	}

	// Invalid result value
	env3 := newValidEnvelope(PkgMsgCompatibilityReport)
	env3.Package.TargetWorkspace = "workspace:docparse"
	env3.Package.Result = "maybe"
	if err := ValidatePackageMessage(env3); err == nil {
		t.Fatal("expected error for invalid result value")
	}
}

func TestValidateContractRegression(t *testing.T) {
	env := newValidEnvelope(PkgMsgContractRegression)
	env.Package.AffectedExports = []string{"validateBearer"}
	env.Package.PrevContract = "accepts any non-empty string"
	env.Package.NewContract = "requires Bearer prefix"
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid contract-regression rejected: %v", err)
	}

	// Missing affected_exports
	env2 := newValidEnvelope(PkgMsgContractRegression)
	env2.Package.PrevContract = "old"
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing affected_exports")
	}
}

func TestValidateMigrationRequest(t *testing.T) {
	env := newValidEnvelope(PkgMsgMigrationRequest)
	env.BlockReason = "effect ceiling policy blocks net effect"
	if err := ValidatePackageMessage(env); err != nil {
		t.Fatalf("valid migration-request rejected: %v", err)
	}

	// Missing block_reason
	env2 := newValidEnvelope(PkgMsgMigrationRequest)
	if err := ValidatePackageMessage(env2); err == nil {
		t.Fatal("expected error for missing block_reason")
	}
}

func TestValidateBaseRequired(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*PackageMessageEnvelope)
	}{
		{"missing schema", func(e *PackageMessageEnvelope) { e.Schema = "" }},
		{"wrong schema", func(e *PackageMessageEnvelope) { e.Schema = "wrong/v1" }},
		{"missing kind", func(e *PackageMessageEnvelope) { e.Kind = "" }},
		{"invalid kind", func(e *PackageMessageEnvelope) { e.Kind = "nonexistent" }},
		{"missing from", func(e *PackageMessageEnvelope) { e.From = "" }},
		{"missing to", func(e *PackageMessageEnvelope) { e.To = nil }},
		{"empty to", func(e *PackageMessageEnvelope) { e.To = []string{} }},
		{"missing timestamp", func(e *PackageMessageEnvelope) { e.Timestamp = time.Time{} }},
		{"missing package name", func(e *PackageMessageEnvelope) { e.Package.Name = "" }},
		{"no delta", func(e *PackageMessageEnvelope) {
			e.Package.FromVersion = ""
			e.Package.ToVersion = ""
			e.Package.FromInterfaceHash = ""
			e.Package.ToInterfaceHash = ""
			e.Package.FromContentHash = ""
			e.Package.ToContentHash = ""
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := newValidEnvelope(PkgMsgUpgradeAvailable)
			tc.modify(env)
			if err := ValidatePackageMessage(env); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestJSONRoundTrip(t *testing.T) {
	brk := true
	env := &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      PkgMsgUpgradeAvailable,
		From:      "pkg:sunholo/auth",
		To:        []string{"workspace:docparse", "workspace:web-api-demo"},
		Timestamp: time.Date(2026, 3, 21, 12, 0, 0, 0, time.UTC),
		Package: PackageRef{
			Name:              "sunholo/auth",
			FromVersion:       "0.1.0",
			ToVersion:         "0.2.0",
			FromInterfaceHash: "sha256:aaa",
			ToInterfaceHash:   "sha256:bbb",
			FromContentHash:   "sha256:ccc",
			ToContentHash:     "sha256:ddd",
			ChangeClass:       "C",
			Breaking:          &brk,
			EffectDelta:       []string{"net"},
		},
		Summary:           "Bearer extraction normalized",
		RecommendedAction: "Run auth compatibility checks",
		Refs: &PackageRefs{
			PackageURL: "pkg:sunholo/auth@0.2.0",
		},
		Status: "open",
	}

	data, err := PackageEnvelopeToJSON(env)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	got, err := PackageEnvelopeFromJSON(data)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if got.Schema != env.Schema {
		t.Errorf("schema: got %q, want %q", got.Schema, env.Schema)
	}
	if got.Kind != env.Kind {
		t.Errorf("kind: got %q, want %q", got.Kind, env.Kind)
	}
	if got.From != env.From {
		t.Errorf("from: got %q, want %q", got.From, env.From)
	}
	if len(got.To) != 2 {
		t.Errorf("to: got %d recipients, want 2", len(got.To))
	}
	if got.Package.Name != "sunholo/auth" {
		t.Errorf("package.name: got %q", got.Package.Name)
	}
	if got.Package.ChangeClass != "C" {
		t.Errorf("change_class: got %q", got.Package.ChangeClass)
	}
	if got.Package.Breaking == nil || !*got.Package.Breaking {
		t.Error("breaking should be true")
	}
	if len(got.Package.EffectDelta) != 1 || got.Package.EffectDelta[0] != "net" {
		t.Errorf("effect_delta: got %v", got.Package.EffectDelta)
	}
	if got.Summary != env.Summary {
		t.Errorf("summary: got %q", got.Summary)
	}
	if got.Refs == nil || got.Refs.PackageURL != "pkg:sunholo/auth@0.2.0" {
		t.Error("refs.package_url not preserved")
	}
}

func TestToInboxMessage(t *testing.T) {
	env := newValidEnvelope(PkgMsgUpgradeAvailable)
	env.Summary = "test upgrade"

	msg, err := env.ToInboxMessage()
	if err != nil {
		t.Fatalf("ToInboxMessage failed: %v", err)
	}

	if msg.ToInbox != "workspace:docparse" {
		t.Errorf("to_inbox: got %q, want workspace:docparse", msg.ToInbox)
	}
	if msg.FromAgent != "pkg:sunholo/auth" {
		t.Errorf("from_agent: got %q", msg.FromAgent)
	}
	if msg.Title != "[upgrade-available] sunholo/auth" {
		t.Errorf("title: got %q", msg.Title)
	}
	if msg.Category != CategoryFeature {
		t.Errorf("category: got %q, want feature", msg.Category)
	}

	// Verify payload round-trips
	extracted, err := ExtractPackageEnvelope(msg)
	if err != nil {
		t.Fatalf("ExtractPackageEnvelope failed: %v", err)
	}
	if extracted == nil {
		t.Fatal("ExtractPackageEnvelope returned nil")
	}
	if extracted.Kind != PkgMsgUpgradeAvailable {
		t.Errorf("extracted kind: got %q", extracted.Kind)
	}
}

func TestExtractPackageEnvelope_NonPackageMessage(t *testing.T) {
	msg := &InboxMessage{Payload: "just a plain text message"}
	env, err := ExtractPackageEnvelope(msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env != nil {
		t.Fatal("expected nil for non-package message")
	}

	// Empty payload
	msg2 := &InboxMessage{}
	env2, err := ExtractPackageEnvelope(msg2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env2 != nil {
		t.Fatal("expected nil for empty payload")
	}

	// JSON but wrong schema
	msg3 := &InboxMessage{Payload: `{"schema": "other/v1", "kind": "test"}`}
	env3, err := ExtractPackageEnvelope(msg3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env3 != nil {
		t.Fatal("expected nil for wrong schema")
	}
}

func TestValidateSimpleKinds(t *testing.T) {
	// These kinds only need base validation
	simpleKinds := []PackageMessageKind{
		PkgMsgCompatibilityReq,
		PkgMsgDeprecationNotice,
		PkgMsgUpgradeComplete,
		PkgMsgBlocked,
		PkgMsgSuperseded,
	}
	for _, kind := range simpleKinds {
		t.Run(string(kind), func(t *testing.T) {
			env := newValidEnvelope(kind)
			if err := ValidatePackageMessage(env); err != nil {
				t.Fatalf("valid %s rejected: %v", kind, err)
			}
		})
	}
}

func TestContractRegressionCategory(t *testing.T) {
	env := newValidEnvelope(PkgMsgContractRegression)
	env.Package.AffectedExports = []string{"validateBearer"}
	env.Package.PrevContract = "old"
	env.Package.NewContract = "new"

	msg, err := env.ToInboxMessage()
	if err != nil {
		t.Fatalf("ToInboxMessage failed: %v", err)
	}
	if msg.Category != CategoryBug {
		t.Errorf("contract-regression should map to bug category, got %q", msg.Category)
	}
}
