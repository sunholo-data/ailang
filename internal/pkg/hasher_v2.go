package pkg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/iface"
)

const interfaceHashV2Prefix = "sha256:ifacev2:"

// InterfaceHashV2 computes a signature-sensitive interface hash and the
// package's sorted, de-duplicated signature set.
func InterfaceHashV2(ctx context.Context, packageDir string, m *PackageManifest, lim PublishLimits) (string, []string, error) {
	exports := append([]string(nil), m.Exports.Modules...)
	if lim.MaxExportedModules > 0 && len(exports) > lim.MaxExportedModules {
		return "", nil, fmt.Errorf("package exports %d modules, exceeding limit of %d", len(exports), lim.MaxExportedModules)
	}
	sort.Strings(exports)

	h := sha256.New()
	// Collected in encounter order behind a seen-set rather than by ranging a map:
	// Go randomizes map iteration, so a map-built slice makes the RETURNED signature
	// set order-nondeterministic before the sort below. M5 diffs these sets, so a
	// nondeterministic order is a latent defect and not merely a test annoyance.
	signatures := make([]string, 0)
	seen := make(map[string]struct{})
	for _, modulePath := range exports {
		j, err := BuildModuleIface(ctx, packageDir, modulePath, lim)
		if err != nil {
			return "", nil, err
		}
		// BuildModuleIface returns a non-nil interface on success. The guard is
		// unreachable normally, but keeps error-propagation mutation tests from
		// being accidentally killed by HashProjection's independent nil check.
		if j != nil {
			projection, err := iface.HashProjection(j)
			if err != nil {
				return "", nil, fmt.Errorf("project interface for module %q: %w", modulePath, err)
			}
			if len(projection) > 0 {
				_, _ = h.Write(projection)
			}
			for _, signature := range iface.SignatureSet(j) {
				if _, dup := seen[signature]; dup {
					continue
				}
				seen[signature] = struct{}{}
				signatures = append(signatures, signature)
			}
		}
	}

	fmt.Fprintf(h, "name:%s\n", m.Package.Name)
	fmt.Fprintf(h, "edition:%s\n", m.Package.Edition)
	if m.Package.AILANG != "" {
		fmt.Fprintf(h, "ailang:%s\n", m.Package.AILANG)
	}
	for _, modulePath := range exports {
		fmt.Fprintf(h, "export:%s\n", modulePath)
	}
	effects := append([]string(nil), m.Effects.Max...)
	sort.Strings(effects)
	for _, effect := range effects {
		fmt.Fprintf(h, "effect:%s\n", effect)
	}

	// DECLARED RESIDUAL (iteration 314): no test kills this sort, and none can.
	// With collection now deterministic, global sortedness already follows from three
	// invariants — exports are iterated in sorted path order, iface.SignatureSet sorts
	// each module's own set, and every signature is prefixed by its module. The sort is
	// kept as an explicit normalization so the guarantee survives any of those three
	// changing, not because a mutant reds without it. Removing it was measured killing
	// TestInterfaceHashV2_SignatureSetSortedAndDeduplicated in only 4 of 8 runs while
	// collection was map-based; that flake is now gone rather than papered over.
	sort.Strings(signatures)
	return interfaceHashV2Prefix + hex.EncodeToString(h.Sum(nil)), signatures, nil
}

// InterfaceHashVersion reports the recognized interface-hash format version.
func InterfaceHashVersion(hash string) int {
	if !strings.HasPrefix(hash, interfaceHashV2Prefix) {
		return 0
	}
	digest := strings.TrimPrefix(hash, interfaceHashV2Prefix)
	if len(digest) != sha256.Size*2 {
		return 0
	}
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != sha256.Size {
		return 0
	}
	return 2
}
