package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// validSHA is a well-formed 40-character object name used as the baseline
// commit in the tests below. It is not a real commit in this repository.
const validSHA = "0123456789abcdef0123456789abcdef01234567"

// okProbe returns a treeProbe that reports head and a clean tree — i.e. the
// state in which binaryMatchesTree must return true. Every refusal arm below
// perturbs exactly ONE input away from this baseline, so a mutation that
// neuters one refusal branch reds exactly one arm rather than a set.
func okProbe(head string) treeProbe {
	return treeProbe{
		head:  func() (string, bool) { return head, true },
		dirty: func([]string) (bool, bool) { return false, true },
	}
}

// TestBinaryMatchesTree_ProvesCurrentBinary is the positive control for the
// whole suppression path: with every precondition satisfied the function must
// return true. Without this arm the refusal arms below are all satisfiable by a
// function that returns false unconditionally.
func TestBinaryMatchesTree_ProvesCurrentBinary(t *testing.T) {
	if !binaryMatchesTree("v0.33.1-103-g0002c9b0b", validSHA, staleCheckDirs, okProbe(validSHA)) {
		t.Fatal("clean tree at the binary's own commit must prove the binary current")
	}
}

// TestBinaryMatchesTree_Refusals drives every refusal branch. The observable is
// a bool, which is over-subscribed by construction, so each case is built from
// the passing baseline and perturbs one field only; `why` names the branch the
// case is meant to reach so a failure identifies which one stopped refusing.
func TestBinaryMatchesTree_Refusals(t *testing.T) {
	cases := []struct {
		why     string
		version string
		commit  string
		probe   treeProbe
	}{
		{
			why:     "ldflags Version carries -dirty: the built tree is not addressed by any sha",
			version: "v0.33.1-103-g0002c9b0b-dirty",
			commit:  validSHA,
			probe:   okProbe(validSHA),
		},
		{
			// The refuser here is the sha-shape check, not a dedicated -dirty
			// branch: "-dirty" cannot occur inside 40 lowercase hex characters.
			why:     "ReadBuildInfo Commit carries -dirty, so it is not a bare sha",
			version: "v0.33.1-103-g0002c9b0b",
			commit:  validSHA + "-dirty",
			probe:   okProbe(validSHA + "-dirty"),
		},
		{
			why:     "commit is the ldflag-less default 'dev'",
			version: "dev",
			commit:  "dev",
			probe:   okProbe("dev"),
		},
		{
			why:     "commit is 'unknown' (git unavailable at build time)",
			version: "v0.33.1",
			commit:  "unknown",
			probe:   okProbe("unknown"),
		},
		{
			why:     "commit is abbreviated, so identity is not comparable",
			version: "v0.33.1",
			commit:  validSHA[:7],
			probe:   okProbe(validSHA[:7]),
		},
		{
			why:     "commit is 40 chars but not hex",
			version: "v0.33.1",
			commit:  strings.Repeat("z", 40),
			probe:   okProbe(strings.Repeat("z", 40)),
		},
		{
			// The value is deliberately the MATCHING sha, so the only thing
			// that can refuse here is the ok flag. A probe returning ("", false)
			// would be refused by the head != commit branch instead, and the arm
			// would pass without ever exercising determinability.
			why:     "HEAD is undeterminable even though the value would match",
			version: "v0.33.1",
			commit:  validSHA,
			probe: treeProbe{
				head:  func() (string, bool) { return validSHA, false },
				dirty: func([]string) (bool, bool) { return false, true },
			},
		},
		{
			why:     "HEAD is a different commit from the one the binary was built at",
			version: "v0.33.1",
			commit:  validSHA,
			probe:   okProbe(strings.Repeat("a", 40)),
		},
		{
			why:     "dirty state is undeterminable",
			version: "v0.33.1",
			commit:  validSHA,
			probe: treeProbe{
				head:  func() (string, bool) { return validSHA, true },
				dirty: func([]string) (bool, bool) { return false, false },
			},
		},
		{
			why:     "the sampled directories hold uncommitted changes",
			version: "v0.33.1",
			commit:  validSHA,
			probe: treeProbe{
				head:  func() (string, bool) { return validSHA, true },
				dirty: func([]string) (bool, bool) { return true, true },
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.why, func(t *testing.T) {
			if binaryMatchesTree(tc.version, tc.commit, staleCheckDirs, tc.probe) {
				t.Fatalf("must refuse to prove the binary current: %s", tc.why)
			}
		})
	}
}

// TestBinaryMatchesTree_DirtyProbeScopedToSampledDirs pins the invariant that
// makes the confirmation sound: the dirty probe must be asked about exactly the
// directories the mtime pre-filter sampled. A narrower scope would suppress a
// warning the pre-filter raised from a directory nobody checked for edits.
func TestBinaryMatchesTree_DirtyProbeScopedToSampledDirs(t *testing.T) {
	var got []string
	p := treeProbe{
		head:  func() (string, bool) { return validSHA, true },
		dirty: func(dirs []string) (bool, bool) { got = append([]string(nil), dirs...); return false, true },
	}
	if !binaryMatchesTree("v0.33.1", validSHA, staleCheckDirs, p) {
		t.Fatal("baseline must prove current")
	}
	if len(got) != len(staleCheckDirs) {
		t.Fatalf("dirty probe saw %d dirs, mtime walk samples %d: %v vs %v",
			len(got), len(staleCheckDirs), got, staleCheckDirs)
	}
	for i, d := range staleCheckDirs {
		if got[i] != d {
			t.Fatalf("dirty probe dir %d = %q, want %q", i, got[i], d)
		}
	}
}

// TestSourcesNewerThan covers the mtime pre-filter, including the two shapes
// that must NOT trip it: a directory that does not exist, and a newer file that
// is not Go source.
func TestSourcesNewerThan(t *testing.T) {
	base := time.Now()
	dir := t.TempDir()
	sub := filepath.Join(dir, "pkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	writeAt := func(name string, mod time.Time) string {
		p := filepath.Join(sub, name)
		if err := os.WriteFile(p, []byte("package pkg\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, mod, mod); err != nil {
			t.Fatal(err)
		}
		return p
	}

	old := writeAt("old.go", base.Add(-time.Hour))
	if sourcesNewerThan(base, []string{sub}) {
		t.Fatalf("a .go file older than the binary must not report stale (%s)", old)
	}

	if sourcesNewerThan(base, []string{filepath.Join(dir, "does-not-exist")}) {
		t.Fatal("a missing directory must contribute nothing")
	}

	writeAt("notes.txt", base.Add(time.Hour))
	if sourcesNewerThan(base, []string{sub}) {
		t.Fatal("a newer non-Go file must not report stale")
	}

	writeAt("new.go", base.Add(time.Hour))
	if !sourcesNewerThan(base, []string{sub}) {
		t.Fatal("a newer .go file must report stale")
	}
}

func TestIsFullSHA(t *testing.T) {
	if !isFullSHA(validSHA) {
		t.Fatalf("%q is a full sha", validSHA)
	}
	// The ReadBuildInfo dirty marker must be unrepresentable as a bare sha.
	// This is what lets binaryMatchesTree drop a dedicated -dirty branch for
	// Commit; if it ever became true, that branch would have to come back.
	if isFullSHA(validSHA + "-dirty") {
		t.Fatal("a -dirty commit must never read as a bare sha")
	}
	for _, bad := range []string{"", "dev", "unknown", validSHA[:39], validSHA + "0",
		strings.ToUpper(validSHA), strings.Repeat("g", 40)} {
		if isFullSHA(bad) {
			t.Fatalf("%q must not read as a full sha", bad)
		}
	}
}

// TestGitProbesAgainstRealCheckout exercises the real subprocess probes rather
// than the stubs above, in both directions. It asserts determinability, never a
// particular dirty state: the package directory is a live checkout and may hold
// a sprint's uncommitted edits.
func TestGitProbesAgainstRealCheckout(t *testing.T) {
	head, ok := gitHead(gitBinary(), ".")
	if !ok {
		t.Fatal("instrument failure: the package directory is a git checkout, so HEAD must be readable")
	}
	if !isFullSHA(head) {
		t.Fatalf("gitHead returned %q, want a full sha", head)
	}
	if _, ok := gitDirty(gitBinary(), ".", staleCheckDirs); !ok {
		t.Fatal("instrument failure: dirty state must be determinable in a checkout")
	}

	// Negative arm. It points at a path that does NOT EXIST rather than at a
	// plain temp directory: `git -C` walks UP the tree looking for a checkout,
	// so a temp dir that happened to sit inside one would make both probes
	// succeed and the arm would red for the runner's TMPDIR layout rather than
	// for the code. A missing path fails identically on every platform.
	outside := filepath.Join(t.TempDir(), "no-such-directory")
	if _, ok := gitHead(gitBinary(), outside); ok {
		t.Fatal("gitHead must not report a HEAD outside a checkout")
	}
	if _, ok := gitDirty(gitBinary(), outside, staleCheckDirs); ok {
		t.Fatal("gitDirty must not report a state outside a checkout")
	}
}

// TestStaleWarningNamesARealTarget pins the emitted advice, not a reconstruction
// of it: the warning tells the user to run `make quick-install`, so that target
// must exist. Reading it out of the make files is what makes this a check on the
// product's own output rather than on the test author's arithmetic.
func TestStaleWarningNamesARealTarget(t *testing.T) {
	found := false
	entries, err := os.ReadDir("../../make")
	if err != nil {
		t.Fatalf("instrument failure: make/ must be readable from the package dir: %v", err)
	}
	seen := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".mk") {
			continue
		}
		seen++
		b, err := os.ReadFile(filepath.Join("../../make", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "quick-install:") {
				found = true
			}
		}
	}
	if seen == 0 {
		t.Fatal("instrument failure: no .mk files found")
	}
	if !found {
		t.Fatal("the stale warning tells the user to run `make quick-install`, but no such target exists")
	}
}

