package mission

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// StagedSuffix is appended to every path this package writes in Phase 1.
//
// install RENDERS; apply PROMOTES. The separation is not cosmetic: the driver
// re-sources ~/.config/ailang/mission-<name>.env on EVERY fire (mission-control.sh
// :63-64 and :73-74, sourced twice), so writing that path in place would apply new
// config on the next interval with nobody reloading anything. Rendering to a staged
// path is what makes "install changes nothing that runs" true rather than merely
// claimed.
const StagedSuffix = ".staged"

// registryOwnedVars are the only env assignments this package authors. Everything
// else in a mission env file — role/model chains, allowlists, PATH, and every
// comment — is PASSTHROUGH, carried through byte-for-byte.
//
// Role and model config is passthrough by ratification (Mark, attended 2026-09-05):
// M-MODEL-REGISTRY-SINGLE-SOURCE owns it, and its M8 (mission driver adoption) is
// PARKED. This package must not become the second source while that milestone sits.
var registryOwnedVars = []string{
	"MISSION_NAME",
	"MISSION_REPO",
	"MISSION_DOC",
	"MISSION_WORKDIR",
}

// passthroughBanner marks the region of a rendered env file this package did not
// author, naming the milestone that will own it.
const passthroughBanner = `# ─────────────────────────────────────────────────────────────────────────────
# BELOW THIS LINE IS PASSTHROUGH — carried verbatim from the previous env file.
# Role/model assignment (MISSION_*_MODEL, MISSION_*_FALLBACK, MISSION_MODEL_PREFS)
# is owned by M-MODEL-REGISTRY-SINGLE-SOURCE, whose M8 is PARKED. The mission
# registry authors schedule and topology only, and copies the rest untouched.
# ─────────────────────────────────────────────────────────────────────────────`

// assignmentKey returns the shell variable a line assigns, or "".
// Handles `K=v`, `export K=v` and `K="${K:-v}"`; ignores comments.
func assignmentKey(line string) string {
	s := strings.TrimSpace(line)
	if s == "" || strings.HasPrefix(s, "#") {
		return ""
	}
	s = strings.TrimPrefix(s, "export ")
	eq := strings.Index(s, "=")
	if eq <= 0 {
		return ""
	}
	k := strings.TrimSpace(s[:eq])
	for _, r := range k {
		if !(r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_') {
			return ""
		}
	}
	return k
}

// renderAssignment writes one registry-owned var, PRESERVING the form the live file
// used. The `${VAR:-value}` idiom is a deliberate rollback ergonomic — it lets an
// operator override a mission from the environment without editing the file — so
// rewriting it to a bare assignment would quietly remove an escape hatch.
func renderAssignment(key, value, prevLine string) string {
	if strings.Contains(prevLine, "${"+key+":-") || overridable[key] {
		return fmt.Sprintf("%s=\"${%s:-%s}\"", key, key, value)
	}
	return fmt.Sprintf("%s=%s", key, value)
}

// overridable names the vars that MUST use the ${VAR:-value} form even when the live
// file had no prior line to copy the idiom from.
//
// MISSION_WORKDIR is the one that bites. pin-root.sh exports it as the PINNED worktree
// and re-execs; the driver derives REPO from it, and then re-sources this env file —
// so a BARE assignment clobbers the pin's export back to the source clone. That exact
// bug was found and fixed by hand in mission-world.env on 2026-08-18 ("Now the pin
// wins"), and a generator emitting a bare assignment would have reintroduced it on
// every mission at once.
var overridable = map[string]bool{"MISSION_WORKDIR": true}

// RenderEnv produces the staged env file for m, given the CURRENT contents of its
// live env file. Registry-owned assignments are replaced with the registry's values;
// every other byte — role config, PATH, comments, blank lines — survives unchanged.
//
// The result is byte-identical to liveEnv whenever the registry agrees with it,
// which is what makes the Phase 2 promotion provably a no-op.
func RenderEnv(m *Mission, liveEnv []byte) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	values := map[string]string{
		"MISSION_NAME":    m.Name,
		"MISSION_REPO":    m.Repo,
		"MISSION_DOC":     m.Doc,
		"MISSION_WORKDIR": m.Workdir,
	}

	// Preserve the file's line ending shape: splitting on \n and rejoining would
	// silently drop a trailing newline and make byte-equality fail for a reason
	// that has nothing to do with content.
	hadTrailingNL := bytes.HasSuffix(liveEnv, []byte("\n"))
	lines := strings.Split(strings.TrimSuffix(string(liveEnv), "\n"), "\n")

	seen := map[string]bool{}
	out := make([]string, 0, len(lines)+8)
	for _, line := range lines {
		k := assignmentKey(line)
		if v, owned := values[k]; owned && !seen[k] {
			seen[k] = true
			out = append(out, renderAssignment(k, v, line))
			continue
		}
		out = append(out, line)
	}

	// Any registry-owned var the live file lacked is appended in a named block, so
	// a new mission (no live file at all) still renders a complete env.
	var missing []string
	for _, k := range registryOwnedVars {
		if !seen[k] {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, "# Authored by the mission registry (missions/"+m.Name+".toml).")
		for _, k := range missing {
			out = append(out, renderAssignment(k, values[k], ""))
		}
		if len(liveEnv) > 0 {
			out = append(out, "", passthroughBanner)
		}
	}

	res := strings.Join(out, "\n")
	if hadTrailingNL || len(liveEnv) == 0 {
		res += "\n"
	}
	return []byte(res), nil
}

