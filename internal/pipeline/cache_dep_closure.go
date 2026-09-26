package pipeline

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/iface"
)

// M-TYPE-NAME-SHADOW M3: what a module's compile reads, and therefore what its
// cache key must cover.
//
// A module's compile reads its DIRECT imports' interfaces (exports, constructors,
// aliases) and the type ALIASES of every module in its transitive import closure
// (M-TRANSITIVE-ALIAS-ENV-IMPORT). The key used to hold only the direct imports'
// iface digests, and an iface digest covers exports and constructors but not
// alias bodies, so an alias edit two hops away served a stale verdict. The key
// now covers the whole closure, each entry digest + alias digest; and the
// transitive alias pull is restricted to the same closure (it used to read every
// module compiled so far, siblings included — order-dependent and uncacheable).

// depClosure returns modID's transitive import closure, excluding modID itself.
func (st *modulePipelineState) depClosure(modID string) map[string]bool {
	seen := make(map[string]bool)
	stack := []string{modID}
	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		mod := st.modules[cur]
		if mod == nil {
			continue
		}
		for _, imp := range mod.Imports {
			if !seen[imp] && imp != modID {
				seen[imp] = true
				stack = append(stack, imp)
			}
		}
	}
	return seen
}

// cacheDepDigest is the per-dependency key component: the iface digest (exports,
// constructors) plus a digest of its alias bodies and params, which the iface
// digest does not cover.
func cacheDepDigest(ifc *iface.Iface) string {
	return ifc.Digest + "+" + aliasDigest(ifc)
}

// aliasDigest hashes an interface's type aliases (name, body, params) in sorted
// order.
func aliasDigest(ifc *iface.Iface) string {
	names := make([]string, 0, len(ifc.TypeAliases))
	for name := range ifc.TypeAliases {
		names = append(names, name)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, name := range names {
		params, _ := ifc.GetTypeAliasParams(name)
		fmt.Fprintf(h, "%s[%s]=%s\n", name, strings.Join(params, ","), ifc.TypeAliases[name])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}
