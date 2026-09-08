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

// knownDriverCopies is a RATCHET, not a target.
//
// There is now exactly ONE copy of mission-control.sh in the fleet. World's fork — 1,047
// lines that silently missed every routing fix for weeks — was deleted on 2026-09-06 once
// the driver's location was decoupled from the mission's workdir.
//
// The count may FALL, never rise. A second copy is how this became a problem in the first
// place, and the fix was never "keep them in sync"; it was "stop having two".
// LOWERED 2 -> 1 on 2026-09-06, when world was de-forked and the ratchet itself said so:
// "driver copies FELL to 1 — good, but lower knownDriverCopies to 1 so the ratchet keeps
// holding at the new level." There is now exactly ONE driver in the fleet, and any second
// one is a regression, not a starting point.
const knownDriverCopies = 1

func TestDriverCopiesDoNotMultiply(t *testing.T) {
	roots := []string{
		filepath.Join("..", "..", "tools", "launchd", "mission-control.sh"),
		filepath.Join("..", "..", "..", "ailang-world", "tools", "launchd", "mission-control.sh"),
		filepath.Join("..", "..", "..", "ailang-docs", "tools", "launchd", "mission-control.sh"),
		filepath.Join("..", "..", "..", "ailang-motoko", "tools", "launchd", "mission-control.sh"),
	}
	// A clone that merely MIRRORS the shared repo is not a separate copy — motoko and
	// docs are clones of sunholo-data/ailang and re-exec from the pin, so their file
	// is the same code by construction. Only a driver in a DIFFERENT repo counts.
	shared, err := os.ReadFile(roots[0])
	if err != nil {
		t.Skipf("shared driver unreadable: %v", err)
	}
	distinct := 1 // the shared one
	var forks []string
	observed := 0
	for _, r := range roots[1:] {
		body, rerr := os.ReadFile(r)
		if rerr != nil {
			continue // that clone is not on this machine
		}
		observed++
		if string(body) == string(shared) {
			continue // an exact mirror, kept in step by the pin
		}
		// A clone of the SAME repo may legitimately lag; only count one that cannot
		// be reached by the pin at all, i.e. has no pin-root helper beside it.
		if _, herr := os.Stat(filepath.Join(filepath.Dir(r), "lib", "pin-root.sh")); herr == nil {
			continue
		}
		distinct++
		forks = append(forks, r)
	}
	// The probe reads SIBLING directories of this checkout, so what it can see is a
	// property of where the test runs, not of the fleet. From the main checkout the
	// siblings are there; from any worktree — and from every CI runner, which clones
	// one repo — none of them is, and `distinct` is 1 by construction. Reporting that
	// as "a fork was removed" would be a fact about the filesystem wearing a finding's
	// clothes, and lowering the ratchet to match would silently retire the invariant
	// while world's fork is still on disk (measured: it is).
	if observed == 0 {
		t.Skipf("no sibling mission clone beside %s — the fleet is not observable from "+
			"this checkout, so the ratchet cannot be evaluated here", filepath.Dir(roots[0]))
	}
	if distinct > knownDriverCopies {
		t.Errorf("driver copies grew to %d (%v). A new fork is how world became invisible to "+
			"every routing fix; add it to the de-fork plan rather than raising this constant.",
			distinct, forks)
	}
	if distinct < knownDriverCopies {
		// A fall is only news if the fleet was VISIBLE. On a fresh clone — every CI
		// runner — none of the sibling checkouts exist, so distinct is 1 by
		// construction and this branch fires for an environment, not for a change.
		// The up-ratchet above is safe either way: it can only fire on a fork this
		// checkout can actually see.
		if observed == 0 {
			t.Skip("no sibling checkout present — the fleet is not observable here, " +
				"so a fall cannot be distinguished from a fresh clone (expected off-rig)")
		}
		t.Errorf("driver copies FELL to %d — good, but lower knownDriverCopies to %d so the "+
			"ratchet keeps holding at the new level", distinct, distinct)
	}
}
