package claude

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// fakeOAuthToken is a structurally valid, obviously fake credential blob.
const fakeOAuthToken = `{"accessToken":"test-access","refreshToken":"test-refresh","expiresAt":1}`

// credAuthEnv points HOME, the artifact root and CLAUDE_CONFIG_DIR at temp dirs.
func credAuthEnv(t *testing.T, configDir func(home, artifacts string) string) (home, artifacts string) {
	t.Helper()
	root := t.TempDir()
	home = filepath.Join(root, "home")
	artifacts = filepath.Join(root, "artifacts")
	for _, d := range []string{home, artifacts} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	testutil.SetHomeDir(t, home)
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", fakeOAuthToken)
	t.Setenv("CLAUDE_CONFIG_DIR", configDir(home, artifacts))
	prev := credentialArtifactRoot
	credentialArtifactRoot = artifacts
	t.Cleanup(func() { credentialArtifactRoot = prev })
	return home, artifacts
}

// credentialFilesUnder lists every .credentials.json beneath dir.
func credentialFilesUnder(t *testing.T, dir string) []string {
	t.Helper()
	var found []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == ".credentials.json" {
			found = append(found, p)
		}
		return nil
	})
	return found
}

// F-H6-1: with a local CLAUDE_CONFIG_DIR the credential lands there (0600) and
// the CLI's refresh — an in-place rewrite or a temp-file rename — still works.
func TestWriteCredentialsFile_LocalConfigDirAndRefresh(t *testing.T) {
	var configDir string
	home, artifacts := credAuthEnv(t, func(home, _ string) string {
		configDir = filepath.Join(home, ".claude-tasks", "task-1")
		return configDir
	})

	if err := writeCredentialsFile(); err != nil {
		t.Fatalf("writeCredentialsFile: %v", err)
	}
	for _, p := range []string{
		filepath.Join(home, ".claude", ".credentials.json"),
		filepath.Join(configDir, ".credentials.json"),
	} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("credential missing at %s: %v", p, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %o, want 0600", p, info.Mode().Perm())
		}
	}

	// Refresh: the CLI rewrites the file in its config dir.
	cred := filepath.Join(configDir, ".credentials.json")
	if err := os.WriteFile(cred, []byte(`{"claudeAiOauth":{"accessToken":"rotated"}}`), 0o600); err != nil {
		t.Fatalf("in-place refresh failed: %v", err)
	}
	tmp := filepath.Join(configDir, ".credentials.json.tmp")
	if err := os.WriteFile(tmp, []byte(`{"claudeAiOauth":{"accessToken":"rotated-2"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, cred); err != nil {
		t.Fatalf("rename refresh failed: %v", err)
	}
	if got, _ := os.ReadFile(cred); string(got) != `{"claudeAiOauth":{"accessToken":"rotated-2"}}` {
		t.Errorf("refreshed credential not readable back: %q", got)
	}

	if leaked := credentialFilesUnder(t, artifacts); len(leaked) > 0 {
		t.Fatalf("credential written under the artifact root: %v", leaked)
	}
}

// F-H6-1 regression: a CLAUDE_CONFIG_DIR on the artifacts mount is refused
// before ANY credential is written — including the ~/.claude copy.
func TestWriteCredentialsFile_RefusesArtifactRoot(t *testing.T) {
	home, artifacts := credAuthEnv(t, func(_, artifacts string) string {
		return filepath.Join(artifacts, "tasks", "task-1", "claude")
	})

	err := writeCredentialsFile()
	if !errors.Is(err, ErrCredentialsUnderArtifactRoot) {
		t.Fatalf("err = %v, want ErrCredentialsUnderArtifactRoot", err)
	}
	if leaked := credentialFilesUnder(t, artifacts); len(leaked) > 0 {
		t.Fatalf("credential written under the artifact root: %v", leaked)
	}
	if found := credentialFilesUnder(t, home); len(found) > 0 {
		t.Errorf("refusal should precede every write, found %v", found)
	}
}

// A local-looking CLAUDE_CONFIG_DIR that is a symlink into the mount is refused too.
func TestWriteCredentialsFile_RefusesSymlinkIntoArtifactRoot(t *testing.T) {
	_, artifacts := credAuthEnv(t, func(home, artifacts string) string {
		target := filepath.Join(artifacts, "tasks", "task-1", "claude")
		if err := os.MkdirAll(target, 0o755); err != nil {
			t.Fatal(err)
		}
		link := filepath.Join(home, "cfg")
		if err := os.Symlink(target, link); err != nil {
			t.Fatal(err)
		}
		return link
	})

	if err := writeCredentialsFile(); !errors.Is(err, ErrCredentialsUnderArtifactRoot) {
		t.Fatalf("err = %v, want ErrCredentialsUnderArtifactRoot", err)
	}
	if leaked := credentialFilesUnder(t, artifacts); len(leaked) > 0 {
		t.Fatalf("credential written under the artifact root: %v", leaked)
	}
}
