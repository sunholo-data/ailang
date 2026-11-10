package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil/prefixpkg"
)

// Whose ceiling bounds a dependency? In a consumer build only the ROOT's
// [effects].max is enforced, and on the root's own code: rule (a) its declared
// rows, rule (b) every reference its code makes to a global it does not own
// (effect_ceiling.go isAuthorityEntry). Once a dependency module reached by
// its module_prefix spelling has its pkg/ identity (M1b), rule (b) is the ONLY
// thing that bounds it, so it must reject a reference to an effectful
// dependency function under BOTH spellings — including a reference that does
// not call it (returned, stored in a record) and one to a dependency function
// whose effect comes from another function inside the dependency.
func TestCanonicalModuleIdentity_RootCeilingBoundsDependencyReferences(t *testing.T) {
	app := prefixpkg.Ceiling(t)
	for sp := range prefixpkg.Spellings {
		for _, c := range prefixpkg.CeilingCases {
			name := c + "_" + sp
			t.Run(name, func(t *testing.T) {
				_, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, name+".ail")})
				if err == nil {
					t.Fatalf("%s: a root capped at [IO] referencing a dependency's effectful function must be rejected", name)
				}
				want := "FS"
				if c == "net" {
					want = "Net"
				}
				msg := err.Error()
				if !strings.Contains(msg, "effect ceiling violation in package acme/app") || !strings.Contains(msg, fmt.Sprintf("effects [%s]", want)) {
					t.Fatalf("%s: want the root's ceiling violation for %s, got: %v", name, want, err)
				}
			})
		}
		t.Run("pure_"+sp, func(t *testing.T) {
			// Control: a pure function of a module whose sibling functions are
			// effectful is not charged, under either spelling. (dev charged the
			// whole module when it was reached by its prefix.)
			res, err := Run(Config{Mode: ModeCheck, NoCache: true}, Source{Filename: filepath.Join(app, "pure_"+sp+".ail")})
			if err != nil || len(res.Errors) > 0 {
				t.Fatalf("pure_%s: %v %v", sp, err, res.Errors)
			}
		})
	}
}
