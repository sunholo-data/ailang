package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/storage"
)

// The approvals path must read the registry of the plane the TASK is on.
//
// It read this machine's config instead (coordinator.LoadAgentRegistry) from
// 2026-09-07 to 2026-09-23. On a laptop that config declares two agents, so
// every cloud task's agent failed to resolve and checkRegistryCanDispatch —
// built to stop an approval that would dispatch nothing — refused every
// approval instead. 83 pending on prod, oldest twelve days, none decidable
// from a terminal. The dashboard was unaffected, which is what made it look
// intermittent rather than broken.
//
// These tests pin the wiring, not the artifact: which registry a plane reads,
// and that the plane survives the trip from bundle.Mode.

func TestResolveInboxRegistryForPlane_GCPReadsThePlanesRegistry(t *testing.T) {
	t.Setenv(string(config.EnvConfigFile), "")

	const sentinel = "gs://the-planes-own-registry"
	restore := loadCloudRegistry
	loadCloudRegistry = func() (*coordinator.AgentRegistry, string, error) {
		return nil, sentinel, nil
	}
	defer func() { loadCloudRegistry = restore }()

	_, source, err := resolveInboxRegistryForPlane("", storage.ModeGCP)
	if err != nil {
		t.Fatalf("resolve on the gcp plane: %v", err)
	}
	if source != sentinel {
		t.Errorf("gcp plane resolved %q, want the plane's own registry %q\n"+
			"a cloud approval judged against this machine's config refuses every cloud agent",
			source, sentinel)
	}
}

func TestResolveInboxRegistryForPlane_LocalDoesNotReachForTheCloud(t *testing.T) {
	t.Setenv(string(config.EnvConfigFile), "")

	called := false
	restore := loadCloudRegistry
	loadCloudRegistry = func() (*coordinator.AgentRegistry, string, error) {
		called = true
		return nil, "cloud", nil
	}
	defer func() { loadCloudRegistry = restore }()

	if _, _, err := resolveInboxRegistryForPlane("", storage.ModeLocal); err != nil {
		t.Fatalf("resolve on the local plane: %v", err)
	}
	if called {
		t.Error("the local plane read the cloud registry; a local task's handoffs come from the local config")
	}
}

// The call site itself: approvalRegistry takes the bundle's Mode LABEL and must
// still reach the plane's registry. This is the test that was missing — the
// resolver tests below pass happily while the call site hands it a label that
// matches no plane, which is how the original bug would have come back.
func TestApprovalRegistry_CloudLabelReachesThePlanesRegistry(t *testing.T) {
	t.Setenv(string(config.EnvConfigFile), "")

	const sentinel = "gs://the-planes-own-registry"
	restore := loadCloudRegistry
	loadCloudRegistry = func() (*coordinator.AgentRegistry, string, error) {
		return nil, sentinel, nil
	}
	defer func() { loadCloudRegistry = restore }()

	// Exactly what coordinatorStoreBundle.Mode holds for the prod plane.
	const bundleMode = "gcp (project ailang-multivac, via config.yaml pubsub.project_id)"

	_, source, err := approvalRegistry(bundleMode)
	if err != nil {
		t.Fatalf("approvalRegistry(%q): %v", bundleMode, err)
	}
	if source != sentinel {
		t.Errorf("approving a task on %q judged against %q, want the plane's registry %q\n"+
			"this is the 2026-09-07 fault: the guard cannot resolve a cloud agent and refuses every approval",
			bundleMode, source, sentinel)
	}
}

func TestApprovalRegistry_LocalLabelStaysLocal(t *testing.T) {
	t.Setenv(string(config.EnvConfigFile), "")

	called := false
	restore := loadCloudRegistry
	loadCloudRegistry = func() (*coordinator.AgentRegistry, string, error) {
		called = true
		return nil, "cloud", nil
	}
	defer func() { loadCloudRegistry = restore }()

	if _, _, err := approvalRegistry("local (/Users/x/.ailang/coordinator.db)"); err != nil {
		t.Fatalf("approvalRegistry on a local bundle: %v", err)
	}
	if called {
		t.Error("a local task's approval read the cloud registry")
	}
}

// bundle.Mode is a human label, not a plane. storage.Mode(bundle.Mode) equals
// no plane at all, falls through to this machine's config, and reinstates the
// whole bug silently — which is why the call site converts with firstWord.
func TestBundleModeConvertsToARealPlane(t *testing.T) {
	cases := []struct {
		mode string
		want storage.Mode
	}{
		{"gcp (project ailang-multivac, via config.yaml pubsub.project_id)", storage.ModeGCP},
		{"local (/Users/x/.ailang/coordinator.db)", storage.ModeLocal},
		{"gcp", storage.ModeGCP},
	}
	for _, tc := range cases {
		if got := storage.Mode(firstWord(tc.mode)); got != tc.want {
			t.Errorf("bundle.Mode %q resolved plane %q, want %q", tc.mode, got, tc.want)
		}
		// The control: the unconverted label must NOT be a usable plane, or
		// this test would pass with the bug present.
		if tc.mode != string(tc.want) && storage.Mode(tc.mode) == tc.want {
			t.Errorf("label %q compared equal to plane %q — the conversion is untested", tc.mode, tc.want)
		}
	}
}
