//go:build !unix && !windows

package proctree

import "os/exec"

// Targets with neither Unix process groups nor Windows Job Objects — js/wasm
// (the browser REPL build, cmd/wasm) and wasip1 chief among them. Neither
// supports os/exec subprocesses at all, so Configure/Kill are unreachable
// here; these exist only so the package still builds for GOOS values with no
// process-group concept, the way procgroup_windows.go does for Windows.
func setProcessGroup(_ *exec.Cmd) {}

func killProcessGroup(_ int) error { return nil }

func killProcess(_ int) error { return nil }
