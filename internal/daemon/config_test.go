package daemon

import "testing"

// TestResolveExtraMessageSourcesWithSub locks in per-device subscription names:
// shared subs work-steal (laptop consumed the rig's prod ping, 2026-07-13), so
// the override must flow through — and empty must stay backward-compatible.
func TestResolveExtraMessageSourcesWithSub(t *testing.T) {
	out, err := ResolveExtraMessageSourcesWithSub("dev", []string{"prod"}, "messages-rig")
	if err != nil || len(out) != 1 {
		t.Fatalf("resolve: %v (n=%d)", err, len(out))
	}
	if out[0].MessagesSub != "messages-rig" {
		t.Errorf("override not applied: got %q", out[0].MessagesSub)
	}
	out, err = ResolveExtraMessageSourcesWithSub("dev", []string{"prod"}, "")
	if err != nil || len(out) != 1 || out[0].MessagesSub != "messages-laptop" {
		t.Errorf("empty override must keep default messages-laptop: %+v err=%v", out, err)
	}
}
