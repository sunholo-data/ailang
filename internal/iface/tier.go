package iface

import (
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/std"
)

// Tiers of the 1.x Stability Promise (docs/docs/reference/stability.md). The
// vocabulary matches builtins.Stability.
const (
	TierStable       = "stable"
	TierExperimental = "experimental"
	TierInternal     = "internal"
)

// stabilityManifest is the parsed std/STABILITY.toml.
type stabilityManifest struct {
	Modules map[string]string `toml:"modules"`
}

var (
	manifestOnce sync.Once
	manifestTier map[string]string
	manifestErr  error
)

// parseStabilityManifest parses manifest text. A malformed manifest or an
// unknown tier is an error, never a silent default: a wrong tier in a golden
// is a wrong promise.
func parseStabilityManifest(text string) (map[string]string, error) {
	var m stabilityManifest
	md, err := toml.Decode(text, &m)
	if err != nil {
		return nil, fmt.Errorf("std/STABILITY.toml: %w", err)
	}
	if undecoded := md.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("std/STABILITY.toml: unknown keys %v", undecoded)
	}
	for mod, tier := range m.Modules {
		switch tier {
		case TierStable, TierExperimental, TierInternal:
		default:
			return nil, fmt.Errorf("std/STABILITY.toml: module %s has unknown tier %q (want stable|experimental|internal)", mod, tier)
		}
	}
	return m.Modules, nil
}

func loadStabilityManifest() (map[string]string, error) {
	manifestOnce.Do(func() {
		manifestTier, manifestErr = parseStabilityManifest(string(std.Stability))
	})
	return manifestTier, manifestErr
}

// StdModuleTiers returns a copy of the manifest: std module path -> tier.
func StdModuleTiers() (map[string]string, error) {
	m, err := loadStabilityManifest()
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out, nil
}

// ModuleTier is the default tier of a module: its std/STABILITY.toml entry, or
// "" for a module the manifest does not list (tiers are a promise about the
// standard library; a package's interface carries a tier only where a function
// says @experimental). Every real std module is listed:
// TestEveryStdModuleHasATier fails when a new one ships without a tier.
func ModuleTier(module string) (string, error) {
	m, err := loadStabilityManifest()
	if err != nil {
		return "", err
	}
	return m[module], nil
}

// FuncTier is the tier of one exported function: experimental when the
// declaration carries @experimental, else the module default. Deprecation is
// a separate axis and never changes the tier.
func FuncTier(moduleDefault string, fd *ast.FuncDecl) string {
	if fd != nil && fd.IsExperimental() {
		return TierExperimental
	}
	return moduleDefault
}
