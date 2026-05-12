package effects

import "strings"

// isAbsoluteCrossPlatform reports whether a path string represents an absolute
// path on any mainstream host OS, not just the current one. This duplicates
// pkg.IsAbsoluteCrossPlatform — the effects package cannot import internal/pkg
// (would create an import cycle, see the AssetsDir comment in pkg.go).
//
// Keep this in sync with pkg.IsAbsoluteCrossPlatform.
//
// We deliberately do not call filepath.IsAbs — its behaviour is GOOS-dependent,
// which is wrong for arguments that arrive from .ail source code (those are
// host-independent string literals, not host paths).
func isAbsoluteCrossPlatform(path string) bool {
	if path == "" {
		return false
	}
	if path[0] == '/' || path[0] == '\\' {
		return true
	}
	if len(path) >= 2 && path[1] == ':' {
		c := path[0]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return true
		}
	}
	if strings.HasPrefix(path, "//") {
		return true
	}
	return false
}
