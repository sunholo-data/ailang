package pkg

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// Staging must preserve D2, including when no registry rewrite is needed. The
// previous lock is provenance for unchanged git, never a raw execution lock.
func TestSmokeStage_GitCarryForward(t *testing.T) {
	for _, mode := range []string{"confined_home", "exclusive_root"} {
		t.Run(mode, func(t *testing.T) {
			repo := makeGitPackage(t)
			home := t.TempDir()
			testutil.SetHomeDir(t, home)
			t.Setenv(config.EnvPackageRoot, "")
			app, m := appWithGitDep(t, repo, "v0.1.0")
			previous := lockUnconfined(t, app, m)
			entry, _ := previous.FindPackage("test/gitlib")
			provisioned, err := lockedGitDir(entry)
			if err != nil {
				t.Fatal(err)
			}
			if err := previous.Save(app); err != nil {
				t.Fatal(err)
			}
			if mode == "exclusive_root" {
				root := t.TempDir()
				dest := filepath.Join(root, "test", "gitlib", "0.1.0")
				if err := os.CopyFS(dest, os.DirFS(provisioned)); err != nil {
					t.Fatal(err)
				}
				t.Setenv(config.EnvPackageRoot, root)
				provisioned = dest
			} else {
				t.Setenv(config.EnvAgentPolicy, "policy.toml")
			}
			// No git executable can be used during stage assembly, in either mode.
			t.Setenv("PATH", t.TempDir())
			before := snapshotTree(t, home)
			stage := t.TempDir()
			if err := stagePublishedPackage(app, stage); err != nil {
				t.Fatal(err)
			}
			got, err := LoadLockFile(stage)
			if err != nil {
				t.Fatal(err)
			}
			carried, ok := got.FindPackage("test/gitlib")
			if !ok || !reflect.DeepEqual(*carried, *entry) {
				t.Fatalf("git provenance changed: got%+v want%+v", carried, entry)
			}
			if snapshotTree(t, home) != before {
				t.Fatal("confined/root staging changed HOME")
			}
			// Removing the pin must not fall back to cached bytes or call git.
			if err := os.Remove(filepath.Join(app, LockFileName)); err != nil {
				t.Fatal(err)
			}
			if err := stagePublishedPackage(app, t.TempDir()); err == nil || !strings.Contains(err.Error(), "no previous ailang.lock") {
				t.Fatalf("missing pin: %v", err)
			}
			if err := previous.Save(app); err != nil {
				t.Fatal(err)
			}
			// A changed tag still refuses, even though the old package is provisioned.
			changedDir, _ := appWithGitDep(t, repo, "v0.2.0")
			manifestPath := filepath.Join(app, ManifestFile)
			originalManifest, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			changedManifest, err := os.ReadFile(filepath.Join(changedDir, ManifestFile))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(manifestPath, changedManifest, 0600); err != nil {
				t.Fatal(err)
			}
			if err := stagePublishedPackage(app, t.TempDir()); err == nil || !strings.Contains(err.Error(), "tag") {
				t.Fatalf("changed tag accepted: %v", err)
			}
			if err := os.WriteFile(manifestPath, originalManifest, 0600); err != nil {
				t.Fatal(err)
			}

			// Restore the same tag but alter its provisioned bytes: hashes must bind it.
			f := filepath.Join(provisioned, "src", "test", "gitlib", "lib.ail")
			if err := os.WriteFile(f, []byte("tampered"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := stagePublishedPackage(app, t.TempDir()); err == nil || !strings.Contains(err.Error(), "provisioned content") {
				t.Fatalf("tampered pin: %v", err)
			}
		})
	}
}
