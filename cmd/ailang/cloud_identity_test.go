package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// noCloudIdentity isolates a test from every source internal/config reads for
// the cloud project and region: both project variables, both region
// variables, the user's config file (via a fixture home, and no AILANG_CONFIG)
// and the GCE metadata server. Any test that asserts "no project resolves"
// must call it — with a real home the machine's own ~/.ailang/config.yaml
// pubsub.project_id resolves and the assertion fails for the wrong reason.
func noCloudIdentity(t *testing.T) {
	t.Helper()
	testutil.SetHomeDir(t, t.TempDir())
	for _, v := range []string{"AILANG_CLOUD_PROJECT", "GOOGLE_CLOUD_PROJECT", "AILANG_CLOUD_REGION",
		"GOOGLE_CLOUD_REGION", "AILANG_MESSAGES_PROJECT", "AILANG_COORDINATOR_SERVICE", "AILANG_CONFIG",
		"AILANG_STRICT_CONFIG"} {
		t.Setenv(v, "")
	}
	t.Setenv("AILANG_NO_METADATA", "1")
}
