package pkg

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// M-PKG-QUALITY-LADDER M4 (design D2, ratified 2026-09-17): every release
// carries a human-readable description in CHANGELOG.md and a machine-readable
// intent in [release] kind. The kind is the author's claim; Sprint 2 checks it
// against the MEASURED change class, so `security` is the label the ladder
// trusts most because it is the one it verifies hardest.

// ChangelogFile is the conventional changelog filename shipped in the tarball.
const ChangelogFile = "CHANGELOG.md"

// ReleaseKinds are the accepted [release] kind values, weakest claim last.
var ReleaseKinds = []string{"security", "fix", "feature", "breaking"}

// ReleaseGateHardFrom is the first ailang version at which PUB001/PUB002
// BLOCK a publish. Below it they are badges — the one-minor grace window the
// design grants (same pattern as --allow-dotted-tool-names in v0.20.x).
const ReleaseGateHardFrom = "0.41.0"

// ChangelogNotesLimit bounds the banked section text.
const ChangelogNotesLimit = 2048

// ReleaseConfig is the [release] section of ailang.toml.
//
//	[release]
//	kind  = "fix"            # security | fix | feature | breaking
//	notes = "optional one-liner; CHANGELOG.md is the canonical description"
type ReleaseConfig struct {
	Kind  string `toml:"kind"`
	Notes string `toml:"notes"`
}

// ValidateReleaseKind accepts an empty kind (grace / not declared) or one of
// ReleaseKinds.
func ValidateReleaseKind(kind string) error {
	if kind == "" {
		return nil
	}
	for _, k := range ReleaseKinds {
		if k == kind {
			return nil
		}
	}
	return fmt.Errorf("[release].kind must be one of %s, got %q", strings.Join(ReleaseKinds, "|"), kind)
}

// changelogHeading matches `## 0.8.2`, `## [0.8.2]`, `## v0.8.2 — 2026-09-17`,
// `## [v0.8.2] - 2026-09-17`. The version token is group 1.
var changelogHeading = regexp.MustCompile(`^##\s*\[?v?(\d+\.\d+\.\d+[^\]\s]*)\]?`)

// ChangelogSection returns the body of the `## <version>` section of
// dir/CHANGELOG.md (trimmed, capped at ChangelogNotesLimit) and whether a
// NON-EMPTY section exists. A missing file, a missing heading and an empty
// section all report false — the caller distinguishes them only for the
// badge text.
func ChangelogSection(dir, version string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, ChangelogFile))
	if err != nil {
		return "", false
	}
	return ChangelogSectionFrom(string(data), version)
}

// ChangelogSectionFrom is ChangelogSection over already-read content.
func ChangelogSectionFrom(content, version string) (string, bool) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var body []string
	inSection := false
	for _, line := range lines {
		if m := changelogHeading.FindStringSubmatch(line); m != nil {
			if inSection {
				break
			}
			if m[1] == version {
				inSection = true
			}
			continue
		}
		if inSection {
			body = append(body, line)
		}
	}
	text := strings.TrimSpace(strings.Join(body, "\n"))
	if text == "" {
		return "", false
	}
	if len(text) > ChangelogNotesLimit {
		text = text[:ChangelogNotesLimit]
	}
	return text, true
}

// ReleaseGatesHard reports whether PUB001/PUB002 block at the given ailang
// version string (e.g. "v0.40.0", "0.41.2-5-gabc", or the `ailang --version`
// banner). A version with no x.y.z token — a plain `go build` reports "dev" —
// is inside the grace window: the badge still shows, and the deployed
// validator (built with the release tag injected) is the binary that decides.
func ReleaseGatesHard(ailangVersion string) bool {
	m := semverToken.FindStringSubmatch(ailangVersion)
	if m == nil {
		return false
	}
	cmp, err := compareSemver(m[1], ReleaseGateHardFrom)
	if err != nil {
		return true
	}
	return cmp >= 0
}

// semverToken finds the first x.y.z in a version string such as
// "v0.40.0-3-gabc-dirty" or the multi-line `ailang --version` banner
// "AILANG v0.39.4-11-g1f78cd2bc\nCommit: …".
var semverToken = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)

// compareSemver compares dotted numeric versions; -1, 0, +1.
func compareSemver(a, b string) (int, error) {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	if len(pa) != 3 || len(pb) != 3 {
		return 0, fmt.Errorf("not a semver triple: %q vs %q", a, b)
	}
	for i := 0; i < 3; i++ {
		var x, y int
		if _, err := fmt.Sscanf(pa[i], "%d", &x); err != nil {
			return 0, err
		}
		if _, err := fmt.Sscanf(pb[i], "%d", &y); err != nil {
			return 0, err
		}
		if x != y {
			if x < y {
				return -1, nil
			}
			return 1, nil
		}
	}
	return 0, nil
}
