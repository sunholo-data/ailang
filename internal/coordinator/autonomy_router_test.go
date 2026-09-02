package coordinator

import (
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func boolPtr(b bool) *bool { return &b }

func TestClassifyChange_AllMessageKinds(t *testing.T) {
	tests := []struct {
		kind     messaging.PackageMessageKind
		breaking *bool
		class    string
		expected ChangeClass
	}{
		{messaging.PkgMsgUpgradeAvailable, nil, "patch", ChangeClassA},
		{messaging.PkgMsgUpgradeAvailable, nil, "minor", ChangeClassB},
		{messaging.PkgMsgUpgradeAvailable, nil, "", ChangeClassA},
		{messaging.PkgMsgInterfaceChange, nil, "minor", ChangeClassB},
		{messaging.PkgMsgInterfaceChange, nil, "major", ChangeClassC},
		{messaging.PkgMsgInterfaceChange, nil, "", ChangeClassB},
		{messaging.PkgMsgEffectWidening, nil, "", ChangeClassC},
		{messaging.PkgMsgEffectWidening, boolPtr(false), "", ChangeClassC}, // Always C regardless of breaking
		{messaging.PkgMsgContractRegression, nil, "", ChangeClassC},
		{messaging.PkgMsgCompatibilityReq, nil, "", ChangeClassA},
		{messaging.PkgMsgCompatibilityReport, nil, "", ChangeClassA},
		{messaging.PkgMsgDeprecationNotice, nil, "", ChangeClassB},
		{messaging.PkgMsgMigrationRequest, nil, "", ChangeClassB},
		{messaging.PkgMsgUpgradeComplete, nil, "", ChangeClassA},
		{messaging.PkgMsgBlocked, nil, "", ChangeClassB},
		{messaging.PkgMsgSuperseded, nil, "", ChangeClassA},
	}

	for _, tt := range tests {
		t.Run(string(tt.kind)+"_"+tt.class, func(t *testing.T) {
			env := &messaging.PackageMessageEnvelope{
				Kind: tt.kind,
				Package: messaging.PackageRef{
					Name:        "sunholo/auth",
					ChangeClass: tt.class,
					Breaking:    tt.breaking,
				},
			}
			got := ClassifyChange(env)
			if got != tt.expected {
				t.Errorf("ClassifyChange(%s, class=%q) = %d, want %d", tt.kind, tt.class, got, tt.expected)
			}
		})
	}
}

func TestClassifyChange_BreakingOverride(t *testing.T) {
	// Breaking flag overrides everything to Class C
	env := &messaging.PackageMessageEnvelope{
		Kind: messaging.PkgMsgUpgradeAvailable,
		Package: messaging.PackageRef{
			Name:        "sunholo/auth",
			ChangeClass: "patch",
			Breaking:    boolPtr(true),
		},
	}
	got := ClassifyChange(env)
	if got != ChangeClassC {
		t.Errorf("expected Class C for breaking=true, got %d", got)
	}
}

func TestAutonomyRouter_UnknownRoutesToReview(t *testing.T) {
	// The "U" check is KIND-INDEPENDENT by design (see ClassifyChange). Both
	// emitters that call M5's classifier can stamp "U", and a per-kind arm is
	// exactly what let a mixed-signature upgrade auto-apply. This table covers
	// every kind that carries a change class; add a row when a kind is added.
	kinds := []messaging.PackageMessageKind{
		messaging.PkgMsgInterfaceChange,
		messaging.PkgMsgUpgradeAvailable,
	}
	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			env := &messaging.PackageMessageEnvelope{
				Kind: kind,
				Package: messaging.PackageRef{
					Name:        "sunholo/auth",
					ChangeClass: "U",
					// Breaking is deliberately nil: "U" is not a known-breaking
					// change, so the breaking override must NOT be what saves us.
				},
			}
			if got := ClassifyChange(env); got != ChangeClassC {
				t.Fatalf("ClassifyChange(%s, U) = %d, want %d (unknown must never auto-apply)", kind, got, ChangeClassC)
			}
		})
	}
}

