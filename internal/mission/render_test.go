package mission

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func mustMission(t *testing.T, body string) *Mission {
	t.Helper()
	p := writeEntry(t, t.TempDir(), "m.toml", body)
	m, err := LoadFile(p)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	return m
}

// ── env rendering ────────────────────────────────────────────────────────────

// THE PROPERTY THAT MAKES PHASE 2 SAFE. If the registry agrees with the live file,
// rendering must reproduce it byte-for-byte — comments, blank lines, role config and
// all. Anything less and promoting a rendered env is a change of unknown size.
func TestRenderEnv_ByteIdenticalWhenRegistryAgrees(t *testing.T) {
	live := `# Ailang World mission profile
MISSION_NAME=world
MISSION_REPO=sunholo-data/ailang-world
MISSION_DOC=design_docs/world-mission.md
MISSION_WORKDIR="${MISSION_WORKDIR:-/Users/x/dev/ailang-world}"

# quota plan comment that must survive
export MISSION_EXECUTOR_MODEL="${MISSION_EXECUTOR_MODEL:-codex:gpt-5.6-sol}"
PATH=/opt/homebrew/bin:$PATH
`
	m := mustMission(t, `
name    = "world"
repo    = "sunholo-data/ailang-world"
workdir = "/Users/x/dev/ailang-world"
doc     = "design_docs/world-mission.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 14400
boot_offset      = 420
`)
	got, err := RenderEnv(m, []byte(live))
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	if string(got) != live {
		t.Errorf("render must be byte-identical when the registry agrees.\n--- want ---\n%s\n--- got ---\n%s", live, got)
	}
}

// Role/model config is PASSTHROUGH by ratification. A line the registry has never
// heard of must survive untouched — that is what keeps this package from becoming a
// second source of model assignment while M8 is parked.
func TestRenderEnv_UnknownRoleLinesPassThroughVerbatim(t *testing.T) {
	live := `MISSION_NAME=world
MISSION_REPO=r/x
MISSION_DOC=d.md
MISSION_WORKDIR=/tmp/w
export MISSION_EVALUATOR_FALLBACK="pi:ollama/minimax-m3:cloud,codex:gpt-6-astra"
export MISSION_SOMETHING_NOBODY_MODELLED="keep me exactly as I am"
`
	m := mustMission(t, `
name    = "world"
repo    = "r/x"
workdir = "/tmp/w"
doc     = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 100
boot_offset      = 1
`)
	got, err := RenderEnv(m, []byte(live))
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	for _, want := range []string{
		`export MISSION_EVALUATOR_FALLBACK="pi:ollama/minimax-m3:cloud,codex:gpt-6-astra"`,
		`export MISSION_SOMETHING_NOBODY_MODELLED="keep me exactly as I am"`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("passthrough line lost:\n  %s", want)
		}
	}
}

// The ${VAR:-value} form is a rollback ergonomic — it lets an operator override a
// mission from the environment without editing the file. Rewriting it to a bare
// assignment would quietly remove that escape hatch.
func TestRenderEnv_PreservesOverrideIdiom(t *testing.T) {
	m := mustMission(t, `
name    = "w"
repo    = "r/x"
workdir = "/tmp/new"
doc     = "d.md"
[schedule]
mode             = "interval"
interval_seconds = 60
boot_offset      = 3
`)
	got, err := RenderEnv(m, []byte("MISSION_WORKDIR=\"${MISSION_WORKDIR:-/tmp/old}\"\n"))
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	want := `MISSION_WORKDIR="${MISSION_WORKDIR:-/tmp/new}"`
	if !strings.Contains(string(got), want) {
		t.Errorf("override idiom not preserved with the new value.\nwant %s\ngot  %s", want, got)
	}
	if strings.Contains(string(got), "/tmp/old") {
		t.Error("stale value survived the render")
	}
}

func TestRenderEnv_AppendsMissingVarsForANewMission(t *testing.T) {
	m := mustMission(t, `
name    = "parse"
repo    = "sunholo-data/ailang-parse"
workdir = "/tmp/parse"
doc     = "design_docs/parse-mission.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 10800
boot_offset      = 1680
`)
	got, err := RenderEnv(m, nil) // no live file: a brand-new mission
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	// MISSION_WORKDIR uses the override form even when appended — see
	// TestRenderEnv_AppendedWorkdirCannotClobberThePin for why that is mandatory.
	for _, want := range []string{"MISSION_NAME=parse", "MISSION_REPO=sunholo-data/ailang-parse",
		"MISSION_DOC=design_docs/parse-mission.md", `MISSION_WORKDIR="${MISSION_WORKDIR:-/tmp/parse}"`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("new-mission render missing %q\ngot:\n%s", want, got)
		}
	}
}

