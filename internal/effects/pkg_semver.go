package effects

import (
	"fmt"
	"strconv"
	"strings"
)

// assetVersion mirrors pkg.ParseSemver's compatibility ordering. Keep both
// parsers in sync; effects deliberately does not depend on the package loader.
func assetVersion(version string) ([3]int, error) {
	v := strings.TrimPrefix(version, "v")
	if v == "" || v == "dev" || v == "unknown" {
		return [3]int{999, 999, 999}, nil
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return [3]int{}, fmt.Errorf("invalid version %q", version)
	}
	out := [3]int{}
	for i, p := range parts {
		if i == 2 {
			p = strings.SplitN(p, "-", 2)[0]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, err
		}
		out[i] = n
	}
	return out, nil
}

// compareAssetVersions uses the name as a stable tie-break for numerically
// equal versions (e.g. v1.0.0 and 1.0.0, or prerelease and release).
func compareAssetVersions(a, b string) int {
	av, _ := assetVersion(a)
	bv, _ := assetVersion(b)
	for i := range av {
		if av[i] < bv[i] {
			return -1
		}
		if av[i] > bv[i] {
			return 1
		}
	}
	return strings.Compare(a, b)
}

// PkgAssetTarget is a locked dependency's directory and version. An empty Dir
// represents a locked but unavailable target, never permission to use fallback.
type PkgAssetTarget struct{ Dir, Version string }
