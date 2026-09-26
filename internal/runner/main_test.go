package runner

import (
	"os"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// Every run path configures capabilities through GrantCapabilities, which
// resolves the storage plane for the secret approver. A developer's shell
// may still export a retired selector (AILANG_MESSAGES_STORE — CLAUDE.md
// told every session to, until M-V1-SIMPLIFY-S3 M3), which is a hard error
// by design and must not leak into tests that are about something else.
func TestMain(m *testing.M) {
	for _, v := range config.RemovedEnvNames() {
		_ = os.Unsetenv(v)
	}
	for _, v := range []string{config.EnvStorage, config.EnvStorageMessaging, config.EnvStorageCoordinator, config.EnvStorageObservatory} {
		_ = os.Unsetenv(v)
	}
	os.Exit(m.Run())
}
