package quorum_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/mission/quorum"
	"github.com/sunholo-data/ailang/internal/policytool"
)

// The wrapper execs this test executable, never a provider or installed ailang.
func writerWrapper(t *testing.T, home string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "writer")
	body := "#!/bin/bash\nexport QUORUM_WRITER_CHILD=1\nexec '" + strings.ReplaceAll(exe, "'", "'\"'\"'") + "' -test.run=^TestQuorumWriterChild$\n"
	if err := os.WriteFile(path, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestQuorumWriterChild(t *testing.T) {
	if os.Getenv("QUORUM_WRITER_CHILD") != "1" {
		return
	}
	if count := os.Getenv("QUORUM_LAUNCH_COUNT"); count != "" {
		f, err := os.OpenFile(count, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("launch\n"); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	path, err := quorum.WriteJSONArtifact("", &quorum.QuorumResult{Doc: "doc.md", ISOTimestamp: "2026-10-14T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("ARTIFACT=" + path)
}

func TestPolicyHostQuorumArtifacts(t *testing.T) {
	for _, name := range []string{"absolute", "invalid", "suppression_diagnostic", "deny_no_child"} {
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			sandbox := filepath.Join(home, "sandbox")
			if err := os.Mkdir(sandbox, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(sandbox, "doc.md"), []byte("# doc"), 0600); err != nil {
				t.Fatal(err)
			}
			external := filepath.Join(home, "external")
			override := external
			if name == "invalid" {
				override = "~/invalid"
			}
			t.Setenv(config.EnvMissionQuorumArtifactDir, override)
			t.Setenv(config.EnvAgentPolicy, "parent-secret-marker")
			count := filepath.Join(home, "launches")
			t.Setenv("QUORUM_LAUNCH_COUNT", count)
			policy := filepath.Join(home, "policy-secret.toml")
			deny := ""
			if name == "deny_no_child" {
				deny = "fs_deny_write = [\".ailang/**\"]\n"
			}
			content := fmt.Sprintf("allowed_caps = [\"IO\", \"FS\"]\nfs_sandbox = %q\nentry = \"main\"\ncli_allow = [\"design-quorum\"]\n%s", filepath.ToSlash(sandbox), deny)
			if err := os.WriteFile(policy, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			host, err := policytool.Open(policy)
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := host.Close(); err != nil {
					t.Error(err)
				}
			}()
			host.SetBinary(writerWrapper(t, home))
			response := host.Dispatch(policytool.Request{Op: "design_quorum", Path: "doc.md"})
			if name == "deny_no_child" {
				if response.OK || !strings.Contains(response.Refused, "fs_deny_write") {
					t.Fatalf("%+v", response)
				}
				if _, err := os.Stat(count); !os.IsNotExist(err) {
					t.Fatal("refusal launched child")
				}
				if _, err := os.Stat(filepath.Join(sandbox, quorum.ArtifactDir)); !os.IsNotExist(err) {
					t.Fatal("denied write")
				}
			} else {
				if !response.OK || response.ExitCode != 0 {
					t.Fatalf("%+v", response)
				}
				launches, err := os.ReadFile(count)
				if err != nil || string(launches) != "launch\n" {
					t.Fatalf("launches %q %v", launches, err)
				}
				local := filepath.Join(sandbox, quorum.ArtifactDir)
				files, err := filepath.Glob(filepath.Join(local, "doc-*.json"))
				if err != nil || len(files) != 1 {
					t.Fatalf("not confined: %v %v; %+v", files, err, response)
				}
				if !strings.Contains(response.Stderr, "suppressed") || !strings.Contains(response.Stderr, local) || strings.Contains(response.Stderr, policy) || strings.Contains(response.Stderr, "parent-secret-marker") {
					t.Fatalf("diagnostic: %s", response.Stderr)
				}
			}
			if _, err := os.Stat(external); !os.IsNotExist(err) {
				t.Fatalf("external artifact: %v", err)
			}
		})
	}
}
