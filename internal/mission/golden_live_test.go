package mission

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run against the LIVE rig. They are the evidence that promoting a
// rendered artifact in Phase 2 is a no-op, so they read the installed files rather
// than fixtures — a fixture would only prove the renderer agrees with itself.
//
// They skip on any machine without the installed fleet, which is every CI runner.
func liveEnvPath(name string) string {
	return filepath.Join(os.Getenv("HOME"), ".config", "ailang", "mission-"+name+".env")
}

// envValue pulls an assignment out of a live env file, unwrapping the
// ${VAR:-value} idiom, so the fixture Mission is built from what the file actually
// says rather than from what this test assumes.
func envValue(body, key string) string {
	for _, line := range strings.Split(body, "\n") {
		if assignmentKey(line) != key {
			continue
		}
		v := strings.TrimSpace(line)
		v = strings.TrimPrefix(v, "export ")
		v = strings.TrimSpace(v[strings.Index(v, "=")+1:])
		v = strings.Trim(v, `"`)
		if strings.HasPrefix(v, "${"+key+":-") {
			v = strings.TrimSuffix(strings.TrimPrefix(v, "${"+key+":-"), "}")
		}
		return v
	}
	return ""
}

// THE PHASE-2 SAFETY PROOF. For every installed mission, rendering from a registry
// entry that agrees with the live file must reproduce that file byte-for-byte. If
// this fails, promoting a rendered env is a change of unknown size and Phase 2 must
// not proceed.
func TestGolden_RenderEnvReproducesEveryInstalledMission(t *testing.T) {
	checked := 0
	for _, name := range []string{"v1", "docs", "motoko", "world"} {
		t.Run(name, func(t *testing.T) {
			body, err := os.ReadFile(liveEnvPath(name))
			if err != nil {
				t.Skipf("no installed env for %s: %v", name, err)
			}
			s := string(body)
			workdir := envValue(s, "MISSION_WORKDIR")
			if workdir == "" {
				// NOT a skip. v1 is the fleet's one implicit mission: its env file
				// is entirely comments (zero assignments), and its identity comes
				// from the driver's own defaults at mission-control.sh:65-67, with
				// the workdir derived from the script's location at :40. That is a
				// real, deliberate state and it is asserted here rather than
				// stepped over, because a silent skip is how the fourth mission
				// stops being checked without anyone noticing.
				assertImplicitMission(t, name, s)
				return
			}
			m := &Mission{
				Name:    envValue(s, "MISSION_NAME"),
				Repo:    envValue(s, "MISSION_REPO"),
				Doc:     envValue(s, "MISSION_DOC"),
				Workdir: workdir,
				Sched:   Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
				Path:    "golden:" + name,
			}
			if err := m.Validate(); err != nil {
				t.Skipf("live env for %s does not form a valid registry entry yet: %v", name, err)
			}
			got, err := RenderEnv(m, body)
			if err != nil {
				t.Fatalf("RenderEnv(%s): %v", name, err)
			}
			if string(got) != s {
				// Report the first differing line: a whole-file diff of a 200-line
				// env is unreadable in test output.
				gl, wl := strings.Split(string(got), "\n"), strings.Split(s, "\n")
				for i := 0; i < len(gl) && i < len(wl); i++ {
					if gl[i] != wl[i] {
						t.Fatalf("%s: render differs at line %d\n  installed: %q\n  rendered:  %q", name, i+1, wl[i], gl[i])
					}
				}
				t.Fatalf("%s: render differs in length (installed %d lines, rendered %d)", name, len(wl), len(gl))
			}
			checked++
		})
	}
	if checked == 0 {
		t.Log("no installed missions found — golden proof did not run (expected off-rig)")
	}
}

