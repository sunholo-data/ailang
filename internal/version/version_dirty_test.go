package version

import "testing"

// TestDirty: Commit alone identifies a build only when it is a clean,
// VCS-stamped SHA. Makefile builds put "-dirty" in Version (not Commit);
// go build/run/test builds carry vcs.modified; no VCS info leaves "dev".
func TestDirty(t *testing.T) {
	saveV, saveC, saveM := Version, Commit, vcsModified
	defer func() { Version, Commit, vcsModified = saveV, saveC, saveM }()

	cases := []struct {
		name     string
		version  string
		commit   string
		modified bool
		want     bool
	}{
		{"clean release build", "v0.43.1", "45a456a95abc", false, false},
		{"Makefile dirty build (Version only)", "v0.43.1-3-gabc-dirty", "45a456a95abc", false, true},
		{"go build of a dirty tree", "dev", "45a456a95abc", true, true},
		{"ReadBuildInfo dirty suffix", "dev", "45a456a95abc-dirty", false, true},
		{"no VCS info at all", "dev", "dev", false, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			Version, Commit, vcsModified = c.version, c.commit, c.modified
			if got := Dirty(); got != c.want {
				t.Errorf("Dirty() = %v, want %v", got, c.want)
			}
		})
	}
}
