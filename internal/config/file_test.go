package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

type ghSection struct {
	DefaultRepo  string   `yaml:"default_repo"`
	ExpectedUser string   `yaml:"expected_user"`
	WatchLabels  []string `yaml:"watch_labels"`
}

func writeConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadHonoursAilangConfigThenHome(t *testing.T) {
	home := t.TempDir()
	testutil.SetHomeDir(t, home)
	t.Setenv(EnvConfigFile, "")
	resetFileCacheForTest()
	t.Cleanup(resetFileCacheForTest)

	if _, err := Load(); !errors.Is(err, ErrConfigNotFound) {
		t.Fatalf("no file: err = %v, want ErrConfigNotFound", err)
	}

	writeConfig(t, filepath.Join(home, ".ailang", "config.yaml"), "github:\n  default_repo: from-home\n")
	f, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var gh ghSection
	if ok, err := f.Section("github", &gh); err != nil || !ok || gh.DefaultRepo != "from-home" {
		t.Fatalf("home: ok=%v repo=%q err=%v", ok, gh.DefaultRepo, err)
	}

	alt := filepath.Join(t.TempDir(), "alt.yaml")
	writeConfig(t, alt, "github:\n  default_repo: from-alt\n")
	t.Setenv(EnvConfigFile, alt)
	f, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if f.Path != alt {
		t.Fatalf("Path = %q, want %q", f.Path, alt)
	}
	gh = ghSection{}
	if _, err := f.Section("github", &gh); err != nil || gh.DefaultRepo != "from-alt" {
		t.Fatalf("%s must win over the home file: %q, %v", EnvConfigFile, gh.DefaultRepo, err)
	}
}

func TestLoadCachesUntilTheFileChanges(t *testing.T) {
	resetFileCacheForTest()
	t.Cleanup(resetFileCacheForTest)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "github:\n  default_repo: one\n")

	f1, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	f2, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if f1 != f2 {
		t.Fatal("an unchanged file must come back from the cache, not be re-parsed")
	}

	// Change the file (different size, and a later mtime for good measure).
	writeConfig(t, path, "github:\n  default_repo: two-longer\n")
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatal(err)
	}
	f3, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if f3 == f1 {
		t.Fatal("a changed file must be re-read")
	}
	var gh ghSection
	if _, err := f3.Section("github", &gh); err != nil || gh.DefaultRepo != "two-longer" {
		t.Fatalf("got %q, %v", gh.DefaultRepo, err)
	}
}

func TestSectionAbsentNullAndInvalid(t *testing.T) {
	f, err := Parse([]byte("github:\n  default_repo: r\nembeddings:\ncoordinator:\n  agents: notalist\n"))
	if err != nil {
		t.Fatal(err)
	}
	var gh ghSection
	if ok, err := f.Section("pubsub", &gh); ok || err != nil {
		t.Fatalf("absent section: ok=%v err=%v; want false, nil", ok, err)
	}
	if ok, err := f.Section("embeddings", &gh); ok || err != nil {
		t.Fatalf("null section: ok=%v err=%v; want false, nil", ok, err)
	}
	if f.Has("embeddings") || !f.Has("github") || f.Has("nope") {
		t.Fatal("Has must be true only for a present, non-null key")
	}
	var coord struct {
		Agents []struct{ ID string } `yaml:"agents"`
	}
	ok, err := f.Section("coordinator", &coord)
	if !ok || !errors.Is(err, ErrConfigInvalid) || !strings.Contains(err.Error(), `section "coordinator"`) {
		t.Fatalf("a section that does not fit must be ErrConfigInvalid naming the key: ok=%v err=%v", ok, err)
	}
}

func TestParseRejectsInvalidYAML(t *testing.T) {
	if _, err := Parse([]byte("github: [unclosed\n")); !errors.Is(err, ErrConfigInvalid) {
		t.Fatalf("err = %v, want ErrConfigInvalid", err)
	}
	resetFileCacheForTest()
	t.Cleanup(resetFileCacheForTest)
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "github: [unclosed\n")
	_, err := LoadFrom(path)
	if !errors.Is(err, ErrConfigInvalid) || !strings.Contains(err.Error(), path) {
		t.Fatalf("a broken file must be ErrConfigInvalid naming the path: %v", err)
	}
}

func TestUnknownKeysNamesOnlyTheAskedType(t *testing.T) {
	type agent struct {
		ID          string `yaml:"id"`
		MergeBranch string `yaml:"merge_branch"`
	}
	type coordinator struct {
		Agents []agent `yaml:"agents"`
	}
	f, err := Parse([]byte("github:\n  default_repo: r\ncoordinator:\n  agents:\n    - id: a\n      push_branch: dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	var wrapper struct {
		Coordinator coordinator `yaml:"coordinator"`
	}
	keys, err := f.UnknownKeys(&wrapper, "config.agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || !strings.HasSuffix(keys[0], "push_branch") || !strings.HasPrefix(keys[0], "line 6") {
		t.Fatalf("want exactly [\"line 6: push_branch\"] (file-relative line), got %v", keys)
	}
	clean, err := Parse([]byte("coordinator:\n  agents:\n    - id: a\n      merge_branch: dev\n"))
	if err != nil {
		t.Fatal(err)
	}
	if keys, err := clean.UnknownKeys(&wrapper, "config.agent"); err != nil || len(keys) != 0 {
		t.Fatalf("clean: %v, %v", keys, err)
	}
}
