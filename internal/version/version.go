// Package version exposes build-time metadata (version, commit, build time).
//
// These vars are overridden at link time via -ldflags:
//
//	-X github.com/sunholo-data/ailang/internal/version.Version=vX.Y.Z
//	-X github.com/sunholo-data/ailang/internal/version.Commit=<sha>
//	-X github.com/sunholo-data/ailang/internal/version.BuildTime=<iso>
//
// For `go run` / `go test` invocations without ldflags, init() populates
// Commit/BuildTime from runtime/debug.ReadBuildInfo() when available so
// the module cache key still differentiates between builds. If no VCS
// info is available (e.g. outside a git checkout), Commit stays "dev"
// and the source-hash component of the cache key still catches edits.
package version

import "runtime/debug"

var (
	// Version is the release version (e.g. "v0.11.4"). "dev" for unreleased builds.
	Version = "dev"
	// Commit is the git commit SHA the binary was built from.
	Commit = "dev"
	// BuildTime is the ISO-8601 build timestamp.
	BuildTime = "unknown"
)

func init() {
	// Fallback for builds without -ldflags (go run, go test): read VCS info
	// from the embedded build info. This keeps the module cache key stable
	// across rebuilds of the same commit but invalidates when the commit changes.
	if Commit != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if s.Value != "" {
				Commit = s.Value
			}
		case "vcs.time":
			if s.Value != "" {
				BuildTime = s.Value
			}
		case "vcs.modified":
			// If the working tree had uncommitted changes at build time,
			// mark the commit as dirty so cache keys change on every edit.
			if s.Value == "true" && Commit != "dev" {
				Commit = Commit + "-dirty"
			}
		}
	}
}
