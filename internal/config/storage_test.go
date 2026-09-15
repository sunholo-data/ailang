package config

import (
	"errors"
	"strings"
	"testing"
)

// clearPlaneEnv unsets every variable StoragePlane and CoordinatorMode read,
// including the retired names (a developer's shell may still export one).
func clearPlaneEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{EnvStorage, EnvStorageMessaging, EnvStorageCoordinator,
		EnvStorageObservatory, EnvCoordinatorMode} {
		t.Setenv(v, "")
	}
	for _, v := range RemovedEnvNames() {
		t.Setenv(v, "")
	}
}

func TestStoragePlaneDefaultsToLocalEverywhere(t *testing.T) {
	clearPlaneEnv(t)
	s, err := StoragePlane()
	if err != nil {
		t.Fatal(err)
	}
	if s.Plane != PlaneLocal || s.PlaneSource != SourceDefault {
		t.Fatalf("plane = %s via %s, want local via default", s.Plane, s.PlaneSource)
	}
	for _, sel := range []StoreSelection{s.Messaging, s.Coordinator, s.Observatory} {
		if sel.Mode != StoreLocal || sel.Source != SourceDefault {
			t.Errorf("%s = %s via %s, want local via default", sel.Store, sel.Mode, sel.Source)
		}
	}
	if s.Shared() || s.AnyGCP() {
		t.Error("a default plane is neither shared nor needs Firestore")
	}
}

func TestStoragePlaneGCPMovesEveryStore(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "gcp")
	s, err := StoragePlane()
	if err != nil {
		t.Fatal(err)
	}
	for _, sel := range []StoreSelection{s.Messaging, s.Coordinator, s.Observatory} {
		if sel.Mode != StoreGCP || sel.Source != Source(EnvStorage) {
			t.Errorf("%s = %s via %s, want gcp via %s", sel.Store, sel.Mode, sel.Source, EnvStorage)
		}
	}
	if !s.Shared() || !s.AnyGCP() {
		t.Error("gcp is shared and needs Firestore")
	}
}

func TestStoragePlaneHybridIsSharedButLocalStores(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "hybrid")
	s, err := StoragePlane()
	if err != nil {
		t.Fatal(err)
	}
	if s.Plane != PlaneHybrid || !s.Shared() {
		t.Fatalf("hybrid must be the shared plane, got %+v", s)
	}
	if s.AnyGCP() {
		t.Error("hybrid opens every store in SQLite; nothing here needs Firestore")
	}
	for _, sel := range []StoreSelection{s.Messaging, s.Coordinator, s.Observatory} {
		if sel.Mode != StoreLocal {
			t.Errorf("%s = %s, want local under hybrid", sel.Store, sel.Mode)
		}
	}
}

func TestStoragePlanePerStoreOverrideNamesItsSource(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "local")
	t.Setenv(EnvStorageMessaging, "gcp")
	s, err := StoragePlane()
	if err != nil {
		t.Fatal(err)
	}
	if s.Messaging.Mode != StoreGCP || s.Messaging.Source != Source(EnvStorageMessaging) {
		t.Errorf("messaging = %s via %s, want gcp via %s", s.Messaging.Mode, s.Messaging.Source, EnvStorageMessaging)
	}
	if s.Coordinator.Mode != StoreLocal || s.Coordinator.Source != Source(EnvStorage) {
		t.Errorf("coordinator = %s via %s, want local via %s", s.Coordinator.Mode, s.Coordinator.Source, EnvStorage)
	}
	if s.Shared() {
		t.Error("a per-store override does not make the plane shared")
	}
	if !s.AnyGCP() {
		t.Error("one gcp store needs Firestore")
	}
	if s.Select(StoreMessaging) != s.Messaging || s.Select(StoreObservatory) != s.Observatory {
		t.Error("Select must return the matching selection")
	}
}

func TestStoragePlaneRejectsRemovedNamesWithReplacement(t *testing.T) {
	for _, tc := range []struct{ name, replacement string }{
		{"AILANG_MESSAGES_STORE", EnvStorageMessaging + "=gcp"},
		{"AILANG_COORDINATOR_REMOTE", EnvStorageCoordinator + "=gcp"},
		{"AILANG_CHAINS_READ", EnvStorageObservatory + "=gcp"},
		{"AILANG_CHAINS_CLOUD", EnvStorageObservatory + "=gcp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearPlaneEnv(t)
			t.Setenv(tc.name, "gcp")
			_, err := StoragePlane()
			if !errors.Is(err, ErrRemovedEnv) {
				t.Fatalf("err = %v, want ErrRemovedEnv", err)
			}
			if !strings.Contains(err.Error(), tc.name+" was removed") || !strings.Contains(err.Error(), "set "+tc.replacement) {
				t.Fatalf("error must name the removed variable and its replacement: %v", err)
			}
			// CoordinatorMode goes through the same gate.
			if _, _, err := CoordinatorMode(); !errors.Is(err, ErrRemovedEnv) {
				t.Fatalf("CoordinatorMode: err = %v, want ErrRemovedEnv", err)
			}
		})
	}
	// Control: an EMPTY retired variable is "unset", not an error — a shell
	// that ran `export AILANG_MESSAGES_STORE=` must not be locked out.
	clearPlaneEnv(t)
	if _, err := StoragePlane(); err != nil {
		t.Fatalf("empty retired variables must not error: %v", err)
	}
}

