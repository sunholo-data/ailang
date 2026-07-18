package executor

import (
	"strings"
	"testing"
)

const testRepoURL = "https://github.com/sunholo-data/ailang.git"
const testSHA = "806b3b4a4c0000000000000000000000000000ab"

func TestBuildClonePreamble_HEADReview(t *testing.T) {
	got, err := BuildClonePreamble(testRepoURL, "")
	if err != nil {
		t.Fatalf("BuildClonePreamble: %v", err)
	}
	// HEAD review = shallow clone of HEAD (probe-R recipe), no fetch-by-SHA.
	if !strings.Contains(got, "git clone --depth 1 "+testRepoURL) {
		t.Errorf("HEAD preamble missing shallow clone command:\n%s", got)
	}
	if strings.Contains(got, "git fetch") {
		t.Errorf("HEAD preamble must NOT use fetch-by-SHA:\n%s", got)
	}
	if !strings.Contains(got, "git rev-parse HEAD") {
		t.Errorf("preamble must ask the agent to echo git rev-parse HEAD:\n%s", got)
	}
}

func TestBuildClonePreamble_ArbitrarySHA_ShallowFetch(t *testing.T) {
	got, err := BuildClonePreamble(testRepoURL, testSHA)
	if err != nil {
		t.Fatalf("BuildClonePreamble: %v", err)
	}
	// Arbitrary SHA = shallow fetch-by-SHA (bounded), NOT a full clone.
	for _, want := range []string{
		"git init",
		"git remote add origin " + testRepoURL,
		"git fetch --depth 1 origin " + testSHA,
		"git checkout --detach FETCH_HEAD",
		"git rev-parse HEAD",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("fetch-by-SHA preamble missing %q:\n%s", want, got)
		}
	}
	// Must remain shallow — no unbounded full clone.
	if strings.Contains(got, "git clone") && !strings.Contains(got, "--depth 1") {
		t.Errorf("fetch-by-SHA preamble must not full-clone:\n%s", got)
	}
}

func TestBuildClonePreamble_Errors(t *testing.T) {
	if _, err := BuildClonePreamble("", ""); err == nil {
		t.Error("empty repo URL must error")
	}
	if _, err := BuildClonePreamble(testRepoURL, "deadbeef"); err == nil {
		t.Error("abbreviated (non-40-hex) SHA must error")
	}
}

func TestValidateCloneFlags(t *testing.T) {
	cases := []struct {
		name          string
		cloneRepo     string
		cloneSHA      string
		apiOnly       bool
		execName      string
		egressCapable bool
		wantErr       bool
		wantEgress    bool
	}{
		{"no flags no-op", "", "", false, "claude", false, false, false},
		{"HEAD clone on egress-capable ok", testRepoURL, "", false, "managed_agents", true, false, true},
		{"pinned clone on egress-capable ok", testRepoURL, testSHA, false, "managed_agents", true, false, true},
		{"clone on non-egress executor errors", testRepoURL, "", false, "claude", false, true, false},
		{"clone with api-only errors", testRepoURL, "", true, "managed_agents", true, true, false},
		{"clone-sha without clone-repo errors", "", testSHA, false, "managed_agents", true, true, false},
		{"invalid sha errors", testRepoURL, "xyz", false, "managed_agents", true, true, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotEgress, err := ValidateCloneFlags(tc.cloneRepo, tc.cloneSHA, tc.apiOnly, tc.execName, tc.egressCapable)
			if tc.wantErr != (err != nil) {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if gotEgress != tc.wantEgress {
				t.Errorf("requiresEgress = %v, want %v", gotEgress, tc.wantEgress)
			}
		})
	}
}
