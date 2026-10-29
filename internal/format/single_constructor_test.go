package format

import (
	"strings"
	"testing"
)

func TestSingleNullaryConstructorKeepsSumIdentity(t *testing.T) {
	for _, src := range []string{
		"module m\nexport type WavEncoding = | IntegerPcm deriving (Eq)",
		"module m\ntype UnitTag = | Tag",
		"module m\ntype Box[a] = Wrap(a)",
		"module m\ntype Choice = A | B",
		"module m\ntype Alias = int",
	} {
		out := assertIdempotentAndRoundTrips(t, src, "test://single-nullary")
		if strings.Contains(src, "= |") && !strings.Contains(out, "= | ") {
			t.Errorf("nullary sum lost leading pipe: %s", out)
		}
	}
}
