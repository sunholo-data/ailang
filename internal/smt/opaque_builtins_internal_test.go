package smt

import (
	"errors"
	"strings"
	"testing"
)

// A declare-fun over a datatype the function does not declare is a skip
// naming the builtin, never a script Z3 rejects; one over primitive sorts
// stays in front of every declaration.
func TestOpaqueDeclarationNeedsItsSort(t *testing.T) {
	activeUninterpreted = map[string]string{}
	defer func() { activeUninterpreted = nil }()
	for _, name := range []string{"_json_encode", "_crypto_sha256hex"} {
		b := opaqueBuiltins[name]
		activeUninterpreted[b.symbol] = "(declare-fun " + b.symbol + " (" + strings.Join(b.params, " ") + ") " + b.result + ")"
	}
	_, _, err := placeUninterpretedDeclarations(map[string]bool{})
	if !errors.Is(err, ErrUnresolvableTypes) || !strings.Contains(err.Error(), "_json_encode needs sort Json") {
		t.Fatalf("Json undeclared: err = %v, want a skip naming _json_encode and Json", err)
	}
	front, after, err := placeUninterpretedDeclarations(map[string]bool{"Json": true})
	if err != nil {
		t.Fatal(err)
	}
	if len(front) != 1 || !strings.Contains(front[0], "$sha256hex") || len(after) != 1 || !strings.Contains(after[0], "$json_encode") {
		t.Fatalf("front %v, after types %v", front, after)
	}
}
