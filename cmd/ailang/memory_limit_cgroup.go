package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
)

// Memory limit resolution (M-V1-MEMORY-FOOTPRINT M4, D-D).
//
// `--max-memory <size>` sets Go's soft memory limit as before. The literal
// `--max-memory cgroup` (or AILANG_MEMLIMIT with the same values) derives it
// from the container's cgroup limit x cgroupFraction. It is OPT-IN: nothing is
// applied unless asked, so an operator who tuned GOGC and expects no limit
// gets none (the quorum's A4 objection to an opt-out default).
//
// What the limit is: best-effort GC tuning. Go collects harder as total
// runtime memory nears it, and Go's default GOGC=100 (serve-api) or the CLI's
// 500 still sets the trigger below it. It can be exceeded by reachable data,
// it is not a per-request bound, and a true overrun still reaches the kernel
// OOM killer — later, and after M1-M3 have removed the copies that used to
// get there first.

// cgroupFraction is the share of the cgroup limit handed to Go: the rest is
// non-heap memory (goroutine stacks, cgo, page cache the kernel charges).
const cgroupFraction = 0.9

// cgroupRootForTest overrides the cgroup mount for tests.
var cgroupRootForTest string

// resolveMemoryLimit turns the --max-memory text (flag beats AILANG_MEMLIMIT)
// into a byte limit and a human source. 0 means none. A malformed value is
// an error: a safety cap never falls back.
func resolveMemoryLimit(flag string) (limit int64, source string, err error) {
	spec := strings.TrimSpace(flag)
	from := "--max-memory"
	if spec == "" {
		spec = strings.TrimSpace(config.MemLimit())
		from = config.EnvMemLimit
	}
	if spec == "" {
		return 0, "none", nil
	}
	if strings.EqualFold(spec, "cgroup") {
		if runtime.GOOS != "linux" && cgroupRootForTest == "" {
			return 0, fmt.Sprintf("%s=cgroup: no cgroups on %s, no limit", from, runtime.GOOS), nil
		}
		root := cgroupRootForTest
		if root == "" {
			root = "/sys/fs/cgroup"
		}
		raw, src, err := cgroupMemoryLimitFrom(root)
		if err != nil {
			return 0, "", fmt.Errorf("%s=cgroup: %w", from, err)
		}
		if raw == 0 {
			return 0, fmt.Sprintf("%s=cgroup: %s", from, src), nil
		}
		return int64(float64(raw) * cgroupFraction), fmt.Sprintf("%s=cgroup: %s x %.2f", from, src, cgroupFraction), nil
	}
	n, err := config.ParseByteSize(spec)
	if err != nil {
		return 0, "", fmt.Errorf("%s: %w (a size such as 256MB or 1GB, or the literal cgroup)", from, err)
	}
	if n <= 0 {
		return 0, "", fmt.Errorf("%s must be positive, got %d", from, n)
	}
	return n, from, nil
}

// cgroupMemoryLimitFrom reads the memory limit under a cgroup mount: v2
// memory.max first, then v1 memory/memory.limit_in_bytes. "max", the v1
// "unlimited" sentinel and a missing file all mean "no limit" — reported, not
// errored, because a container without a limit is a normal state.
func cgroupMemoryLimitFrom(root string) (limit int64, source string, err error) {
	for _, rel := range []string{"memory.max", filepath.Join("memory", "memory.limit_in_bytes")} {
		p := filepath.Join(root, rel)
		b, readErr := os.ReadFile(p)
		if errors.Is(readErr, os.ErrNotExist) {
			continue
		}
		if readErr != nil {
			return 0, "", fmt.Errorf("reading %s: %w", p, readErr)
		}
		s := strings.TrimSpace(string(b))
		if s == "max" {
			return 0, rel + " is max (no limit)", nil
		}
		n, perr := strconv.ParseInt(s, 10, 64)
		if perr != nil {
			return 0, "", fmt.Errorf("%s: unparseable %q", p, s)
		}
		// cgroup v1 reports "unlimited" as a page-rounded int64 max.
		if n <= 0 || n >= 1<<62 {
			return 0, rel + " is unlimited (no limit)", nil
		}
		return n, rel, nil
	}
	return 0, "no cgroup memory file under " + root + " (no limit)", nil
}

// applyResolvedMemoryLimit resolves and applies the limit for a command, and
// says what it did on stderr unless quiet. Returns the resolved limit so a
// caller can report it.
func applyResolvedMemoryLimit(flag string, quiet bool) (int64, error) {
	limit, source, err := resolveMemoryLimit(flag)
	if err != nil {
		return 0, err
	}
	if limit > 0 {
		debug.SetMemoryLimit(limit)
		if !quiet {
			fmt.Fprintf(os.Stderr, "memory limit: %d MB (%s)\n", limit>>20, source)
		}
	} else if !quiet && source != "none" {
		fmt.Fprintf(os.Stderr, "memory limit: none (%s)\n", source)
	}
	return limit, nil
}
