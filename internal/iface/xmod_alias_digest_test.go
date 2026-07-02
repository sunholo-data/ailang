package iface

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

// TestXModAlias_DigestIgnoresTypeAliases locks the "digest-neutral" property of
// M-XMOD-ALIAS: computeDigest serializes only Module/Schema/Exports/Constructors,
// NOT TypeAliases. Registering a non-record alias (the fix) therefore does not
// change a module's interface digest — so it triggers no dependent-package cascade
// rebuilds. If computeDigest ever starts folding TypeAliases into the digest, this
// test fails and the no-cascade claim in the design doc must be revisited.
func TestXModAlias_DigestIgnoresTypeAliases(t *testing.T) {
	b := NewBuilder("test/mod", types.NewTypeEnv())

	base := NewIface("test/mod")
	base.Schema = "ailang.iface/v1"

	withAlias := NewIface("test/mod")
	withAlias.Schema = "ailang.iface/v1"
	// The exact shape M-XMOD-ALIAS now registers: `type Row = Json`.
	withAlias.AddTypeAlias("Row", &types.TCon{Name: "Json"})

	d1, err := b.computeDigest(base)
	if err != nil {
		t.Fatalf("computeDigest(base): %v", err)
	}
	d2, err := b.computeDigest(withAlias)
	if err != nil {
		t.Fatalf("computeDigest(withAlias): %v", err)
	}

	if d1 != d2 {
		t.Fatalf("interface digest changed when a non-record alias was added (%q vs %q) — "+
			"M-XMOD-ALIAS would trigger cascade rebuilds; digest must ignore TypeAliases", d1, d2)
	}
}
