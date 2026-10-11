//go:build unix

package runlife

import "syscall"

func closeOnExec(fd int) { syscall.CloseOnExec(fd) }
