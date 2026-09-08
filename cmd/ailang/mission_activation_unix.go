//go:build darwin || linux

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

func activationProcessAlive(pid int) (bool, error) {
	err := unix.Kill(pid, 0)
	if errors.Is(err, unix.ESRCH) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
func activationLegacyIdle(ctx context.Context, home string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("local Docs scheduler verification currently requires macOS launchd")
	}
	bounded, cancel := context.WithTimeout(ctx, activationCheckTimeout)
	defer cancel()
	out, err := exec.CommandContext(bounded, "launchctl", "print", fmt.Sprintf("gui/%d/dev.ailang.mission-docs", os.Getuid())).Output()
	if err != nil {
		return fmt.Errorf("cannot establish Docs launchd ownership: %w", err)
	}
	idle := false
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "pid =") {
			return errors.New("Docs launchd job has a live PID")
		}
		if line == "state = waiting" || line == "state = not running" {
			idle = true
		}
		if line == "state = running" {
			return errors.New("Docs launchd job is running")
		}
	}
	if !idle {
		return errors.New("Docs launchd state is unknown; inspect before activation")
	}
	b, err := os.ReadFile(filepath.Join(home, ".ailang", "state", "mission-docs.pid"))
	if err == nil {
		pid, e := strconv.Atoi(strings.TrimSpace(string(b)))
		if e != nil || pid < 1 {
			return errors.New("Docs pidfile has unknown process identity")
		}
		alive, e := activationProcessAlive(pid)
		if e != nil {
			return e
		}
		if alive {
			return errors.New("Docs pidfile process is alive")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	// The driver writes its PID late; catch a manually launched preflight or
	// recovery driver as well. Failure to obtain the inventory is not idle.
	out, err = exec.CommandContext(bounded, "ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return fmt.Errorf("cannot inventory Docs driver processes: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if (strings.Contains(line, "mission-control.sh") || strings.Contains(line, "mission-recovery.sh")) && (strings.Contains(line, " docs") || strings.Contains(line, "mission-docs")) {
			return errors.New("Docs legacy driver process is present")
		}
	}
	return nil
}

func activationSessionStopped(ctx context.Context, session int) error {
	if session < 1 {
		return errors.New("unknown owned session ID")
	}
	bounded, cancel := context.WithTimeout(ctx, activationCheckTimeout)
	defer cancel()
	out, err := exec.CommandContext(bounded, "ps", "-axo", "pid=").Output()
	if err != nil {
		return fmt.Errorf("cleanup_pending: cannot inventory owned session: %w", err)
	}
	pids := strings.Fields(string(out))
	if len(pids) == 0 {
		return errors.New("cleanup_pending: empty process inventory is not stop proof")
	}
	for _, value := range pids {
		pid, e := strconv.Atoi(value)
		if e != nil || pid < 1 {
			return errors.New("cleanup_pending: malformed process inventory")
		}
		sid, e := unix.Getsid(pid)
		if errors.Is(e, unix.ESRCH) {
			continue
		}
		if e != nil {
			return fmt.Errorf("cleanup_pending: cannot establish session for PID %d: %w", pid, e)
		}
		if sid == session {
			return fmt.Errorf("cleanup_pending: owned session %d still contains PID %d", session, pid)
		}
	}
	return nil
}
func superviseActivationChild(ctx context.Context, dir string, p *activationProcess, out io.Writer) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(executable, "mission", "activation", "child", "docs", "--operation", p.OperationID)
	return superviseActivationCommand(ctx, dir, p, out, cmd)
}
func superviseActivationCommand(ctx context.Context, dir string, p *activationProcess, out io.Writer, cmd *exec.Cmd) error {
	read, write, err := os.Pipe()
	if err != nil {
		return err
	}
	defer read.Close()
	defer write.Close()
	// A session includes provider-created process groups, unlike a group-only
	// check. Current executor adapters do not detach sessions.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.WaitDelay = activationCheckTimeout
	cmd.Stdin = read
	cmd.Stdout = out
	cmd.Stderr = out
	if err = cmd.Start(); err != nil {
		return err
	}
	_ = read.Close()
	p.SessionID = cmd.Process.Pid
	p.Phase = "armed"
	if err = saveActivationProcess(dir, *p); err != nil {
		_ = write.Close()
		_ = cmd.Wait()
		return err
	}
	if _, err = write.Write([]byte{'R'}); err != nil {
		_ = write.Close()
		_ = cmd.Wait()
		return err
	}
	_ = write.Close()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = cmd.Process.Signal(os.Interrupt)
		select {
		case err = <-done:
		case <-time.After(activationCheckTimeout):
			// Kill only our direct group; escaped provider groups keep cleanup held.
			_ = unix.Kill(-p.SessionID, unix.SIGKILL)
			err = <-done
		}
	}
	latest, readErr := readActivationProcess(dir, p.OperationID)
	if readErr != nil {
		return errors.Join(err, readErr)
	}
	p.NoDispatch = latest.NoDispatch
	p.Phase = "exited"
	return errors.Join(err, saveActivationProcess(dir, *p))
}
