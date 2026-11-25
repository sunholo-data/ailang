package messaging

import "testing"

// TST-PKGPAR-19
// Contract-aware classification (M-PKG-PARALLEL-AGENTS D7). Contract lines never
// count as additive, any contract difference is C, and a set without the
// marker (produced before contracts were recorded) makes the pair U.
func TestClassifyChange_Contracts(t *testing.T) {
	const (
		fn   = "m:func:f:(int)->int:"
		req  = "m:contract:f:requires (n > 0)"
		ens  = "m:contract:f:ensures (result > 0)"
		fn2  = "m:func:g:(int)->int:"
		mark = SignatureContractsMarker
	)
	set := func(lines ...string) PackageVersionInfo { return PackageVersionInfo{Signatures: lines} }
	tests := []struct {
		name     string
		old, new PackageVersionInfo
		want     string
	}{
		{"requires added, same type", set(fn, mark), set(fn, req, mark), "C"},
		{"function's last ensures removed", set(fn, ens, mark), set(fn, mark), "C"},
		{"clause changed", set(fn, req, mark), set(fn, "m:contract:f:requires (n >= 0)", mark), "C"},
		{"package's first requires", set(fn, fn2, mark), set(fn, fn2, req, mark), "C"},
		{"contract change hidden by an addition", set(fn, mark), set(fn, fn2, req, mark), "C"},
		{"identical contracts, addition", set(fn, req, mark), set(fn, fn2, req, mark), "B"},
		{"identical contracts, nothing else", set(fn, req, mark), set(fn, req, mark), "A"},
		{"old side pre-marker, would read A", set(fn), set(fn, mark), "U"},
		{"old side pre-marker, would read B", set(fn), set(fn, fn2, mark), "U"},
		{"pre-D7 package drops all contracts", set(fn), set(fn), "U"},
		{"new side lacks marker", set(fn, mark), set(fn), "U"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyChange(tc.old, tc.new); got != tc.want {
				t.Errorf("classifyChange = %q, want %q", got, tc.want)
			}
		})
	}
}
