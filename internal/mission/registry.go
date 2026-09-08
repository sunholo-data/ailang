// Package mission owns the declarative registry that describes WHICH mission loops
// exist, where they run and on what schedule.
//
// It deliberately does NOT own which MODEL runs a role. That is
// M-MODEL-REGISTRY-SINGLE-SOURCE, ratified 2026-08-27 (D3(a)), whose read path is
// `ailang models role`. An earlier draft of the design gave this registry a [roles]
// block; it was cut because a second source of model assignment is precisely the
// failure this package exists to remove. Load rejects such a block on sight so the
// cut cannot regress silently.
package mission

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

// ScheduleMode is how launchd is asked to run the mission.
type ScheduleMode string

const (
	// ModeKeepAlive renders KeepAlive + ThrottleInterval. ThrottleInterval is
	// measured from the job's START, so the period is max(duration, throttle).
	ModeKeepAlive ScheduleMode = "keepalive"
	// ModeInterval renders StartInterval, which re-arms from the job's EXIT —
	// measured 2026-09-05 across four gaps on two missions (5403/5372/5379s
	// against 5400s; 14390s against 14400s). The period is duration + interval,
	// i.e. a full interval of idle every cycle. Kept because three missions
	// deliberately want spacing, not back-to-back.
	ModeInterval ScheduleMode = "interval"
)

// Schedule is the [schedule] table.
type Schedule struct {
	Mode ScheduleMode `toml:"mode"`
	// ThrottleSeconds applies to keepalive; IntervalSeconds to interval.
	// Exactly one must be set, matching Mode.
	ThrottleSeconds int `toml:"throttle_seconds"`
	IntervalSeconds int `toml:"interval_seconds"`
	// BootOffset staggers this mission out of a boot stampede. Every mission
	// plist carries RunAtLoad, so a boot fires all of them within seconds; on
	// 2026-09-05 that put 33 claude processes on the box inside ten minutes.
	// Today this lives as a case arm inside the driver; the registry is its home.
	BootOffset int `toml:"boot_offset"`
}

// Mission is one registry entry: one TOML file in missions/ (HD-1(a), ratified
// 2026-09-05).
type Mission struct {
	Name    string   `toml:"name"`
	Repo    string   `toml:"repo"`
	Workdir string   `toml:"workdir"`
	Doc     string   `toml:"doc"`
	Sched   Schedule `toml:"schedule"`

	// Roles is a REJECTION TRAP, not a field. It is never populated: Validate
	// fails if the TOML carries a [roles] table, so a future edit that
	// reintroduces model assignment here fails loudly at load instead of
	// quietly becoming a competing source of truth.
	Roles map[string]any `toml:"roles"`

	// Driver optionally overrides where this mission's driver code comes from.
	//
	// EMPTY MEANS THE SHARED DRIVER, and that is the point: one centralized place to
	// change every loop. Set it only for a mission that must run its own copy, which is
	// a fork and will be reported as one.
	//
	// This exists because "where the driver LIVES" and "where the mission WORKS" were the
	// same thing, and that conflation is what made world unfixable: its work repo is
	// sunholo-data/ailang-world, so its driver had to live there too, so every shared fix
	// had to be hand-ported and silently was not. Separating them means a mission in
	// another repo entirely is just a different `workdir` — no second driver.
	Driver string `toml:"driver"`

	// Path is the file this entry was loaded from; used in error messages so a
	// validation failure names the file, never just "invalid config".
	Path string `toml:"-"`

	// root is the registry directory's repo, i.e. where the shared driver lives.
	root string `toml:"-"` //nolint:unused // set by Load, read by DriverPath
}

