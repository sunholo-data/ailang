//go:build linux

package effects

import "golang.org/x/sys/unix"

func terminalRetainSignals(fd int) error {
	state, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return err
	}
	state.Lflag |= unix.ISIG
	state.Cc[unix.VMIN] = 0
	state.Cc[unix.VTIME] = 0
	return unix.IoctlSetTermios(fd, unix.TCSETS, state)
}
