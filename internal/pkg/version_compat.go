package pkg

import (
	"fmt"
	"strconv"
	"strings"
)

// semverTuple holds parsed major.minor.patch components.
type semverTuple struct {
	Major, Minor, Patch int
}

// ParseSemver parses a version string like "0.9.5" or "v0.9.5" into components.
// Returns an error if the format is invalid.
func ParseSemver(version string) (semverTuple, error) {
	v := strings.TrimPrefix(version, "v")

	// Handle "dev" or empty — always satisfies
	if v == "" || v == "dev" || v == "unknown" {
		return semverTuple{Major: 999, Minor: 999, Patch: 999}, nil
	}

	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return semverTuple{}, fmt.Errorf("invalid version %q: expected major.minor[.patch]", version)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semverTuple{}, fmt.Errorf("invalid major version in %q: %w", version, err)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semverTuple{}, fmt.Errorf("invalid minor version in %q: %w", version, err)
	}

	patch := 0
	if len(parts) == 3 {
		// Strip any pre-release suffix (e.g., "5-rc1")
		patchStr := strings.SplitN(parts[2], "-", 2)[0]
		patch, err = strconv.Atoi(patchStr)
		if err != nil {
			return semverTuple{}, fmt.Errorf("invalid patch version in %q: %w", version, err)
		}
	}

	return semverTuple{Major: major, Minor: minor, Patch: patch}, nil
}

// gte returns true if a >= b.
func (a semverTuple) gte(b semverTuple) bool {
	if a.Major != b.Major {
		return a.Major > b.Major
	}
	if a.Minor != b.Minor {
		return a.Minor > b.Minor
	}
	return a.Patch >= b.Patch
}

// ParseVersionConstraint extracts the minimum version from a constraint string.
// Supported format: ">=X.Y.Z" (only >= is supported, no ranges).
// Returns the minimum version string.
func ParseVersionConstraint(constraint string) (string, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return "", fmt.Errorf("empty version constraint")
	}

	// Reject range syntax
	if strings.Contains(constraint, ",") || strings.Contains(constraint, "||") {
		return "", fmt.Errorf("unsupported version constraint %q: only >=X.Y.Z is supported (no ranges)", constraint)
	}

	if strings.HasPrefix(constraint, ">=") {
		return strings.TrimSpace(strings.TrimPrefix(constraint, ">=")), nil
	}

	// Bare version treated as >=
	if _, err := ParseSemver(constraint); err == nil {
		return constraint, nil
	}

	return "", fmt.Errorf("unsupported version constraint %q: only >=X.Y.Z is supported", constraint)
}

// SatisfiesAILANGVersion checks if currentVersion meets a package's AILANG requirement.
// Returns true if:
//   - requirement is empty (no constraint — backward compat)
//   - currentVersion is "dev" or "unknown" (local builds bypass checks)
//   - currentVersion >= required minimum
func SatisfiesAILANGVersion(requirement, currentVersion string) (bool, error) {
	if requirement == "" {
		return true, nil // No constraint
	}

	current, err := ParseSemver(currentVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse current AILANG version %q: %w", currentVersion, err)
	}

	// dev/unknown always satisfy (parsed as 999.999.999)
	if currentVersion == "dev" || currentVersion == "unknown" {
		return true, nil
	}

	minVersion, err := ParseVersionConstraint(requirement)
	if err != nil {
		return false, fmt.Errorf("invalid ailang constraint %q: %w", requirement, err)
	}

	required, err := ParseSemver(minVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse required version %q: %w", minVersion, err)
	}

	return current.gte(required), nil
}

// BumpSemver increments a version string by the specified bump type.
// bumpType must be "patch", "minor", or "major".
func BumpSemver(version, bumpType string) (string, error) {
	sv, err := ParseSemver(version)
	if err != nil {
		return "", err
	}
	switch bumpType {
	case "patch":
		sv.Patch++
	case "minor":
		sv.Minor++
		sv.Patch = 0
	case "major":
		sv.Major++
		sv.Minor = 0
		sv.Patch = 0
	default:
		return "", fmt.Errorf("invalid bump type %q: must be patch, minor, or major", bumpType)
	}
	return fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch), nil
}

// FormatVersionConstraint creates a >=X.Y.Z constraint from a version string.
func FormatVersionConstraint(version string) string {
	v := strings.TrimPrefix(version, "v")
	if v == "dev" || v == "unknown" || v == "" {
		return ""
	}
	return ">=" + v
}