// Label is the launchd job label for this mission.
func (m *Mission) Label() string { return "dev.ailang.mission-" + m.launchdSuffix() }

// launchdSuffix preserves the ONE historical irregularity in the fleet: the v1
// mission's job is `dev.ailang.mission-control`, not `dev.ailang.mission-v1`. That
// name predates MISSION_PROFILE and is load-bearing — the installed plist, the
// recovery job's target, the pidfile and the log paths all key off it. Renaming it
// is a migration, not a rendering decision.
func (m *Mission) launchdSuffix() string {
	if m.Name == "v1" {
		return "control"
	}
	return m.Name
}

// DriverPath is the script launchd executes for this mission.
func (m *Mission) DriverPath() string {
	return filepath.Join(m.Workdir, "tools", "launchd", "mission-control.sh")
}

// missionPATH is the PATH every mission plist gets.
//
// /usr/sbin IS LOAD-BEARING and its absence is why this is centralised. The v1 and
// docs plists set a PATH omitting it, so `sysctl` was unreachable, _mc_uptime_secs
// could not read kern.boottime, and the BOOT STAGGER — half the answer to the boot
// stampede behind the 09-04/09-05 OOM events — shipped inert on two of four missions,
// announcing itself only as a log line. Rendering the PATH here means no plist can
// reintroduce that by omission.
const missionPATH = "/Users/voightkampff/go/bin:/Users/voightkampff/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// RenderPlist produces the staged launchd plist for m.
//
// Fully authored, unlike the env file. Two deliberate normalisations, enumerated
// here rather than discovered later:
//
//  1. ProgramArguments is always ["/bin/bash", <driver>]. Two of the four installed
//     plists invoke the script directly and rely on its shebang; an explicit
//     interpreter does not, and cannot break if the shebang is edited.
//  2. Rationale comments are NOT reproduced. The v1 plist carries ~40 lines of
//     ratified measurement (the 09-02 cadence trial) that a generator cannot
//     regenerate. That rationale moves to missions/<name>.toml, which is the
//     reviewable artifact; the plist becomes a build product with a pointer to it.
//     Losing that text silently would be the worst outcome here, so M4 migrates it
//     before any plist is promoted.
func RenderPlist(m *Mission) ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<!--\n")
	b.WriteString("  GENERATED by `ailang mission install " + xmlEscape(m.Name) + "` — DO NOT EDIT.\n")
	b.WriteString("  Source of truth: missions/" + xmlEscape(m.Name) + ".toml (rationale lives there, in TOML comments).\n")
	b.WriteString("  Hand edits are reported by `ailang mission doctor` and lost on the next install.\n")
	b.WriteString("-->\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")

	b.WriteString("\t<key>Label</key>\n\t<string>" + xmlEscape(m.Label()) + "</string>\n")
	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	b.WriteString("\t\t<string>/bin/bash</string>\n")
	b.WriteString("\t\t<string>" + xmlEscape(m.DriverPath()) + "</string>\n")
	b.WriteString("\t</array>\n")

	b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
	b.WriteString("\t\t<key>HOME</key>\n\t\t<string>" + xmlEscape(os.Getenv("HOME")) + "</string>\n")
	b.WriteString("\t\t<key>MISSION_PROFILE</key>\n\t\t<string>" + xmlEscape(m.Name) + "</string>\n")
	b.WriteString("\t\t<key>PATH</key>\n\t\t<string>" + missionPATH + "</string>\n")
	b.WriteString("\t</dict>\n")

	b.WriteString("\t<key>WorkingDirectory</key>\n\t<string>" + xmlEscape(m.Workdir) + "</string>\n")

	// RunAtLoad: fire once at every login/boot so a reboot cannot silently kill the
	// cadence (the 2026-07-20 18h outage). The kill switch gates deliberate offs.
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")

	switch m.Sched.Mode {
	case ModeKeepAlive:
		b.WriteString("\t<key>KeepAlive</key>\n\t<true/>\n")
		b.WriteString(fmt.Sprintf("\t<key>ThrottleInterval</key>\n\t<integer>%d</integer>\n", m.Sched.ThrottleSeconds))
	case ModeInterval:
		b.WriteString(fmt.Sprintf("\t<key>StartInterval</key>\n\t<integer>%d</integer>\n", m.Sched.IntervalSeconds))
	default:
		return nil, fmt.Errorf("%s: unrenderable schedule mode %q", m.Path, m.Sched.Mode)
	}

	// LAUNCHD'S STREAM GETS ITS OWN FILE, and the suffix is load-bearing. The
	// driver writes its own $LOG at /tmp/ailang-mission-<suffix>.log — the file
	// mission-recovery.sh reads and the slot-verdict tooling greps. Pointing
	// launchd's stdout/stderr at the same path would have both appending to it,
	// interleaving launchd's spawn noise into the driver's record. world and
	// motoko already use the .launchd.log convention; this makes it uniform.
	// (Caught by diffing a staged plist against the installed one before applying,
	// which is the entire reason install and apply are separate verbs.)
	log := "/tmp/ailang-mission-" + m.launchdSuffix() + ".launchd.log"
	b.WriteString("\t<key>StandardOutPath</key>\n\t<string>" + xmlEscape(log) + "</string>\n")
	b.WriteString("\t<key>StandardErrorPath</key>\n\t<string>" + xmlEscape(log) + "</string>\n")
	b.WriteString("</dict>\n</plist>\n")
	return []byte(b.String()), nil
}

