package effects

// PkgAssetTarget is a locked dependency's directory and version. An empty Dir
// represents a locked but unavailable target, never permission to use fallback.
type PkgAssetTarget struct{ Dir, Version string }
