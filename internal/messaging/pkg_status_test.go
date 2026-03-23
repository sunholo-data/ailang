package messaging

import (
	"testing"
	"time"
)

func TestTriagePackageMessage(t *testing.T) {
	tests := []struct {
		name        string
		kind        PackageMessageKind
		changeClass string
		result      string
		wantAction  TriageActionability
	}{
		{"internal upgrade", PkgMsgUpgradeAvailable, "A", "", TriageNoAction},
		{"contract change", PkgMsgUpgradeAvailable, "C", "", TriageMigrate},
		{"content change", PkgMsgUpgradeAvailable, "B", "", TriageVerifyLocal},
		{"interface change", PkgMsgInterfaceChange, "", "", TriageVerifyLocal},
		{"effect widening", PkgMsgEffectWidening, "", "", TriagePolicyBlock},
		{"contract regression", PkgMsgContractRegression, "", "", TriageEscalate},
		{"compat pass", PkgMsgCompatibilityReport, "", "pass", TriageNoAction},
		{"compat fail", PkgMsgCompatibilityReport, "", "fail", TriageMigrate},
		{"blocked", PkgMsgBlocked, "", "", TriageEscalate},
		{"deprecation", PkgMsgDeprecationNotice, "", "", TriageMigrate},
		{"superseded", PkgMsgSuperseded, "", "", TriageNoAction},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &PackageMessageEnvelope{
				Kind: tc.kind,
				Package: PackageRef{
					Name:        "sunholo/auth",
					ChangeClass: tc.changeClass,
					Result:      tc.result,
				},
			}
			result := TriagePackageMessage(env)
			if result.Action != tc.wantAction {
				t.Errorf("got %q, want %q", result.Action, tc.wantAction)
			}
			if result.Reason == "" {
				t.Error("expected non-empty reason")
			}
		})
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		from, to string
		valid    bool
	}{
		{PkgStatusOpen, PkgStatusAcknowledged, true},
		{PkgStatusOpen, PkgStatusRejected, true},
		{PkgStatusOpen, PkgStatusSuperseded, true},
		{PkgStatusOpen, PkgStatusCompleted, false},
		{PkgStatusAcknowledged, PkgStatusInProgress, true},
		{PkgStatusInProgress, PkgStatusCompleted, true},
		{PkgStatusInProgress, PkgStatusBlocked, true},
		{PkgStatusBlocked, PkgStatusInProgress, true},
		{PkgStatusCompleted, PkgStatusOpen, false},
		{PkgStatusRejected, PkgStatusOpen, false},
		{PkgStatusSuperseded, PkgStatusOpen, false},
	}
	for _, tc := range tests {
		t.Run(tc.from+"→"+tc.to, func(t *testing.T) {
			got := isValidTransition(tc.from, tc.to)
			if got != tc.valid {
				t.Errorf("isValidTransition(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.valid)
			}
		})
	}
}

func TestUpdatePackageMessageStatus(t *testing.T) {
	store := newTestStore(t)

	// Create a package message
	brk := false
	env := &PackageMessageEnvelope{
		Schema:    PackageMessageSchema,
		Kind:      PkgMsgUpgradeAvailable,
		From:      "pkg:sunholo/auth",
		To:        []string{"workspace:docparse"},
		Timestamp: time.Now(),
		Package: PackageRef{
			Name:              "sunholo/auth",
			FromVersion:       "0.1.0",
			ToVersion:         "0.2.0",
			FromInterfaceHash: "sha256:old",
			ToInterfaceHash:   "sha256:new",
			ChangeClass:       "C",
			Breaking:          &brk,
		},
		Status: "open",
	}
	msg, err := env.ToInboxMessage()
	if err != nil {
		t.Fatalf("ToInboxMessage failed: %v", err)
	}
	if err := store.InsertInboxMessage(msg); err != nil {
		t.Fatalf("InsertInboxMessage failed: %v", err)
	}

	// Valid transition: open → acknowledged
	if err := store.UpdatePackageMessageStatus(msg.ID, PkgStatusAcknowledged); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}

	// Verify the status was updated in the payload
	updated, err := store.GetInboxMessage(msg.ID)
	if err != nil {
		t.Fatalf("GetInboxMessage failed: %v", err)
	}
	updatedEnv, err := ExtractPackageEnvelope(updated)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if updatedEnv.Status != PkgStatusAcknowledged {
		t.Errorf("status: got %q, want acknowledged", updatedEnv.Status)
	}

	// Invalid transition: acknowledged → completed (skips in_progress)
	if err := store.UpdatePackageMessageStatus(msg.ID, PkgStatusCompleted); err == nil {
		t.Error("expected error for invalid transition acknowledged → completed")
	}
}

func TestSupersedeOlderMessages(t *testing.T) {
	store := newTestStore(t)

	// Create two upgrade-available messages for same package
	for _, version := range []string{"0.2.0", "0.3.0"} {
		brk := false
		env := &PackageMessageEnvelope{
			Schema:    PackageMessageSchema,
			Kind:      PkgMsgUpgradeAvailable,
			From:      "pkg:sunholo/auth",
			To:        []string{"pkg:sunholo/auth"},
			Timestamp: time.Now(),
			Package: PackageRef{
				Name:              "sunholo/auth",
				FromVersion:       "0.1.0",
				ToVersion:         version,
				FromInterfaceHash: "sha256:old",
				ToInterfaceHash:   "sha256:new-" + version,
				ChangeClass:       "C",
				Breaking:          &brk,
			},
			Status: "open",
		}
		msg, _ := env.ToInboxMessage()
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	// Supersede older messages for version 0.3.0
	count, err := store.SupersedeOlderMessages("sunholo/auth", "0.3.0")
	if err != nil {
		t.Fatalf("SupersedeOlderMessages failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 superseded, got %d", count)
	}
}

func TestDeduplicatePackageReports(t *testing.T) {
	store := newTestStore(t)

	// Create 3 compatibility reports for same package+version
	for i := 0; i < 3; i++ {
		brk := false
		env := &PackageMessageEnvelope{
			Schema:    PackageMessageSchema,
			Kind:      PkgMsgCompatibilityReport,
			From:      "workspace:docparse",
			To:        []string{"pkg:sunholo/auth"},
			Timestamp: time.Now(),
			Package: PackageRef{
				Name:            "sunholo/auth",
				FromVersion:     "0.1.0",
				ToVersion:       "0.2.0",
				FromContentHash: "sha256:c1",
				ToContentHash:   "sha256:c2",
				TargetWorkspace: "workspace:docparse",
				Result:          "pass",
				Breaking:        &brk,
			},
			Status: "open",
		}
		msg, _ := env.ToInboxMessage()
		if err := store.InsertInboxMessage(msg); err != nil {
			t.Fatalf("insert failed: %v", err)
		}
	}

	deduped, err := store.DeduplicatePackageReports("sunholo/auth")
	if err != nil {
		t.Fatalf("dedupe failed: %v", err)
	}
	if deduped != 2 {
		t.Errorf("expected 2 deduped, got %d", deduped)
	}
}