// writeAtomic writes data to path via a same-directory temp file and a rename.
//
// The inventory found NO named atomic-write helper in the repo — the temp+rename
// pattern is ad-hoc across 11 os.Rename sites. This one is deliberately scoped to
// this package rather than being a repo-wide refactor, which is its own change.
func writeAtomic(path string, data []byte, perm os.FileMode) (err error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file in %s: %w", dir, err)
	}
	tmp := f.Name()
	defer func() {
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = f.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to write %s: %w", tmp, err)
	}
	// fsync before rename: a rename is atomic with respect to readers, but without
	// the sync a crash can leave the new name pointing at unflushed content.
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("failed to sync %s: %w", tmp, err)
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", tmp, err)
	}
	if err = os.Chmod(tmp, perm); err != nil {
		return fmt.Errorf("failed to chmod %s: %w", tmp, err)
	}
	if err = os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}

// Staged is the set of paths a render produced, and where each will be promoted to.
type Staged struct {
	// Source is the file passthrough content was read from — the reviewed copy when
	// one exists, otherwise the installed file. Reported so an operator can see which,
	// rather than having to infer it.
	Source      string
	EnvStaged   string
	EnvTarget   string
	PlistStaged string
	PlistTarget string
}

// RenderStaged renders both artifacts and writes them to STAGED paths only.
//
// It never opens a live target for writing. Promotion is `apply`'s job, and pairing it
// with the launchd reload is the only point at which behaviour changes.
//
// PASSTHROUGH COMES FROM THE REVIEWED COPY when one exists, not from the installed
// file. That inversion is the whole point of this function's `reviewedEnv` argument:
//
//   - Rendering from the INSTALLED file makes the generated output a copy of what
//     already runs, so an edit to the reviewed copy in the repo never reaches the
//     fleet — which is V5 exactly, merely narrowed. That is how the docs planner
//     allowlist sat undeployed behind a green test.
//   - Rendering from the REVIEWED copy makes `tools/launchd/mission-env/<name>.env`
//     the thing you edit, and after an apply installed == reviewed BY CONSTRUCTION,
//     so `env-source-drift` becomes unreportable rather than merely unreported.
//
// Falls back to the installed file when no reviewed copy exists, so a mission that has
// not been brought under review still renders.
func RenderStaged(m *Mission, envTarget, plistTarget string) (*Staged, error) {
	return RenderStagedFrom(m, envTarget, plistTarget, "")
}

// RenderStagedFrom is RenderStaged with an explicit reviewed-copy path.
func RenderStagedFrom(m *Mission, envTarget, plistTarget, reviewedEnv string) (*Staged, error) {
	source := envTarget
	if reviewedEnv != "" {
		if _, statErr := os.Stat(reviewedEnv); statErr == nil {
			source = reviewedEnv
		}
	}
	live, err := os.ReadFile(source) //nolint:gosec // caller-supplied mission path
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read env source %s: %w", source, err)
	}
	envBytes, err := RenderEnv(m, live)
	if err != nil {
		return nil, err
	}
	plistBytes, err := RenderPlist(m)
	if err != nil {
		return nil, err
	}
	s := &Staged{
		Source:      source,
		EnvStaged:   envTarget + StagedSuffix,
		EnvTarget:   envTarget,
		PlistStaged: plistTarget + StagedSuffix,
		PlistTarget: plistTarget,
	}
	if err := writeAtomic(s.EnvStaged, envBytes, 0o600); err != nil {
		return nil, err
	}
	if err := writeAtomic(s.PlistStaged, plistBytes, 0o644); err != nil { //nolint:gosec // launchd must read it
		return nil, err
	}
	return s, nil
}
