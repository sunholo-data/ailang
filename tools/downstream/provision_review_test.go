package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Review of #2220: the candidate's own `ailang lock`, entries the loader never
// type-checked, and the provisioning paths the first suite left unpinned.

func depsFixtureWith(t *testing.T, extra map[string]string) *fixture {
	f := newFixture(t, depsCorpus, "")
	files := depsFiles()
	for k, v := range extra {
		files[k] = v
	}
	f.remote.repos["owner/deps"] = files
	mkPkg(t, filepath.Join(f.dir, "pkgroot"), "acme/cached", "1.0.0")
	return f
}

func bucketCodes(rep *Report) map[string]int {
	out := map[string]int{}
	for _, b := range rep.Buckets {
		out[b.Code] += b.Count
	}
	return out
}

// Must-fix 1: where the gate generates a lock, the candidate's own `ailang lock`
// is run and compared. A failure, or a different resolved set, is a break; the
// arms themselves both check against the baseline's lock.
func TestCandidateLockFailureAndDriftAreBreaks(t *testing.T) {
	f := depsFixtureWith(t, map[string]string{
		"lf/ailang.toml": "[package]\nname = \"acme/lf\"\n# CANDLOCKFAIL\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n",
		"lf/main.ail":    "module fixture/lf -- NEEDDEPS\n",
		"ld/ailang.toml": "[package]\nname = \"acme/ld\"\n# CANDLOCKDIFF\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n",
		"ld/main.ail":    "module fixture/ld -- NEEDDEPS\n",
	})
	o := f.options(f.base, f.cand)
	code, rep := f.run(o)
	if code != exitBreak || len(rep.Failures) > 0 {
		t.Fatalf("exit %d, want 2 (a candidate lock break); failures %v", code, rep.Failures)
	}
	rows := rowStates(t, o.Out)
	for rel, want := range map[string]string{
		"lf/ailang.lock":   stateNewlyFailing,
		"ld/ailang.lock":   stateNewlyFailing,
		"app/ailang.lock":  statePassBoth,
		"gone/ailang.lock": stateNotMeasured, // both arms' locks fail: not a new break
		"lf/main.ail":      statePassBoth,    // checked against the baseline's lock
		"ld/main.ail":      statePassBoth,
	} {
		if got := rows[rel].State; got != want {
			t.Errorf("%s: %q, want %q", rel, got, want)
		}
	}
	bc := bucketCodes(rep)
	if bc[codeLockFailed] != 1 || bc[codeLockDiffers] != 1 || len(bc) != 2 {
		t.Errorf("buckets %v, want one %s and one %s", bc, codeLockFailed, codeLockDiffers)
	}
	cp := corpusProvision(t, rep, "deps")
	for _, l := range cp.Locks {
		switch l.Dir {
		case "ld":
			if !strings.HasPrefix(l.Candidate, "DIFFERS") || !strings.Contains(l.Candidate, "acme/extra@9.0.0") || !strings.Contains(l.Candidate, "baseline's lock") {
				t.Errorf("ld candidate lock %q, want DIFFERS naming acme/extra and the tree both arms used", l.Candidate)
			}
		case "lf":
			if !strings.HasPrefix(l.Candidate, "FAILED") {
				t.Errorf("lf candidate lock %q, want FAILED", l.Candidate)
			}
		case "app":
			if l.Candidate != "same" {
				t.Errorf("app candidate lock %q, want same", l.Candidate)
			}
		}
	}
	if s := corpusSummary(rep, "deps"); s.Locks != 5 {
		t.Errorf("%d lock entries, want 5 (app, gone, stub, lf, ld)", s.Locks)
	}
}

func corpusSummary(rep *Report, name string) CorpusSummary {
	for _, c := range rep.Corpora {
		if c.Name == name {
			return c
		}
	}
	return CorpusSummary{}
}

