package executor

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestGitCredentialScopes(t *testing.T) {
	got := GitCredentialScopes("https://github.com/sunholo-data/ailang.git")
	want := []string{"https://github.com/sunholo-data/ailang", "https://github.com/sunholo-data/ailang.git"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("scopes = %v, want %v", got, want)
	}
	for _, u := range []string{"git@gh-deploy:owner/repo.git", "sunholo-data/ailang", "http://github.com/a/b", "https://u:p@github.com/a/b", "https://github.com/", ""} {
		if s := GitCredentialScopes(u); s != nil {
			t.Errorf("GitCredentialScopes(%q) = %v, want nil", u, s)
		}
	}
}

func TestWriteGitCredentialFile_PermissionsAndCleanup(t *testing.T) {
	scopes := GitCredentialScopes("https://github.com/owner/repo")
	path, cleanup, err := WriteGitCredentialFile(scopes, "tok-123\n")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "https://x-access-token:tok-123@github.com\n" {
		t.Errorf("credential line = %q", data)
	}
	if runtime.GOOS != "windows" {
		fi, _ := os.Stat(path)
		di, _ := os.Stat(strings.TrimSuffix(path, "/credentials"))
		if fi.Mode().Perm() != 0o600 || di.Mode().Perm() != 0o700 {
			t.Errorf("file %v dir %v, want 0600 / 0700", fi.Mode().Perm(), di.Mode().Perm())
		}
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("cleanup must remove the credential file")
	}
}

// The scoped credential answers for the task's repository and for nothing
// else — checked against real git, the thing that interprets the config.
func TestGitCredentialEnv_ScopedToTheTaskRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	scopes := GitCredentialScopes("https://github.com/owner/repo.git")
	path, cleanup, err := WriteGitCredentialFile(scopes, "tok-xyz")
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	fill := func(repoPath string) string {
		cmd := exec.Command("git", "credential", "fill")
		cmd.Env = append([]string{
			"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=" + os.DevNull,
			"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=", "SSH_ASKPASS=",
		}, GitCredentialEnv(path, scopes)...)
		if runtime.GOOS == "windows" {
			cmd.Env = append(cmd.Env, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"))
		}
		cmd.Stdin = strings.NewReader("protocol=https\nhost=github.com\npath=" + repoPath + "\n\n")
		out, _ := cmd.Output()
		return string(out)
	}
	for _, p := range []string{"owner/repo.git", "owner/repo"} {
		if out := fill(p); !strings.Contains(out, "password=tok-xyz") {
			t.Errorf("task repo %s: git credential fill = %q, want the scoped token", p, out)
		}
	}
	if out := fill("owner/other.git"); strings.Contains(out, "tok-xyz") {
		t.Errorf("another repository received the token: %q", out)
	}
}
