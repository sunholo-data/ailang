package loader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/importhint"
	"github.com/sunholo-data/ailang/internal/stdlibroot"
)

// BinaryVersion is the version of the ailang binary.
// Set by main.go from build-time ldflags.
// Used to validate stdlib version compatibility.
var BinaryVersion = "dev"

// stdlibVersionWarningShown tracks if we've already shown the version mismatch warning
// M-DX21: Show warning only once per process to reduce noise
var stdlibVersionWarningShown bool

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

// StdlibResolver resolves stdlib module names against the ONE stdlib root of the
// process (internal/stdlibroot, M-STDLIB-ROOT-RESOLUTION). Modules are never
// searched for one by one across several roots: a module missing from the chosen
// root is an error naming that root, so a run cannot mix modules from two stdlibs.
type StdlibResolver struct {
	// CLI override path (--stdlib-path); "" uses the process configuration.
	cliOverridePath string

	// Enable trace logging (--trace-loader); ORed with the process configuration.
	traceEnabled bool

	// Strict mode (--strict: fail on version mismatch); ORed likewise.
	strictMode bool

	// Expected stdlib version (embedded at compile time)
	expectedVersion string
}

// NewStdlibResolver creates a new stdlib resolver. Empty/false arguments defer to
// the process configuration set by stdlibroot.Configure.
func NewStdlibResolver(cliPath string, traceEnabled, strictMode bool) *StdlibResolver {
	return &StdlibResolver{
		cliOverridePath: cliPath,
		traceEnabled:    traceEnabled,
		strictMode:      strictMode,
		expectedVersion: BinaryVersion, // Uses package-level variable set by main.go
	}
}

func (r *StdlibResolver) tracing() bool { return r.traceEnabled || stdlibroot.Current().Trace }

func (r *StdlibResolver) strict() bool { return r.strictMode || stdlibroot.Current().StrictVersion }

// Root returns the stdlib root this resolver reads from, after the version check
// (on-disk roots only: the embedded copy matches the binary by construction).
func (r *StdlibResolver) Root() (stdlibroot.Root, error) {
	root, err := stdlibroot.Resolve(r.cliOverridePath)
	if err != nil {
		return root, err
	}
	if root.Embedded() {
		return root, nil
	}
	if err := r.checkStdlibVersion(root.Dir); err != nil {
		if r.strict() {
			return root, err
		}
		// M-DX21: Non-strict: log warning only once per process
		// AILANG_NO_VERSION_WARNINGS: suppress entirely
		// AILANG_QUIET_WARNINGS: suppress in JSON/quiet mode (set by CLI)
		if !stdlibVersionWarningShown && !config.StdlibVersionWarningsSuppressed() {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			stdlibVersionWarningShown = true
		}
	}
	return root, nil
}

// ResolveStdlib resolves a stdlib module name to the path of its source in the
// process stdlib root: <root>/<module>.ail on disk, "<embedded>/std/<module>.ail"
// for the copy built into the binary.
func (r *StdlibResolver) ResolveStdlib(moduleName string) (string, error) {
	path, _, err := r.resolve(moduleName)
	return path, err
}

// ReadStdlib resolves a stdlib module and returns its path plus, for the embedded
// root, its source. For an on-disk root content is nil: the caller reads the file
// itself (through the source snapshot, so admission and execution see one read).
func (r *StdlibResolver) ReadStdlib(moduleName string) (string, []byte, error) {
	path, root, err := r.resolve(moduleName)
	if err != nil || !root.Embedded() {
		return path, nil, err
	}
	content, err := fs.ReadFile(root.FS, strings.TrimPrefix(path, "<embedded>/std/"))
	return path, content, err
}