// nameOK reports whether s is a usable mission name. Mission names become
// filenames, launchd labels and state-file keys, so the character set is
// deliberately narrow.
func nameOK(s string) bool {
	if s == "" || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// Validate checks one entry in isolation. Cross-entry checks (boot-offset
// collisions) live in Registry.Validate, because they cannot be decided from a
// single file.
func (m *Mission) Validate() error {
	where := m.Path
	if where == "" {
		where = "<unknown file>"
	}

	// The scope cut, enforced. Named owner in the message: a reader who hits
	// this needs to know where role config DOES belong, not merely that it is
	// unwelcome here.
	if len(m.Roles) > 0 {
		return fmt.Errorf("%s: [roles] is not a mission-registry field — model and role "+
			"assignment belongs to M-MODEL-REGISTRY-SINGLE-SOURCE (models.yml, read via "+
			"`ailang models role`), ratified 2026-08-27 D3(a). This registry owns which "+
			"missions exist, not what they run", where)
	}

	if !nameOK(m.Name) {
		return fmt.Errorf("%s: name %q must be 1-32 chars of [a-z0-9-]", where, m.Name)
	}
	if m.Repo == "" {
		return fmt.Errorf("%s: repo is required (e.g. sunholo-data/ailang)", where)
	}
	if m.Doc == "" {
		return fmt.Errorf("%s: doc is required (the mission charter path, repo-relative)", where)
	}
	if m.Workdir == "" {
		return fmt.Errorf("%s: workdir is required", where)
	}
	if strings.HasPrefix(m.Workdir, "~") {
		return fmt.Errorf("%s: workdir %q must be an absolute path, not ~-relative — "+
			"the driver runs under launchd, where ~ is not expanded", where, m.Workdir)
	}
	// "Absolute" means POSIX-absolute OR host-absolute, and it needs both arms.
	// filepath.IsAbs alone is wrong: on Windows it rejects "/Users/..." — every real
	// rig workdir — so the whole package went red on test-windows. A POSIX-only check
	// alone is wrong too: the fleet fixtures build their workdirs from t.TempDir(),
	// which is C:\Users\... on a Windows runner. On unix the two arms are the SAME
	// function (filepath.IsAbs is strings.HasPrefix(p, "/") there), so this is
	// byte-identical to the old behaviour on the rig. What it still rejects is what
	// the check is actually for: a relative path, and the ~-form handled above.
	if !strings.HasPrefix(m.Workdir, "/") && !filepath.IsAbs(m.Workdir) {
		return fmt.Errorf("%s: workdir %q must be absolute", where, m.Workdir)
	}

	switch m.Sched.Mode {
	case ModeKeepAlive:
		if m.Sched.ThrottleSeconds <= 0 {
			return fmt.Errorf("%s: schedule.throttle_seconds must be > 0 for mode %q", where, ModeKeepAlive)
		}
		if m.Sched.IntervalSeconds != 0 {
			return fmt.Errorf("%s: schedule.interval_seconds is meaningless for mode %q — "+
				"set throttle_seconds only", where, ModeKeepAlive)
		}
	case ModeInterval:
		if m.Sched.IntervalSeconds <= 0 {
			return fmt.Errorf("%s: schedule.interval_seconds must be > 0 for mode %q", where, ModeInterval)
		}
		if m.Sched.ThrottleSeconds != 0 {
			return fmt.Errorf("%s: schedule.throttle_seconds is meaningless for mode %q — "+
				"set interval_seconds only", where, ModeInterval)
		}
	case "":
		return fmt.Errorf("%s: schedule.mode is required (%q or %q)", where, ModeKeepAlive, ModeInterval)
	default:
		return fmt.Errorf("%s: schedule.mode %q is not one of %q, %q", where, m.Sched.Mode, ModeKeepAlive, ModeInterval)
	}

	if m.Sched.BootOffset < 0 {
		return fmt.Errorf("%s: schedule.boot_offset must be >= 0", where)
	}
	if m.Driver != "" && !filepath.IsAbs(m.Driver) {
		return fmt.Errorf("%s: driver %q must be an absolute path when set (leave it EMPTY to use "+
			"the shared driver, which is what you almost certainly want)", where, m.Driver)
	}
	return nil
}

// Registry is every mission entry, ordered by name so iteration is deterministic
// regardless of directory read order.
type Registry struct {
	Missions []*Mission
}

// Get returns the entry with the given name.
func (r *Registry) Get(name string) (*Mission, bool) {
	for _, m := range r.Missions {
		if m.Name == name {
			return m, true
		}
	}
	return nil, false
}

// Names returns the registered mission names, sorted.
func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.Missions))
	for _, m := range r.Missions {
		out = append(out, m.Name)
	}
	sort.Strings(out)
	return out
}

// Validate runs the cross-entry checks that a single file cannot decide.
func (r *Registry) Validate() error {
	seenName := map[string]string{}
	seenOffset := map[int]string{}
	for _, m := range r.Missions {
		if prev, dup := seenName[m.Name]; dup {
			return fmt.Errorf("duplicate mission name %q in %s and %s", m.Name, prev, m.Path)
		}
		seenName[m.Name] = m.Path

		// A colliding boot offset silently recreates the stampede the offsets
		// exist to prevent, and nothing downstream would report it. Offset 0 is
		// exempt: it means "go first", and exactly one mission (v1) holds it,
		// but two missions at 0 is the collision, so it is checked like any other.
		if prev, dup := seenOffset[m.Sched.BootOffset]; dup {
			return fmt.Errorf("%s: schedule.boot_offset %d collides with %s — "+
				"colliding offsets recreate the boot stampede the offsets exist to prevent",
				m.Path, m.Sched.BootOffset, prev)
		}
		seenOffset[m.Sched.BootOffset] = m.Path
	}
	return nil
}

// LoadFile reads and validates one registry entry.
func LoadFile(path string) (*Mission, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from the registry dir listing
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var m Mission
	if _, err := toml.Decode(string(data), &m); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	m.Path = path
	// The shared driver lives in the repo that holds the registry: missions/ is beside
	// tools/launchd/. Self-locating, so no machine-specific path is ever configured.
	if abs, aerr := filepath.Abs(filepath.Join(filepath.Dir(path), "..")); aerr == nil {
		m.root = abs
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Load reads every *.toml in dir. The result is sorted by name, so callers get a
// stable order no matter what the filesystem hands back.
func Load(dir string) (*Registry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read mission registry %s: %w", dir, err)
	}
	reg := &Registry{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		m, err := LoadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		reg.Missions = append(reg.Missions, m)
	}
	sort.Slice(reg.Missions, func(i, j int) bool { return reg.Missions[i].Name < reg.Missions[j].Name })
	if err := reg.Validate(); err != nil {
		return nil, err
	}
	return reg, nil
}
