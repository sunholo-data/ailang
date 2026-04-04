package apiserver

import "strings"

// portableToolName creates a machine-portable MCP tool name from a module path
// and function name. Strips machine-specific prefixes (absolute paths, pkg/ dirs)
// to produce names like "docparse.services.parseCsv" instead of
// "Users.mark.dev.sunholo.ailang-parse.docparse.services.parseCsv".
func portableToolName(modPath, funcName string) string {
	// Clean the path and convert separators to dots.
	name := strings.ReplaceAll(modPath, "/", ".")

	// Strip leading dots (from leading slashes).
	name = strings.TrimLeft(name, ".")

	// If this looks like an absolute path (contains machine-specific segments),
	// find the package root. Heuristic: look for common package markers.
	// Packages loaded from pkg/ directories have paths like:
	//   pkg/sunholo/ailang-parse/docparse/services/samples
	// We want: docparse.services.samples
	if idx := strings.Index(name, "pkg."); idx >= 0 {
		// Skip "pkg.<org>.<repo>." prefix (3 segments after pkg.)
		after := name[idx+len("pkg."):]
		parts := strings.SplitN(after, ".", 3)
		if len(parts) >= 3 {
			// parts[0] = org, parts[1] = repo, parts[2] = rest
			name = parts[2]
		} else if len(parts) == 2 {
			name = parts[1]
		}
	} else if looksLikeAbsolutePath(name) {
		// Absolute path converted to dots, e.g.:
		//   "Users.mark.dev.sunholo.ailang-parse.docparse.services"
		// Heuristic: find the first segment containing a hyphen (likely a repo/project
		// name like "ailang-parse") and keep from there. If no hyphen segment, keep
		// the last 3 segments as a reasonable default.
		parts := strings.Split(name, ".")
		for i, p := range parts {
			if strings.Contains(p, "-") {
				name = strings.Join(parts[i:], ".")
				break
			}
		}
	}

	if name == "" {
		name = modPath
	}

	return name + "." + funcName
}

// looksLikeAbsolutePath returns true if the dot-separated name appears to
// originate from an absolute filesystem path (starts with Users, home, etc.).
func looksLikeAbsolutePath(name string) bool {
	first := name
	if idx := strings.Index(name, "."); idx > 0 {
		first = name[:idx]
	}
	lower := strings.ToLower(first)
	return lower == "users" || lower == "home" || lower == "tmp" ||
		lower == "var" || lower == "opt"
}
