//go:build !windows

package eval_harness

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// SetProcessGroup configures the command to run in its own process group (Unix only)
func SetProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// KillProcessGroup kills the entire process group (Unix only)
// Uses negative PID to kill all processes in the group
func KillProcessGroup(pid int) error {
	return syscall.Kill(-pid, syscall.SIGKILL)
}

// KillProcess kills a single process
func KillProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

// ProcessGroupRSS returns the total resident set size (bytes) of every process
// in pid's process group. pid must be a group leader started via
// SetProcessGroup (Setpgid makes pgid == pid). Sampling shells out to
// `ps -axo pgid=,rss=` — portable across macOS and Linux, and the reliable
// option on macOS where RLIMIT_AS is not dependably enforced. ok=false means
// the sample failed (ps error, or the group has no live members).
func ProcessGroupRSS(pid int) (int64, bool) {
	out, err := exec.Command("ps", "-axo", "pgid=,rss=").Output()
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
