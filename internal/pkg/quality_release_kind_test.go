package pkg

import (
	"strings"
	"testing"
)

// TST-PKGPAR-04
// The declared [release] kind is checked against the measured class: a
// non-breaking claim over a measured C warns (then gates), "breaking" over a
// measured A is informational, and an unmeasurable pair raises nothing.
func TestPUB025_ReleaseKindVsMeasured(t *testing.T) {
	cases := []struct {
		kind, class string
		hard        bool
		wantGate    bool
		wantBadge   string // "" = no PUB025 badge
	}{
		{"fix", "C", false, false, "warn"},
		{"security", "C", false, false, "warn"},
		{"feature", "C", false, false, "warn"},
		{"fix", "C", true, true, ""},
		{"breaking", "C", true, false, ""},
		{"feature", "B", true, false, ""},
		{"fix", "A", true, false, ""},
		{"breaking", "A", false, false, "info"},
		{"fix", "U", true, false, ""},
		{"fix", "", true, false, ""},
	}
	for _, tc := range cases {
		m := qualityManifest("experimental", []string{})
		m.Release.Kind = tc.kind
		in := cleanInputs()
		in.MeasuredChangeClass, in.MeasuredAgainst, in.ReleaseKindGatesHard = tc.class, "0.9.0", tc.hard
		r := BuildQualityReport(m, ModeServer, in, false)
		if got := hasCode(r.Gates, "PUB025"); got != tc.wantGate {
			t.Errorf("kind=%s class=%s hard=%v: gate=%v, want %v (%v)", tc.kind, tc.class, tc.hard, got, tc.wantGate, r.Gates)
		}
		level := ""
		for _, b := range r.Badges {
			if b.Code == "PUB025" {
				level = b.Level
				if !strings.Contains(b.Msg, "0.9.0") {
					t.Errorf("PUB025 must name the version it measured against: %q", b.Msg)
				}
			}
		}
		if level != tc.wantBadge {
			t.Errorf("kind=%s class=%s hard=%v: badge level %q, want %q", tc.kind, tc.class, tc.hard, level, tc.wantBadge)
		}
		if r.Release.MeasuredClass != tc.class || (tc.class != "" && r.Release.MeasuredAgainst != "0.9.0") {
			t.Errorf("release section = %+v", r.Release)
		}
	}
}

func TestReleaseKindGatesHard_GraceWindow(t *testing.T) {
	for v, want := range map[string]bool{"v0.63.0": false, "0.63.9-3-gabc": false, "v0.64.0": true, "v1.0.0": true, "dev": false} {
		if got := ReleaseKindGatesHard(v); got != want {
			t.Errorf("ReleaseKindGatesHard(%q) = %v, want %v", v, got, want)
		}
	}
	// The refactor must leave the PUB001/PUB002 schedule untouched.
	if ReleaseGatesHard("v0.40.9") || !ReleaseGatesHard("v0.41.0") {
		t.Error("ReleaseGatesHard schedule changed")
	}
}

func TestPreviousPublishedVersion(t *testing.T) {
	idx := &RegistryIndex{Packages: []IndexEntry{
		{Name: "other/x", Versions: []string{"9.9.9"}},
		{Name: "test/q", Versions: []string{"0.1.0", "0.2.0", "0.3.0"}},
	}}
	cases := map[string]string{"0.4.0": "0.3.0", "0.3.0": "0.2.0"}
	for newV, want := range cases {
		if got := PreviousPublishedVersion(idx, "test/q", newV); got != want {
			t.Errorf("previous of %s = %q, want %q", newV, got, want)
		}
	}
	if PreviousPublishedVersion(idx, "absent/p", "1.0.0") != "" || PreviousPublishedVersion(nil, "test/q", "1.0.0") != "" {
		t.Error("absent package / nil index must yield \"\"")
	}
	first := &RegistryIndex{Packages: []IndexEntry{{Name: "test/q", Versions: []string{"0.1.0"}}}}
	if PreviousPublishedVersion(first, "test/q", "0.1.0") != "" {
		t.Error("a first publish has no previous version")
	}
}
