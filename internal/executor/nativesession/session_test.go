package nativesession

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFind(t *testing.T) {
	for _, provider := range []string{"pi", "codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			root := t.TempDir()
			names := map[string]string{"pi": "stamp_id.jsonl", "codex": "rollout-stamp-id.jsonl", "claude": "id.jsonl"}
			match := filepath.Join(root, names[provider])
			if err := os.WriteFile(match, []byte("native"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chtimes(match, time.Unix(1, 0), time.Unix(1, 0)); err != nil {
				t.Fatal(err)
			}
			other := filepath.Join(root, "rollout-other.jsonl")
			if err := os.WriteFile(other, []byte("newer"), 0600); err != nil {
				t.Fatal(err)
			}
			if got := Find(root, provider, "id", true); got != match {
				t.Fatalf("match = %q, want %q", got, match)
			}
			if got := Find(root, provider, "missing", false); got != "" {
				t.Fatalf("unexpected match %q", got)
			}
			if provider != "claude" {
				if got := Find(root, provider, "missing", true); got != other {
					t.Fatalf("fallback = %q", got)
				}
			}
			if got := Find(filepath.Join(root, "absent"), provider, "id", true); got != "" {
				t.Fatal(got)
			}
		})
	}
}

func TestFindThroughProjectsSymlink(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "real-projects")
	if err := os.MkdirAll(projects, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(projects, "id.jsonl")
	if err := os.WriteFile(path, []byte("native"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "projects")
	if err := os.Symlink(projects, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if got := Find(link, "claude", "id", false); got != path {
		t.Fatalf("symlink lookup = %q, want %q", got, path)
	}
}
