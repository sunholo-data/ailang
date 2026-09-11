package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
)

// Every effect the runtime can execute must be discoverable from the --caps help
// text. Four hand-typed copies of this list drifted apart and none of them named
// Process, so agents grepping `ailang --help` concluded the effect did not exist.
func TestCapsList_CoversEveryRegisteredEffect(t *testing.T) {
	listed := map[string]bool{}
	for _, c := range strings.Split(CapsList, ",") {
		listed[c] = true
	}
	for name := range effects.Registry {
		if name == "Debug" {
			continue // ghost effect: always granted, never passed to --caps
		}
		if !listed[name] {
			t.Errorf("effect %q is in effects.Registry but missing from CapsList (help text) — add it", name)
		}
	}
	for _, must := range []string{"IO", "FS", "Net", "Env", "Process", "AI"} {
		if !listed[must] {
			t.Errorf("CapsList must name %q", must)
		}
	}
}
