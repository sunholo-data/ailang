package main

import (
	"fmt"
	"runtime"

	"github.com/sunholo-data/ailang/internal/config"
)

// runDoctorMemory prints what `ailang run` / `serve-api` would resolve for the
// process memory controls, and where each came from (M-V1-MEMORY-FOOTPRINT M4).
// Nothing is applied.
func runDoctorMemory() {
	fmt.Println("Memory controls")
	limit, source, err := resolveMemoryLimit("")
	switch {
	case err != nil:
		fmt.Printf("  soft limit:   ERROR %v\n", err)
	case limit > 0:
		fmt.Printf("  soft limit:   %d MB (%s)\n", limit>>20, source)
	default:
		fmt.Printf("  soft limit:   none (%s) — pass --max-memory <size|cgroup> or set %s\n", source, config.EnvMemLimit)
	}
	if v := config.GOMEMLIMIT(); v != "" {
		fmt.Printf("  GOMEMLIMIT:   %s (Go applies it itself)\n", v)
	}
	if config.GOGCSet() {
		fmt.Printf("  GOGC:         %s (operator-set)\n", config.GOGC())
	} else {
		fmt.Println("  GOGC:         unset — run/exec use 500; serve-api uses Go's default 100")
	}
	if runtime.GOOS == "linux" {
		raw, src, cgErr := cgroupMemoryLimitFrom("/sys/fs/cgroup")
		switch {
		case cgErr != nil:
			fmt.Printf("  cgroup:       ERROR %v\n", cgErr)
		case raw > 0:
			fmt.Printf("  cgroup:       %d MB (%s); --max-memory cgroup would apply %d MB\n", raw>>20, src, int64(float64(raw)*cgroupFraction)>>20)
		default:
			fmt.Printf("  cgroup:       %s\n", src)
		}
	} else {
		fmt.Printf("  cgroup:       not linux (%s); --max-memory cgroup applies nothing here\n", runtime.GOOS)
	}
	fmt.Println()
	fmt.Println("Other caps: --fs-max-bytes / AILANG_FS_MAX_BYTES (FS reads), Net 5 MB body cap,")
	fmt.Println("  trace values 1 KB each / 32 MB retained, AILANG_EVAL_MAX_RSS (eval harness only).")
	fmt.Println("Probe: /usr/bin/time -l ailang run ... (macOS) or -v (Linux) — see docs/docs/guides/debugging.md#memory")
}
