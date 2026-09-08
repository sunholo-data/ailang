package main

import "testing"

// The CLI must act on the plane the operator named.
//
// `ailang coordinator approve` opened a hardcoded local SQLite path. On a machine
// whose real coordinator is Firestore that is not a missing feature — it is the
// CLI confidently operating on the wrong store. Measured this session:
// `coordinator list` returned a stale local task from May while production held
// live work. An approve would have resolved nothing and reported success.

func TestRemoteCoordinatorSelected_ExplicitFlag(t *testing.T) {
	t.Setenv("AILANG_COORDINATOR_REMOTE", "")
	t.Setenv("AILANG_STORAGE", "")

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
	t.Setenv("AILANG_STORAGE", "")
	t.Setenv("AILANG_COORDINATOR_REMOTE", "gcp")
	if !remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_COORDINATOR_REMOTE=gcp must select the remote plane")
	}

	t.Setenv("AILANG_COORDINATOR_REMOTE", "")
	t.Setenv("AILANG_STORAGE", "gcp")
	if !remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_STORAGE=gcp must select the remote plane")
	}

	t.Setenv("AILANG_STORAGE", "local")
	if remoteCoordinatorSelected([]string{"task-x"}) {
		t.Error("AILANG_STORAGE=local must stay on the local path")
	}
}

// The explicit flag must win over the environment, so an operator can act on a
// specific plane without unsetting whatever their shell exports.
func TestRemoteCoordinatorSelected_FlagBeatsEnvironment(t *testing.T) {
	t.Setenv("AILANG_STORAGE", "gcp")
	if remoteCoordinatorSelected([]string{"task-x", "--remote", "local"}) {
		t.Error("--remote local must override AILANG_STORAGE=gcp")
	}
}

// A local open must not depend on cloud configuration being present.
func TestOpenCoordinatorStore_LocalNeedsNoCloudConfig(t *testing.T) {
	t.Setenv("AILANG_COORDINATOR_REMOTE", "")
	t.Setenv("AILANG_STORAGE", "")
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
