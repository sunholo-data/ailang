package main

import (
	"errors"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// clearCoordinatorPlaneEnv isolates a test from the plane variables and the
// retired selectors a developer's shell may still export.
func clearCoordinatorPlaneEnv(t *testing.T) {
	t.Helper()
	for _, v := range []string{config.EnvStorage, config.EnvStorageCoordinator, config.EnvStorageMessaging, config.EnvStorageObservatory} {
		t.Setenv(v, "")
	}
	for _, v := range config.RemovedEnvNames() {
		t.Setenv(v, "")
	}
}

// The CLI must act on the plane the operator named.
//
// `ailang coordinator approve` opened a hardcoded local SQLite path. On a machine
// whose real coordinator is Firestore that is not a missing feature — it is the
// CLI confidently operating on the wrong store. Measured this session:
// `coordinator list` returned a stale local task from May while production held
// live work. An approve would have resolved nothing and reported success.

func TestRemoteCoordinatorSelected_ExplicitFlag(t *testing.T) {
	clearCoordinatorPlaneEnv(t)

	cases := []struct {
		name string
		args []string
		want bool
	}{
		{"space form", []string{"task-x", "--remote", "gcp"}, true},
		{"equals form", []string{"task-x", "--remote=gcp"}, true},
		{"explicit local", []string{"task-x", "--remote", "local"}, false},
		{"no flag", []string{"task-x"}, false},
		{"empty value", []string{"task-x", "--remote", ""}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := remoteCoordinatorSelected(tc.args); got != tc.want {
				t.Errorf("remoteCoordinatorSelected(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}

func TestRemoteCoordinatorSelected_Environment(t *testing.T) {
	clearCoordinatorPlaneEnv(t)
	t.Setenv("AILANG_STORAGE_COORDINATOR", "gcp")
	if !remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_STORAGE_COORDINATOR=gcp must select the remote plane")
	}
	if mode, src, err := coordinatorPlane(""); err != nil || mode != "gcp" || src != config.Source("AILANG_STORAGE_COORDINATOR") {
		t.Errorf("coordinatorPlane = %q via %q, %v; want gcp via AILANG_STORAGE_COORDINATOR", mode, src, err)
	}

	t.Setenv("AILANG_STORAGE_COORDINATOR", "")
	t.Setenv("AILANG_STORAGE", "gcp")
	if !remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_STORAGE=gcp must select the remote plane")
	}

	t.Setenv("AILANG_STORAGE", "local")
	if remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_STORAGE=local must stay on the local path")
	}
}

// AILANG_COORDINATOR_REMOTE was retired by M-V1-SIMPLIFY-S3 M3. It must not
// be silently ignored (that would route an approve to local SQLite on a
// machine whose coordinator is Firestore): the pre-parse check routes to the
// remote path and the open fails naming the replacement.
func TestRemoteCoordinatorSelected_RemovedSelectorIsAHardError(t *testing.T) {
	clearCoordinatorPlaneEnv(t)
	t.Setenv("AILANG_COORDINATOR_REMOTE", "gcp")
	if !remoteCoordinatorSelected([]string{"task-x"}) {
		t.Fatal("a retired selector must not fall through to the local path")
	}
	_, err := openCoordinatorStore(t.Context(), "", t.TempDir())
	if !errors.Is(err, config.ErrRemovedEnv) {
		t.Fatalf("err = %v, want config.ErrRemovedEnv", err)
	}
}

// The explicit flag must win over the environment, so an operator can act on a
// specific plane without unsetting whatever their shell exports.
func TestRemoteCoordinatorSelected_FlagBeatsEnvironment(t *testing.T) {
	clearCoordinatorPlaneEnv(t)
	t.Setenv("AILANG_STORAGE", "gcp")
	if remoteCoordinatorSelected([]string{"task-x", "--remote", "local"}) {
		t.Error("--remote local must override AILANG_STORAGE=gcp")
	}
}

// A local open must not depend on cloud configuration being present.
func TestOpenCoordinatorStore_LocalNeedsNoCloudConfig(t *testing.T) {
	clearCoordinatorPlaneEnv(t)
	// t.Setenv, not os.Unsetenv: the latter is never restored and leaks into
	// every test that runs afterwards in the same process.
	t.Setenv("AILANG_CLOUD_PROJECT", "")

	bundle, err := openCoordinatorStore(t.Context(), "", t.TempDir())
	if err != nil {
		t.Fatalf("local open failed: %v", err)
	}
	defer bundle.Close()

	if bundle.Store == nil {
		t.Fatal("local open returned no store")
	}
	// The mode string is printed before every mutation, so an operator can see
	// which plane they just changed. An empty one would defeat that.
	if bundle.Mode == "" {
		t.Error("mode must be reported: 'approved' against the wrong store looks exactly like success")
	}
}
