//go:build unix

package proctree

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ReapWorkspace SIGKILLs every process whose working directory is dir or below it, together
// with its process group, and returns the pids it found. The calling process, its ancestors
// and its own process group are never touched.
//
// Why a group kill of the agent is not enough (measured 2026-10-02 on the rig): pi's bash
// tool spawns every command DETACHED, in a process group of its own, so killing the agent's
// group never reaches them. Two `find /` shells from a qwen3.8 gauntlet run reparented to
// launchd and ran for 7h and 33h after their run had ended, in disk-wait, stalling git on
// the whole machine. Their cwd was the run's workspace, which is the one thing every such
// straggler has in common, whichever harness started it.
func ReapWorkspace(dir string) ([]int, error) {
	if dir == "" {
		return nil, nil
	}
	root, err := canonicalDir(dir)
	if err != nil {
		return nil, err
	}
	// A workspace of "/", a top-level dir (/tmp is /private/tmp on macOS) or a bare home
	// would match half the machine. Real workspaces are per-run temp dirs, several levels deep.
	if strings.Count(strings.Trim(root, string(filepath.Separator)), string(filepath.Separator)) < 2 || root == canonicalHome() {
		return nil, errors.New("proctree: refusing to reap a root or home directory")
	}
	cwds, err := processCwds()
	if err != nil {
		return nil, err
	}
	protected := protectedPids()
	ownGroup, _ := syscall.Getpgid(os.Getpid())
	var found []int
	for pid, cwd := range cwds {
		if protected[pid] {
			continue
		}
		c := cwd
		if resolved, err := filepath.EvalSymlinks(cwd); err == nil {
			c = resolved
		}
		if c != root && !strings.HasPrefix(c, root+string(filepath.Separator)) {
			continue
		}
		found = append(found, pid)
		if pgid, err := syscall.Getpgid(pid); err == nil && pgid > 1 && pgid != ownGroup && !protected[pgid] {
			_ = killProcessGroup(pgid)
		}
		_ = killProcess(pid)
	}
	return found, nil
}

// canonicalDir resolves symlinks so /var/folders/... (what the harness creates) matches
// /private/var/folders/... (what the kernel reports as a cwd on macOS).
func canonicalDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved, nil
	}
	// The workspace may already be deleted; its stragglers still report the old path.
	if parent, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		return filepath.Join(parent, filepath.Base(abs)), nil
	}
	return abs, nil
}

func canonicalHome() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(h); err == nil {
		return r
	}
	return h
}

// protectedPids is this process and every ancestor up to init.
func protectedPids() map[int]bool {
	p := map[int]bool{0: true, 1: true}
	pid := os.Getpid()
	for i := 0; i < 64 && pid > 1; i++ {
		p[pid] = true
		next, err := parentPid(pid)
		if err != nil || next == pid {
			break
		}
		pid = next
	}
	return p
}

func parentPid(pid int) (int, error) {
	out, err := exec.Command(psPath(), "-o", "ppid=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

func psPath() string {
	for _, p := range []string{"/bin/ps", "/usr/bin/ps"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "ps"
}

// processCwds maps pid -> cwd for every process this user can inspect.
func processCwds() (map[int]string, error) {
	if runtime.GOOS == "linux" {
		return procCwds()
	}
	return lsofCwds()
}

func procCwds() (map[int]string, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	out := map[int]string{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if cwd, err := os.Readlink(filepath.Join("/proc", e.Name(), "cwd")); err == nil {
			out[pid] = cwd
		}
	}
	return out, nil
}

// lsofCwds reads cwds through lsof (macOS has no /proc). Bounded: a wedged lsof must not
// turn cleanup into its own straggler.
func lsofCwds() (map[int]string, error) {
	lsof := "/usr/sbin/lsof"
	if _, err := os.Stat(lsof); err != nil {
		lsof = "lsof"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, lsof, "-n", "-w", "-a", "-d", "cwd", "-F", "pn")
	out, err := cmd.Output()
	// lsof exits 1 when some processes could not be read; the rows it did print are good.
	if err != nil && len(out) == 0 {
		return nil, err
	}
	cwds := map[int]string{}
	pid := 0
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
		case 'n':
			if pid > 0 {
				cwds[pid] = line[1:]
			}
		}
	}
	return cwds, nil
}
