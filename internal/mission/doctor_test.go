package mission

import (
	"errors"
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
		env, "", false) // unpinned, exactly like the real world mission
	// A FORK IS A DECLARED CHOICE since the driver location was decoupled from the
	// workdir. Working in another repo no longer implies running your own driver — that
	// is the whole point of the decoupling — so the fixture must SAY it forks.
	m.Driver = m.DriverPath()
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

// ── the FILE is not the JOB ───────────────────────────────────────────────────
// launchd reads a plist ONCE, at bootstrap. Editing the file afterwards changes
// nothing until a reload, so every file-based check can pass while the mission runs
// old settings. Phase 2 created this blind spot the moment it promoted v1's plist
// with --no-reload, and these tests are what close it.

// printingLaunchCtl returns canned `launchctl print` output.
type printingLaunchCtl struct {
	out string
	err error
}

func (p *printingLaunchCtl) Bootout(string) error         { return nil }
func (p *printingLaunchCtl) Bootstrap(string) error       { return nil }
func (p *printingLaunchCtl) Print(string) (string, error) { return p.out, p.err }

func TestDoctor_LoadedConfigOlderThanTheFileIsDrift(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
		env, "<key>StandardOutPath</key><string>/tmp/new.launchd.log</string><key>PATH</key><string>/usr/bin:/usr/sbin</string>", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600)

	// launchd still holds the PRE-adoption config: old log path, no /usr/sbin.
	lc := &printingLaunchCtl{out: "\tstate = running\n\tstdout path = /tmp/old.stdout\n\tstderr path = /tmp/old.stderr\n\t\tPATH => /usr/bin:/bin\n"}

	rep := DoctorWith(f.reg, f.p, lc)
	got := findingsOfKind(rep, "loaded-stale")
	if len(got) == 0 {
		t.Fatalf("a loaded config older than the file must be DRIFT. Findings:\n%s", strings.Join(findingStrings(rep), "\n"))
	}
	d := got[0].Detail
	if !strings.Contains(d, "/tmp/old.stdout") || !strings.Contains(d, "/tmp/new.launchd.log") {
		t.Errorf("the finding must name BOTH sides of the disagreement; got: %s", d)
	}
	if !strings.Contains(d, "mission apply alpha") {
		t.Errorf("the finding should name the command that fixes it; got: %s", d)
	}
}

// The V8 surface specifically: a loaded PATH without /usr/sbin means the boot stagger
// is still inert on the RUNNING job however good the file is.
func TestDoctor_LoadedPATHWithoutSysctlIsDriftEvenWhenTheFileIsFixed(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
		env, "<key>StandardOutPath</key><string>/tmp/x.launchd.log</string><key>PATH</key><string>/usr/bin:/usr/sbin</string>", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600)

	lc := &printingLaunchCtl{out: "\tstate = running\n\tstdout path = /tmp/x.launchd.log\n\t\tPATH => /usr/bin:/bin\n"}
	rep := DoctorWith(f.reg, f.p, lc)
	found := false
	for _, fi := range findingsOfKind(rep, "loaded-stale") {
		if strings.Contains(fi.Detail, "/usr/sbin") {
			found = true
		}
	}
	if !found {
		t.Errorf("a loaded PATH lacking /usr/sbin must be reported even though the FILE has it. Findings:\n%s",
			strings.Join(findingStrings(rep), "\n"))
	}
}

// The complement, and it carries the weight: a job whose loaded config MATCHES must
// not be flagged, or the check is a constant and gets ignored.
func TestDoctor_MatchingLoadedConfigIsNotFlagged(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
		env, "<key>StandardOutPath</key><string>/tmp/match.launchd.log</string><key>PATH</key><string>/usr/bin:/usr/sbin</string>", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600)

	lc := &printingLaunchCtl{out: "\tstate = running\n\tstdout path = /tmp/match.launchd.log\n\t\tPATH => /usr/bin:/usr/sbin\n"}
	if got := findingsOfKind(DoctorWith(f.reg, f.p, lc), "loaded-stale"); len(got) != 0 {
		t.Errorf("a matching loaded config must NOT be flagged; got %v", got)
	}
}

// An installed plist that launchd has never bootstrapped will never run.
func TestDoctor_InstalledButNotLoadedIsDrift(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, env, "", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600)

	lc := &printingLaunchCtl{err: errors.New("could not find service")}
	rep := DoctorWith(f.reg, f.p, lc)
	got := findingsOfKind(rep, "job-not-loaded")
	if len(got) == 0 {
		t.Fatal("an installed-but-unloaded job must be reported — it will never run")
	}
	// SEVERITY IS THE POINT, and a mutation caught this assertion missing: a job
	// launchd has never bootstrapped does not run at all. Downgrading that to a note
	// would let `doctor` exit 0 on a mission that is simply absent.
	if got[0].Severity != Drift {
		t.Errorf("an unloaded job is %s, not %s — the mission is not running", Drift, got[0].Severity)
	}
	if !rep.HasDrift() || rep.ExitCode() != 1 {
		t.Errorf("an unloaded job must fail the run: HasDrift=%v exit=%d", rep.HasDrift(), rep.ExitCode())
	}
}

// Passing no LaunchCtl skips the check rather than inventing an answer.
func TestDoctor_NilLaunchCtlSkipsTheLoadedCheck(t *testing.T) {
	f := newFleet(t)
	env := "MISSION_NAME=alpha\nMISSION_REPO=sunholo-data/ailang\nMISSION_DOC=d.md\n"
	m := f.addMission("alpha", sharedDriverRepo, Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400}, env, "", true)
	rendered, _ := RenderEnv(m, []byte(env))
	_ = os.WriteFile(f.p.EnvPath("alpha"), rendered, 0o600)

	rep := DoctorWith(f.reg, f.p, nil)
	for _, k := range []string{"loaded-stale", "job-not-loaded"} {
		if len(findingsOfKind(rep, k)) != 0 {
			t.Errorf("a nil LaunchCtl must skip the loaded check, not report %q", k)
		}
	}
}