// Must-fix 2: the loader rejects these before type checking, so they are
// NOT_MEASURED with their own reason — and a dependency's bad manifest is a
// dependency problem, not the entry's.
func TestUncheckedEntriesAreNotMeasured(t *testing.T) {
	f := depsFixtureWith(t, map[string]string{
		"own/ailang.toml": "[package]\nname = \"acme/own\"\n",
		"own/main.ail":    "module fixture/own -- OWNBADMANIFEST\n",
		"dep/ailang.toml": "[package]\nname = \"acme/dep\"\n",
		"dep/main.ail":    "module fixture/dep -- DEPBADMANIFEST\n",
		"exp/ailang.toml": "[package]\nname = \"acme/exp\"\n",
		"exp/main.ail":    "module fixture/exp -- UNEXPORTED\n",
	})
	o := f.options(f.base, f.cand)
	code, rep := f.run(o)
	if code != exitIncomplete {
		t.Fatalf("exit %d, want 4; failures %v", code, rep.Failures)
	}
	rows := rowStates(t, o.Out)
	for rel, want := range map[string]string{
		"own/main.ail": reasonManifest,
		"dep/main.ail": reasonDeps,
		"exp/main.ail": reasonExport,
	} {
		if r := rows[rel]; r.State != stateNotMeasured || r.NotMeasured != want {
			t.Errorf("%s: %s/%q, want NOT_MEASURED/%q (codes %v)", rel, r.State, r.NotMeasured, want, r.Base.Codes)
		}
	}
	if s := corpusSummary(rep, "deps"); s.NotMeasured[reasonManifest] != 1 || s.NotMeasured[reasonExport] != 1 {
		t.Errorf("not-measured reasons %v", s.NotMeasured)
	}
}

