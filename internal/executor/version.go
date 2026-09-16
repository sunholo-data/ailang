package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/sunholo-data/ailang/internal/proctree"
)

// VersionProbeTimeout bounds the POST-RUN `<cli> --version` stamp (Identity).
// A binary that does not answer promptly is not a version we can bank, and it
// must never hold up the run it is stamping (the proctree kill-promptness
// tests measure exactly that; a fake CLI that backgrounds `sleep 60` hung the
// post-run probe for 60s). A deliberate HealthCheck passes its own, more
// generous deadline through ctx — Raw honours a caller deadline and applies
// this bound only when there is none.
const VersionProbeTimeout = 3 * time.Second

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
	failed  bool // Identity() gave up once; Raw() (HealthCheck) still retries
}

// Raw returns the trimmed first line of `--version`, probing on first use.
func (p *VersionProbe) Raw(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.version != "" {
		return p.version, nil
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, VersionProbeTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, p.Path, "--version")
	proctree.Configure(cmd)
	// Kill the whole tree on timeout: a probe that backgrounds a child would
	// otherwise keep the stdout pipe open and Output() would wait on it.
	cmd.WaitDelay = 500 * time.Millisecond
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
// A probe that fails once is remembered as failed for this executor, so a
// bad binary costs one bounded probe, not one per run.
func (p *VersionProbe) Identity(ctx context.Context) string {
	p.mu.Lock()
	failed := p.failed
	p.mu.Unlock()
	if failed {
		return ""
	}
	ctx, cancel := context.WithTimeout(ctx, VersionProbeTimeout)
	defer cancel()
	v, err := p.Raw(ctx)
	if err != nil {
		p.mu.Lock()
		p.failed = true
		p.mu.Unlock()
		return ""
	}
	return p.CLI + "@" + v
}
