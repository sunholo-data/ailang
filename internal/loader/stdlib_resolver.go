package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// validateModuleName validates a stdlib module name for security
// Prevents directory traversal and other attacks
func validateModuleName(name string) error {
	// Remove std/ prefix if present
	name = strings.TrimPrefix(name, "std/")

	// Check for empty name
	if name == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	// Check for directory traversal attempts
	if strings.Contains(name, "..") {
		return fmt.Errorf("module name cannot contain '..': %s", name)
	}

	// Check for null bytes (security)
	if strings.Contains(name, "\x00") {
		return fmt.Errorf("module name cannot contain null bytes: %s", name)
	}

	// Allow only alphanumeric, underscore, hyphen, forward slash
	// This prevents shell injection and other attacks
	// Note: Checked BEFORE filepath.IsAbs() to ensure consistent error messages across platforms
	// (Windows drive letters like "c:" and UNC paths contain invalid chars, not just absolute paths)
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_/-]+$`)
	if !validPattern.MatchString(name) {
		return fmt.Errorf("module name contains invalid characters (only [a-zA-Z0-9_/-] allowed): %s", name)
	}

	// Check for absolute paths BEFORE suspicious patterns
	// This ensures consistent error messages (absolute paths are rejected uniformly)
	// Note: On Unix, /etc/passwd is absolute; on Windows it's not (hence falls to suspicious check)
	if filepath.IsAbs(name) {
		return fmt.Errorf("module name cannot be an absolute path: %s", name)
	}

	// Check for suspicious patterns (after IsAbs, to catch platform-specific edge cases)
	// On Windows: /etc/passwd isn't absolute, so caught here
	// On Unix: /etc/passwd caught by IsAbs above
	suspicious := []string{
		"/etc/", "/usr/", "/var/", "/sys/", "/proc/", // Unix system dirs
		"c:", "C:", "d:", "D:", // Windows drive letters (caught by regex, but defense in depth)
		"\\\\", // UNC paths (caught by regex, but defense in depth)
	}
	lowerName := strings.ToLower(name)
	for _, pattern := range suspicious {
		if strings.Contains(lowerName, pattern) {
			return fmt.Errorf("module name contains suspicious pattern: %s", name)
		}
	}

	return nil
}

// getUserDataDir returns the platform-specific user data directory for AILANG stdlib
// Returns empty string if unable to determine (caller should skip this path)
func getUserDataDir() string {
	var baseDir string

	switch runtime.GOOS {
	case "linux", "freebsd", "openbsd", "netbsd":
		// Linux/BSD: Use XDG_DATA_HOME or ~/.local/share
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			baseDir = xdg
		} else if home := os.Getenv("HOME"); home != "" {
			baseDir = filepath.Join(home, ".local", "share")
		}

	case "darwin":
		// macOS: Use ~/Library/Application Support
		if home := os.Getenv("HOME"); home != "" {
			baseDir = filepath.Join(home, "Library", "Application Support")
		}

	case "windows":
		// Windows: Use %APPDATA%
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			baseDir = appdata
		}

	default:
		// Unknown OS - return empty
		return ""
	}

	if baseDir == "" {
		return ""
	}

	// Append ailang/std to base directory
	return filepath.Join(baseDir, "ailang", "std")
}

// getPathSeparator returns the path separator for environment variables
// Windows uses semicolon, Unix uses colon
func getPathSeparator() string {
	if runtime.GOOS == "windows" {
		return ";"
	}
	return ":"
}

// StdlibResolver resolves stdlib module paths using a search path strategy
type StdlibResolver struct {
	// Search paths (computed once and cached)
	searchPaths []string

	// Negative cache: module name → paths tried
	// Avoids repeated filesystem hits for missing modules
	negativeCache map[string][]string

	// CLI override path (highest priority)
	cliOverridePath string

	// Enable trace logging (for --trace-loader flag)
	traceEnabled bool

	// Strict mode (fail on version mismatch)
	strictMode bool

	// Expected stdlib version (embedded at compile time)
	expectedVersion string
}

// NewStdlibResolver creates a new stdlib resolver
func NewStdlibResolver(cliPath string, traceEnabled, strictMode bool) *StdlibResolver {
	return &StdlibResolver{
		negativeCache:   make(map[string][]string),
		cliOverridePath: cliPath,
		traceEnabled:    traceEnabled,
		strictMode:      strictMode,
		expectedVersion: "v0.4.4", // TODO: Embed from build flags
	}
}

// ResolveStdlib resolves a stdlib module name to an absolute file path
// Returns the resolved path or an error with search trace
func (r *StdlibResolver) ResolveStdlib(moduleName string) (string, error) {
	// Validate module name for security
	if err := validateModuleName(moduleName); err != nil {
		return "", err
	}

	// Remove std/ prefix if present (we'll add it back)
	moduleName = strings.TrimPrefix(moduleName, "std/")

	// Check negative cache first
	if triedPaths, found := r.checkNegativeCache(moduleName); found {
		return "", r.errWithSearchTrace(moduleName, triedPaths)
	}

	// Initialize search paths (done once)
	if r.searchPaths == nil {
		r.initializeSearchPaths()
	}

	// Try each search path
	var triedPaths []string
	for _, searchPath := range r.searchPaths {
		fullPath := filepath.Join(searchPath, moduleName+".ail")
		triedPaths = append(triedPaths, fullPath)

		if r.traceEnabled {
			fmt.Fprintf(os.Stderr, "[trace-loader] Checking: %s\n", fullPath)
		}

		if _, err := os.Stat(fullPath); err == nil {
			// Found! Check version if this is the stdlib root
			if err := r.checkStdlibVersion(searchPath); err != nil {
				if r.strictMode {
					return "", err
				}
				// Non-strict: log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
			return fullPath, nil
		}
	}

	// Not found - cache negative result
	r.cacheNegative(moduleName, triedPaths)
	return "", r.errWithSearchTrace(moduleName, triedPaths)
}

// initializeSearchPaths initializes the search path list
// Search order (highest priority first):
// 1. CLI flag (--stdlib-path)
// 2. Binary-relative (../std from binary location)
// 3. AILANG_STDLIB_PATH environment variable (colon/semicolon separated)
// 4. User data directory (platform-specific)
// 5. System directories (/usr/local/share/ailang/std, /usr/share/ailang/std)
func (r *StdlibResolver) initializeSearchPaths() {
	var paths []string

	// 1. CLI override (highest priority)
	if r.cliOverridePath != "" {
		paths = append(paths, r.cliOverridePath)
	}

	// 2. Binary-relative path
	if binPath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(binPath)
		stdPath := filepath.Join(binDir, "..", "std")
		if absPath, err := filepath.Abs(stdPath); err == nil {
			paths = append(paths, absPath)
		}
	}

	// 3. AILANG_STDLIB_PATH environment variable (multi-path)
	if envPath := os.Getenv("AILANG_STDLIB_PATH"); envPath != "" {
		sep := getPathSeparator()
		for _, p := range strings.Split(envPath, sep) {
			p = strings.TrimSpace(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
	}

	// 4. User data directory (platform-specific)
	if userDir := getUserDataDir(); userDir != "" {
		paths = append(paths, userDir)
	}

	// 5. System directories (Unix only)
	if runtime.GOOS != "windows" {
		paths = append(paths,
			"/usr/local/share/ailang/std",
			"/usr/share/ailang/std",
		)
	}

	r.searchPaths = paths

	if r.traceEnabled {
		fmt.Fprintf(os.Stderr, "[trace-loader] Search paths initialized (%d paths):\n", len(paths))
		for i, p := range paths {
			fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, p)
		}
	}
}

// checkNegativeCache checks if a module lookup previously failed
// Returns (tried paths, found)
func (r *StdlibResolver) checkNegativeCache(moduleName string) ([]string, bool) {
	paths, found := r.negativeCache[moduleName]
	return paths, found
}

// cacheNegative caches a failed module lookup
func (r *StdlibResolver) cacheNegative(moduleName string, triedPaths []string) {
	r.negativeCache[moduleName] = triedPaths
}

// checkStdlibVersion checks if the stdlib VERSION file matches expected version
// Returns error if version mismatch (severity depends on strictMode)
func (r *StdlibResolver) checkStdlibVersion(stdlibRoot string) error {
	versionFile := filepath.Join(stdlibRoot, "VERSION")
	content, err := os.ReadFile(versionFile)
	if err != nil {
		// VERSION file missing - not necessarily an error
		return nil
	}

	version := strings.TrimSpace(string(content))
	if version != r.expectedVersion {
		return fmt.Errorf("stdlib version mismatch: expected %s, found %s at %s",
			r.expectedVersion, version, stdlibRoot)
	}

	return nil
}

// errWithSearchTrace returns a detailed error with search trace
func (r *StdlibResolver) errWithSearchTrace(moduleName string, triedPaths []string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("stdlib module not found: std/%s\n", moduleName))
	sb.WriteString("searched:\n")
	for _, p := range triedPaths {
		sb.WriteString(fmt.Sprintf("  - %s\n", p))
	}
	sb.WriteString("\ntip: set AILANG_STDLIB_PATH=/path/to/ailang/std or use --stdlib-path flag\n")
	return fmt.Errorf("%s", sb.String())
}
