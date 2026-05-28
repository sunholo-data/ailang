package notify

import "testing"

// Compile-time assertion that MacOSChannel satisfies Channel.
var _ Channel = MacOSChannel{}

func TestMacOSChannelName(t *testing.T) {
	if got := (MacOSChannel{}).Name(); got != "macos" {
		t.Fatalf("MacOSChannel.Name() = %q, want macos", got)
	}
}
