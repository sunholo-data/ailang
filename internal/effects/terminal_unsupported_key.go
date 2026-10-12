//go:build !darwin && !linux && !windows

package effects

import "fmt"

func terminalDeviceKey(fd int) (string, error) { return fmt.Sprintf("fd:%d", fd), nil }
