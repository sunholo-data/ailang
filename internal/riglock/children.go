// children.go — the rig lock's record of the GPU work its holder spawned.
//
// Why this exists: the lock records the PID of the process that took it, and
// holderAlive lets the next job steal the lock once that PID is gone. That is
// right when the holder crashed with nothing left running, and wrong when it
// died leaving its agent subprocess behind. proctree installs context
// cancellation that kills the agent's process group, but cmd.Cancel only fires
// while the parent lives: a SIGKILLed or abruptly-terminated eval-suite never
// runs it, and macOS has no PDEATHSIG to make the child follow. The agent is
// reparented to launchd and keeps streaming against ollama, the lock reads
// free, and the next job is admitted alongside it — two tenants on a
// single-GPU rig, the exact thrash the lock exists to prevent.
//
// Observed 2026-09-22: two orphaned `opencode run` processes, 4h34m and 1h30m
// old, both PPID 1 and each holding an ESTABLISHED socket to 127.0.0.1:11434,
// while /Users/Shared/ailang/rig.lock.d did not exist at all. Both were the
// hung-stream failure described in the package doc, so the rig was carrying
// two dead-weight tenants at once.
//
// So a holder registers every GPU child it starts, and the steal path consults
// the record: a dead holder whose child still lives does NOT free the lock,
// and a steal that does go ahead reaps those orphans first.
package riglock

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/proctree"
)

// childrenFile is the append-only record inside the lock directory. It is
// removed with the directory on release, so it never outlives its lock.
const childrenFile = "children"

// childRef is one registered GPU child: its PID and the base name of the
// binary it was started from.
//
// The name is a PID-reuse guard, and it is the reason this file stores more
// than a bare integer. A recorded PID can be recycled by an unrelated process
// between registration and the next acquire; without the name we would either
// refuse to steal because of a stranger, or — far worse — SIGKILL that
// stranger's process group while reaping. Both checks below therefore confirm
// the live process still runs the binary we registered, and treat a mismatch
// as "this child is gone".
type childRef struct {
	pid  int
	name string
}

// RegisterChild records a GPU subprocess against the currently held rig lock.
//
// It is a no-op unless this process (or an ancestor) holds the lock, which
// makes it safe to call from any executor: eval-suite only takes the lock for
// runs that actually touch the local GPU, so a child spawned under a held lock
// is by definition rig work, and a child spawned without one has nothing to
// register against. Failures are silent — the record is a coordination aid,
// and a missing entry degrades to exactly the old behaviour.
//
// execPath is the binary the child was started from; its base name is stored
// alongside the PID to guard against PID reuse. Call after cmd.Start.
func RegisterChild(pid int, execPath string) {
	if pid <= 0 || !config.RigLockHeld() {
		return
	}
	dir := lockDir()
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, childrenFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = fmt.Fprintf(f, "%d %s\n", pid, filepath.Base(execPath))
}

// readChildren parses the children file. A missing or unreadable file means
// "no registered children", which restores the pre-existing steal semantics.
func readChildren(dir string) []childRef {
	f, err := os.Open(filepath.Join(dir, childrenFile))
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var refs []childRef
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) != 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid <= 0 {
			continue
		}
		refs = append(refs, childRef{pid: pid, name: fields[1]})
	}
	return refs
}

// commOf returns the current base command name for each pid that still exists,
// in one ps call. A pid absent from the result is not running. ps is already
// this repo's portable process probe (see eval_harness.ProcessGroupRSS); a ps
// failure returns nil, which callers read as "no live children" so an
// unreadable probe can never hold the rig hostage.
func commOf(pids []int) map[int]string {
	if len(pids) == 0 {
		return nil
	}
	args := make([]string, 0, len(pids)+2)
	args = append(args, "-o", "pid=,comm=", "-p")
	strs := make([]string, 0, len(pids))
	for _, p := range pids {
		strs = append(strs, strconv.Itoa(p))
	}
	args = append(args, strings.Join(strs, ","))

	out, err := exec.Command(psBinPath(), args...).Output()
	if err != nil {
		// ps exits non-zero when NO listed pid exists, which is the common
		// "all children finished" case, not an error worth reporting.
		return nil
	}
	res := make(map[int]string, len(pids))
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		res[pid] = filepath.Base(fields[1])
	}
	return res
}

// psBinPath returns ps at its fixed system location, falling back to PATH.
// Mirrors eval_harness.psBinPath: a fixed path keeps this working under
// launchd's restricted PATH and avoids PATH hijacking.
func psBinPath() string {
	for _, p := range []string{"/bin/ps", "/usr/bin/ps"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "ps"
}

// liveChildren returns the registered children that are still running the
// binary they were registered with. A PID that now belongs to something else
// has been recycled and is reported as gone.
func liveChildren(dir string) []childRef {
	refs := readChildren(dir)
	if len(refs) == 0 {
		return nil
	}
	pids := make([]int, 0, len(refs))
	for _, r := range refs {
		pids = append(pids, r.pid)
	}
	comms := commOf(pids)

	var live []childRef
	for _, r := range refs {
		if comm, ok := comms[r.pid]; ok && comm == r.name {
			live = append(live, r)
		}
	}
	return live
}

// reapChildren SIGKILLs the process group of every live registered child.
//
// Called only on the steal path and only when the holder is positively gone,
// so every process killed here is orphaned GPU work: nothing is left to
// enforce its timeout, sample its memory, cancel it, or record its result. It
// cannot finish usefully and it is occupying the rig, so the honest recovery
// is to end it rather than admit a second tenant beside it or wait out the
// staleness window. Each child was started under proctree.SetGroup and is its
// own group leader, so KillGroup also takes the descendants (opencode's
// server, its MCP children) that orphaned with it.
//
// It returns the children it killed so the caller can report them; a name
// mismatch from PID reuse has already excluded strangers.
func reapChildren(dir string) []childRef {
	live := liveChildren(dir)
	for _, c := range live {
		// Group first, then the process itself. KillGroup targets the group led
		// by pid, which exists only because the agent was started under
		// proctree.SetGroup — but it reports a MISSING group as success (ESRCH
		// is deliberately not an error there), so it cannot tell us whether
		// anything actually died, and an `if err != nil` fallback never fires.
		// Following it unconditionally with KillProcess makes the reap correct
		// for a child that is not a group leader, and is a harmless no-op when
		// the group kill already took it.
		_ = proctree.KillGroup(c.pid)
		_ = proctree.KillProcess(c.pid)
	}
	return live
}
