package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeSyntheticRepo pins the TOML to mission_registry_env_test.go:14 and the log to rotate_test.go:13-19.
func writeSyntheticRepo(t *testing.T, name string, shared bool, n int) (root, reg, work string) {
	root, work = t.TempDir(), t.TempDir()
	reg = filepath.Join(root, "missions")
	repo := map[bool]string{true: sharedRepoSlug, false: "example/external"}[shared]
	toml := fmt.Sprintf("name = %q\nrepo = %q\nworkdir = %q\ndoc = \"README.md\"\n[schedule]\nmode = \"interval\"\ninterval_seconds = 21600\nboot_offset = 840\n", name, repo, filepath.ToSlash(work))
	synthWrite(t, filepath.Join(reg, name+".toml"), toml)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Mission Log\n\nPreamble that must survive rotation.\n", name)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "\n## %d — 2026-09-%02d — Did thing number %d [TAG%d]\n\n**Shipped.** body line for %d\nmore body\n", i, (i%28)+1, i, i%3, i)
	}
	synthWrite(t, filepath.Join(root, "design_docs", name+"-mission-log.md"), b.String())
	synthWrite(t, filepath.Join(work, "design_docs", name+"-mission-log.md"), b.String())
	synthWrite(t, filepath.Join(root, "design_docs", name+"-mission-status-archive.md"), b.String())
	return root, reg, work
}
func read(t *testing.T, p string) string {
	b, err := os.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(b)
}
func synthWrite(t *testing.T, p, s string) {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
}
func use(t *testing.T, root, reg string) {
	t.Setenv("AILANG_MISSION_REGISTRY", reg)
	t.Chdir(root)
}
func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}
func snapshot(t *testing.T, paths ...string) {
	m := map[string]string{}
	for _, p := range paths {
		m[p] = read(t, p)
	}
	t.Cleanup(func() {
		for p, want := range m {
			if read(t, p) != want {
				t.Errorf("%s changed (expected byte-identical)", p)
			}
		}
	})
}

