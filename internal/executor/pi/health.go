package pi

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// HealthCheck verifies the pi binary exists on PATH and responds to --version.
// The version it reports is captured (see Version), not discarded.
func (e *PiExecutor) HealthCheck(ctx context.Context) error {
	if _, err := exec.LookPath(e.piPath); err != nil {
		if _, statErr := os.Stat(e.piPath); statErr != nil {
			return fmt.Errorf("pi CLI not found: %w (install with: npm i -g %s@%s)", err, ExpectedPackage, ExpectedVersion)
		}
	}
	_, err := e.probe.Raw(ctx)
	return err
}

// Version returns the harness identity as "pi@<version>", where <version> is
// what the binary printed for --version. Empty when the probe fails: absent
// means UNMEASURED, never a guess.
func (e *PiExecutor) Version(ctx context.Context) string {
	return e.probe.Identity(ctx)
}
