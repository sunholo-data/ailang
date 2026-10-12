//go:build windows

package effects

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// terminalDeviceKey borrows the caller's HANDLE without creating an os.File,
// closing it, or installing a finalizer. Disk aliases share the identity supplied
// by Windows. Non-disk stdin retains handle identity: querying disk metadata on
// a console or pipe would reject ordinary line IO. Native sessions remain
// Unsupported on Windows, so these handles cannot acquire raw terminal mode.
func terminalDeviceKey(fd int) (string, error) {
	handle := windows.Handle(uintptr(fd))
	kind, err := windows.GetFileType(handle)
	if err != nil {
		return "", fmt.Errorf("query input handle type: %w", err)
	}
	switch kind {
	case windows.FILE_TYPE_DISK:
		var info windows.ByHandleFileInformation
		if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
			return "", fmt.Errorf("query input file identity: %w", err)
		}
		return fmt.Sprintf("windows:disk:%08x:%08x%08x", info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow), nil
	case windows.FILE_TYPE_CHAR, windows.FILE_TYPE_PIPE:
		return fmt.Sprintf("windows:handle:%d:%x", kind, uintptr(handle)), nil
	default:
		return "", fmt.Errorf("input handle type %d has no supported identity", kind)
	}
}
