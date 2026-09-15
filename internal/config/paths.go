package config

// Locations: where AILANG keeps state and where it finds its inputs.
const (
	EnvStateDir          = "AILANG_STATE_DIR"
	EnvCacheDir          = "AILANG_CACHE_DIR"
	EnvStdlibPath        = "AILANG_STDLIB_PATH"
	EnvProjectRoot       = "AILANG_PROJECT_ROOT"
	EnvExamples          = "AILANG_EXAMPLES"
	EnvZ3Path            = "AILANG_Z3_PATH"
	EnvBrowserProfileDir = "AILANG_BROWSER_PROFILE_DIR"
	EnvHome              = "HOME"
	EnvXDGCacheHome      = "XDG_CACHE_HOME"
	EnvXDGDataHome       = "XDG_DATA_HOME"
	EnvAppData           = "APPDATA"
)

var pathVars = []Var{
	{EnvStateDir, "~/.ailang", AreaPaths, "Directory for every local store (SQLite databases, ledgers, locks); statedir.Dir cleans and returns it."},
	{EnvCacheDir, "", AreaPaths, "Root of the compile cache (<dir>/compile) and the prompt cache; unset means <project>/.ailang/cache and $XDG_CACHE_HOME/ailang (else ~/.cache/ailang) respectively."},
	{EnvStdlibPath, "", AreaPaths, "Path-list (OS separator) of stdlib roots searched before the bundled and installed copies; the embed engine sets it for child processes when unset."},
	{EnvProjectRoot, "", AreaPaths, "Root the embed engine resolves module paths against; must contain the requested module or Load fails."},
	{EnvExamples, "", AreaPaths, "Directory `ailang examples` reads instead of searching upward from the binary."},
	{EnvZ3Path, "", AreaPaths, "Path of the z3 binary, tried before PATH and the usual install locations."},
	{EnvBrowserProfileDir, "", AreaPaths, "Root for browser profiles used by the browser commands; unset derives one under the state dir."},
	{EnvHome, "", AreaPaths, "The user's home directory as the shell set it; used where a plist or a data-dir convention needs the literal value rather than os.UserHomeDir."},
	{EnvXDGCacheHome, "", AreaPaths, "XDG cache base; the prompt cache lives at $XDG_CACHE_HOME/ailang when AILANG_CACHE_DIR is unset."},
	{EnvXDGDataHome, "", AreaPaths, "XDG data base on Linux/BSD; the installed stdlib lives under it (else ~/.local/share)."},
	{EnvAppData, "", AreaPaths, "Windows application-data base; the installed stdlib lives under it."},
}

// StateDir returns AILANG_STATE_DIR uncleaned, "" when unset. statedir.Dir
// is the resolver that applies the ~/.ailang default.
func StateDir() string { return get(EnvStateDir) }

// CacheDir returns AILANG_CACHE_DIR, "" when unset.
func CacheDir() string { return get(EnvCacheDir) }

// StdlibPath returns AILANG_STDLIB_PATH verbatim (a path-list), "" when unset.
func StdlibPath() string { return get(EnvStdlibPath) }

// ProjectRoot returns AILANG_PROJECT_ROOT, "" when unset.
func ProjectRoot() string { return get(EnvProjectRoot) }

// ExamplesDir returns AILANG_EXAMPLES, "" when unset.
func ExamplesDir() string { return get(EnvExamples) }

// Z3Path returns AILANG_Z3_PATH, "" when unset.
func Z3Path() string { return get(EnvZ3Path) }

// BrowserProfileDir returns AILANG_BROWSER_PROFILE_DIR, "" when unset.
func BrowserProfileDir() string { return get(EnvBrowserProfileDir) }

// Home returns $HOME verbatim, "" when unset. Prefer os.UserHomeDir unless
// the literal variable is what is wanted (rendering a plist, a data-dir
// convention that reads $HOME by contract).
func Home() string { return get(EnvHome) }

// XDGCacheHome returns XDG_CACHE_HOME, "" when unset.
func XDGCacheHome() string { return get(EnvXDGCacheHome) }

// XDGDataHome returns XDG_DATA_HOME, "" when unset.
func XDGDataHome() string { return get(EnvXDGDataHome) }

// AppData returns APPDATA, "" when unset.
func AppData() string { return get(EnvAppData) }
