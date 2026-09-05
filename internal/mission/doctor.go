package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Paths locates the two artifact trees. Injected rather than read from the
// environment directly so the doctor can be tested against a fixture fleet — the
// alternative is a detector nobody can test except on the rig it is meant to check.
type Paths struct {
	Home string
	// ReviewedEnvDir holds the VERSIONED copies of the env files
	// (tools/launchd/mission-env/ in the shared repo). Empty disables the
	// installed-vs-reviewed check.
	ReviewedEnvDir string
}

// DefaultPaths uses the real $HOME and the shared repo's mission-env directory.
func DefaultPaths() Paths {
	home := os.Getenv("HOME")
	return Paths{
		Home:           home,
		ReviewedEnvDir: filepath.Join(home, "dev", "sunholo-data", "ailang", "tools", "launchd", "mission-env"),
	}
}

// ReviewedEnvPath is the versioned, code-reviewed copy of a mission's env file.
func (p Paths) ReviewedEnvPath(name string) string {
	if p.ReviewedEnvDir == "" {
		return ""
	}
	return filepath.Join(p.ReviewedEnvDir, "mission-"+name+".env")
}

// EnvPath is the env file the driver sources on every fire.
func (p Paths) EnvPath(name string) string {
	return filepath.Join(p.Home, ".config", "ailang", "mission-"+name+".env")
}

// PlistPath is the installed launchd job.
func (p Paths) PlistPath(m *Mission) string {
	return filepath.Join(p.Home, "Library", "LaunchAgents", m.Label()+".plist")
}

// Severity separates "this is wrong" from "this is worth knowing".
type Severity string

const (
	// Drift means the installed artifact disagrees with the registry. Exit 1.
	Drift Severity = "DRIFT"
	// Note is reported but does not fail: a true statement about the fleet that
	// the registry does not control.
	Note Severity = "note"
)

// Finding is one thing the doctor noticed.
type Finding struct {
	Mission  string
	Kind     string
	Severity Severity
	Detail   string
}

func (f Finding) String() string {
	return fmt.Sprintf("[%s] %-8s %-22s %s", f.Severity, f.Mission, f.Kind, f.Detail)
}

// Row is the per-mission summary line. It replaces the reach truth-table that was
// maintained by hand in a comment at mission-control.sh:752 — the point of computing
// it is that a hand-maintained table goes stale unseen.
type Row struct {
	Name     string
	Repo     string
	Schedule string
	Driver   string
	Pinned   bool
	Fork     bool
}

// Report is a whole-fleet doctor run.
type Report struct {
	Rows     []Row
	Findings []Finding
}

// HasDrift reports whether anything should fail the run.
func (r *Report) HasDrift() bool {
	for _, f := range r.Findings {
		if f.Severity == Drift {
			return true
		}
	}
	return false
}

// ExitCode is 0 clean, 1 drift. A registry load error is the caller's 2.
func (r *Report) ExitCode() int {
	if r.HasDrift() {
		return 1
	}
	return 0
}

// sharedDriverRepo is the repo whose tools/launchd/mission-control.sh is canonical.
const sharedDriverRepo = "sunholo-data/ailang"

// pinSentinel is the line that makes a clone re-exec from the committed driver pin
// rather than running whatever its working tree holds. Its ABSENCE is what made the
// world fork invisible to every routing fix landed upstream.
const pinSentinel = `. "$REPO/tools/launchd/lib/pin-root.sh"`

// Doctor inspects every registered mission. It is READ-ONLY: it opens no file for
// writing. It may READ launchd's loaded state via the injected LaunchCtl (print only),
// which is how it sees the difference between a plist on disk and the job actually
// running. Pass a nil LaunchCtl to skip that check.
func Doctor(reg *Registry, p Paths) *Report { return DoctorWith(reg, p, nil) }

