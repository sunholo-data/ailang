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
	signatures := make(map[string]struct{})
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
				signatures[signature] = struct{}{}
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

	resultSignatures := make([]string, 0, len(signatures))
	for signature := range signatures {
		resultSignatures = append(resultSignatures, signature)
	}
	sort.Strings(resultSignatures)
	return interfaceHashV2Prefix + hex.EncodeToString(h.Sum(nil)), resultSignatures, nil
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
