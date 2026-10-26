package pkg

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// runGit runs git for test fixtures, isolated from the developer's config.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// makeGitPackage creates a repo holding test/gitlib, tagged v0.1.0 and
// v0.2.0 with different module content. Returns the repo dir (usable as a
// git URL).
func makeGitPackage(t *testing.T) string {
	t.Helper()
	requireGit(t)
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	writeFile(t, filepath.Join(repo, "ailang.toml"), "[package]\nname = \"test/gitlib\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	writeFile(t, filepath.Join(repo, "src", "test", "gitlib", "lib.ail"), "module test/gitlib/lib\nexport func one() -> int = 1\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "v1")
	runGit(t, repo, "tag", "v0.1.0")
	writeFile(t, filepath.Join(repo, "src", "test", "gitlib", "lib.ail"), "module test/gitlib/lib\nexport func one() -> int = 2\n")
	runGit(t, repo, "commit", "-qam", "v2")
	runGit(t, repo, "tag", "v0.2.0")
	return repo
}

func appWithGitDep(t *testing.T, url, tag string) (string, *PackageManifest) {
	t.Helper()
	app := t.TempDir()
	writeManifest(t, app, fmt.Sprintf("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/gitlib\" = { git = %q, tag = %q }\n", filepath.ToSlash(url), tag))
	m, err := LoadManifest(app)
	if err != nil {
		t.Fatal(err)
	}
	return app, m
}

// snapshotTree records every path under root with its size and mode, so a
// confined call can be shown to leave it byte-for-byte alone.
func snapshotTree(t *testing.T, root string) string {
	t.Helper()
	var lines []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		lines = append(lines, fmt.Sprintf("%s %v %d %d", filepath.ToSlash(rel), info.Mode(), info.Size(), info.ModTime().UnixNano()))
		return nil
	})
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// lockUnconfined resolves as an operator would (no policy), into home.
func lockUnconfined(t *testing.T, app string, m *PackageManifest) *LockFile {
	t.Helper()
	t.Setenv(config.EnvAgentPolicy, "")
	lf, err := ResolveLockFrom(m, app, "test", nil)
	if err != nil {
		t.Fatalf("unconfined lock: %v", err)
	}
	return lf
}

