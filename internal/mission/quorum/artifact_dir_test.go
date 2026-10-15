package quorum

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

func artifactEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(config.EnvMissionQuorumArtifactDir, "")
	t.Setenv(config.EnvAgentPolicy, "")
	return home
}

func TestResolveArtifactDir_Default(t *testing.T) {
	home := artifactEnv(t)
	for _, name := range []string{"unset", "empty"} {
		t.Run(name, func(t *testing.T) {
			if name == "unset" {
				if err := os.Unsetenv(config.EnvMissionQuorumArtifactDir); err != nil {
					t.Fatal(err)
				}
			}
			for _, arg := range []string{"", ArtifactDir} {
				got, err := ResolveArtifactDir(arg)
				if err != nil || got != ArtifactDir {
					t.Fatalf("got %q %v", got, err)
				}
			}
			entries, err := os.ReadDir(home)
			if err != nil || len(entries) != 0 {
				t.Fatalf("resolution wrote: %v %v", entries, err)
			}
		})
	}
}

func TestResolveArtifactDir_ConstantPinned(t *testing.T) {
	artifactEnv(t)
	if ArtifactDir != ".ailang/state/mission-quorum" {
		t.Fatal(ArtifactDir)
	}
}

func TestResolveArtifactDir_ExplicitWins(t *testing.T) {
	home := artifactEnv(t)
	t.Setenv(config.EnvMissionQuorumArtifactDir, "invalid")
	t.Setenv(config.EnvAgentPolicy, " ")
	for name, arg := range map[string]string{"absolute": filepath.Join(home, "explicit"), "relative": "explicit"} {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveArtifactDir(arg)
			if err != nil || got != arg {
				t.Fatalf("%q %v", got, err)
			}
		})
	}
}

func TestResolveArtifactDir_RawPolicyMarker(t *testing.T) {
	home := artifactEnv(t)
	t.Run("whitespace", func(t *testing.T) {
		t.Setenv(config.EnvAgentPolicy, " ")
		for _, override := range []string{filepath.Join(home, "outside"), "invalid"} {
			t.Setenv(config.EnvMissionQuorumArtifactDir, override)
			got, err := ResolveArtifactDir("")
			if err != nil || got != ArtifactDir {
				t.Fatalf("%q %v", got, err)
			}
			if config.MissionQuorumArtifactDir() != override {
				t.Fatal("getter changed raw bytes")
			}
		}
	})
}

func TestValidateArtifactDir_NUL(t *testing.T) {
	home := artifactEnv(t)
	if validateArtifactDir(filepath.Join(home, "nul")+"\x00") == nil {
		t.Fatal("accepted NUL")
	}
}

func TestQuorumArtifactRegistry(t *testing.T) {
	artifactEnv(t)
	t.Run("raw_and_default", func(t *testing.T) {
		t.Setenv(config.EnvMissionQuorumArtifactDir, " /path with spaces ")
		if config.MissionQuorumArtifactDir() != " /path with spaces " {
			t.Fatal("trimmed override")
		}
		count := 0
		for _, v := range config.Registry {
			if v.Name == config.EnvMissionQuorumArtifactDir {
				count++
				if v.Default != "" || v.Where != config.AreaMission || v.Doc == "" {
					t.Fatalf("bad row: %+v", v)
				}
			}
		}
		if count != 1 {
			t.Fatalf("rows=%d", count)
		}
	})
}

func artifactResult() *QuorumResult {
	return &QuorumResult{Doc: "doc.md", ISOTimestamp: "2026-10-14T00:00:00Z"}
}

func TestWriteJSONArtifact_OverrideSpaces(t *testing.T) {
	home := artifactEnv(t)
	t.Chdir(home)
	dir := filepath.Join(home, "override with spaces")
	t.Setenv(config.EnvMissionQuorumArtifactDir, dir)
	q := artifactResult()
	path, err := WriteJSONArtifact("", q)
	if err != nil || filepath.Dir(path) != dir {
		t.Fatalf("%q %v", path, err)
	}
	want, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(want) {
		t.Fatalf("bytes=%q %v", got, err)
	}
	if _, err := os.Stat(ArtifactDir); !os.IsNotExist(err) {
		t.Fatalf("local write: %v", err)
	}
}

func TestWriteJSONArtifact_InvalidOverride(t *testing.T) {
	home := artifactEnv(t)
	t.Chdir(home)
	for name, override := range map[string]string{"relative": "relative", "tilde": "~/artifacts", "whitespace": " \t "} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(config.EnvMissionQuorumArtifactDir, override)
			path, err := WriteJSONArtifact("", artifactResult())
			if err == nil || path != "" {
				t.Fatalf("%q %v", path, err)
			}
			entries, err := os.ReadDir(home)
			if err != nil || len(entries) != 0 {
				t.Fatalf("unexpected writes: %v %v", entries, err)
			}
		})
	}
}

func TestWriteJSONArtifact_DestinationError(t *testing.T) {
	home := artifactEnv(t)
	t.Chdir(home)
	file := filepath.Join(home, "file")
	if err := os.WriteFile(file, []byte("sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(config.EnvMissionQuorumArtifactDir, filepath.Join(file, "child"))
	path, err := WriteJSONArtifact("", artifactResult())
	if err == nil || path != "" {
		t.Fatalf("%q %v", path, err)
	}
	if _, err := os.Stat(ArtifactDir); !os.IsNotExist(err) {
		t.Fatal("local fallback")
	}
	t.Run("permission", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows ignores POSIX directory modes, so a 0500 dir is still writable")
		}
		dir := filepath.Join(home, "read-only")
		if err := os.Mkdir(dir, 0500); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.Chmod(dir, 0700); err != nil {
				t.Error(err)
			}
		})
		t.Setenv(config.EnvMissionQuorumArtifactDir, dir)
		if _, err := WriteJSONArtifact("", artifactResult()); err == nil {
			t.Fatal("permission refusal not exercised")
		}
	})
}

func TestWriteJSONArtifact_OverrideNoClobber(t *testing.T) {
	home := artifactEnv(t)
	dir := filepath.Join(home, "override")
	t.Setenv(config.EnvMissionQuorumArtifactDir, dir)
	q := artifactResult()
	a, err := WriteJSONArtifact("", q)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(a)
	if err != nil {
		t.Fatal(err)
	}
	q.Doc = "DOC.md"
	b, err := WriteJSONArtifact("", q)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(a)
	if err != nil || a == b || string(before) != string(after) {
		t.Fatalf("clobber: %q %q %v", a, b, err)
	}
	second, err := os.ReadFile(b)
	if err != nil || string(second) == string(before) {
		t.Fatal("distinct bytes missing")
	}
}

func TestResolveArtifactDir_CWDUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows cannot remove a process's current working directory")
	}
	if os.Getenv("QUORUM_CWD_CHILD") == "1" {
		if err := os.Remove(os.Getenv("QUORUM_CWD_DIR")); err != nil {
			t.Fatal(err)
		}
		if _, err := ResolveArtifactDir(""); err == nil {
			t.Fatal("unavailable CWD accepted")
		}
		return
	}
	home := artifactEnv(t)
	dir := filepath.Join(home, "cwd")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestResolveArtifactDir_CWDUnavailable$")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "QUORUM_CWD_CHILD=1", "QUORUM_CWD_DIR="+dir, config.EnvAgentPolicy+"= ", config.EnvMissionQuorumArtifactDir+"=invalid")
	out, err := cmd.CombinedOutput()
	if err != nil || strings.Contains(string(out), "suppressed") {
		t.Fatalf("%s %v", out, err)
	}
}
