package main

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormaliseCode(t *testing.T) {
	cases := []struct{ code, msg, want string }{
		{"PAR001", "anything", "PAR001"},
		{"ERROR", "module loading failed: MOD010: module path mismatch", "MOD010"},
		{"ERROR", "parse: TC_ARITY_001 wrong arity", "TC_ARITY_001"},
		{"ERROR", "IMP012_UNSUPPORTED_NAMESPACE: no", "IMP012_UNSUPPORTED_NAMESPACE"},
		{"ERROR", "effect checking failed in a/b: x\n  The pure keyword contradicts the declared effect row ! {IO}.", "EFFECT/pure-with-row"},
		{"ERROR", "type error: information-flow violation: <email>", "IFC/flow-violation"},
		{"ERROR", "type error in m (decl 0): cannot unify string vs int", "TYPE_ERROR/unify"},
		{"ERROR", "type error in m: undefined variable foo", "TYPE_ERROR/unbound"},
		{"ERROR", "failed to resolve package import sunholo/x", "DEP/unresolved"},
		{"ERROR", "effect checking failed in x: something new", "EFFECT/other"},
		{"ERROR", "MAX_SIZE exceeded in user constant", uncodedPrefix + "max-size-exceeded-in-user-constant"},
	}
	for _, c := range cases {
		if got := normaliseCode(c.code, c.msg); got != c.want {
			t.Errorf("normaliseCode(%q, %q) = %q, want %q", c.code, c.msg, got, c.want)
		}
	}
	// One cause in different modules is one bucket.
	a := normaliseCode("ERROR", "weird failure in pkg/alpha/mod at alpha/mod.ail:3:4")
	b := normaliseCode("ERROR", "weird failure in pkg/beta/other at beta/other.ail:9:1")
	if a != b || !strings.HasPrefix(a, uncodedPrefix) {
		t.Errorf("fingerprints differ across modules: %q vs %q", a, b)
	}
}

func TestGlobRegexp(t *testing.T) {
	cases := []struct {
		glob, path string
		want       bool
	}{
		{"src/**/*.ail", "src/a/b/c.ail", true},
		{"src/**/*.ail", "src/c.ail", true},
		{"src/**/*.ail", "srcx/c.ail", false},
		{"*.ail", "a/b.ail", false},
		{".agent/**", ".agent/x/y.ail", true},
		{"**/fixtures/reject_*.ail", "a/fixtures/reject_1.ail", true},
		{"**/fixtures/reject_*.ail", "fixtures/reject_1.ail", true},
		{"examples/expected_fail/**", "examples/expected_fail/x.ail", true},
	}
	for _, c := range cases {
		re, err := globRegexp(c.glob)
		if err != nil {
			t.Fatal(err)
		}
		if got := re.MatchString(c.path); got != c.want {
			t.Errorf("glob %q on %q = %v, want %v", c.glob, c.path, got, c.want)
		}
	}
}

func TestFileEntryRootRule(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("repo/src/a/b.ail", "-- c\nmodule a/b\n")
	write("pkgs/p/ailang.toml", "")
	write("pkgs/p/lib/x.ail", "module wrong/name\n")
	write("loose/y.ail", "module other\n")
	for rel, want := range map[string][3]string{
		"repo/src/a/b.ail": {"repo/src", "a/b.ail", "module-suffix"},
		"pkgs/p/lib/x.ail": {"pkgs/p", "lib/x.ail", "package-root"},
		"loose/y.ail":      {"loose", "y.ail", "dirname"},
	} {
		e := fileEntry("c", root, rel)
		gotCwd, _ := filepath.Rel(root, e.Cwd)
		if filepath.ToSlash(gotCwd) != want[0] || e.Arg != want[1] || e.Rule != want[2] {
			t.Errorf("%s: cwd=%s arg=%s rule=%s, want %v", rel, gotCwd, e.Arg, e.Rule, want)
		}
	}
}

func TestVersionAfter(t *testing.T) {
	if !versionAfter("AILANG v0.58.1-3-gabc", "v0.58.0") || versionAfter("v0.58.0", "v0.58.0") ||
		versionAfter("v0.57.9", "v0.58.0") || versionAfter("dev", "v0.1.0") {
		t.Error("versionAfter ordering wrong")
	}
}

func TestExtractTarRejectsEscape(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "top/../../evil.ail", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	dest := t.TempDir()
	if _, err := extractTar(&buf, dest, 1); err == nil || !strings.Contains(err.Error(), "unsafe") {
		t.Fatalf("extractTar accepted an escaping path: %v", err)
	}
}

func TestManifestValidation(t *testing.T) {
	dir := t.TempDir()
	for name, body := range map[string]string{
		"unknown key":  "- name: a\n  repo: o/a\n  tier: {ci: off, release: blocking}\n  blokcing: true\n",
		"missing tier": "- name: a\n  repo: o/a\n  tier: {ci: off}\n",
		"bad repo":     "- name: a\n  repo: a\n  tier: {ci: off, release: blocking}\n",
		"duplicate":    "- name: a\n  repo: o/a\n  tier: {ci: off, release: off}\n- name: a\n  repo: o/b\n  tier: {ci: off, release: off}\n",
	} {
		p := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".yaml")
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadCorpora(p); err == nil {
			t.Errorf("%s: LoadCorpora accepted it", name)
		}
	}
}

func TestSelectCorpora(t *testing.T) {
	all := []Corpus{
		{Name: "a", Tier: TierSet{CI: tierAdvisory, Release: tierBlocking}},
		{Name: "b", Tier: TierSet{CI: tierOff, Release: tierBlocking}},
		{Name: "c", Tier: TierSet{CI: tierAdvisory, Release: tierAdvisory}},
	}
	names := func(cs []Corpus) string {
		var out []string
		for _, c := range cs {
			out = append(out, c.Name)
		}
		return strings.Join(out, ",")
	}
	for _, tc := range []struct{ tier, want, got string }{
		{"release", "", "a,b,c"},
		{"ci", "", "a,c"},
		{"release", "b", "b"}, // narrowing to one must not re-admit the rest
		{"release", "a,c", "a,c"},
	} {
		got, err := selectCorpora(all, tc.tier, tc.want)
		if err != nil || names(got) != tc.got {
			t.Errorf("tier %s CORPORA=%q: got %s (%v), want %s", tc.tier, tc.want, names(got), err, tc.got)
		}
	}
	for _, bad := range []struct{ tier, want string }{{"release", "zzz"}, {"ci", "b"}} {
		if _, err := selectCorpora(all, bad.tier, bad.want); err == nil {
			t.Errorf("tier %s CORPORA=%q accepted", bad.tier, bad.want)
		}
	}
}

// The committed manifest and expected-breaks file must load: they are what the
// release tier reads.
func TestCommittedManifestsLoad(t *testing.T) {
	cs, err := LoadCorpora("corpora.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadExpected("expected-breaks.yaml", cs); err != nil {
		t.Fatal(err)
	}
	for _, c := range cs {
		if c.Repo == selfRepo && len(selfPaths(c)) == 0 {
			t.Errorf("in-repo corpus %s has no literal include root", c.Name)
		}
	}
}