// staleFixture builds a temp source dir whose .go file is NEWER than the
// binary time it returns, i.e. the state in which the mtime pre-filter fires.
// This is the shape a freshly created git worktree is always in.
func staleFixture(t *testing.T) (binaryTime time.Time, dirs []string) {
	t.Helper()
	binaryTime = time.Now().Add(-time.Hour)
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	if err := os.WriteFile(src, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(src, time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	return binaryTime, []string{dir}
}

// TestWarnIfStale_EmitsWhenTreeCannotProveCurrency is the positive control for
// the emitting path, and it asserts the MESSAGE TEXT rather than "some output":
// every silent arm below is a claim about absence, which many failures satisfy
// equally, so the suite needs one arm that pins what is actually written.
func TestWarnIfStale_EmitsWhenTreeCannotProveCurrency(t *testing.T) {
	binaryTime, dirs := staleFixture(t)
	var buf strings.Builder
	// Perturbed one field from the suppressing baseline: the tree is dirty.
	p := treeProbe{
		head:  func() (string, bool) { return validSHA, true },
		dirty: func([]string) (bool, bool) { return true, true },
	}
	warnIfStale(&buf, binaryTime, dirs, "v0.33.1", validSHA, p)

	got := buf.String()
	if !strings.Contains(got, "Binary may be stale (source files modified after build)") {
		t.Fatalf("missing the staleness sentence; got %q", got)
	}
	if !strings.Contains(got, "make quick-install") {
		t.Fatalf("missing the remedy command; got %q", got)
	}
}

// TestWarnIfStale_SilentWhenBinaryProvenCurrent is ailang#687 itself: a fresh
// worktree stamps every file with a current mtime, so the pre-filter fires on a
// binary that is byte-identical to the tree. With the commit matching and the
// tree clean, nothing may be written.
func TestWarnIfStale_SilentWhenBinaryProvenCurrent(t *testing.T) {
	binaryTime, dirs := staleFixture(t)
	if !sourcesNewerThan(binaryTime, dirs) {
		t.Fatal("instrument failure: the fixture must trip the mtime pre-filter")
	}
	var buf strings.Builder
	warnIfStale(&buf, binaryTime, dirs, "v0.33.1", validSHA, okProbe(validSHA))
	if got := buf.String(); got != "" {
		t.Fatalf("a binary proven current must warn about nothing; got %q", got)
	}
}

// TestWarnIfStale_SilentWhenSourcesOlder pins the pre-filter's own short
// circuit: with no source newer than the binary there is nothing to confirm, and
// in particular no git subprocess is run. The probe panics to prove that.
func TestWarnIfStale_SilentWhenSourcesOlder(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	if err := os.WriteFile(src, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(src, past, past); err != nil {
		t.Fatal(err)
	}
	neverProbe := treeProbe{
		head:  func() (string, bool) { panic("git must not be consulted when no source is newer") },
		dirty: func([]string) (bool, bool) { panic("git must not be consulted when no source is newer") },
	}
	var buf strings.Builder
	warnIfStale(&buf, time.Now(), []string{dir}, "v0.33.1", validSHA, neverProbe)
	if got := buf.String(); got != "" {
		t.Fatalf("an up-to-date binary must warn about nothing; got %q", got)
	}
}

// TestGitBinaryIsAbsolute pins the resolution contract: whatever git we run, we
// run it by absolute path. A relative or unresolvable result must degrade to
// "no git", which makes both probes undeterminable and therefore shows the
// warning rather than suppressing it.
func TestGitBinaryIsAbsolute(t *testing.T) {
	got := gitBinary()
	if got == "" {
		t.Skip("no git on PATH in this environment")
	}
	if !filepath.IsAbs(got) {
		t.Fatalf("git resolved to %q, which is not an absolute path", got)
	}
}

// TestResolveGitRefusesAnythingNotAbsolute drives the real resolver, not a
// re-implementation of its predicate, across every way it can fail. The
// positive arm is what stops the refusals passing vacuously.
func TestResolveGitRefusesAnythingNotAbsolute(t *testing.T) {
	// Derive the absolute path at runtime rather than writing a POSIX literal:
	// filepath.IsAbs("/usr/bin/git") is FALSE on windows, which needs a volume
	// name, so a hardcoded path would red for the platform and not for the code.
	absGit := filepath.Join(t.TempDir(), "git")
	if !filepath.IsAbs(absGit) {
		t.Fatalf("instrument failure: %q must be absolute on this platform", absGit)
	}
	if got := resolveGit(func() (string, error) { return absGit, nil }); got != absGit {
		t.Fatalf("an absolute path must be accepted; got %q", got)
	}
	for _, tc := range []struct {
		why  string
		path string
		err  error
	}{
		{"git is not on PATH at all", "", exec.ErrNotFound},
		{"lookup errored but still returned a usable-looking path", "ERRPATH", exec.ErrNotFound},
		{"git resolved to a RELATIVE path", "bin/git", nil},
		{"git resolved to a bare name", "git", nil},
		{"git resolved to a CWD-relative path", "./git", nil},
	} {
		t.Run(tc.why, func(t *testing.T) {
			path := tc.path
			if path == "ERRPATH" {
				path = absGit // absolute AND erroring: only the error may refuse it
			}
			if got := resolveGit(func() (string, error) { return path, tc.err }); got != "" {
				t.Fatalf("must refuse %q (%s); got %q", tc.path, tc.why, got)
			}
		})
	}
}

// TestGitProbesRefuseWithoutResolvableGit pins the CONTRACT for an unresolvable
// git: undeterminable, never a confident answer. Note honestly what it does not
// do — neutering the empty-git guard leaves this green, because exec refuses an
// empty command name with the same error. The guard is a declared fast path,
// not a pinned branch, and this arm pins the outcome rather than the mechanism.
func TestGitProbesRefuseWithoutResolvableGit(t *testing.T) {
	if _, ok := gitHead("", "."); ok {
		t.Fatal("with no resolvable git, HEAD must be undeterminable")
	}
	if _, ok := gitDirty("", ".", staleCheckDirs); ok {
		t.Fatal("with no resolvable git, dirty state must be undeterminable")
	}
	// Control: the same call in the same directory IS determinable with a real
	// git, so the refusals above are about the empty binary and not the path.
	if _, ok := gitHead(gitBinary(), "."); !ok {
		t.Fatal("instrument failure: HEAD must be readable here with a real git")
	}
}
