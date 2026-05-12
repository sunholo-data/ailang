package pkg

import "strings"

// IsAbsoluteCrossPlatform reports whether a path string represents an absolute
// path on any mainstream host OS, not just the current one. Go's filepath.IsAbs
// asks "is this absolute on the host running this binary?" — which is the wrong
// question for validating cross-platform artifacts like published package
// manifests. A manifest entry like "/etc/passwd" must be rejected whether the
// validator runs on Linux, macOS, or Windows.
//
// Detected forms:
//   - Unix absolute:    starts with "/"
//   - Windows drive:    "C:\\foo" or "C:/foo" (drive letter + colon + sep)
//   - Windows UNC:      "\\\\server\\share" or "//server/share"
//
// We deliberately do not call filepath.IsAbs — its behaviour depends on GOOS.
func IsAbsoluteCrossPlatform(path string) bool {
	if path == "" {
		return false
	}
	// Unix absolute or root-relative.
	if path[0] == '/' || path[0] == '\\' {
		return true
	}
	// Windows drive letter: X: or X:/ or X:\
	if len(path) >= 2 && path[1] == ':' {
		c := path[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	// UNC paths normalized to forward slashes.
	if strings.HasPrefix(path, "//") {
		return true
	}
	return false
}
