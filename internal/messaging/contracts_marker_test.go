package messaging

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/iface"
)

// The marker is defined twice so internal/messaging need not import iface. The
// check lives here, not in iface: a core package must not import a dashboard
// package, even from a test (make check-boundaries).
func TestContractsMarker_MatchesIface(t *testing.T) {
	if iface.ContractsMarker != SignatureContractsMarker {
		t.Fatalf("iface %q != messaging %q", iface.ContractsMarker, SignatureContractsMarker)
	}
}
