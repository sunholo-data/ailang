package mission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeFleet builds a fixture $HOME with the two artifact trees, so the doctor can be
// exercised deterministically on any machine — including CI, where the real fleet
// does not exist.
type fakeFleet struct {
	t    *testing.T
	p    Paths
	reg  *Registry
	root string
}

func newFleet(t *testing.T) *fakeFleet {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{
		filepath.Join(root, ".config", "ailang"),
		filepath.Join(root, "Library", "LaunchAgents"),
	} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	return &fakeFleet{t: t, p: Paths{Home: root}, reg: &Registry{}, root: root}
}

// addMission writes a driver, an env file and a plist, then registers the mission.
func (f *fakeFleet) addMission(name, repo string, sched Schedule, env, plistExtra string, pinned bool) *Mission {
	f.t.Helper()
	workdir := filepath.Join(f.root, "repos", name)
	if err := os.MkdirAll(filepath.Join(workdir, "tools", "launchd"), 0o750); err != nil {
		f.t.Fatal(err)
	}
	driver := "#!/bin/bash\n"
	if pinned {
		driver += pinSentinel + "\n"
	}
	if err := os.WriteFile(filepath.Join(workdir, "tools", "launchd", "mission-control.sh"), []byte(driver), 0o600); err != nil {
		f.t.Fatal(err)
	}
	m := &Mission{Name: name, Repo: repo, Doc: "d.md", Workdir: workdir, Sched: sched, Path: "fixture:" + name}
	if err := m.Validate(); err != nil {
		f.t.Fatalf("fixture mission invalid: %v", err)
	}
	if err := os.WriteFile(f.p.EnvPath(name), []byte(env), 0o600); err != nil {
		f.t.Fatal(err)
	}
	knob, _ := scheduleKnob(sched)
	plist := "<plist><dict><key>Label</key><string>" + m.Label() + "</string>\n" + knob + "\n" + plistExtra + "</dict></plist>\n"
	if err := os.WriteFile(f.p.PlistPath(m), []byte(plist), 0o600); err != nil {
		f.t.Fatal(err)
	}
	f.reg.Missions = append(f.reg.Missions, m)
	return m
}

func (f *fakeFleet) run() *Report { return Doctor(f.reg, f.p) }

func findingsOfKind(r *Report, kind string) []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Kind == kind {
			out = append(out, f)
		}
	}
	return out
}

// A doctor that reports a clean fleet is indistinguishable from one that is not
// looking. This is the control: a fleet with nothing wrong must produce nothing.
func TestDoctor_CleanFleetIsClean(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, "", "", true)
	// Re-write the env so it agrees with the registry exactly.
	rendered, err := RenderEnv(m, []byte(env))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600); err != nil {
		t.Fatal(err)
	}
	rep := f.run()
	if rep.HasDrift() {
		t.Fatalf("a clean fleet must report no drift; got:\n%s", strings.Join(findingStrings(rep), "\n"))
	}
	if rep.ExitCode() != 0 {
		t.Errorf("ExitCode = %d, want 0", rep.ExitCode())
	}
}

// ── THE GATE ─────────────────────────────────────────────────────────────────
// Three divergences measured on the live rig 2026-09-05. The doctor is not trusted
// unless it finds each. These are fixtures so they run in CI; the live counterparts
// are below.

