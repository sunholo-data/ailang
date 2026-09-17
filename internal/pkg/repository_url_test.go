package pkg

import "testing"

func TestParseRepositoryURL(t *testing.T) {
	cases := []struct {
		in   string
		want RepoRef
		ok   bool
	}{
		{"https://github.com/sunholo-data/ailang-packages/tree/main/packages/gcp-auth", RepoRef{"sunholo-data/ailang-packages", "main", "packages/gcp-auth"}, true},
		{"https://github.com/sunholo-data/daneel/tree/main/ext/calendar", RepoRef{"sunholo-data/daneel", "main", "ext/calendar"}, true},
		{"https://github.com/sunholo-data/ailang-parse", RepoRef{"sunholo-data/ailang-parse", "", ""}, true},
		{"https://github.com/sunholo-data/ailang-parse.git", RepoRef{"sunholo-data/ailang-parse", "", ""}, true},
		{"https://github.com/sunholo-data/ailang-parse/tree/dev", RepoRef{"sunholo-data/ailang-parse", "dev", ""}, true},
		{"https://github.com/sunholo-data/ailang/blob/dev/README.md", RepoRef{}, false},
		{"https://gitlab.com/x/y", RepoRef{}, false},
		{"", RepoRef{}, false},
		{"not a url", RepoRef{}, false},
	}
	for _, c := range cases {
		got, ok := ParseRepositoryURL(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("ParseRepositoryURL(%q) = %+v,%v want %+v,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPackageAgentID(t *testing.T) {
	if got := PackageAgentID("sunholo/gcp_auth"); got != "pkg-sunholo-gcp-auth" {
		t.Errorf("got %q", got)
	}
	if got := PackageAgentID("sunholo/daneel_ext_help"); got != "pkg-sunholo-daneel-ext-help" {
		t.Errorf("got %q", got)
	}
}
