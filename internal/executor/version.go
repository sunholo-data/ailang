package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// VersionProbe captures what a CLI prints for `--version`, once, so every
// Result an executor returns can carry Result.ExecutorVersion.
//
// M-PI-HARNESS-UPGRADE M1. Every CLI executor's HealthCheck already ran
// `<cli> --version` and threw the output away; this is the shared capture so
// the next executor adopts it in two lines rather than a fourth copy. A failed
// probe is not cached — the coordinator builds executors before PATH is final.
type VersionProbe struct {
	CLI  string // identity prefix, e.g. "pi"
	Path string // binary to run

	mu      sync.Mutex
	version string
}

// Raw returns the trimmed first line of `--version`, probing on first use.
func (p *VersionProbe) Raw(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.version != "" {
		return p.version, nil
	}
	cmd := exec.CommandContext(ctx, p.Path, "--version")
	proctree.Configure(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("%s --version failed: %w", p.CLI, err)
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if line == "" {
		return "", fmt.Errorf("%s --version printed nothing", p.CLI)
	}
	p.version = line
	return line, nil
}

// Identity returns "<cli>@<version>", or "" when the probe fails. Empty is
// the documented UNMEASURED value of Result.ExecutorVersion — never a guess.
func (p *VersionProbe) Identity(ctx context.Context) string {
	v, err := p.Raw(ctx)
	if err != nil {
		return ""
	}
	return p.CLI + "@" + v
}
