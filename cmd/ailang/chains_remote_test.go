package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// legNames renders a leg set for failure messages.
func legNames(legs []observatory.IterationLeg) string {
	names := make([]string, 0, len(legs))
	for _, l := range legs {
		names = append(names, l.Name)
	}
	return strings.Join(names, "+")
}

// M-MISSION-LOOP-UNIFIED-TELEMETRY M3 — node-generic destination selection.
//
// These assert the WIRING decisions, which is where "node-generic" is either
// true or false: which legs exist must come from configuration alone, with no
// host-specific branch anywhere in the path.

func TestIterationLegs_NoCloudConfiguredIsLocalOnly(t *testing.T) {
	dir := t.TempDir()
	for _, mode := range []string{"", "local", "hybrid"} {
		t.Run("AILANG_STORAGE="+mode, func(t *testing.T) {
			t.Setenv("AILANG_STORAGE", mode)

			legs, closeLegs := iterationLegs(context.Background(),
				filepath.Join(dir, "observatory.db"), filepath.Join(dir, "spool.jsonl"))
			defer closeLegs()

			if len(legs) != 1 || legs[0].Name != "local" {
				t.Fatalf("got %d leg(s) %v, want exactly the local leg — behaviour must be identical to before M3", len(legs), legNames(legs))
			}
		})
	}
}

// "hybrid" resolves the observatory to local SQLite, so treating it as a cloud
// leg would dual-write the same database twice. Covered above; called out here
// because it is the one mode whose name suggests otherwise.

func TestIterationLegs_GCPAddsACloudLeg(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "gcp")
	// Deliberately no AILANG_CLOUD_PROJECT: the cloud leg must still EXIST so the
	// post is buffered, rather than being silently dropped because the
	// destination could not be opened.
	t.Setenv("AILANG_CLOUD_PROJECT", "")

	dir := t.TempDir()
	legs, closeLegs := iterationLegs(context.Background(),
		filepath.Join(dir, "observatory.db"), filepath.Join(dir, "spool.jsonl"))
	defer closeLegs()

	if len(legs) != 2 {
		t.Fatalf("got %d leg(s) %v, want local+cloud", len(legs), legNames(legs))
	}
	cloud := legs[1]
	if cloud.Name != "cloud" {
		t.Fatalf("second leg is %q, want %q", cloud.Name, "cloud")
	}
	if cloud.Sink != nil || cloud.Err == nil {
		t.Error("an unopenable cloud leg must carry a nil sink AND a reason, so PostToLegs spools instead of dropping")
	}
	if cloud.Spool.Path == legs[0].Spool.Path {
		t.Error("legs share a spool file; a shared spool replays posts the other leg already stored")
	}
}

func TestCloudSpoolPath_IsDistinctAndNextToTheLocalOne(t *testing.T) {
	tests := []struct{ local, want string }{
		{"/x/y/chains-iteration-spool.jsonl", "/x/y/chains-iteration-spool-cloud.jsonl"},
		{"spool.jsonl", "spool-cloud.jsonl"},
		{"/x/y/spool", "/x/y/spool-cloud"},
	}
	for _, tt := range tests {
		if got := cloudSpoolPath(tt.local); got != tt.want {
			t.Errorf("cloudSpoolPath(%q) = %q, want %q", tt.local, got, tt.want)
		}
	}
}

// TestOpenChainBackend_RemoteWithoutCloudFailsLoudly: the ratified read decision
// is opt-in remote, and an opt-in that quietly answers from the local store
// would report a cloud-side record as absent when it was never queried.
func TestOpenChainBackend_RemoteWithoutCloudFailsLoudly(t *testing.T) {
	for _, mode := range []string{"", "local", "hybrid"} {
		t.Run("AILANG_STORAGE="+mode, func(t *testing.T) {
			t.Setenv("AILANG_STORAGE", mode)

			backend, closeBackend, err := openChainBackend(context.Background(), true)
			defer closeBackend()

			if err == nil {
				t.Fatal("openChainBackend(--remote) succeeded with no cloud configured; want a loud error, never a silent local fallback")
			}
			if backend != nil {
				t.Error("openChainBackend returned a backend alongside its error")
			}
			if !strings.Contains(err.Error(), "AILANG_STORAGE") {
				t.Errorf("error does not say how to configure a cloud observatory: %v", err)
			}
		})
	}
}