// ── plist rendering ──────────────────────────────────────────────────────────

func TestRenderPlist_ScheduleKnobsAreNeverBoth(t *testing.T) {
	ka := mustMission(t, `
name = "w"
repo = "r/x"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 14400
boot_offset      = 1
`)
	b, err := RenderPlist(ka)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "<key>KeepAlive</key>") || !strings.Contains(s, "<key>ThrottleInterval</key>\n\t<integer>14400</integer>") {
		t.Errorf("keepalive must render KeepAlive + ThrottleInterval:\n%s", s)
	}
	if strings.Contains(s, "StartInterval") {
		t.Error("keepalive must NOT render StartInterval — the combined launchd behaviour is unmeasured")
	}

	iv := mustMission(t, `
name = "d"
repo = "r/x"
workdir = "/tmp/d"
doc = "d.md"
[schedule]
mode             = "interval"
interval_seconds = 21600
boot_offset      = 2
`)
	b2, err := RenderPlist(iv)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	s2 := string(b2)
	if !strings.Contains(s2, "<key>StartInterval</key>\n\t<integer>21600</integer>") {
		t.Errorf("interval must render StartInterval:\n%s", s2)
	}
	if strings.Contains(s2, "KeepAlive") || strings.Contains(s2, "ThrottleInterval") {
		t.Error("interval must NOT render KeepAlive/ThrottleInterval")
	}
}

// V8, enforced at render time. /usr/sbin's absence from the v1 and docs plists made
// `sysctl` unreachable and shipped the boot stagger inert on two of four missions.
// No generated plist may reintroduce that by omission.
func TestRenderPlist_PATHAlwaysReachesSysctl(t *testing.T) {
	m := mustMission(t, `
name = "w"
repo = "r/x"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 100
boot_offset      = 1
`)
	b, err := RenderPlist(m)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	if !strings.Contains(string(b), "/usr/sbin") {
		t.Error("rendered PATH must contain /usr/sbin or the boot stagger goes inert (V8)")
	}
}

// The v1 job is dev.ailang.mission-CONTROL, not -v1. That name predates
// MISSION_PROFILE and the recovery job, pidfile and logs all key off it.
func TestRenderPlist_V1KeepsItsHistoricalLabel(t *testing.T) {
	v1 := mustMission(t, `
name = "v1"
repo = "sunholo-data/ailang"
workdir = "/tmp/ailang"
doc = "design_docs/v1-mission.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 5400
boot_offset      = 0
`)
	if got := v1.Label(); got != "dev.ailang.mission-control" {
		t.Errorf("v1 label = %q, want dev.ailang.mission-control", got)
	}
	other := mustMission(t, `
name = "world"
repo = "r/x"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 100
boot_offset      = 1
`)
	if got := other.Label(); got != "dev.ailang.mission-world" {
		t.Errorf("world label = %q", got)
	}
}

// Every rendered plist must be a plist. plutil is the same parser launchd uses.
func TestRenderPlist_PassesPlutilLint(t *testing.T) {
	if _, err := exec.LookPath("plutil"); err != nil {
		t.Skip("plutil unavailable (not macOS)")
	}
	m := mustMission(t, `
name = "world"
repo = "sunholo-data/ailang-world"
workdir = "/Users/x/dev/ailang-world"
doc = "design_docs/world-mission.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 14400
boot_offset      = 420
`)
	b, err := RenderPlist(m)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	p := filepath.Join(t.TempDir(), "x.plist")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("plutil", "-lint", p).CombinedOutput(); err != nil {
		t.Fatalf("plutil -lint rejected the rendered plist: %v\n%s\n--- plist ---\n%s", err, out, b)
	}
}

// A value carrying XML metacharacters must not be able to produce a malformed plist.
func TestRenderPlist_EscapesXMLMetacharacters(t *testing.T) {
	m := mustMission(t, `
name = "w"
repo = "org/a&b<c>"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 100
boot_offset      = 1
`)
	b, err := RenderPlist(m)
	if err != nil {
		t.Fatalf("RenderPlist: %v", err)
	}
	if strings.Contains(string(b), "a&b<c>") {
		t.Error("XML metacharacters must be escaped")
	}
}