// A path dependency that leaves the snapshot is not followed for fetching.
func TestProvisionDoesNotFollowPathsOutOfTheSnapshot(t *testing.T) {
	f := depsFixtureWith(t, map[string]string{
		"escape/ailang.toml": "[package]\nname = \"acme/escape\"\n\n[dependencies]\n\"acme/evil\" = { path = \"../../../../evil\" }\n",
		"escape/main.ail":    "module fixture/escape\n",
	})
	// <dir>/out/corpora/deps/escape/../../../../evil = <dir>/evil, outside the snapshot.
	if err := os.MkdirAll(filepath.Join(f.dir, "evil"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, "evil", "ailang.toml"), []byte("[package]\nname = \"acme/evil\"\n\n[dependencies]\n\"acme/outside\" = \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	registry := filepath.Join(f.dir, "registry")
	mkPkg(t, registry, "acme/old", "0.1.0")
	mkPkg(t, registry, "acme/outside", "1.0.0")
	t.Setenv("DOWNSTREAM_FAKE_REGISTRY", registry)
	o := f.options(f.base, f.cand)
	o.FetchDeps = true
	_, rep := f.run(o)
	for _, p := range rep.Provisioning.Fetched {
		if p.Name == "acme/outside" {
			t.Error("followed a path dependency out of the snapshot and fetched what it names")
		}
	}
}

// A write into the provisioned tree during the arms is an instrument failure.
func TestProvisionedTreeWriteIsAudited(t *testing.T) {
	f := depsFixtureWith(t, map[string]string{"app/write.ail": "module fixture/w -- WRITE_CWD\n"})
	o := f.options(f.base, f.cand)
	code, rep := f.run(o)
	found := false
	for _, a := range rep.Audit {
		found = found || (a.Scope == "provisioned tree deps" && strings.Contains(a.Path, "leaked.txt"))
	}
	if code != exitInstrument || !found {
		t.Fatalf("exit %d, audit %v; want 3 with the provisioned tree's leaked.txt", code, rep.Audit)
	}
}

// A hung `ailang lock` times out under the per-check deadline.
func TestProvisionLockHasADeadline(t *testing.T) {
	f := depsFixtureWith(t, map[string]string{
		"hang/ailang.toml": "[package]\nname = \"acme/hang\"\n# HANGLOCK\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n",
		"hang/main.ail":    "module fixture/hang -- NEEDDEPS\n",
	})
	o := f.options(f.base, f.cand)
	o.Timeout = 1
	t0 := time.Now()
	_, rep := f.run(o)
	if el := time.Since(t0); el > 20*time.Second {
		t.Errorf("run took %s: the hung lock was not killed", el)
	}
	cp := corpusProvision(t, rep, "deps")
	if got := lockOf(cp, "hang"); got != "FAILED" {
		t.Errorf("hang lock %q, want FAILED", got)
	}
	if !strings.Contains(strings.Join(cp.Unprovisioned, "\n"), "timed out") {
		t.Errorf("unprovisioned does not say the lock timed out: %v", cp.Unprovisioned)
	}
}

// A registry 5xx is not "no such version": exit 3, not NOT_MEASURED.
func TestRegistry5xxIsAnInstrumentFailure(t *testing.T) {
	f := newDepsFixture(t)
	registry := filepath.Join(f.dir, "registry")
	mkPkg(t, registry, "acme/old", "0.1.0")
	if err := os.WriteFile(filepath.Join(registry, "acme", "old", "0.1.0", "FAIL503"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOWNSTREAM_FAKE_REGISTRY", registry)
	o := f.options(f.base, f.cand)
	o.FetchDeps = true
	code, rep := f.run(o)
	if code != exitInstrument || !strings.Contains(strings.Join(rep.Failures, "\n"), "HTTP 503") {
		t.Fatalf("exit %d, want 3 naming HTTP 503; failures %v", code, rep.Failures)
	}
}

// Fetch mode follows the registry dependencies of what it fetched.
func TestProvisionFetchesDependenciesOfFetchedPackages(t *testing.T) {
	f := newDepsFixture(t)
	registry := filepath.Join(f.dir, "registry")
	mkPkg(t, registry, "acme/old", "0.1.0")
	mkPkg(t, registry, "acme/leaf", "2.0.0")
	if err := os.WriteFile(filepath.Join(registry, "acme", "old", "0.1.0", "ailang.toml"),
		[]byte("[package]\nname = \"acme/old\"\nversion = \"0.1.0\"\n\n[dependencies]\n\"acme/leaf\" = \"2.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOWNSTREAM_FAKE_REGISTRY", registry)
	o := f.options(f.base, f.cand)
	o.FetchDeps = true
	_, rep := f.run(o)
	got := false
	for _, p := range rep.Provisioning.Fetched {
		got = got || (p.Name == "acme/leaf" && p.Version == "2.0.0")
	}
	if !got {
		t.Errorf("acme/leaf@2.0.0 (a dependency of the fetched acme/old) was not fetched: %v", rep.Provisioning.Fetched)
	}
}

// `ailang lock` never gets the network; an exhausted budget is exit 3; a
// committed lock the gate cannot read is a finding.
func TestProvisionLockIsOfflineBudgetedAndLoud(t *testing.T) {
	f := newDepsFixture(t)
	o := f.options(f.base, f.cand)
	_, rep := f.run(o)
	env, err := os.ReadFile(filepath.Join(o.Out, "arms", "provision", "state", "lock-env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kv := range []string{"HTTPS_PROXY=" + deadProxy, "HTTP_PROXY=" + deadProxy, "https_proxy=" + deadProxy} {
		if !strings.Contains("\n"+string(env), "\n"+kv+"\n") {
			t.Errorf("`ailang lock` ran without %s", kv)
		}
	}
	if !strings.Contains(strings.Join(rep.Provisioning.Findings, "\n"), "stub/ailang.lock cannot be read") {
		t.Errorf("no finding for the unreadable committed lock: %v", rep.Provisioning.Findings)
	}
	md, _ := os.ReadFile(filepath.Join(o.Out, "report.md"))
	if !strings.Contains(string(md), "FINDING for the corpus owner") {
		t.Error("report.md does not lead with the finding")
	}

	f2 := newDepsFixture(t)
	o2 := f2.options(f2.base, f2.cand)
	o2.provisionBudget = time.Nanosecond
	code, rep2 := f2.run(o2)
	if code != exitInstrument || !strings.Contains(strings.Join(rep2.Failures, "\n"), "budget") {
		t.Errorf("exhausted budget: exit %d, failures %v; want 3 naming the budget", code, rep2.Failures)
	}

	// The budget also ends a command that is still running: a hung lock under a
	// long per-check deadline is cut at the budget, and that is exit 3 too.
	f3 := depsFixtureWith(t, map[string]string{
		"hang/ailang.toml": "[package]\nname = \"acme/hang\"\n# HANGLOCK\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n",
	})
	o3 := f3.options(f3.base, f3.cand)
	o3.Timeout, o3.provisionBudget = 600, 2*time.Second
	t0 := time.Now()
	code, rep3 := f3.run(o3)
	if code != exitInstrument || !strings.Contains(strings.Join(rep3.Failures, "\n"), "budget") || time.Since(t0) > 20*time.Second {
		t.Errorf("budget ran out mid-lock: exit %d after %s, failures %v; want 3 naming the budget, quickly", code, time.Since(t0), rep3.Failures)
	}
}