// V4/V5: the installed env disagrees with the reviewed source, and the disagreeing
// key is MISSION_PLANNER_ALLOWLIST — which is why docs work routed to opus instead of
// codex while a test asserting the repo copy stayed green. The finding must NAME the
// key, not merely say the files differ.
func TestDoctor_GATE_V4_V5_EnvDriftIsFoundAndNamed(t *testing.T) {
	f := newFleet(t)
	// The installed file carries a NARROWER allowlist than the registry-rendered
	// one would — exactly the docs situation.
	installed := "MISSION_NAME=docs\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n" +
		`MISSION_PLANNER_ALLOWLIST="tools/*|docs/*"` + "\n"
	m := f.addMission("docs", sharedDriverRepo, Schedule{Mode: ModeInterval, IntervalSeconds: 21600}, installed, "", true)
	// The registry says the workdir is elsewhere: a concrete, named disagreement.
	m.Workdir = filepath.Join(f.root, "repos", "docs-moved")
	if err := os.MkdirAll(filepath.Join(m.Workdir, "tools", "launchd"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(m.Workdir, "tools", "launchd", "mission-control.sh"),
		[]byte("#!/bin/bash\n"+pinSentinel+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	rep := f.run()
	got := findingsOfKind(rep, "env-drift")
	if len(got) == 0 {
		t.Fatalf("GATE FAILED: env drift not detected. Findings:\n%s", strings.Join(findingStrings(rep), "\n"))
	}
	if !strings.Contains(got[0].Detail, "MISSION_WORKDIR") {
		t.Errorf("drift report must NAME the disagreeing key; got: %s", got[0].Detail)
	}
	if !rep.HasDrift() || rep.ExitCode() != 1 {
		t.Errorf("env drift must fail the run: HasDrift=%v ExitCode=%d", rep.HasDrift(), rep.ExitCode())
	}
}

// V8: a plist that sets its own PATH without /usr/sbin. `sysctl` becomes unreachable,
// _mc_uptime_secs cannot read kern.boottime, and the boot stagger goes inert while
// announcing itself only as a log line.
func TestDoctor_GATE_V8_PlistPATHWithoutSysctlIsFound(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=v1\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("v1", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
		env, "<key>PATH</key><string>/usr/local/bin:/usr/bin:/bin</string>", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("v1"), rendered, 0o600)

	rep := f.run()
	got := findingsOfKind(rep, "path-no-sysctl")
	if len(got) == 0 {
		t.Fatalf("GATE FAILED: a PATH without /usr/sbin was not detected. Findings:\n%s", strings.Join(findingStrings(rep), "\n"))
	}
	if !strings.Contains(got[0].Detail, "sysctl") || !strings.Contains(got[0].Detail, "stagger") {
		t.Errorf("the finding must explain the consequence, not just the omission; got: %s", got[0].Detail)
	}
}

// The complement, and it is load-bearing: motoko and world set NO PATH key and
// inherit launchd's default, which DOES include /usr/sbin. Firing on them would make
// the check noise and train the reader to ignore it.
func TestDoctor_V8_DoesNotFireWhenThePlistSetsNoPATH(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=motoko\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("motoko", sharedDriverRepo, Schedule{Mode: ModeInterval, IntervalSeconds: 46800}, env, "", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("motoko"), rendered, 0o600)

	if got := findingsOfKind(f.run(), "path-no-sysctl"); len(got) != 0 {
		t.Errorf("must NOT fire on a plist with no PATH key (it inherits one containing /usr/sbin); got: %v", got)
	}
	_ = m
}

// The reach question, computed instead of maintained by hand in a comment.
func TestDoctor_ReportsForkAndMissingPin(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=world\nMISSION_REPO=sunholo-data/ailang-world\nMISSION_DOC=d.md\n"
	m := f.addMission("world", "sunholo-data/ailang-world", Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 14400},
		env, "", false) // unpinned fork, exactly like the real world mission
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("world"), rendered, 0o600)

	rep := f.run()
	if len(findingsOfKind(rep, "no-pin")) == 0 {
		t.Error("an unpinned driver must be reported — it runs its working tree, so upstream fixes never reach it")
	}
	if len(findingsOfKind(rep, "driver-fork")) == 0 {
		t.Error("a driver outside the shared repo must be reported as a fork")
	}
	if rep.Rows[0].Pinned || !rep.Rows[0].Fork {
		t.Errorf("row should say unpinned fork; got %+v", rep.Rows[0])
	}
	// The fork is a NOTE, not drift: it is true and worth knowing, but it is not
	// something the registry can fix, and failing on it every run would make the
	// exit code useless.
	for _, fi := range findingsOfKind(rep, "driver-fork") {
		if fi.Severity != Note {
			t.Errorf("fork should be %s, not %s", Note, fi.Severity)
		}
	}
}

func TestDoctor_MissingArtifactsAreDrift(t *testing.T) {
	f := newFleet(t)
	m := &Mission{Name: "ghost", Repo: sharedDriverRepo, Doc: "d.md",
		Workdir: filepath.Join(f.root, "nope"),
		Sched:   Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 100}, Path: "fixture"}
	f.reg.Missions = append(f.reg.Missions, m)

	rep := f.run()
	for _, kind := range []string{"env-missing", "plist-missing", "driver-missing"} {
		if len(findingsOfKind(rep, kind)) == 0 {
			t.Errorf("missing artifact %q not reported", kind)
		}
	}
	if rep.ExitCode() != 1 {
		t.Errorf("ExitCode = %d, want 1", rep.ExitCode())
	}
}

// Phase 1 is read-only. A detector that writes is a detector that can break the thing
// it inspects.
func TestDoctor_IsReadOnly(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, env, "", true)

	walk := func() map[string]os.FileInfo {
		out := map[string]os.FileInfo{}
		_ = filepath.Walk(f.root, func(p string, fi os.FileInfo, err error) error {
			if err == nil && !fi.IsDir() {
				out[p] = fi
			}
			return nil
		})
		return out
	}
	snap := walk()
	f.run()
	after := walk()
	if len(after) != len(snap) {
		t.Fatalf("doctor changed the file set: %d before, %d after", len(snap), len(after))
	}
	for p, before := range snap {
		a, ok := after[p]
		if !ok {
			t.Errorf("doctor removed %s", p)
			continue
		}
		if !a.ModTime().Equal(before.ModTime()) || a.Size() != before.Size() {
			t.Errorf("doctor modified %s", p)
		}
	}
}

func findingStrings(r *Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.String())
	}
	return out
}
