//go:build !windows

// process_unix.go — the eval memory watchdog's process-group RSS sampler.
// Process-group creation and kill live in internal/proctree (the one recipe for
// every subprocess AILANG spawns); this file only measures a group proctree
// created. M-V1-SIMPLIFY-S4 M3A.

package eval_harness

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// psBinPath returns the ps binary at its fixed system location (/bin/ps on
// macOS, /usr/bin/ps on most Linux). A fixed path keeps the watchdog working
// under restricted-PATH contexts (launchd) and avoids PATH-hijack concerns
// (go:S4036) for a safety-critical sampler. Falls back to PATH lookup only if
// neither canonical location exists.
func psBinPath() string {
	for _, p := range []string{"/bin/ps", "/usr/bin/ps"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "ps"
}

// ProcessGroupRSS returns the total resident set size (bytes) of every process
// in pid's process group. pid must be a group leader started via
// proctree.SetGroup (Setpgid makes pgid == pid). Sampling shells out to
// `ps -axo pgid=,rss=` — portable across macOS and Linux, and the reliable
// option on macOS where RLIMIT_AS is not dependably enforced. ok=false means
// the sample failed (ps error, or the group has no live members).
func ProcessGroupRSS(pid int) (int64, bool) {
	out, err := exec.Command(psBinPath(), "-axo", "pgid=,rss=").Output()
	if err != nil {
		return 0, false
	}
	var total int64
	found := false
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		pgid, err := strconv.Atoi(fields[0])
		if err != nil || pgid != pid {
			continue
		}
		rssKB, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		total += rssKB * 1024 // ps reports rss in 1024-byte units
		found = true
	}
	return total, found
}