// assertImplicitMission handles a mission that declares nothing and rides the
// driver's defaults. Rendering such a mission ADDS the four identity assignments, so
// byte-equality cannot hold and would be the wrong property to demand. What must hold
// is that making the implicit explicit does not change any VALUE — and that the
// passthrough half (every comment in the file) still survives intact.
func assertImplicitMission(t *testing.T, name, live string) {
	t.Helper()
	if n := countAssignments(live); n != 0 {
		t.Fatalf("%s was treated as implicit but declares %d assignments — the exception no longer applies", name, n)
	}
	// The values the driver would have used, read from the driver rather than
	// restated here, so this test fails if a default is ever changed underneath it.
	drv, err := os.ReadFile(filepath.Join("..", "..", "tools", "launchd", "mission-control.sh"))
	if err != nil {
		t.Skipf("driver unreadable: %v", err)
	}
	def := func(key string) string {
		for _, line := range strings.Split(string(drv), "\n") {
			pre := key + "=\"${" + key + ":-"
			if strings.HasPrefix(line, pre) {
				return strings.TrimSuffix(strings.TrimPrefix(line, pre), "}\"")
			}
		}
		return ""
	}
	m := &Mission{
		Name: def("MISSION_NAME"), Repo: def("MISSION_REPO"), Doc: def("MISSION_DOC"),
		Workdir: filepath.Join(os.Getenv("HOME"), "dev", "sunholo-data", "ailang"),
		Sched:   Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400},
		Path:    "golden-implicit:" + name,
	}
	if m.Name == "" || m.Repo == "" || m.Doc == "" {
		t.Fatalf("could not read the driver defaults this mission relies on (name=%q repo=%q doc=%q)", m.Name, m.Repo, m.Doc)
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("driver defaults do not form a valid registry entry: %v", err)
	}
	got, err := RenderEnv(m, []byte(live))
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	// Every value the driver would have used is now stated explicitly...
	for _, want := range []string{
		"MISSION_NAME=" + m.Name, "MISSION_REPO=" + m.Repo, "MISSION_DOC=" + m.Doc,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("making %s explicit lost %q", name, want)
		}
	}
	// ...and nothing that was in the file was dropped.
	for _, line := range strings.Split(live, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.Contains(string(got), line) {
			t.Errorf("%s: passthrough dropped a line: %q", name, line)
		}
	}
}

// countAssignments counts real shell assignments, ignoring comments and blanks.
func countAssignments(body string) int {
	n := 0
	for _, line := range strings.Split(body, "\n") {
		if assignmentKey(line) != "" {
			n++
		}
	}
	return n
}

// The plist is fully authored, so byte-equality is neither achievable nor wanted:
// the installed v1 plist carries ~40 lines of ratified rationale a generator cannot
// regenerate. What must hold is SEMANTIC equality on the fields launchd acts on, and
// every difference must be one of the two enumerated normalisations.
func TestGolden_RenderPlistMatchesInstalledSemantics(t *testing.T) {
	if _, err := exec.LookPath("/usr/libexec/PlistBuddy"); err != nil {
		t.Skip("PlistBuddy unavailable (not macOS)")
	}
	type spec struct {
		name, label string
		mode        ScheduleMode
		secs        int
	}
	for _, sp := range []spec{
		{"v1", "dev.ailang.mission-control", ModeKeepAlive, 5400},
		{"world", "dev.ailang.mission-world", ModeKeepAlive, 14400},
		{"docs", "dev.ailang.mission-docs", ModeInterval, 21600},
		{"motoko", "dev.ailang.mission-motoko", ModeInterval, 46800},
	} {
		t.Run(sp.name, func(t *testing.T) {
			installed := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", sp.label+".plist")
			if _, err := os.Stat(installed); err != nil {
				t.Skipf("not installed: %s", installed)
			}
			read := func(key string) string {
				out, err := exec.Command("/usr/libexec/PlistBuddy", "-c", "Print :"+key, installed).Output()
				if err != nil {
					return ""
				}
				return strings.TrimSpace(string(out))
			}
			body, err := os.ReadFile(liveEnvPath(sp.name))
			if err != nil {
				t.Skipf("no env for %s", sp.name)
			}
			workdir := envValue(string(body), "MISSION_WORKDIR")
			m := &Mission{
				Name: sp.name, Repo: envValue(string(body), "MISSION_REPO"),
				Doc: envValue(string(body), "MISSION_DOC"), Workdir: workdir,
				Path: "golden:" + sp.name,
			}
			switch sp.mode {
			case ModeKeepAlive:
				m.Sched = Schedule{Mode: ModeKeepAlive, ThrottleSeconds: sp.secs}
			case ModeInterval:
				m.Sched = Schedule{Mode: ModeInterval, IntervalSeconds: sp.secs}
			}
			if err := m.Validate(); err != nil {
				t.Skipf("%s: %v", sp.name, err)
			}
			rendered, err := RenderPlist(m)
			if err != nil {
				t.Fatalf("RenderPlist: %v", err)
			}
			rs := string(rendered)

			// 1. Label must match exactly — launchd identity.
			if got := read("Label"); got != sp.label {
				t.Errorf("installed Label = %q, want %q", got, sp.label)
			}
			if !strings.Contains(rs, "<string>"+sp.label+"</string>") {
				t.Errorf("rendered plist does not carry Label %q", sp.label)
			}
			// 2. Same driver script, whatever the invocation shape.
			if !strings.Contains(rs, m.DriverPath()) {
				t.Errorf("rendered plist does not target the driver %q", m.DriverPath())
			}
			// 3. Same schedule knob and value as installed.
			switch sp.mode {
			case ModeKeepAlive:
				if got := read("ThrottleInterval"); got != itoa(sp.secs) {
					t.Errorf("installed ThrottleInterval = %q, want %d", got, sp.secs)
				}
			case ModeInterval:
				if got := read("StartInterval"); got != itoa(sp.secs) {
					t.Errorf("installed StartInterval = %q, want %d", got, sp.secs)
				}
			}
			// 4. The rendered PATH fixes V8 even where the installed one does not.
			if !strings.Contains(rs, "/usr/sbin") {
				t.Error("rendered PATH lost /usr/sbin")
			}
		})
	}
}