// checkRotated asserts the live log shrank (no `## 1`) but kept the newest (`## 10`),
// and that each named artifact was written beside it.
func checkRotated(t *testing.T, dir, live string, extra ...string) {
	if got := read(t, filepath.Join(dir, live)); strings.Contains(got, "## 1 —") || !strings.Contains(got, "## 10 —") {
		t.Errorf("%s not rotated: %s", live, got)
	}
	for _, f := range extra {
		if read(t, filepath.Join(dir, f)) == "" {
			t.Errorf("missing artifact %s", f)
		}
	}
}
func TestMissionRotateLogSharedRepoTargetsRegistryOrigin(t *testing.T) {
	aRoot, aReg, aWork := writeSyntheticRepo(t, "rotcanary", true, 10)
	bRoot, _, _ := writeSyntheticRepo(t, "rotcanary", true, 10)
	use(t, bRoot, aReg)
	snapshot(t, filepath.Join(bRoot, "design_docs", "rotcanary-mission-log.md"), filepath.Join(aWork, "design_docs", "rotcanary-mission-log.md"))
	must(t, missionRotateLog([]string{"rotcanary", "--keep", "5"}))
	checkRotated(t, filepath.Join(aRoot, "design_docs"), "rotcanary-mission-log.md", "rotcanary-mission-log-archive.md", "rotcanary-mission-index.md")
}
func TestMissionRotateLogExternalMissionUsesWorkdir(t *testing.T) {
	root, reg, work := writeSyntheticRepo(t, "extcanary", false, 10)
	use(t, root, reg)
	snapshot(t, filepath.Join(root, "design_docs", "extcanary-mission-log.md"))
	must(t, missionRotateLog([]string{"extcanary", "--keep", "5"}))
	checkRotated(t, filepath.Join(work, "design_docs"), "extcanary-mission-log.md")
}
func TestMissionRotateLogStatusRejectedBeforeRegistryLoad(t *testing.T) {
	use(t, t.TempDir(), "")
	err := missionRotateLog([]string{"x", "--status"})
	if err == nil || !strings.Contains(err.Error(), "--stream status") || strings.Contains(err.Error(), "mission registry") {
		t.Fatalf("bad --status rejection: %v", err)
	}
}
func TestMissionRotateLogStatusRejectedLeavesFilesByteIdentical(t *testing.T) {
	root, reg, _ := writeSyntheticRepo(t, "rejcanary", true, 10)
	use(t, root, reg)
	d := filepath.Join(root, "design_docs")
	var ps []string
	for _, f := range []string{"log", "log-archive", "index", "status-archive", "status-archive-old", "status-index"} {
		ps = append(ps, filepath.Join(d, "rejcanary-mission-"+f+".md"))
	}
	snapshot(t, ps...)
	if err := missionRotateLog([]string{"rejcanary", "--status"}); err == nil || !strings.Contains(err.Error(), "D-FLEET-9") {
		t.Fatalf("bad migration rejection: %v", err)
	}
}
func TestMissionRotateLogStreamStatusRotatesStatusArchive(t *testing.T) {
	root, reg, _ := writeSyntheticRepo(t, "statuscanary", true, 10)
	use(t, root, reg)
	d := filepath.Join(root, "design_docs")
	snapshot(t, filepath.Join(d, "statuscanary-mission-log.md"))
	must(t, missionRotateLog([]string{"statuscanary", "--stream", "status", "--keep", "5"}))
	checkRotated(t, d, "statuscanary-mission-status-archive.md", "statuscanary-mission-status-archive-old.md", "statuscanary-mission-status-index.md")
}
func TestMissionRotateLogRejectsUnknownStream(t *testing.T) {
	root, reg, _ := writeSyntheticRepo(t, "badstream", true, 10)
	use(t, root, reg)
	snapshot(t, filepath.Join(root, "design_docs", "badstream-mission-log.md"))
	if err := missionRotateLog([]string{"badstream", "--stream", "bogus"}); err == nil || !strings.Contains(err.Error(), "log|status") {
		t.Fatalf("bad unknown-stream rejection: %v", err)
	}
}
func TestMissionNormalizeSharedRepoTargetsRegistryOrigin(t *testing.T) {
	aRoot, aReg, _ := writeSyntheticRepo(t, "normcanary", true, 10)
	bRoot, _, _ := writeSyntheticRepo(t, "normcanary", true, 10)
	wordy := "# normcanary Mission Log\n\nPreamble that must survive rotation.\n\n## Iteration 3 — 2026-09-30 — Did the thing\n\nbody\n"
	aLog := filepath.Join(aRoot, "design_docs", "normcanary-mission-log.md")
	bLog := filepath.Join(bRoot, "design_docs", "normcanary-mission-log.md")
	synthWrite(t, aLog, wordy)
	synthWrite(t, bLog, wordy)
	use(t, bRoot, aReg)
	snapshot(t, bLog)
	must(t, missionNormalize([]string{"--apply"}))
	if got := read(t, aLog); !strings.Contains(got, "## 3 — 2026-09-30 — Did the thing") {
		t.Errorf("shared normalize did not rewrite at registry origin: %s", got)
	}
}
func TestMissionNormalizeNoRegistryStillFailsLoudly(t *testing.T) {
	use(t, t.TempDir(), "")
	if err := missionNormalize(nil); err == nil || !strings.Contains(err.Error(), "mission registry") {
		t.Fatalf("no-registry normalize did not fail loudly: %v", err)
	}
}
func TestMissionRegistryRootAccessor(t *testing.T) {
	_, reg, _ := writeSyntheticRepo(t, "rootcanary", true, 10)
	t.Setenv("AILANG_MISSION_REGISTRY", reg)
	loaded, err := loadMissionRegistry()
	must(t, err)
	m, ok := loaded.Get("rootcanary")
	if !ok || m.Root() != filepath.Dir(reg) || m.Root() == reg {
		t.Fatalf("Root() = %q, want parent %q", m.Root(), filepath.Dir(reg))
	}
}

func TestRegistryWalkCandidatesTerminatesAtVolumeRoot(t *testing.T) {
	wd := t.TempDir()
	done := make(chan []string, 1)
	go func() { done <- registryWalkCandidates(wd) }()
	var got []string
	select {
	case got = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("registryWalkCandidates did not terminate")
	}
	depth := strings.Count(filepath.ToSlash(wd), "/") + 1
	if len(got) == 0 || len(got) > depth {
		t.Fatalf("got %d candidates for depth %d: %v", len(got), depth, got)
	}
	if got[0] != filepath.Join(wd, missionRegistryDir) {
		t.Errorf("first candidate = %q, want one under wd", got[0])
	}
	last := filepath.Dir(filepath.Dir(got[len(got)-1]))
	root := filepath.VolumeName(wd) + string(filepath.Separator)
	if last != root {
		t.Errorf("last candidate's grandparent = %q, want volume root %q", last, root)
	}
}
