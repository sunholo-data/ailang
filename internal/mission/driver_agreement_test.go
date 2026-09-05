package mission

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The driver is bash and cannot read TOML, and making it shell out to `ailang` on
// every fire would add a dependency and a failure mode to the hot path. So the
// boot-offset case arm STAYS in the driver — but the registry is authoritative, and
// these tests make silent divergence impossible. That is the same guarantee the
// deleted truth-table failed to provide, achieved by checking instead of by trusting.

func driverPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "tools", "launchd", "mission-control.sh")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("driver not present: %v", err)
	}
	return p
}

func registryDir(t *testing.T) string {
	t.Helper()
	p := filepath.Join("..", "..", "missions")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("registry not present: %v", err)
	}
	return p
}

// The offsets in _mc_boot_offset must equal the registry's, mission for mission. A
// colliding or stale offset silently recreates the boot stampede the offsets exist to
// prevent — 33 claude processes inside ten minutes on 2026-09-05 — and nothing
// downstream reports it.
func TestDriverBootOffsetsMatchTheRegistry(t *testing.T) {
	body, err := os.ReadFile(driverPath(t))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := Load(registryDir(t))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Scope the scan to the _mc_boot_offset function, or a `case` arm elsewhere in a
	// 1500-line driver could satisfy this test by accident.
	src := string(body)
	i := strings.Index(src, "_mc_boot_offset() {")
	if i < 0 {
		t.Fatal("_mc_boot_offset not found — if it was removed, this test must be replaced, not deleted")
	}
	fn := src[i:]
	if j := strings.Index(fn, "\n}"); j > 0 {
		fn = fn[:j]
	}

	arm := regexp.MustCompile(`(?m)^\s*([a-z0-9-]+)\)\s*echo\s+(\d+)`)
	got := map[string]int{}
	for _, m := range arm.FindAllStringSubmatch(fn, -1) {
		n, _ := strconv.Atoi(m[2])
		got[m[1]] = n
	}
	if len(got) == 0 {
		t.Fatal("parsed no offsets from _mc_boot_offset — the instrument is broken, not the driver")
	}

	for _, m := range reg.Missions {
		g, ok := got[m.Name]
		if !ok {
			t.Errorf("%s is registered but has no _mc_boot_offset arm — an unknown mission "+
				"silently gets 0 and joins the boot stampede", m.Name)
			continue
		}
		if g != m.Sched.BootOffset {
			t.Errorf("%s: driver says boot_offset %d, registry says %d — the registry is "+
				"authoritative; update the driver's case arm", m.Name, g, m.Sched.BootOffset)
		}
	}
	for name := range got {
		if _, ok := reg.Get(name); !ok {
			t.Errorf("the driver has an offset arm for %q, which is not a registered mission", name)
		}
	}
}

// The hand-maintained reach truth-table is deleted. This is what stops it coming back:
// it went stale unseen once, and a comment cannot assert its own freshness.
func TestDriverDoesNotReadoptTheHandMaintainedReachTable(t *testing.T) {
	body, err := os.ReadFile(driverPath(t))
	if err != nil {
		t.Fatal(err)
	}
	src := string(body)
	// The old table's shape: per-mission rows pairing a checkout with a reach verdict.
	// Signatures unique to the TABLE's shape (a checkout column pointing at a reach
	// verdict), not to any word it happened to contain. A first draft flagged
	// "-> astra" and matched the designer ROTATION comment, which is unrelated — a
	// guard that fires on innocent text gets deleted by the next reader.
	for _, sig := range []string{
		"astra ✅",
		"re-execs from ~/.ailang-driver-pin/",
		"ailang-motoko/   ->",
		"ailang-world/    ->",
		"deliberate per-mission pin, mission-docs.env",
	} {
		if strings.Contains(src, sig) {
			t.Errorf("the hand-maintained reach table is back (%q). It is COMPUTED now — "+
				"`ailang mission list` — because the written one went stale and that is how "+
				"the world fork stayed invisible", sig)
		}
	}
	// And the pointer to the computed answer must survive.
	if !strings.Contains(src, "ailang mission list") {
		t.Error("the driver should point a reader at `ailang mission list` where the table used to be")
	}
}