// D2: a confined lock carries a provisioned, pinned git entry forward with
// no git at all; every other shape is the canonical refusal.
func TestResolveGitConfined_CarryForwardMatrix(t *testing.T) {
	repo := makeGitPackage(t)
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	app, m := appWithGitDep(t, repo, "v0.1.0")
	prev := lockUnconfined(t, app, m)
	entry, ok := prev.FindPackage("test/gitlib")
	if !ok || entry.GitTag != "v0.1.0" || entry.GitRev == "" {
		t.Fatalf("unconfined lock entry: %+v", entry)
	}
	if _, err := os.Stat(filepath.Join(home, ".ailang", "cache", "git")); err != nil {
		t.Fatalf("instrument blind: the unconfined lock did not clone into HOME: %v", err)
	}

	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	before := snapshotTree(t, home)
	// PATH without git: the carry-forward must not need it.
	t.Setenv("PATH", t.TempDir())

	got, err := ResolveLockFrom(m, app, "test", prev)
	if err != nil {
		t.Fatalf("carry-forward refused: %v", err)
	}
	carried, _ := got.FindPackage("test/gitlib")
	if carried == nil || !reflect.DeepEqual(*carried, *entry) {
		t.Fatalf("carried entry differs:\n got %+v\nwant %+v", carried, entry)
	}

	refused := func(name string, m *PackageManifest, prev *LockFile, want string) {
		t.Helper()
		_, err := ResolveDependenciesFrom(m, app, prev)
		if !IsConfinedError(err) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want confined refusal containing %q, got %v", name, want, err)
		}
	}
	refused("no prior lock", m, nil, "no previous ailang.lock")
	_, m2 := appWithGitDep(t, repo, "v0.2.0")
	refused("tag changed", m2, prev, `tag "v0.2.0"`)
	_, m3 := appWithGitDep(t, repo+"-other", "v0.1.0")
	refused("url changed", m3, prev, "git url changed")
	m4 := *m
	m4.Dependencies = map[string]Dependency{"test/gitlib": {Git: m.Dependencies["test/gitlib"].Git, Tag: "v0.1.0", Subdir: "pkg"}}
	refused("subdir changed", &m4, prev, "subdir changed")
	old := &LockFile{Packages: []LockedPackage{*entry}}
	old.Packages[0].GitTag = ""
	refused("lock predates git_tag", m, old, "is not the one ailang.lock pins")

	if after := snapshotTree(t, home); after != before {
		t.Fatalf("confined resolution changed HOME:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	// Tampered provisioned content: refusal, not a silent carry.
	dir, err := gitPackageDir(entry)
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(dir, "src", "test", "gitlib", "lib.ail")
	if err := os.WriteFile(f, []byte("module test/gitlib/lib\nexport func one() -> int = 99\n"), 0644); err != nil {
		t.Fatal(err)
	}
	refused("content tampered", m, prev, "does not match locked")
}

// Unconfined, an explicit rev records no tag; carry-forward then matches on rev.
func TestResolveGitConfined_RevPin(t *testing.T) {
	repo := makeGitPackage(t)
	testutil.SetHomeDir(t, t.TempDir())
	rev := runGit(t, repo, "rev-parse", "v0.1.0")
	app := t.TempDir()
	writeManifest(t, app, fmt.Sprintf("[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/gitlib\" = { git = %q, rev = %q }\n", filepath.ToSlash(repo), rev))
	m, err := LoadManifest(app)
	if err != nil {
		t.Fatal(err)
	}
	prev := lockUnconfined(t, app, m)
	if e, _ := prev.FindPackage("test/gitlib"); e.GitTag != "" || e.GitRev != rev {
		t.Fatalf("rev pin entry: %+v", e)
	}
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	if _, err := ResolveDependenciesFrom(m, app, prev); err != nil {
		t.Fatalf("rev carry-forward refused: %v", err)
	}
}

// T1/T2 (D4): git content is verified at check time and drift is fatal.
func TestValidateGitContentHashes(t *testing.T) {
	repo := makeGitPackage(t)
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	app, m := appWithGitDep(t, repo, "v0.1.0")
	lf := lockUnconfined(t, app, m)
	if err := lf.ValidateContentHashesFrom(app); err != nil {
		t.Fatalf("fresh lock does not validate: %v", err)
	}
	entry, _ := lf.FindPackage("test/gitlib")
	clone, _ := gitPackageDir(entry)

	// Another project checks out a different tag in the shared URL-keyed clone.
	runGit(t, clone, "fetch", "-q", "origin", "tag", "v0.2.0")
	runGit(t, clone, "checkout", "-q", "v0.2.0")
	err := lf.ValidateContentHashesFrom(app)
	var rce *RegistryContentError
	if !errors.As(err, &rce) || !strings.Contains(err.Error(), "test/gitlib content changed") || !strings.Contains(err.Error(), entry.ContentHash) {
		t.Fatalf("shared-clone drift not fatal: %v", err)
	}
	runGit(t, clone, "checkout", "-q", "v0.1.0")
	if err := lf.ValidateContentHashesFrom(app); err != nil {
		t.Fatalf("restored clone does not validate: %v", err)
	}

	// Missing clone: fatal too.
	if err := os.RemoveAll(clone); err != nil {
		t.Fatal(err)
	}
	if err := lf.ValidateContentHashesFrom(app); !errors.As(err, &rce) {
		t.Fatalf("missing git dep not fatal: %v", err)
	}
}

// D3: with AILANG_PACKAGE_ROOT set, git packages resolve only from
// <root>/<vendor>/<name>/<version>, and a confined loader never touches HOME.
func TestGitPackageRoot(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	root := t.TempDir()
	t.Setenv(config.EnvPackageRoot, root)
	t.Setenv(config.EnvAgentPolicy, "policy.toml")
	locked := LockedPackage{Name: "test/gitlib", Version: "0.1.0", Source: "git", GitURL: "https://example.invalid/x.git"}
	loader := NewPackageLoader(&LockFile{Packages: []LockedPackage{locked}}, t.TempDir())

	if _, err := loader.packageDir(&locked); err == nil || !strings.Contains(err.Error(), "not provisioned in AILANG_PACKAGE_ROOT") {
		t.Fatalf("missing package: %v", err)
	}
	want := filepath.Join(root, "test", "gitlib", "0.1.0")
	writeFile(t, filepath.Join(want, "ailang.toml"), "[package]\nname = \"test/gitlib\"\nversion = \"0.1.0\"\nedition = \"1\"\n")
	if err := os.Chmod(want, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(want, 0755) })
	got, err := loader.packageDir(&locked)
	if err != nil || got != want {
		t.Fatalf("packageDir = %q, %v; want %q", got, err, want)
	}
	assertHomeUntouched(t, home)
}

// Row 14: the loader's git case is write-free (NewGitCache created
// ~/.ailang/cache/git on every check of a project with a git dependency).
func TestLoaderGitCaseCreatesNothing(t *testing.T) {
	home := confine(t)
	if _, err := NewGitCache(); err != nil {
		t.Fatal(err)
	}
	locked := LockedPackage{Name: "test/gitlib", Version: "0.1.0", Source: "git", GitURL: "https://example.invalid/x.git"}
	loader := NewPackageLoader(&LockFile{Packages: []LockedPackage{locked}}, t.TempDir())
	if _, err := loader.packageDir(&locked); err == nil {
		t.Fatal("expected a cache-not-found error")
	}
	assertHomeUntouched(t, home)
}

// Rows 9–13: a confined git cache operation is refused before any write.
func TestGitCacheConfinedResolve(t *testing.T) {
	repo := makeGitPackage(t)
	home := confine(t)
	cache, err := NewGitCache()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := cache.Resolve(repo, "v0.1.0", "", ""); !IsConfinedError(err) {
		t.Fatalf("confined clone not refused: %v", err)
	}
	assertHomeUntouched(t, home)
}

// AC10: git argv hardening at the manifest.
func TestManifestRejectsLeadingDashGitValues(t *testing.T) {
	for _, dep := range []string{
		`{ git = "--upload-pack=touch x", tag = "v1" }`,
		`{ git = "https://example.com/r", tag = "--branch=x" }`,
		`{ git = "https://example.com/r", rev = "-x" }`,
		`{ git = "https://example.com/r", tag = "v1", subdir = "-p" }`,
	} {
		dir := t.TempDir()
		writeManifest(t, dir, "[package]\nname = \"test/app\"\nversion = \"0.1.0\"\nedition = \"1\"\n[dependencies]\n\"test/x\" = "+dep+"\n")
		if _, err := LoadManifest(dir); err == nil || !strings.Contains(err.Error(), "must not begin with '-'") {
			t.Errorf("%s: not rejected: %v", dep, err)
		}
	}
}
