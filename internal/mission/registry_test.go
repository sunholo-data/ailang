package mission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validEntry is the shape every negative test mutates, so a failure is
// attributable to the one field under test rather than to an unrelated gap.
const validEntry = `
name    = "parse"
repo    = "sunholo-data/ailang-parse"
workdir = "/Users/x/dev/sunholo-data/ailang-parse"
doc     = "design_docs/parse-mission.md"

[schedule]
mode             = "keepalive"
throttle_seconds = 10800
boot_offset      = 1680
`

func writeEntry(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestLoadFile_ValidEntryRoundTripsEveryField(t *testing.T) {
	dir := t.TempDir()
	p := writeEntry(t, dir, "parse.toml", validEntry)

	m, err := LoadFile(p)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if m.Name != "parse" {
		t.Errorf("Name = %q, want parse", m.Name)
	}
	if m.Repo != "sunholo-data/ailang-parse" {
		t.Errorf("Repo = %q", m.Repo)
	}
	if m.Workdir != "/Users/x/dev/sunholo-data/ailang-parse" {
		t.Errorf("Workdir = %q", m.Workdir)
	}
	if m.Doc != "design_docs/parse-mission.md" {
		t.Errorf("Doc = %q", m.Doc)
	}
	if m.Sched.Mode != ModeKeepAlive {
		t.Errorf("Mode = %q", m.Sched.Mode)
	}
	if m.Sched.ThrottleSeconds != 10800 {
		t.Errorf("ThrottleSeconds = %d", m.Sched.ThrottleSeconds)
	}
	if m.Sched.BootOffset != 1680 {
		t.Errorf("BootOffset = %d", m.Sched.BootOffset)
	}
	if m.Path != p {
		t.Errorf("Path = %q, want %q", m.Path, p)
	}
}

// THE SCOPE CUT, ENFORCED. Role/model assignment belongs to
// M-MODEL-REGISTRY-SINGLE-SOURCE (ratified D3(a) 2026-08-27). An earlier draft of
// the design gave this registry a [roles] block; if a future edit reintroduces it,
// this test is what stops it becoming a second silent source of model assignment.
func TestLoadFile_RejectsRolesBlock_AndNamesTheOwner(t *testing.T) {
	dir := t.TempDir()
	p := writeEntry(t, dir, "parse.toml", validEntry+`
[roles]
executor = "codex:gpt-5.6-sol"
`)
	_, err := LoadFile(p)
	if err == nil {
		t.Fatal("a [roles] block must be REJECTED — it would be a competing source of model assignment")
	}
	// The message must point the reader at the owning doc; "invalid config" would
	// leave them to rediscover a ratified decision.
	for _, want := range []string{"models.yml", "ailang models role", "D3(a)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("rejection message must name %q so the reader knows where roles DO belong; got: %v", want, err)
		}
	}
}

func TestValidate_RejectsBadFields(t *testing.T) {
	cases := []struct {
		name, body, wantSubstr string
	}{
		{"empty name", strings.Replace(validEntry, `name    = "parse"`, `name = ""`, 1), "name"},
		{"uppercase name", strings.Replace(validEntry, `name    = "parse"`, `name = "Parse"`, 1), "[a-z0-9-]"},
		{"missing repo", strings.Replace(validEntry, `repo    = "sunholo-data/ailang-parse"`, `repo = ""`, 1), "repo is required"},
		{"missing doc", strings.Replace(validEntry, `doc     = "design_docs/parse-mission.md"`, `doc = ""`, 1), "doc is required"},
		{"tilde workdir", strings.Replace(validEntry, `workdir = "/Users/x/dev/sunholo-data/ailang-parse"`, `workdir = "~/dev/x"`, 1), "not ~-relative"},
		{"relative workdir", strings.Replace(validEntry, `workdir = "/Users/x/dev/sunholo-data/ailang-parse"`, `workdir = "dev/x"`, 1), "must be absolute"},
		{"unknown mode", strings.Replace(validEntry, `mode             = "keepalive"`, `mode = "cron"`, 1), "not one of"},
		{"missing mode", strings.Replace(validEntry, `mode             = "keepalive"`, ``, 1), "schedule.mode is required"},
		{"keepalive without throttle", strings.Replace(validEntry, `throttle_seconds = 10800`, ``, 1), "throttle_seconds must be > 0"},
		{"negative boot offset", strings.Replace(validEntry, `boot_offset      = 1680`, `boot_offset = -1`, 1), "boot_offset must be >= 0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := writeEntry(t, t.TempDir(), "m.toml", tc.body)
			_, err := LoadFile(p)
			if err == nil {
				t.Fatalf("expected rejection for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error must mention %q; got: %v", tc.wantSubstr, err)
			}
			// Every validation error names the file it came from.
			if !strings.Contains(err.Error(), p) {
				t.Errorf("error must name the file %q; got: %v", p, err)
			}
		})
	}
}

