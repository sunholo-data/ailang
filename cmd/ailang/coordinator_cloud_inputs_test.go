package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/messaging"
)

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Inputs test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Inputs test", "GIT_COMMITTER_EMAIL=test@example.com")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
	return strings.TrimSpace(string(out))
}

func inputFixture(t *testing.T) (string, string, taskInputFetcher) {
	t.Helper()
	workspace, source := t.TempDir(), t.TempDir()
	runGit(t, workspace, "init")
	writeInputFile(t, source, "payload/photo.png", []byte{0, 255, 1, 2})
	fetcher := taskInputFetcher{limit: maxTaskInputBytes, clone: func(_ context.Context, input messaging.TaskInput, dest string) (string, error) {
		if input.Ref == "missing" {
			return "", fmt.Errorf("missing ref")
		}
		return strings.Repeat("a", 40), copyInputFixture(source, dest)
	}}
	return workspace, source, fetcher
}

func writeInputFile(t *testing.T, root, name string, data []byte) {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Fixture copies preserve symlinks so the production fetcher must reject them.
func copyInputFixture(src, dest string) error {
	return filepath.WalkDir(src, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, e := os.Readlink(p)
			if e != nil {
				return e
			}
			return os.Symlink(link, target)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func TestTaskInputsFetchDelivery(t *testing.T) {
	for _, tc := range []struct{ name, path, dest, target string }{
		{"default file", "payload/photo.png", "", ".incoming/1/photo.png"},
		{"explicit file", "payload/photo.png", "images/banner.png", "images/banner.png"},
		{"directory", "payload", "images/", "images/photo.png"},
		{"whole tree", "", "", ".incoming/1/payload/photo.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			workspace, source, fetcher := inputFixture(t)
			writeInputFile(t, source, ".git/config", []byte("must not copy"))
			inputs := []messaging.TaskInput{{Repo: "org/data", Ref: "incoming/demo", Path: tc.path, Dest: tc.dest}}
			got, err := fetcher.fetch(context.Background(), workspace, inputs)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(workspace, tc.target))
			if err != nil || string(data) != string([]byte{0, 255, 1, 2}) {
				t.Fatalf("bytes %v, %v", data, err)
			}
			if len(got) != 1 || got[0].Commit != strings.Repeat("a", 40) || !strings.Contains(taskInputReport(got), "org/data") {
				t.Fatalf("provenance %+v", got)
			}
			if _, err := os.Stat(filepath.Join(workspace, ".incoming/1/.git")); !os.IsNotExist(err) {
				t.Fatalf("copied git metadata: %v", err)
			}
			if tc.dest == "" {
				if out := runGit(t, workspace, "status", "--porcelain"); strings.Contains(out, ".incoming") {
					t.Fatalf("default input tracked: %s", out)
				}
			}
		})
	}
}

func TestTaskInputsFetchRefusals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input messaging.TaskInput
		setup func(*testing.T, string, string)
		limit int64
	}{
		{name: "missing ref", input: messaging.TaskInput{Ref: "missing"}},
		{name: "missing path", input: messaging.TaskInput{Path: "absent"}},
		{name: "traversal", input: messaging.TaskInput{Dest: "../escape"}},
		{name: "checksum mismatch", input: messaging.TaskInput{Path: "payload/photo.png", SHA256: strings.Repeat("0", 64)}},
		{name: "directory digest", input: messaging.TaskInput{Path: "payload", SHA256: strings.Repeat("0", 64)}},
		{name: "source symlink", setup: func(t *testing.T, _, src string) {
			if err := os.Symlink("photo.png", filepath.Join(src, "payload/link")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "source ancestor symlink", input: messaging.TaskInput{Path: "alias/photo.png"}, setup: func(t *testing.T, _, src string) {
			if err := os.Symlink("payload", filepath.Join(src, "alias")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "destination symlink", input: messaging.TaskInput{Dest: "images/"}, setup: func(t *testing.T, ws, _ string) {
			if err := os.Symlink(t.TempDir(), filepath.Join(ws, "images")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "existing file", input: messaging.TaskInput{Path: "payload/photo.png", Dest: "existing.png"}, setup: func(t *testing.T, ws, _ string) { writeInputFile(t, ws, "existing.png", []byte("keep")) }},
		{name: "nested instructions", input: messaging.TaskInput{Dest: "data/nested/AGENTS.md/"}},
		{name: "source instructions", setup: func(t *testing.T, _, src string) {
			writeInputFile(t, src, "payload/nested/.claude/config", []byte("deny"))
		}},
		{name: "size limit", limit: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ws, src, f := inputFixture(t)
			if tc.setup != nil {
				tc.setup(t, ws, src)
			}
			if tc.limit != 0 {
				f.limit = tc.limit
			}
			in := tc.input
			in.Repo = "org/data"
			if in.Ref == "" {
				in.Ref = "main"
			}
			if in.Path == "" {
				in.Path = "payload"
			}
			_, err := f.fetch(context.Background(), ws, []messaging.TaskInput{in})
			if err == nil || !strings.Contains(err.Error(), "inputs[0]") {
				t.Fatalf("indexed refusal: %v", err)
			}
			if _, err := os.Stat(filepath.Join(ws, ".incoming/1")); !os.IsNotExist(err) {
				t.Fatalf("partial delivery: %v", err)
			}
		})
	}
}

func TestTaskInputsChecksumsAndAtomicStaging(t *testing.T) {
	ws, src, f := inputFixture(t)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte{0, 255, 1, 2}))
	writeInputFile(t, src, "payload/manifest.sha256", []byte(digest+"  photo.png\n"))
	got, err := f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}})
	if err != nil || len(got[0].Verified) != 1 {
		t.Fatalf("manifest: %+v %v", got, err)
	}
	for _, manifest := range []string{"bad", strings.Repeat("0", 64) + "  photo.png\n", digest + "  missing\n", digest + "  ../escape\n", digest + "  photo.png\n" + digest + "  photo.png\n"} {
		ws, src, f := inputFixture(t)
		writeInputFile(t, src, "payload/manifest.sha256", []byte(manifest))
		_, err := f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}})
		if err == nil {
			t.Fatalf("accepted %q", manifest)
		}
	}
	ws, _, f = inputFixture(t)
	_, err = f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}, {Repo: "org/data", Ref: "missing", Path: "payload"}})
	if err == nil || !strings.Contains(err.Error(), "inputs[1]") {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".incoming")); !os.IsNotExist(err) {
		t.Fatal("first input delivered before second verified")
	}
	ws, _, f = inputFixture(t)
	f.limit = 7
	_, err = f.fetch(context.Background(), ws, []messaging.TaskInput{{Repo: "org/data", Ref: "main", Path: "payload"}, {Repo: "org/data", Ref: "main", Path: "payload"}})
	if err == nil {
		t.Fatal("aggregate byte cap not enforced")
	}
}