// TestAutonomyRouter_MixedSignaturePairRoutesToReview is the CROSS-BOUNDARY lock
// for the "U" path: it emits a real mixed-signature pair through
// internal/messaging and routes the emitted envelope through this package.
//
// A mixed pair (exactly one side carrying v2 signatures) is the unavoidable
// shape of every package's FIRST post-migration publish, so this is the path
// the feature takes on its first real use — not an edge case. Before this lock
// existed, the upgrade-available half of it classified as ChangeClassA, i.e.
// fully autonomous auto-merge of a change the classifier had just declared
// unmeasurable.
func TestAutonomyRouter_MixedSignaturePairRoutesToReview(t *testing.T) {
	old := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "0.1.0",
		InterfaceHash: "sha256:aaa", ContentHash: "sha256:ccc",
		Signatures: []string{"auth:func:login:String -> Bool"},
	}
	newer := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "0.2.0",
		InterfaceHash: "sha256:bbb", ContentHash: "sha256:ddd",
		// no Signatures: the old side is v2, the new side is legacy => "U"
	}

	cases := []struct {
		name string
		emit func(*messaging.Store) (string, error)
	}{
		{"upgrade-available", func(st *messaging.Store) (string, error) {
			return messaging.EmitUpgradeAvailable(st, old, newer, []string{"workspace:probe"})
		}},
		{"interface-change", func(st *messaging.Store) (string, error) {
			return messaging.EmitInterfaceChangeNotice(st, old, newer, []string{"workspace:probe"})
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := messaging.OpenStore(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("OpenStore: %v", err)
			}
			t.Cleanup(func() { store.Close() })

			if _, err := tc.emit(store); err != nil {
				t.Fatalf("emit: %v", err)
			}
			msgs, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: "workspace:probe", Limit: 10})
			if err != nil {
				t.Fatalf("ListInboxMessages: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 emitted message, got %d", len(msgs))
			}
			env, err := messaging.ExtractPackageEnvelope(&msgs[0])
			if err != nil {
				t.Fatalf("ExtractPackageEnvelope: %v", err)
			}
			if env.Package.ChangeClass != "U" {
				t.Fatalf("emitted change_class = %q, want U for a mixed-signature pair", env.Package.ChangeClass)
			}
			// The breaking override must NOT be what rescues this: "U" is not a
			// known-breaking change, so the flag is false and the class alone
			// has to carry the decision.
			if env.Package.Breaking == nil || *env.Package.Breaking {
				t.Fatalf("emitted breaking = %v, want non-nil false for a mixed pair classified U", env.Package.Breaking)
			}
			if got := ClassifyChange(env); got != ChangeClassC {
				t.Fatalf("ClassifyChange = %d, want %d — an unknown change class must never auto-apply", got, ChangeClassC)
			}
		})
	}
}

func TestAutonomyRouter_LegacyEnvelopeRoutingUnchanged(t *testing.T) {
	// A wholly-legacy pair: neither side carries signature metadata, and the
	// interface hash changed, so the hash-only classifier yields "C".
	old := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "0.1.0",
		InterfaceHash: "sha256:aaa", ContentHash: "sha256:ccc",
	}
	newer := messaging.PackageVersionInfo{
		Name: "sunholo/auth", Version: "0.2.0",
		InterfaceHash: "sha256:bbb", ContentHash: "sha256:ddd",
	}

	cases := []struct {
		name string
		emit func(*messaging.Store) (string, error)
		want ChangeClass
	}{
		{
			name: "legacy upgrade-available stays auto-apply",
			emit: func(st *messaging.Store) (string, error) {
				return messaging.EmitUpgradeAvailable(st, old, newer, []string{"workspace:probe"})
			},
			want: ChangeClassA,
		},
		{
			name: "legacy interface-change stays semi-autonomous",
			emit: func(st *messaging.Store) (string, error) {
				return messaging.EmitInterfaceChangeNotice(st, old, newer, []string{"workspace:probe"})
			},
			want: ChangeClassB,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := messaging.OpenStore(filepath.Join(t.TempDir(), "test.db"))
			if err != nil {
				t.Fatalf("OpenStore: %v", err)
			}
			t.Cleanup(func() { store.Close() })

			if _, err := tc.emit(store); err != nil {
				t.Fatalf("emit: %v", err)
			}
			msgs, err := store.ListInboxMessages(messaging.InboxListOptions{Inbox: "workspace:probe", Limit: 10})
			if err != nil {
				t.Fatalf("ListInboxMessages: %v", err)
			}
			if len(msgs) != 1 {
				t.Fatalf("expected 1 emitted message, got %d", len(msgs))
			}
			env, err := messaging.ExtractPackageEnvelope(&msgs[0])
			if err != nil {
				t.Fatalf("ExtractPackageEnvelope: %v", err)
			}
			if env.Package.Breaking != nil {
				t.Errorf("emitted breaking = %v, want nil for a legacy pair", *env.Package.Breaking)
			}
			if got := ClassifyChange(env); got != tc.want {
				t.Fatalf("ClassifyChange = %d, want %d — legacy cascade routing changed", got, tc.want)
			}
		})
	}
}

