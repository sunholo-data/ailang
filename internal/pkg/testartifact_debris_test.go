package pkg

// Packages that already carry a `_namedtest_body_<n>.ail` left by an
// interrupted `ailang test` of an older binary (#1502): quality refuses them
// (PUB024), and the tarball, content hash, smoke staging and source
// discovery are identical to the clean tree.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const debrisName = "_namedtest_body_3176816492.ail"

// debrisPackage writes a minimal package; with debris, it also writes a stale
// body copy at the root and one in src/.
func debrisPackage(t *testing.T, withDebris bool) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		ManifestFile:    "[package]\nname = \"test/pkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[exports]\nmodules = [\"test/pkg/core\"]\n",
		"core.ail":      "module test/pkg/core\n\nexport pure func one() -> int = 1\n",
		"core_test.ail": "module test/pkg/core_test\n\ntest \"one\" { 1 == 1 }\n",
	}
	if withDebris {
		files[debrisName] = "module test/pkg/core_test\n\ntest \"one\" { 1 == 1 }\n{ 1 == 1 }\n"
		files[filepath.Join("src", "_namedtest_body_7.ail")] = "module test/pkg/src/x\n"
	}
	for rel, body := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestFindNamedTestBodyFiles(t *testing.T) {
	got, err := FindNamedTestBodyFiles(debrisPackage(t, true))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{debrisName, "src/_namedtest_body_7.ail"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("FindNamedTestBodyFiles = %q, want %q (sorted, slash-separated, relative)", got, want)
	}
	clean, err := FindNamedTestBodyFiles(debrisPackage(t, false))
	if err != nil || len(clean) != 0 {
		t.Errorf("clean package: %q, %v; want none", clean, err)
	}
}

func TestCreateTarball_ExcludesNamedTestBodies(t *testing.T) {
	clean, err := CreateTarball(debrisPackage(t, false))
	if err != nil {
		t.Fatal(err)
	}
	dirty, err := CreateTarball(debrisPackage(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if TarballHash(clean) != TarballHash(dirty) {
		t.Errorf("tarball with a stale named-test body differs from the clean tree's")
	}
}

func TestContentHash_ExcludesNamedTestBodies(t *testing.T) {
	clean, err := ContentHash(debrisPackage(t, false))
	if err != nil {
		t.Fatal(err)
	}
	dirty, err := ContentHash(debrisPackage(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if clean != dirty {
		t.Errorf("content hash depends on a stale named-test body: %s vs %s", clean, dirty)
	}
}

func TestShouldStageFile_ExcludesNamedTestBodies(t *testing.T) {
	if shouldStageFile(debrisName) || shouldStageFile(filepath.Join("src", "_namedtest_body_7.ail")) {
		t.Errorf("a stale named-test body is staged into the smoke workspace")
	}
	if !shouldStageFile("core.ail") {
		t.Errorf("core.ail is no longer staged")
	}
}

func TestDiscoverPackageSources_SkipsNamedTestBodies(t *testing.T) {
	src, err := DiscoverPackageSources(debrisPackage(t, true))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range append(append([]string{}, src.OrphanFiles...), src.TestFiles...) {
		if IsNamedTestBodyFile(filepath.Base(f)) {
			t.Errorf("stale named-test body discovered as a package file: %s", f)
		}
	}
}

func TestScaffoldGitignore(t *testing.T) {
	dir := t.TempDir()
	if err := ScaffoldGitignore(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), NamedTestBodyGitignore+"\n") {
		t.Fatalf("new .gitignore lacks %q:\n%s", NamedTestBodyGitignore, got)
	}

	// An existing .gitignore keeps its lines, gains the entry once, and a
	// second run changes nothing.
	existing := "build/\n*.log" // no trailing newline
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := ScaffoldGitignore(dir); err != nil {
			t.Fatal(err)
		}
	}
	got, _ = os.ReadFile(filepath.Join(dir, ".gitignore"))
	if want := existing + "\n" + NamedTestBodyGitignore + "\n"; string(got) != want {
		t.Errorf(".gitignore = %q, want %q", got, want)
	}
}

func TestBuildQualityReport_PUB024NamesStaleBodies(t *testing.T) {
	m := qualityManifest("experimental", []string{})
	if r := BuildQualityReport(m, ModePublisher, cleanInputs(), false); hasCode(r.Gates, "PUB024") || hasCode(r.Badges, "PUB024") {
		t.Fatalf("clean inputs raised PUB024: %v %v", r.Gates, r.Badges)
	}
	in := cleanInputs()
	in.NamedTestBodyFiles = []string{debrisName, "src/_namedtest_body_7.ail"}
	r := BuildQualityReport(m, ModePublisher, in, false)
	if !hasCode(r.Gates, "PUB024") {
		t.Fatalf("stale named-test bodies must be a PUB024 gate (blocks publish), got gates %v", r.Gates)
	}
	for _, f := range r.Gates {
		if f.Code != "PUB024" {
			continue
		}
		for _, want := range []string{debrisName, "src/_namedtest_body_7.ail", "delete"} {
			if !strings.Contains(f.Msg, want) {
				t.Errorf("PUB024 message %q does not mention %q", f.Msg, want)
			}
		}
	}
}
