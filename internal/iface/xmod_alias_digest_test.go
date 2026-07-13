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

// TestXModAliasPoly_DigestIgnoresParameterizedAlias extends the digest-neutral
// lock to M-XMOD-ALIAS-POLY: adding a PARAMETERIZED alias (body + AliasParams)
// must ALSO leave the interface digest unchanged. computeDigest excludes both
// TypeAliases and AliasParams, so a module gaining `type Box[a] = { items: [a] }`
// triggers no dependent-package cascade. If AliasParams ever enters the digest,
// this fails and the no-cascade claim must be revisited.
func TestXModAliasPoly_DigestIgnoresParameterizedAlias(t *testing.T) {
	b := NewBuilder("test/mod", types.NewTypeEnv())

	base := NewIface("test/mod")
	base.Schema = "ailang.iface/v1"

	withPoly := NewIface("test/mod")
	withPoly.Schema = "ailang.iface/v1"
	// `type Box[a] = { items: [a] }` — record body + one param.
	withPoly.AddTypeAlias("Box", &types.TRecord{
		Fields: map[string]types.Type{
			"items": &types.TList{Element: &types.TVar2{Name: "a", Kind: types.Star}},
		},
	})
	withPoly.AddTypeAliasParams("Box", []string{"a"})

	d1, err := b.computeDigest(base)
	if err != nil {
		t.Fatalf("computeDigest(base): %v", err)
	}
	d2, err := b.computeDigest(withPoly)
	if err != nil {
		t.Fatalf("computeDigest(withPoly): %v", err)
	}

	if d1 != d2 {
		t.Fatalf("interface digest changed when a PARAMETERIZED alias was added (%q vs %q) — "+
			"M-XMOD-ALIAS-POLY would trigger cascade rebuilds; digest must ignore AliasParams", d1, d2)
	}
}