// DoctorWith is Doctor plus the loaded-job check.
func DoctorWith(reg *Registry, p Paths, lc LaunchCtl) *Report {
	rep := &Report{}
	for _, m := range reg.Missions {
		row := Row{Name: m.Name, Repo: m.Repo, Schedule: describeSchedule(m.Sched), Driver: m.DriverPath()}
		add := func(kind string, sev Severity, format string, args ...any) {
			rep.Findings = append(rep.Findings, Finding{m.Name, kind, sev, fmt.Sprintf(format, args...)})
		}

		// ── 1. env drift ─────────────────────────────────────────────────────
		envPath := p.EnvPath(m.Name)
		live, err := os.ReadFile(envPath) //nolint:gosec // path derived from the registry
		switch {
		case os.IsNotExist(err):
			add("env-missing", Drift, "%s does not exist — the mission has never been installed", envPath)
		case err != nil:
			add("env-unreadable", Drift, "%s: %v", envPath, err)
		default:
			want, rerr := RenderEnv(m, live)
			if rerr != nil {
				add("env-unrenderable", Drift, "%v", rerr)
			} else if string(want) != string(live) {
				keys := changedKeys(string(live), string(want))
				add("env-drift", Drift, "%s disagrees with the registry on %s", envPath, strings.Join(keys, ", "))
			}
		}

		// ── 1b. installed vs REVIEWED ────────────────────────────────────────
		// The check that would have caught the docs bug, and the reason it is
		// separate from 1: V5 is a disagreement between the versioned copy and the
		// installed copy, and the registry owns NEITHER side of it. The allowlist
		// is passthrough, so a registry-vs-installed render can never differ on it.
		//
		// Measured 2026-09-05: mission-docs.env in the repo widened
		// MISSION_PLANNER_ALLOWLIST to admit scripts/* (PR #1010) and its own
		// comment declared itself "the sprint's only deployment surface". It was
		// never copied. Docs work routed to opus instead of codex for days, with a
		// green CI arm — because the test asserts against the repo copy while the
		// driver reads the installed one. Nothing compared the two. This does.
		if rp := p.ReviewedEnvPath(m.Name); rp != "" && len(live) > 0 {
			reviewed, rerr := os.ReadFile(rp) //nolint:gosec // path derived from the registry
			switch {
			case os.IsNotExist(rerr):
				add("env-unreviewed", Note, "no versioned copy at %s — this mission's config is unreviewable", rp)
			case rerr != nil:
				add("env-unreviewed", Note, "%s: %v", rp, rerr)
			case string(reviewed) != string(live):
				add("env-source-drift", Drift,
					"%s (reviewed) and %s (what the driver reads) disagree on %s — the reviewed copy is NOT deployed",
					rp, envPath, strings.Join(changedKeys(string(live), string(reviewed)), ", "))
			}
		}

		// ── 2. plist: does it exist, and does its schedule match? ────────────
		plistPath := p.PlistPath(m)
		plistBody, perr := os.ReadFile(plistPath) //nolint:gosec // path derived from the registry
		switch {
		case os.IsNotExist(perr):
			add("plist-missing", Drift, "%s does not exist", plistPath)
		case perr != nil:
			add("plist-unreadable", Drift, "%s: %v", plistPath, perr)
		default:
			s := string(plistBody)
			if want, ok := scheduleKnob(m.Sched); ok && !strings.Contains(s, want) {
				add("schedule-drift", Drift, "%s does not carry %s", plistPath, strings.TrimSpace(want))
			}
			// V8. /usr/sbin's absence made `sysctl` unreachable, so
			// _mc_uptime_secs could not read kern.boottime and the BOOT STAGGER
			// shipped inert on two of four missions, announcing itself only as a
			// log line. The driver now appends /usr/sbin itself, which is the
			// mitigation — but a plist that omits it is still wrong, and anything
			// not routed through that export still loses.
			if hasPATHKey(s) && !strings.Contains(s, "/usr/sbin") {
				add("path-no-sysctl", Drift,
					"%s sets a PATH without /usr/sbin — `sysctl` unreachable, boot stagger inert (mitigated in-driver, still wrong here)", plistPath)
			}
		}

		// ── 2b. the FILE is not the JOB ──────────────────────────────────────
		// The blind spot Phase 2 created, and it bit immediately: v1's plist was
		// promoted with --no-reload while it was mid-iteration, so the file on disk
		// is the generated one while launchd still holds the pre-adoption config.
		// Every file-based check passes and the mission runs old settings.
		//
		// launchd reads a plist ONCE, at bootstrap. Editing the file afterwards
		// changes nothing until a reload, so "installed" and "loaded" are two
		// different questions and only one of them is about what is running.
		if lc != nil && len(plistBody) > 0 {
			loaded, lerr := lc.Print(m.Label())
			switch {
			case lerr != nil:
				add("job-not-loaded", Drift, "%s is installed but launchd has no such job — it will not run until bootstrapped", m.Label())
			default:
				for _, mm := range loadedMismatches(string(plistBody), loaded) {
					add("loaded-stale", Drift,
						"launchd is running an OLDER config than %s: %s. The file changed but the job was never reloaded (`ailang mission apply %s`)",
						plistPath, mm, m.Name)
				}
			}
		}

		// ── 3. reach: pin, and fork ──────────────────────────────────────────
		driver, derr := os.ReadFile(m.DriverPath()) //nolint:gosec // path derived from the registry
		switch {
		case derr != nil:
			add("driver-missing", Drift, "%s: %v", m.DriverPath(), derr)
		default:
			row.Pinned = strings.Contains(string(driver), pinSentinel)
			row.Fork = m.Repo != sharedDriverRepo
			if !row.Pinned {
				add("no-pin", Drift,
					"%s does not source pin-root.sh — it runs whatever its working tree holds, so upstream driver fixes never reach it", m.DriverPath())
			}
			if row.Fork {
				add("driver-fork", Note,
					"driver lives in %s, not %s — every shared-driver change must be ported by hand until it is de-forked", m.Repo, sharedDriverRepo)
			}
		}
		rep.Rows = append(rep.Rows, row)
	}
	sort.Slice(rep.Rows, func(i, j int) bool { return rep.Rows[i].Name < rep.Rows[j].Name })
	return rep
}

