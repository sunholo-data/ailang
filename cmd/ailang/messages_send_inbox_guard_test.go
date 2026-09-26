package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func guardReg(t *testing.T) *coordinator.AgentRegistry {
	t.Helper()
	reg := coordinator.NewAgentRegistry()
	if err := reg.Register(&coordinator.AgentConfig{
		ID: "design-doc-creator-daneel", Inbox: "daneel-design", Workspace: "/tmp/t",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}
	reg.SetTriageOnlyInboxes([]string{"user", "controlplane"})
	return reg
}

func TestCheckSendInbox_ServedAndTriagePass(t *testing.T) {
	reg := guardReg(t)
	for _, inbox := range []string{"daneel-design", "user", "controlplane"} {
		v := checkSendInbox(reg, inbox, true, false)
		if !v.Allow || v.Message != "" {
			t.Errorf("%q should send silently, got allow=%v msg=%q", inbox, v.Allow, v.Message)
		}
	}
}

// The measured failure: five design requests to daneel-design-ailang, all filed,
// all bounced, none dispatched.
func TestCheckSendInbox_UnknownInboxIsRefusedWithASuggestion(t *testing.T) {
	v := checkSendInbox(guardReg(t), "daneel-design-ailang", true, false)
	if v.Allow {
		t.Fatal("an unknown inbox on the authoritative registry must be refused")
	}
	if !strings.Contains(v.Message, "daneel-design") {
		t.Errorf("the refusal must name the likely intended inbox: %q", v.Message)
	}
	if !strings.Contains(v.Message, "FILED, NOT DISPATCHED") {
		t.Errorf("the refusal must say what would have happened: %q", v.Message)
	}
}

func TestCheckSendInbox_ForceSends(t *testing.T) {
	v := checkSendInbox(guardReg(t), "diag-probe", true, true)
	if !v.Allow {
		t.Error("--force must allow a deliberate probe")
	}
	if v.Message == "" {
		t.Error("--force must still say what it is doing")
	}
}

// The guard's own blind spot. resolveInboxRegistry falls back to this machine's
// config, which may hold two agents; refusing against it would block every
// legitimate send on a laptop without GCS credentials — a guard failing closed
// on its own ignorance. It must warn and send instead.
func TestCheckSendInbox_NonAuthoritativeRegistryWarnsButSends(t *testing.T) {
	v := checkSendInbox(guardReg(t), "sprint-planner", false, false)
	if !v.Allow {
		t.Fatal("a non-authoritative registry must never block a send")
	}
	if !strings.Contains(v.Message, "THIS MACHINE") {
		t.Errorf("the warning must say the verdict is not the plane's: %q", v.Message)
	}
}

func TestCheckSendInbox_NilRegistryNeverBlocks(t *testing.T) {
	if v := checkSendInbox(nil, "anything", true, false); !v.Allow {
		t.Error("no registry means no verdict, not a refusal")
	}
}
