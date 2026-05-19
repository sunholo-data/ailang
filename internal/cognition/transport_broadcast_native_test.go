//go:build !js

package cognition

import (
	"testing"
)

// ============================================================================
// BroadcastChannel non-js compile stub
// ============================================================================
//
// The BroadcastChannelTransport is build-tagged `js && wasm` — it only
// compiles in browser WASM. This test file (tagged !js) just documents
// that absence and asserts the substrate types are available on the
// native side without the BroadcastChannel impl. Real browser tests
// live in the M-COG-MESH browser integration suite.

// TestBroadcastChannel_NotAvailableOnNative is a documentary anchor:
// BroadcastChannel is browser-only. Native code must not assume the
// transport is present. The compile-time guard is the build tag on
// transport_broadcast_js.go; this test exists to make the deliberate
// absence visible in the test suite.
func TestBroadcastChannel_NotAvailableOnNative(t *testing.T) {
	t.Log("BroadcastChannelTransport is build-tagged 'js && wasm' — native runtime uses LocalWorker")
}
