package pkg

import (
	"slices"
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
)

// Through the real build (parser -> iface builder -> BuildModuleIface): adding a
// contract changes the v2 signature set by exactly one contract line and leaves
// interface_hash_v2 byte-identical, which is the compatibility promise of
// M-PKG-PARALLEL-AGENTS D7 (registry identity checks are unaffected).
func TestInterfaceHashV2_ContractsInSignaturesNotHash(t *testing.T) {
	plain := "export func pos(n: int) -> int { n }\n"
	withContract := "export func pos(n: int) -> int\n  requires { n > 0 }\n{ n }\n"

	dirA, mA := writeV2Package(t, []v2Module{{"test/pkg/main", plain}}, nil)
	dirB, mB := writeV2Package(t, []v2Module{{"test/pkg/main", withContract}}, nil)
	hashA, sigsA := mustV2Hash(t, dirA, mA, DefaultPublishLimits())
	hashB, sigsB := mustV2Hash(t, dirB, mB, DefaultPublishLimits())

	if hashA != hashB {
		t.Fatalf("a contract moved interface_hash_v2: %s -> %s", hashA, hashB)
	}
	for _, sigs := range [][]string{sigsA, sigsB} {
		if !slices.Contains(sigs, iface.ContractsMarker) {
			t.Fatalf("set lacks the contracts marker: %#v", sigs)
		}
	}
	want := "test/pkg/main:contract:pos:requires (n > 0)"
	if slices.Contains(sigsA, want) || !slices.Contains(sigsB, want) || len(sigsB) != len(sigsA)+1 {
		t.Fatalf("want exactly one added contract line %q:\nA=%#v\nB=%#v", want, sigsA, sigsB)
	}
}