func TestStoragePlaneRejectsBadValues(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "dynamodb")
	if _, err := StoragePlane(); !errors.Is(err, ErrBadPlane) || !strings.Contains(err.Error(), EnvStorage) {
		t.Fatalf("bad plane: %v", err)
	}
	clearPlaneEnv(t)
	t.Setenv(EnvStorageObservatory, "bigquery")
	if _, err := StoragePlane(); !errors.Is(err, ErrBadPlane) || !strings.Contains(err.Error(), EnvStorageObservatory) {
		t.Fatalf("bad override: %v", err)
	}
	clearPlaneEnv(t)
	t.Setenv(EnvStorageCoordinator, "hybrid")
	_, err := StoragePlane()
	if !errors.Is(err, ErrBadPlane) || !strings.Contains(err.Error(), "hybrid is a plane value") {
		t.Fatalf("hybrid as a per-store value must be refused and explained: %v", err)
	}
}

func TestStorageForPlaneReadsNoEnvironment(t *testing.T) {
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "gcp")
	t.Setenv(EnvStorageMessaging, "local")
	s := StorageForPlane(PlaneHybrid)
	if s.Plane != PlaneHybrid || s.Messaging.Mode != StoreLocal || s.Coordinator.Mode != StoreLocal {
		t.Fatalf("explicit hybrid: %+v", s)
	}
	if g := StorageForPlane(PlaneGCP); g.Observatory.Mode != StoreGCP {
		t.Fatalf("explicit gcp: %+v", g)
	}
}

func TestParsePlane(t *testing.T) {
	for in, want := range map[string]Plane{"": PlaneLocal, "local": PlaneLocal, " gcp ": PlaneGCP, "hybrid": PlaneHybrid} {
		if got, err := ParsePlane(in); err != nil || got != want {
			t.Errorf("ParsePlane(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParsePlane("firestore"); !errors.Is(err, ErrBadPlane) {
		t.Errorf("ParsePlane(firestore) = %v, want ErrBadPlane", err)
	}
}

func TestCoordinatorModeValidatesAgainstPlane(t *testing.T) {
	// Unset: local, whatever the plane — the rig runs a local-execution
	// coordinator on the gcp plane (dev.ailang.coordinator.plist, 2026-09-15).
	clearPlaneEnv(t)
	t.Setenv(EnvStorage, "gcp")
	m, src, err := CoordinatorMode()
	if err != nil || m != CoordinatorModeLocal || src != SourceDefault {
		t.Fatalf("unset on gcp: %s via %s, %v; want local via default", m, src, err)
	}

	// cloud on gcp agrees.
	t.Setenv(EnvCoordinatorMode, "cloud")
	m, src, err = CoordinatorMode()
	if err != nil || m != CoordinatorModeCloud || src != Source(EnvCoordinatorMode) {
		t.Fatalf("cloud on gcp: %s via %s, %v", m, src, err)
	}

	// cloud with the coordinator store in SQLite is the silent disagreement
	// this exists to catch: Pub/Sub push intake against a database no Cloud
	// Run Job can see.
	cases := []struct{ set, val string }{
		{EnvStorage, "local"},
		{EnvStorage, "hybrid"},
		{EnvStorageCoordinator, "local"},
		{EnvStorageMessaging, "local"},
	}
	for _, c := range cases {
		clearPlaneEnv(t)
		t.Setenv(EnvStorage, "gcp")
		t.Setenv(c.set, c.val)
		t.Setenv(EnvCoordinatorMode, "cloud")
		_, _, err := CoordinatorMode()
		if !errors.Is(err, ErrCoordinatorModeDisagrees) {
			t.Errorf("%s=%s with COORDINATOR_MODE=cloud: err = %v, want ErrCoordinatorModeDisagrees", c.set, c.val, err)
		} else if !strings.Contains(err.Error(), "set "+EnvStorage+"=gcp") {
			t.Errorf("error must say what to set: %v", err)
		}
	}

	clearPlaneEnv(t)
	t.Setenv(EnvCoordinatorMode, "serverless")
	if _, _, err := CoordinatorMode(); !errors.Is(err, ErrBadPlane) {
		t.Errorf("unknown mode: %v", err)
	}
	clearPlaneEnv(t)
	t.Setenv(EnvCoordinatorMode, "local")
	if m, src, err := CoordinatorMode(); err != nil || m != CoordinatorModeLocal || src != Source(EnvCoordinatorMode) {
		t.Errorf("explicit local: %s via %s, %v", m, src, err)
	}
}
