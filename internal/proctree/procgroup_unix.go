//go:build unix

package proctree

import (
	"errors"
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func killProcessGroup(pid int) error {
	return killProcessGroupWith(pid, syscall.Kill)
}

func killProcessGroupWith(pid int, kill func(int, syscall.Signal) error) error {
	err := kill(-pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	// Darwin can find a group while its last member disappears, then return
	// EPERM because there is nobody left to signal. Confirm absence without
	// sending another termination signal; retain every real permission error.
	return resolveGroupErrorWith(pid, err, kill)
}

func killProcess(pid int) error {
	err := syscall.Kill(pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

func resolveGroupError(pid int, err error) error {
	return resolveGroupErrorWith(pid, err, syscall.Kill)
}

func resolveGroupErrorWith(pid int, err error, probe func(int, syscall.Signal) error) error {
	// Do not clear wrapped/joined errors that might also carry I/O failures.
	if err == syscall.EPERM && probe(-pid, 0) == syscall.ESRCH {
		return nil
	}
	return err
}
