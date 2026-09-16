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
	v, err := e.probe.Raw(ctx)
	if err != nil {
		return err
	}
	// M2: a mismatch is an ERROR naming both versions, not a warning. The
	// parser and the extension suite are written against ExpectedVersion; a
	// different pi keeps producing NDJSON that mostly parses, which is exactly
	// the silent-drift shape this sprint exists to remove.
	if v != ExpectedVersion {
		return fmt.Errorf("pi version mismatch: expected %s@%s, running pi@%s (repin ExpectedVersion in internal/executor/pi after re-verifying the wire, or install the pinned version: npm i -g %s@%s)",
			ExpectedPackage, ExpectedVersion, v, ExpectedPackage, ExpectedVersion)
	}
	return nil
}

// Version returns the harness identity as "pi@<version>", where <version> is
// what the binary printed for --version. Empty when the probe fails: absent
// means UNMEASURED, never a guess.
func (e *PiExecutor) Version(ctx context.Context) string {
	return e.probe.Identity(ctx)
}