// ── staging ──────────────────────────────────────────────────────────────────

// Phase 1's central safety property: nothing this package writes is a path the fleet
// reads. Asserted by leaving the live targets ABSENT and proving they stay absent.
func TestRenderStaged_NeverWritesALivePath(t *testing.T) {
	dir := t.TempDir()
	envTarget := filepath.Join(dir, "mission-world.env")
	plistTarget := filepath.Join(dir, "dev.ailang.mission-world.plist")

	m := mustMission(t, `
name = "world"
repo = "r/x"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 14400
boot_offset      = 420
`)
	s, err := RenderStaged(m, envTarget, plistTarget)
	if err != nil {
		t.Fatalf("RenderStaged: %v", err)
	}
	for _, live := range []string{envTarget, plistTarget} {
		if _, err := os.Stat(live); !os.IsNotExist(err) {
			t.Errorf("LIVE path %s was created — install must render only, never apply", live)
		}
	}
	for _, staged := range []string{s.EnvStaged, s.PlistStaged} {
		if !strings.HasSuffix(staged, StagedSuffix) {
			t.Errorf("staged path %q lacks %q", staged, StagedSuffix)
		}
		if _, err := os.Stat(staged); err != nil {
			t.Errorf("staged path %s not written: %v", staged, err)
		}
	}
	// And the directory must contain nothing but the two staged files.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected exactly 2 staged files, got %v", names)
	}
}

