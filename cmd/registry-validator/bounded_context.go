package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// checkCapSource is set together with checkRSSCap at startup. It includes the
// measured instance limit, or explicitly identifies the fallback.
var checkCapSource = "instance limit unavailable; fallback"

func envValue(env []string, key string) string {
	value := "unavailable"
	for _, entry := range env {
		if strings.HasPrefix(entry, key+"=") {
			value = strings.TrimPrefix(entry, key+"=")
		}
	}
	return value
}

func refusalContext(cap uint64, env []string, stats checkStats) string {
	if env == nil {
		env = os.Environ()
	}
	return fmt.Sprintf("\nEnvironment: cap=%s; source=%s; platform=%s/%s; GOMAXPROCS=%s; GOMEMLIMIT=%s; sampling=%s; sampled_peak=%d bytes; wall=%s.\n%s",
		mib(cap), checkCapSource, runtime.GOOS, runtime.GOARCH, envValue(env, "GOMAXPROCS"), envValue(env, "GOMEMLIMIT"), rssPollEvery, stats.PeakRSS, stats.Wall, reproduction(envValue(env, "GOMEMLIMIT")))
}

func reproduction(memory string) string {
	image := os.Getenv("VALIDATOR_IMAGE")
	if image == "" {
		return "Reproduction: image identity unavailable (VALIDATOR_IMAGE unset); ask the operator for the deployed image. See package-publishing.md#budgets."
	}
	// Quote operator-provided values as POSIX shell literals.
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
	return "Reproduce from the extracted published tarball directory (disposable writable staging, fresh dependency cache):\n" +
		"docker run --rm --platform linux/amd64 --cpus 2 --memory 1g -e AILANG_NO_CACHE=1 -e GOMAXPROCS=2 -e " + quote("GOMEMLIMIT="+memory) + " -v \"$PWD\":/input:ro " + quote(image) +
		" sh -ec 'stage=$(mktemp -d); export HOME=$(mktemp -d); cp -a /input/. \"$stage/\"; cd \"$stage\"; rm -f ailang.lock; ailang lock; ailang check --package .'\n" +
		"Docker does not reproduce gVisor. Use registry-version dependencies as uploaded by current publishers; legacy path dependencies require validator rewriting first. GOMEMLIMIT is a soft Go runtime target; sampled RSS can miss spikes."
}