// The two schedule modes are mutually exclusive in their knobs. Accepting both
// would let a plist render StartInterval AND ThrottleInterval, whose combined
// launchd behaviour nobody has measured.
func TestValidate_ScheduleKnobsAreExclusive(t *testing.T) {
	dir := t.TempDir()
	both := strings.Replace(validEntry, "throttle_seconds = 10800", "throttle_seconds = 10800\ninterval_seconds = 3600", 1)
	if _, err := LoadFile(writeEntry(t, dir, "a.toml", both)); err == nil {
		t.Fatal("keepalive + interval_seconds must be rejected")
	}

	iv := `
name    = "iv"
repo    = "r/x"
workdir = "/tmp/x"
doc     = "d.md"
[schedule]
mode             = "interval"
interval_seconds = 21600
boot_offset      = 5
`
	m, err := LoadFile(writeEntry(t, dir, "b.toml", iv))
	if err != nil {
		t.Fatalf("valid interval entry rejected: %v", err)
	}
	if m.Sched.Mode != ModeInterval || m.Sched.IntervalSeconds != 21600 {
		t.Errorf("interval entry mis-parsed: %+v", m.Sched)
	}
}

// Deterministic and order-independent: the registry is iterated to render
// artifacts and to report drift, so a filesystem-order-dependent result would make
// both non-reproducible.
func TestLoad_IsSortedAndOrderIndependent(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string, off int) string {
		return strings.NewReplacer(
			`name    = "parse"`, `name = "`+name+`"`,
			`boot_offset      = 1680`, `boot_offset = `+itoa(off),
		).Replace(validEntry)
	}
	// Written in deliberately non-alphabetical order.
	writeEntry(t, dir, "zz.toml", mk("world", 420))
	writeEntry(t, dir, "aa.toml", mk("motoko", 1260))
	writeEntry(t, dir, "mm.toml", mk("docs", 840))

	reg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Assert on reg.Missions, NOT reg.Names(): Names() sorts internally, so it
	// reports sorted output even when Load did not sort — a mutation test caught
	// exactly that vacuity here. The slice order is what callers iterate.
	want := []string{"docs", "motoko", "world"}
	if len(reg.Missions) != len(want) {
		t.Fatalf("loaded %d missions, want %d", len(reg.Missions), len(want))
	}
	for i, w := range want {
		if reg.Missions[i].Name != w {
			var got []string
			for _, m := range reg.Missions {
				got = append(got, m.Name)
			}
			t.Fatalf("Missions order = %v, want %v (sorted by name, not by filename)", got, want)
		}
	}
	// Filenames sort to motoko(aa), docs(mm), world(zz) — a DIFFERENT order from
	// the names. Without that divergence the assertion above could not fail.
	if reg.Missions[0].Path == filepath.Join(dir, "aa.toml") {
		t.Fatal("fixture is degenerate: filename order must differ from name order or this test proves nothing")
	}
	if gotN := reg.Names(); len(gotN) != len(want) {
		t.Errorf("Names() = %v", gotN)
	}
	if _, ok := reg.Get("world"); !ok {
		t.Error("Get(world) must find the entry")
	}
	if _, ok := reg.Get("nope"); ok {
		t.Error("Get on an unregistered name must report false")
	}
}

// A colliding offset silently recreates the boot stampede the offsets exist to
// prevent, and nothing downstream would notice.
func TestLoad_RejectsCollidingBootOffsets(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string) string {
		return strings.Replace(validEntry, `name    = "parse"`, `name = "`+name+`"`, 1)
	}
	writeEntry(t, dir, "a.toml", mk("alpha")) // both at boot_offset 1680
	writeEntry(t, dir, "b.toml", mk("beta"))

	_, err := Load(dir)
	if err == nil {
		t.Fatal("colliding boot_offset must be rejected")
	}
	if !strings.Contains(err.Error(), "collides") || !strings.Contains(err.Error(), "stampede") {
		t.Errorf("error should explain the consequence, not just the collision; got: %v", err)
	}
}

func TestLoad_RejectsDuplicateNames(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "a.toml", validEntry)
	writeEntry(t, dir, "b.toml", strings.Replace(validEntry, `boot_offset      = 1680`, `boot_offset = 99`, 1))
	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "duplicate mission name") {
		t.Fatalf("duplicate names must be rejected; got: %v", err)
	}
}

func TestLoad_IgnoresNonTomlAndDirs(t *testing.T) {
	dir := t.TempDir()
	writeEntry(t, dir, "parse.toml", validEntry)
	writeEntry(t, dir, "README.md", "not a mission")
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(reg.Missions) != 1 {
		t.Fatalf("expected 1 mission, got %d", len(reg.Missions))
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