// An existing live env is READ (for passthrough) and left untouched.
func TestRenderStaged_ReadsLiveEnvWithoutModifyingIt(t *testing.T) {
	dir := t.TempDir()
	envTarget := filepath.Join(dir, "mission-world.env")
	original := "MISSION_NAME=world\nexport MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol\n"
	if err := os.WriteFile(envTarget, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	m := mustMission(t, `
name = "world"
repo = "r/x"
workdir = "/tmp/w"
doc = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 14400
boot_offset      = 420
`)
	s, err := RenderStaged(m, envTarget, filepath.Join(dir, "x.plist"))
	if err != nil {
		t.Fatalf("RenderStaged: %v", err)
	}
	after, _ := os.ReadFile(envTarget)
	if string(after) != original {
		t.Errorf("the LIVE env was modified.\nbefore: %q\nafter:  %q", original, after)
	}
	staged, _ := os.ReadFile(s.EnvStaged)
	if !strings.Contains(string(staged), "MISSION_EXECUTOR_MODEL=codex:gpt-5.6-sol") {
		t.Error("staged render lost the passthrough role line")
	}
}

func TestWriteAtomic_ReplacesContentAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f")
	if err := writeAtomic(p, []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(p, []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "two" {
		t.Errorf("content = %q, want two", got)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("temp files left behind: %d entries", len(entries))
	}
}

// BUG FOUND BY DIFFING A STAGED PLIST BEFORE APPLYING IT. The driver writes its own
// $LOG at /tmp/ailang-mission-<suffix>.log — the file mission-recovery.sh reads. An
// earlier renderer pointed launchd's stdout AND stderr at that same path, which would
// have interleaved launchd's spawn noise into the driver's own record.
func TestRenderPlist_LaunchdStreamDoesNotShareTheDriversLogFile(t *testing.T) {
	for _, name := range []string{"v1", "world", "docs", "motoko"} {
		m := &Mission{Name: name, Repo: "r/x", Doc: "d.md", Workdir: "/tmp/w",
			Sched: Schedule{Mode: ModeKeepAlive, ThrottleSeconds: 100}, Path: "t"}
		b, err := RenderPlist(m)
		if err != nil {
			t.Fatalf("RenderPlist(%s): %v", name, err)
		}
		driverLog := "/tmp/ailang-mission-" + m.launchdSuffix() + ".log"
		if strings.Contains(string(b), "<string>"+driverLog+"</string>") {
			t.Errorf("%s: launchd stdout/stderr points at the DRIVER's own log %s — "+
				"both would append to the file mission-recovery reads", name, driverLog)
		}
		if !strings.Contains(string(b), ".launchd.log") {
			t.Errorf("%s: expected launchd's stream on a .launchd.log path", name)
		}
	}
}

// BUG FOUND THE SAME WAY. pin-root.sh exports MISSION_WORKDIR as the PINNED worktree
// and re-execs; the driver then re-sources the env file, so a BARE assignment clobbers
// the pin back to the source clone. That was found and fixed by hand in
// mission-world.env on 2026-08-18; a generator emitting a bare assignment would have
// reintroduced it across every mission at once.
func TestRenderEnv_AppendedWorkdirCannotClobberThePin(t *testing.T) {
	m := mustMission(t, `
name    = "v1"
repo    = "sunholo-data/ailang"
workdir = "/Users/x/dev/sunholo-data/ailang"
doc     = "design_docs/v1-mission.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 5400
boot_offset      = 0
`)
	// A comments-only live file, exactly like v1's: nothing to copy the idiom from.
	got, err := RenderEnv(m, []byte("# nothing but comments\n"))
	if err != nil {
		t.Fatalf("RenderEnv: %v", err)
	}
	want := `MISSION_WORKDIR="${MISSION_WORKDIR:-/Users/x/dev/sunholo-data/ailang}"`
	if !strings.Contains(string(got), want) {
		t.Errorf("an APPENDED MISSION_WORKDIR must use the override form or it clobbers "+
			"the pin's export.\nwant: %s\ngot:\n%s", want, got)
	}
	if strings.Contains(string(got), "\nMISSION_WORKDIR=/Users") {
		t.Error("bare MISSION_WORKDIR assignment would clobber pin-root's export")
	}
}

// M9: passthrough comes from the REVIEWED copy, not the installed one.
//
// Rendering from the installed file makes the output a copy of what already runs, so
// an edit to the repo copy never reaches the fleet — V5 exactly, merely narrowed.
// Rendering from the reviewed copy is what makes "edit the repo file, install, apply"
// actually deploy.
func TestRenderStagedFrom_PrefersTheReviewedCopy(t *testing.T) {
	dir := t.TempDir()
	envTarget := filepath.Join(dir, "installed.env")
	reviewed := filepath.Join(dir, "reviewed.env")

	// The two disagree exactly the way docs did: the reviewed copy carries a widened
	// allowlist that was never deployed.
	if err := os.WriteFile(envTarget, []byte("MISSION_NAME=docs\nMISSION_PLANNER_ALLOWLIST=\"tools/*\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reviewed, []byte("MISSION_NAME=docs\nMISSION_PLANNER_ALLOWLIST=\"tools/*|scripts/*\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := mustMission(t, `
name    = "docs"
repo    = "sunholo-data/ailang"
workdir = "/tmp/docs"
doc     = "d.md"
[schedule]
mode             = "interval"
interval_seconds = 21600
boot_offset      = 840
`)
	s, err := RenderStagedFrom(m, envTarget, filepath.Join(dir, "x.plist"), reviewed)
	if err != nil {
		t.Fatalf("RenderStagedFrom: %v", err)
	}
	if s.Source != reviewed {
		t.Errorf("Source = %s, want the reviewed copy %s", s.Source, reviewed)
	}
	got, _ := os.ReadFile(s.EnvStaged)
	if !strings.Contains(string(got), "scripts/*") {
		t.Errorf("the REVIEWED allowlist must reach the staged render — this is the V5 workflow, fixed.\ngot:\n%s", got)
	}
	// And the installed file is still only read.
	after, _ := os.ReadFile(envTarget)
	if strings.Contains(string(after), "scripts/*") {
		t.Error("the installed file must not be written by a render")
	}
}

// A mission with no reviewed copy still renders, from the installed file.
func TestRenderStagedFrom_FallsBackToInstalledWhenUnreviewed(t *testing.T) {
	dir := t.TempDir()
	envTarget := filepath.Join(dir, "installed.env")
	if err := os.WriteFile(envTarget, []byte("MISSION_NAME=solo\nexport MISSION_X=keep\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := mustMission(t, `
name    = "solo"
repo    = "r/x"
workdir = "/tmp/solo"
doc     = "d.md"
[schedule]
mode             = "keepalive"
throttle_seconds = 900
boot_offset      = 7
`)
	s, err := RenderStagedFrom(m, envTarget, filepath.Join(dir, "x.plist"), filepath.Join(dir, "absent.env"))
	if err != nil {
		t.Fatalf("RenderStagedFrom: %v", err)
	}
	if s.Source != envTarget {
		t.Errorf("with no reviewed copy the source must be the installed file; got %s", s.Source)
	}
	got, _ := os.ReadFile(s.EnvStaged)
	if !strings.Contains(string(got), "MISSION_X=keep") {
		t.Error("fallback render lost the passthrough")
	}
}
