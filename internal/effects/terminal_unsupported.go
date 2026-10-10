//go:build !darwin && !linux

package effects

import "fmt"

func terminalSupported() bool { return false }
func terminalIsTTY(int) bool  { return false }
func terminalQuerySize(int) (int, int, error) {
	return 0, 0, fmt.Errorf("native terminals unsupported")
}
func terminalDeviceKey(fd int) (string, error) { return fmt.Sprintf("fd:%d", fd), nil }
func terminalActivate(int, int) (terminalDevice, error) {
	return nil, fmt.Errorf("native terminals unsupported")
}
