//go:build windows

package main

import "os/exec"

// ownGroup on Windows kills only the compiler on timeout (no process groups
// here); the bounded WaitDelay still closes pipes a grandchild holds open.
// The gate is run on macOS and Linux; this keeps the package building.
func ownGroup(cmd *exec.Cmd) {}

func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