// hasPATHKey reports whether the plist sets its own PATH. A plist with no PATH key
// inherits launchd's default, which DOES include /usr/sbin — that is exactly why
// motoko and world could read kern.boottime while v1 and docs could not, and why the
// check must not fire on them.
func hasPATHKey(plist string) bool { return strings.Contains(plist, "<key>PATH</key>") }

func describeSchedule(s Schedule) string {
	switch s.Mode {
	case ModeKeepAlive:
		return fmt.Sprintf("keepalive/%d", s.ThrottleSeconds)
	case ModeInterval:
		return fmt.Sprintf("interval/%d", s.IntervalSeconds)
	}
	return string(s.Mode)
}

func scheduleKnob(s Schedule) (string, bool) {
	switch s.Mode {
	case ModeKeepAlive:
		return fmt.Sprintf("<integer>%d</integer>", s.ThrottleSeconds), true
	case ModeInterval:
		return fmt.Sprintf("<integer>%d</integer>", s.IntervalSeconds), true
	}
	return "", false
}

// changedKeys names the assignments that differ, so a drift report says WHICH
// setting is wrong. "the file differs" would leave the reader to diff it themselves,
// which is how the docs allowlist sat undeployed for days behind a green test.
func changedKeys(live, want string) []string {
	parse := func(body string) map[string]string {
		out := map[string]string{}
		for _, line := range strings.Split(body, "\n") {
			if k := assignmentKey(line); k != "" {
				out[k] = strings.TrimSpace(line)
			}
		}
		return out
	}
	l, w := parse(live), parse(want)
	seen := map[string]bool{}
	var keys []string
	for k, lv := range l {
		if wv, ok := w[k]; !ok || wv != lv {
			keys = append(keys, k)
			seen[k] = true
		}
	}
	for k := range w {
		if _, ok := l[k]; !ok && !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		// The files differ but no assignment does: a comment or blank line moved.
		return []string{"(comments/whitespace only)"}
	}
	return keys
}

// loadedPathRe pulls the stdout/stderr path out of `launchctl print` output.
var loadedPathRe = regexp.MustCompile(`std(?:out|err) path = (\S+)`)

// loadedMismatches compares the fields of a plist FILE against what launchd reports
// as loaded. It deliberately checks only fields launchd prints verbatim and that this
// package authors — comparing everything would produce noise from launchd's own
// derived keys and train the reader to ignore the finding.
func loadedMismatches(plistFile, loaded string) []string {
	var out []string

	// Log paths: the clearest tell, because this package changed them and launchd
	// prints them plainly.
	wantLog := ""
	if m := regexp.MustCompile(`<key>StandardOutPath</key>\s*<string>([^<]+)</string>`).FindStringSubmatch(plistFile); m != nil {
		wantLog = m[1]
	}
	if wantLog != "" {
		found := false
		for _, m := range loadedPathRe.FindAllStringSubmatch(loaded, -1) {
			if m[1] == wantLog {
				found = true
				break
			}
		}
		if !found {
			got := "none"
			if m := loadedPathRe.FindStringSubmatch(loaded); m != nil {
				got = m[1]
			}
			out = append(out, fmt.Sprintf("stdout path is %s, file says %s", got, wantLog))
		}
	}

	// PATH: the V8 surface. A loaded PATH without /usr/sbin means the boot stagger
	// is still inert on that job however good the file is.
	if strings.Contains(plistFile, "/usr/sbin") && !strings.Contains(loaded, "/usr/sbin") {
		out = append(out, "loaded PATH has no /usr/sbin although the file does (boot stagger still inert on the running job)")
	}

	// Schedule: a loaded StartInterval where the file says KeepAlive means the
	// cadence change never took effect.
	fileKeepAlive := strings.Contains(plistFile, "<key>KeepAlive</key>")
	loadedInterval := strings.Contains(loaded, "run interval =") || strings.Contains(loaded, "StartInterval")
	if fileKeepAlive && loadedInterval {
		out = append(out, "file says KeepAlive but the loaded job still has an interval — the cadence change has not taken effect")
	}
	return out
}
