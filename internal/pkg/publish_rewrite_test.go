package pkg

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRewritePathDeps_UsesOriginalDirectoryAndIsIdempotent(t *testing.T) {
	_, author := smokeStageFixture(t)
	stage := t.TempDir()
	if err := copyPackageContents(author, stage); err != nil {
		t.Fatal(err)
	}
	got, err := RewritePathDeps(stage, author)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []PathDepRewrite{{Name: "test/b", Path: "../b", Version: "0.1.0"}}) {
		t.Fatalf("rewrite facts: %+v", got)
	}
	before, err := os.ReadFile(filepath.Join(stage, ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	again, err := RewritePathDeps(stage, author)
	if err != nil || len(again) != 0 {
		t.Fatalf("not idempotent: %+v %v", again, err)
	}
	after, _ := os.ReadFile(filepath.Join(stage, ManifestFile))
	if string(before) != string(after) {
		t.Fatal("idempotent rewrite changed bytes")
	}
	original, _ := os.ReadFile(filepath.Join(author, ManifestFile))
	if !strings.Contains(string(original), `{path = "../b"}`) {
		t.Fatal("author changed")
	}
}

func TestRewritePathDeps_ErrorDoesNotWritePartialManifest(t *testing.T) {
	_, author := smokeStageFixture(t)
	path := filepath.Join(author, ManifestFile)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A parseable spelling the approved existing rewrite does not support must
	// fail clearly, never silently publish an unrewritten path dependency.
	bad := strings.Replace(string(original), `"test/b" = {path = "../b"}`, `'test/b' = {path = '../b'}`, 1)
	if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := RewritePathDeps(author, author); err == nil {
		t.Fatal("unrewritten path dep accepted")
	}
	after, _ := os.ReadFile(path)
	if string(after) != bad {
		t.Fatal("failed rewrite changed author bytes")
	}
}
