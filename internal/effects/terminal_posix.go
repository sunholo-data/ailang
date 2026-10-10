//go:build darwin || linux

package effects

import (
	"fmt"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

type posixTerminal struct {
	input, output int
	state         *term.State
}

func terminalSupported() bool                    { return true }
func terminalIsTTY(fd int) bool                  { return term.IsTerminal(fd) }
func terminalQuerySize(fd int) (int, int, error) { return term.GetSize(fd) }
func terminalDeviceKey(fd int) (string, error) {
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%d", stat.Dev, stat.Ino, stat.Rdev), nil
}
func terminalActivate(input, output int) (terminalDevice, error) {
	old, err := term.MakeRaw(input)
	if err != nil {
		return nil, err
	}
	if err = terminalRetainSignals(input); err != nil {
		restoreErr := term.Restore(input, old)
		if restoreErr != nil {
			return nil, fmt.Errorf("activation: %w; restore: %v", err, restoreErr)
		}
		return nil, err
	}
	return &posixTerminal{input: input, output: output, state: old}, nil
}
func (p *posixTerminal) size() (int, int, error)       { return terminalQuerySize(p.output) }
func (p *posixTerminal) restore() error                { return term.Restore(p.input, p.state) }
func (p *posixTerminal) read(data []byte) (int, error) { return unix.Read(p.input, data) }
func (p *posixTerminal) poll(timeout int) (bool, error) {
	fds := []unix.PollFd{{Fd: int32(p.input), Events: unix.POLLIN}}
	_, err := unix.Poll(fds, timeout)
	if err == unix.EINTR {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if fds[0].Revents&unix.POLLNVAL != 0 {
		return false, fmt.Errorf("terminal descriptor is invalid")
	}
	return fds[0].Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) != 0, nil
}