func (r *StdlibResolver) resolve(moduleName string) (string, stdlibroot.Root, error) {
	// Validate module name for security
	if err := validateModuleName(moduleName); err != nil {
		return "", stdlibroot.Root{}, err
	}
	moduleName = strings.TrimPrefix(moduleName, "std/")

	root, err := r.Root()
	if err != nil {
		return "", root, err
	}
	name := moduleName + ".ail"
	if st, statErr := fs.Stat(root.FS, name); statErr != nil || st.IsDir() {
		return "", root, r.errWithSearchTrace(moduleName, root)
	}
	path := root.DisplayPath(name)
	if r.tracing() {
		fmt.Fprintf(os.Stderr, "[trace-loader] std/%s -> %s\n", moduleName, path)
	}
	return path, root, nil
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
	// Compare BASE semver only — a dev/CI binary built from a release tag carries a
	// git-describe suffix (e.g. "v0.25.0-177-g5878c2204-dirty"), but the v0.25.0 stdlib
	// IS the correct stdlib for it. Without this, every dev/eval-rig run emitted a spurious
	// "stdlib version mismatch" warning into stderr — 291 runs in the 2026-06-20 rotation —
	// polluting the model's BashExec context. Real mismatches (different base) still warn.
	if baseVersion(version) != baseVersion(r.expectedVersion) {
		return fmt.Errorf("stdlib version mismatch: expected %s, found %s at %s",
			r.expectedVersion, version, stdlibRoot)
	}

	return nil
}

// baseVersion strips a git-describe dev suffix ("-<n>-g<hash>[-dirty]") from a version
// string, leaving the base semver. Compatibility is determined by the base release only.
func baseVersion(v string) string {
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i]
	}
	return v
}

// errWithSearchTrace returns a detailed error for a module the stdlib root lacks.
func (r *StdlibResolver) errWithSearchTrace(moduleName string, root stdlibroot.Root) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("stdlib module not found: std/%s\n", moduleName))
	if root.Embedded() {
		sb.WriteString("stdlib root: the copy built into this binary\n")
	} else {
		sb.WriteString(fmt.Sprintf("stdlib root: %s (from %s)\n", root.Dir, rootSourceLabel(root.Source)))
		sb.WriteString("the stdlib root is chosen once per run; a module missing from it is not looked up anywhere else\n")
		sb.WriteString("\ntip: point AILANG_STDLIB_PATH or --stdlib-path at a complete std/ directory, or remove a partial ./std\n")
	}

	// M-DX-AI-DISCOVERY M3: recover a mistyped stdlib MODULE name. A curated alias
	// table (time->clock, ...) first, then Levenshtein <= 2 over the live module
	// list — both via internal/importhint (reusing its levenshtein; no parallel
	// engine). Exactly one "did you mean" line, only if a confident match exists.
	if suggestion := importhint.ModuleSuggestion(moduleName); suggestion != "" {
		sb.WriteString(fmt.Sprintf("\ndid you mean: %s?\n", suggestion))
	}
	// Always show the available module list (or an explicit unavailable note — never
	// a silent skip). Alias suggestions above still print even when this is nil.
	if importhint.ModuleLocator != nil {
		if mods := importhint.ModuleLocator(); len(mods) > 0 {
			sb.WriteString(fmt.Sprintf("available: %s (%d modules)\n", strings.Join(mods, ", "), len(mods)))
		} else {
			sb.WriteString("available: (module list unavailable — the import-hint index could not read the stdlib root)\n")
		}
	} else {
		sb.WriteString("available: (module list unavailable — the import-hint index could not read the stdlib root)\n")
	}

	return fmt.Errorf("%s", sb.String())
}

// rootSourceLabel names where a root came from, for error text.
func rootSourceLabel(source string) string {
	switch source {
	case "flag":
		return "--stdlib-path"
	case "env":
		return config.EnvStdlibPath
	case "cwd":
		return "./std in the working directory"
	case "binary":
		return "the std/ next to the ailang binary"
	case "user":
		return "the user data directory"
	case "system":
		return "a system directory"
	}
	return source
}