// The two normalisations are INTENDED, and this test exists so they are asserted
// rather than discovered. If either stops being true, the enumeration in
// RenderPlist's doc comment is stale.
func TestGolden_EnumeratedNormalisationsHold(t *testing.T) {
	m := &Mission{
		Name: "motoko", Repo: "sunholo-data/ailang", Doc: "design_docs/motoko-mission.md",
		Workdir: "/Users/x/dev/ailang-motoko",
		Sched:   Schedule{Mode: ModeInterval, IntervalSeconds: 46800, BootOffset: 1260},
		Path:    "norm-test",
	}
	b, err := RenderPlist(m)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	s := string(b)
	// Normalisation 1: explicit interpreter, even where the installed plist relies
	// on the script's shebang (motoko and world do).
	if !strings.Contains(s, "<string>/bin/bash</string>") {
		t.Error("normalisation 1 broken: ProgramArguments must name /bin/bash explicitly")
	}
	// Normalisation 2: the plist is a build product and says so, because the
	// rationale it cannot regenerate now lives in the TOML.
	if !strings.Contains(s, "DO NOT EDIT") || !strings.Contains(s, "missions/motoko.toml") {
		t.Error("normalisation 2 broken: a generated plist must point at its source of truth")
	}
}

// ── the doctor, against the LIVE rig ─────────────────────────────────────────

// liveRegistry builds registry entries for the real fleet from what is INSTALLED,
// so this test measures the rig rather than a fixture's opinion of it. M4 replaces
// this with missions/*.toml; until then, this is what proves the doctor works on the
// thing it was built for.
func liveRegistry(t *testing.T) *Registry {
	t.Helper()
	home := os.Getenv("HOME")
	type spec struct {
		name, repo, doc, workdir string
		sched                    Schedule
	}
	specs := []spec{
		{"v1", "sunholo-data/ailang", "design_docs/v1-mission.md",
			filepath.Join(home, "dev/sunholo-data/ailang"), Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 5400, BootOffset: 0}},
		{"world", "sunholo-data/ailang-world", "design_docs/world-mission.md",
			filepath.Join(home, "dev/sunholo-data/ailang-world"), Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 14400, BootOffset: 420}},
		{"docs", "sunholo-data/ailang", "design_docs/docs-mission.md",
			filepath.Join(home, "dev/sunholo-data/ailang-docs"), Schedule{Mode: ModeInterval, IntervalSeconds: 21600, BootOffset: 840}},
		{"motoko", "sunholo-data/ailang", "design_docs/motoko-mission.md",
			filepath.Join(home, "dev/sunholo-data/ailang-motoko"), Schedule{Mode: ModeInterval, IntervalSeconds: 46800, BootOffset: 1260}},
	}
	reg := &Registry{}
	for _, s := range specs {
		if _, err := os.Stat(s.workdir); err != nil {
			continue // that mission is not on this machine
		}
		m := &Mission{Name: s.name, Repo: s.repo, Doc: s.doc, Workdir: s.workdir, Sched: s.sched, Path: "live:" + s.name}
		if err := m.Validate(); err != nil {
			t.Fatalf("live spec for %s is not a valid registry entry: %v", s.name, err)
		}
		reg.Missions = append(reg.Missions, m)
	}
	if err := reg.Validate(); err != nil {
		t.Fatalf("live registry invalid: %v", err)
	}
	return reg
}

