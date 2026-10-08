package eval_harness

import (
	"path/filepath"
	"regexp"
	"strings"
)

// gitDescribeSuffix matches the `-<commits>-g<sha>` metadata that `git describe` appends to a tag.
var gitDescribeSuffix = regexp.MustCompile(`-\d+-g[0-9a-f]+`)

// ReleaseTag reduces a build version to its RELEASE TAG so results bucket by
// release, not by every dev build. Drops the git-describe dev metadata and a
// -dirty suffix:
//
//	"v0.26.0-26-g9249a66bf"        -> "v0.26.0"   (dev build buckets under its release)
//	"v0.26.0-26-g9249a66bf-dirty"  -> "v0.26.0"
//	"v0.26.0-rc1-5-gabc1234"       -> "v0.26.0-rc1" (pre-release tags preserved)
//	"v0.26.0" / "dev"              -> unchanged
func ReleaseTag(v string) string {
	return strings.TrimSuffix(gitDescribeSuffix.ReplaceAllString(v, ""), "-dirty")
}

// versionDir matches a directory named for a release: v0.52.3, 0.3.14, v0.26.0-rc1.
var versionDir = regexp.MustCompile(`^v?\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

// versionFromPath returns the release named by the nearest version-named
// ancestor directory of a result file ("" if none). Rotation and baseline banks
// are laid out by version, so this recovers the AILANG version of rows written
// before RunMetrics carried it.
func versionFromPath(path string) string {
	for dir := filepath.Dir(path); ; {
		base := filepath.Base(dir)
		if versionDir.MatchString(base) {
			if !strings.HasPrefix(base, "v") {
				base = "v" + base
			}
			return base
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}
