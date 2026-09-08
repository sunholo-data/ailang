package pi

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/executor/proctree"
	"os"
	"os/exec"
)

// HealthCheck verifies the pi binary exists on PATH and responds.
func (e *PiExecutor) HealthCheck(ctx context.Context) error {
	piPath := e.piPath
	if _, err := exec.LookPath(piPath); err != nil {
		if _, statErr := os.Stat(piPath); statErr != nil {
			return fmt.Errorf("pi CLI not found: %w (install with: npm i -g @mariozechner/pi-coding-agent)", err)
		}
	}
	checkCmd := exec.CommandContext(ctx, piPath, "--version")
	proctree.Configure(checkCmd)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("pi --version failed: %w", err)
	}
	return nil
}