// THE GATE, ON THE REAL RIG. The design doc's own rule: a drift detector that
// reports a clean fleet is indistinguishable from one that is not looking. This test
// FAILS if the doctor comes back clean, because three divergences were measured on
// this machine on 2026-09-05 and none of them has been fixed by Phase 1.
func TestLive_DoctorReproducesTheMeasuredDivergences(t *testing.T) {
	reg := liveRegistry(t)
	if len(reg.Missions) == 0 {
		t.Skip("no missions installed on this machine (expected off-rig)")
	}
	rep := Doctor(reg, DefaultPaths())
	for _, f := range rep.Findings {
		t.Logf("%s", f)
	}

	if !rep.HasDrift() {
		t.Fatal("GATE FAILED: the doctor reports a clean fleet, but three divergences were " +
			"measured on this rig and none is fixed. A clean report here means the doctor is not looking.")
	}

	// V8 — a plist setting a PATH without /usr/sbin. Measured on v1 and docs.
	if got := findingsOfKind(rep, "path-no-sysctl"); len(got) == 0 {
		t.Error("GATE: expected a path-no-sysctl finding (v1/docs plists omit /usr/sbin)")
	} else {
		for _, f := range got {
			if f.Mission != "v1" && f.Mission != "docs" {
				t.Errorf("unexpected mission %q flagged for PATH; motoko/world set no PATH key", f.Mission)
			}
		}
	}

	// V4/V5 — the reviewed copy is not what runs.
	//
	// FIXED 2026-09-05, BY THIS TOOL FINDING IT. The docs allowlist was deployed
	// during Phase 2 and the finding disappeared, which broke this assertion — the
	// most useful way a gate can fail. It is now CONDITIONAL: the permanent,
	// deterministic proof that the doctor can find and name this class lives in
	// TestDoctor_GATE_V4_V5_EnvDriftIsFoundAndNamed, which uses a fixture and cannot
	// be silently fixed out from under itself. A live test that demands the fleet
	// stay broken is not a gate, it is a hostage.
	for _, f := range findingsOfKind(rep, "env-source-drift") {
		if f.Mission == "docs" && !strings.Contains(f.Detail, "MISSION_PLANNER_ALLOWLIST") {
			t.Errorf("if docs drifts again the finding must still NAME the key that routes "+
				"work to opus; got: %s", f.Detail)
		}
	}

	// The reach question: world is an unpinned fork. Both must be reported.
	if _, ok := reg.Get("world"); ok {
		if len(findingsOfKind(rep, "driver-fork")) == 0 {
			t.Error("GATE: world's driver is a fork in sunholo-data/ailang-world and must be reported")
		}
		var worldRow *Row
		for i := range rep.Rows {
			if rep.Rows[i].Name == "world" {
				worldRow = &rep.Rows[i]
			}
		}
		if worldRow == nil {
			t.Fatal("no row for world")
		}
		if worldRow.Pinned {
			t.Error("world must be reported as UNPINNED — it has no lib/pin-root.sh")
		}
		if !worldRow.Fork {
			t.Error("world must be reported as a fork")
		}
	}

	// And the pinned missions must NOT be reported as unpinned — otherwise the
	// check is a constant, not a measurement.
	for _, row := range rep.Rows {
		if row.Name == "world" {
			continue
		}
		if !row.Pinned {
			t.Errorf("%s should be pin-backed but was reported unpinned", row.Name)
		}
	}
}
