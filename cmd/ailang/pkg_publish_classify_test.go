package main

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func sigInfo(hash string, sigs ...string) messaging.PackageVersionInfo {
	return messaging.PackageVersionInfo{Name: "sunholo/core", InterfaceHash: hash,
		Signatures: append(sigs, messaging.SignatureContractsMarker)}
}

// TST-PKGPAR-28
// Contracts are kept out of the interface hash (D7), so equal hashes must not
// short-circuit to A when the signature sets disagree on a contract.
func TestMapChangeClassToSchema_ContractChangeUnderEqualHash(t *testing.T) {
	old := sigInfo("same", "m:func:f:int:")
	new := sigInfo("same", "m:func:f:int:", "m:contract:f:requires (n > 0)")
	if got := mapChangeClassToSchema("patch", old, new); got != "C" {
		t.Fatalf("class = %q, want C", got)
	}
}

// TST-PKGPAR-03
// A measured additive change reaches the deterministic cascade path; before
// signatures were threaded through publish, an in-module addition was C and
// went to AI (V6, V7).
func TestMapChangeClassToSchema_AdditiveDispatchesDeterministically(t *testing.T) {
	old := sigInfo("h1", "m:func:a:int:")
	new := sigInfo("h2", "m:func:a:int:", "m:func:b:int:")
	class := mapChangeClassToSchema("minor", old, new)
	if class != "B" || classifyDispatchPath(class) != DispatchDeterministic {
		t.Fatalf("class=%q path=%q, want B/deterministic", class, classifyDispatchPath(class))
	}
	removed := sigInfo("h3", "m:func:b:int:")
	if c := mapChangeClassToSchema("minor", old, removed); c != "C" || classifyDispatchPath(c) != DispatchAI {
		t.Fatalf("removal class=%q path=%q, want C/AI", c, classifyDispatchPath(c))
	}
	preMarker := messaging.PackageVersionInfo{InterfaceHash: "h0", Signatures: []string{"m:func:a:int:"}}
	if c := mapChangeClassToSchema("minor", preMarker, new); c != "U" || classifyDispatchPath(c) != DispatchAI {
		t.Fatalf("pre-D7 pair class=%q path=%q, want U/AI", c, classifyDispatchPath(c))
	}
}

// Pairs without signatures keep the pre-C4 heuristic exactly; a pair with a
// signature set on only one side is U on both channels.
func TestMapChangeClassToSchema_LegacyUnchanged(t *testing.T) {
	cases := []struct {
		name     string
		old, new messaging.PackageVersionInfo
		want     string
	}{
		{"equal hash", messaging.PackageVersionInfo{InterfaceHash: "h"}, messaging.PackageVersionInfo{InterfaceHash: "h"}, "A"},
		{"first publish", messaging.PackageVersionInfo{}, messaging.PackageVersionInfo{InterfaceHash: "h"}, "A"},
		{"effects widened", messaging.PackageVersionInfo{InterfaceHash: "a"}, messaging.PackageVersionInfo{InterfaceHash: "b", Effects: []string{"IO"}}, "C"},
		{"module added", messaging.PackageVersionInfo{InterfaceHash: "a", Exports: []string{"m/x"}}, messaging.PackageVersionInfo{InterfaceHash: "b", Exports: []string{"m/x", "m/y"}}, "B"},
		{"same modules, hash moved", messaging.PackageVersionInfo{InterfaceHash: "a", Exports: []string{"m/x"}}, messaging.PackageVersionInfo{InterfaceHash: "b", Exports: []string{"m/x"}}, "C"},
		{"one side has signatures: U, as the inbox says", messaging.PackageVersionInfo{InterfaceHash: "h"}, sigInfo("h", "m:func:a:int:"), "U"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := mapChangeClassToSchema("patch", tc.old, tc.new); got != tc.want {
				t.Errorf("class = %q, want %q", got, tc.want)
			}
		})
	}
}