func TestAdjustAutonomyForChangeClass_ClassA(t *testing.T) {
	agent := &AgentConfig{
		ID:                  "pkg-auth",
		SkipApproval:        false,
		AutoMerge:           false,
		AutoApproveHandoffs: false,
	}
	msg := &messaging.InboxMessage{
		Payload: `{"schema":"ailang.package-message/v1","kind":"upgrade-available","from":"registry","to":["pkg:sunholo/auth"],"timestamp":"2026-01-01T00:00:00Z","package":{"name":"sunholo/auth","to_version":"0.2.0","change_class":"patch"}}`,
	}

	result := AdjustAutonomyForChangeClass(agent, msg)
	if !result.SkipApproval {
		t.Error("expected SkipApproval=true for Class A")
	}
	if !result.AutoMerge {
		t.Error("expected AutoMerge=true for Class A")
	}
	if !result.AutoApproveHandoffs {
		t.Error("expected AutoApproveHandoffs=true for Class A")
	}
	// Original should be unchanged
	if agent.SkipApproval {
		t.Error("original agent should not be modified")
	}
}

func TestAdjustAutonomyForChangeClass_ClassB(t *testing.T) {
	agent := &AgentConfig{
		ID:                  "pkg-auth",
		SkipApproval:        true,
		AutoMerge:           true,
		AutoApproveHandoffs: false,
	}
	msg := &messaging.InboxMessage{
		Payload: `{"schema":"ailang.package-message/v1","kind":"interface-change-notice","from":"registry","to":["pkg:sunholo/auth"],"timestamp":"2026-01-01T00:00:00Z","package":{"name":"sunholo/auth","to_version":"0.2.0","change_class":"minor","from_interface_hash":"abc","to_interface_hash":"def"}}`,
	}

	result := AdjustAutonomyForChangeClass(agent, msg)
	if result.SkipApproval {
		t.Error("expected SkipApproval=false for Class B")
	}
	if result.AutoMerge {
		t.Error("expected AutoMerge=false for Class B")
	}
	if !result.AutoApproveHandoffs {
		t.Error("expected AutoApproveHandoffs=true for Class B")
	}
}

func TestAdjustAutonomyForChangeClass_ClassC(t *testing.T) {
	agent := &AgentConfig{
		ID:                  "pkg-auth",
		SkipApproval:        true,
		AutoMerge:           true,
		AutoApproveHandoffs: true,
	}
	msg := &messaging.InboxMessage{
		Payload: `{"schema":"ailang.package-message/v1","kind":"effect-widening-warning","from":"registry","to":["pkg:sunholo/auth"],"timestamp":"2026-01-01T00:00:00Z","package":{"name":"sunholo/auth","to_version":"0.2.0","prev_effect_ceiling":["Net"],"new_effect_ceiling":["Net","FS"]}}`,
	}

	result := AdjustAutonomyForChangeClass(agent, msg)
	if result.SkipApproval {
		t.Error("expected SkipApproval=false for Class C")
	}
	if result.AutoMerge {
		t.Error("expected AutoMerge=false for Class C")
	}
	if result.AutoApproveHandoffs {
		t.Error("expected AutoApproveHandoffs=false for Class C")
	}
}

func TestAdjustAutonomyForChangeClass_NonPackageMessage(t *testing.T) {
	agent := &AgentConfig{
		ID:           "coordinator",
		SkipApproval: false,
		AutoMerge:    false,
	}
	msg := &messaging.InboxMessage{
		Payload: "This is a regular text message, not a package envelope",
	}

	result := AdjustAutonomyForChangeClass(agent, msg)
	// Should return unchanged for non-package messages
	if result.SkipApproval != agent.SkipApproval {
		t.Error("non-package message should not change SkipApproval")
	}
	if result.AutoMerge != agent.AutoMerge {
		t.Error("non-package message should not change AutoMerge")
	}
}

func TestAdjustAutonomyForChangeClass_NilInputs(t *testing.T) {
	if AdjustAutonomyForChangeClass(nil, nil) != nil {
		t.Error("nil agent should return nil")
	}

	agent := &AgentConfig{ID: "test"}
	if AdjustAutonomyForChangeClass(agent, nil) != agent {
		t.Error("nil message should return original agent")
	}
}
