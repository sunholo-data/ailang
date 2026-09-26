// Package version exposes build-time metadata (version, commit, build time).
//
// These vars are overridden at link time via -ldflags:
//
//	-X github.com/sunholo-data/ailang/internal/version.Version=vX.Y.Z
//	-X github.com/sunholo-data/ailang/internal/version.Commit=<sha>
//	-X github.com/sunholo-data/ailang/internal/version.BuildTime=<iso>
//
// For `go run` / `go test` invocations without ldflags, init() populates
// Commit/BuildTime from runtime/debug.ReadBuildInfo() when available.
// Commit alone does NOT identify a build: see Dirty.
package version

import (
	"runtime/debug"
	"strings"
)

var (
	// Version is the release version (e.g. "v0.11.4"). "dev" for unreleased builds.
	Version = "dev"
	// Commit is the git commit SHA the binary was built from.
	Commit = "dev"
	// BuildTime is the ISO-8601 build timestamp.
	BuildTime = "unknown"
)

// vcsModified records the build info's vcs.modified flag, which `go build`
// embeds whether or not -ldflags stamped Commit.
var vcsModified bool

func init() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	// Fallback for builds without -ldflags (go run, go test): read VCS info
	// from the embedded build info so Commit names the real revision.
	fillCommit := Commit == "dev"
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if fillCommit && s.Value != "" {
				Commit = s.Value
			}
		case "vcs.time":
			if fillCommit && s.Value != "" {
				BuildTime = s.Value
			}
		case "vcs.modified":
			vcsModified = s.Value == "true"
			// A constant marker: it does NOT change per edit, so it cannot
			// key a cache on its own. See Dirty and M-COMPILE-CACHE-DIRTY-BUILD-KEY.
			if fillCommit && vcsModified && Commit != "dev" {
				Commit = Commit + "-dirty"
			}
		}
	}
}

// Dirty reports whether this binary's source is not fully identified by
// Commit: built from a tree with uncommitted changes, or with no VCS
// information at all. Two such builds of one commit can compile the same
// source differently, so anything keyed by Commit (the module compile cache)
// must add a per-build fingerprint. It reads both build paths: Makefile
// builds stamp Commit with a bare SHA and put "-dirty" only in Version;
// plain go build/run/test builds carry vcs.modified.
func Dirty() bool {
	return Commit == "dev" || vcsModified ||
		strings.HasSuffix(Commit, "-dirty") || strings.Contains(Version, "-dirty")
}
