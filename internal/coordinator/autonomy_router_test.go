package coordinator

import (
	"testing"

	"github.com/sunholo/ailang/internal/messaging"
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
