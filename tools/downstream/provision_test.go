package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Dependency provisioning (provision.go): one fixture corpus whose package dirs
// need each outcome — a path dependency (resolved from the snapshot), a cached
// registry dependency (from the package root copy), a missing one (stays
// NOT_MEASURED, named), a committed lock pinning a version the root lacks
// (fetched only under --fetch-deps), and a committed lock that does not parse
// (regenerated in the provisioned copy).

const depsCorpus = `
- name: deps
  repo: owner/deps
  modes: [files]
  tier: {ci: advisory, release: blocking}
`

func depsFiles() map[string]string {
	return map[string]string{
		"lib/ailang.toml":    "[package]\nname = \"acme/lib\"\nversion = \"0.1.0\"\n",
		"lib/util.ail":       "module acme/lib/util\nexport func u() -> int = 1\n",
		"app/ailang.toml":    "[package]\nname = \"acme/app\"\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n\"acme/cached\" = \"1.0.0\"\n",
		"app/main.ail":       "module fixture/app -- NEEDDEPS\n",
		"gone/ailang.toml":   "[package]\nname = \"acme/gone\"\n\n[dependencies]\n\"acme/missing\" = \"9.9.9\"\n",
		"gone/main.ail":      "module fixture/gone -- NEEDDEPS\n",
		"pinned/ailang.toml": "[package]\nname = \"acme/pinned\"\n\n[dependencies]\n\"acme/old\" = \"0.1.0\"\n",
		"pinned/ailang.lock": `{"schema":"ailang.lock/v1","packages":[{"name":"acme/old","version":"0.1.0","source":"registry","content_hash":"sha256:old"}]}` + "\n",
		"pinned/main.ail":    "module fixture/pinned -- NEEDDEPS\n",
		"stub/ailang.toml":   "[package]\nname = \"acme/stub\"\n\n[dependencies]\n\"acme/lib\" = { path = \"../lib\" }\n",
		"stub/ailang.lock":   `{"_comment": "names only", "packages": [{"name": "acme/lib"}]}` + "\n",
		"stub/main.ail":      "module fixture/stub -- NEEDDEPS\n",
	}
}

func newDepsFixture(t *testing.T) *fixture {
	f := newFixture(t, depsCorpus, "")
	f.remote.repos["owner/deps"] = depsFiles()
	mkPkg(t, filepath.Join(f.dir, "pkgroot"), "acme/cached", "1.0.0")
	return f
}

func mkPkg(t *testing.T, root, name, version string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(name), version)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte("[package]\nname = \""+name+"\"\nversion = \""+version+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rowStates reads every row (rows.jsonl, pass_both included) keyed by rel.
func rowStates(t *testing.T, out string) map[string]Row {
	t.Helper()
	fh, err := os.Open(filepath.Join(out, "rows.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer fh.Close()
	rows := map[string]Row{}
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var r Row
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		rows[r.Rel] = r
	}
	return rows
}

func corpusProvision(t *testing.T, rep *Report, name string) CorpusProvision {
	t.Helper()
	if rep.Provisioning == nil {
		t.Fatal("report has no provisioning section")
	}
	for _, c := range rep.Provisioning.Corpora {
		if c.Corpus == name {
			return c
		}
	}
	t.Fatalf("no provisioning record for %s", name)
	return CorpusProvision{}
}

func lockOf(cp CorpusProvision, dir string) string {
	for _, l := range cp.Locks {
		if l.Dir == dir {
			return l.Lock
		}
	}
	return ""
}

func pkgSource(cp CorpusProvision, name string) string {
	for _, p := range cp.Packages {
		if p.Name == name {
			return p.Source
		}
	}
	return ""
}

// Offline: the path and cached deps are provisioned and measured; the missing
// and the lock-pinned-but-absent ones stay NOT_MEASURED, named, and the gate
// exits 4. Nothing is written into the snapshot or the operator's package root.
func TestProvisionOfflineThreeOutcomes(t *testing.T) {
	f := newDepsFixture(t)
	o := f.options(f.base, f.cand)
	code, rep := f.run(o)
	if code != exitIncomplete || len(rep.Failures) > 0 {
		t.Fatalf("exit %d, want 4 (incomplete) with no instrument failure; failures %v", code, rep.Failures)
	}
	rows := rowStates(t, o.Out)
	for rel, want := range map[string]string{
		"app/main.ail":    statePassBoth,    // path dep from the snapshot + cached registry dep
		"stub/main.ail":   statePassBoth,    // unreadable committed lock regenerated
		"gone/main.ail":   stateNotMeasured, // acme/missing is nowhere
		"pinned/main.ail": stateNotMeasured, // committed lock pins a version the root lacks
	} {
		if got := rows[rel].State; got != want {
			t.Errorf("%s: state %q, want %q (baseline %+v)", rel, got, want, rows[rel].Base)
		}
	}
	cp := corpusProvision(t, rep, "deps")
	for dir, want := range map[string]string{"app": "generated", "stub": "regenerated", "gone": "FAILED", "pinned": "committed"} {
		if got := lockOf(cp, dir); got != want {
			t.Errorf("lock of %s: %q, want %q", dir, got, want)
		}
	}
	if got := pkgSource(cp, "acme/cached"); got != "package root copy" {
		t.Errorf("acme/cached source %q, want package root copy", got)
	}
	if got := pkgSource(cp, "acme/lib"); !strings.HasPrefix(got, "path ") {
		t.Errorf("acme/lib source %q, want a path into the snapshot", got)
	}
	un := strings.Join(cp.Unprovisioned, "\n")
	for _, want := range []string{"acme/missing@9.9.9", "acme/old@0.1.0 (pinned by pinned/ailang.lock)", "FETCH_DEPS=1", "gone: no committed ailang.lock; `ailang lock`: "} {
		if !strings.Contains(un, want) {
			t.Errorf("unprovisioned list does not name %q:\n%s", want, un)
		}
	}
	// Hermetic: locks were written only in the provisioned copy.
	snap := filepath.Join(o.Out, "corpora", "deps")
	if _, err := os.Stat(filepath.Join(snap, "app", "ailang.lock")); err == nil {
		t.Error("a lock was written into the snapshot")
	}
	if data, _ := os.ReadFile(filepath.Join(snap, "stub", "ailang.lock")); !strings.Contains(string(data), "names only") {
		t.Error("the snapshot's committed lock was changed")
	}
	if cp.Tree == "" {
		t.Error("no provisioned tree recorded although locks were generated")
	}
	// One provisioner for both arms: the released (baseline) binary.
	if !strings.Contains(rep.Provisioning.Binary, "(base)") {
		t.Errorf("provisioned by %q, want the baseline binary", rep.Provisioning.Binary)
	}
	if len(rep.Audit) > 0 {
		t.Errorf("write audit found %v", rep.Audit)
	}
}

// Fetch mode: the lock-pinned version is fetched from the registry into the
// run's package root copy (never the operator's), recorded, and measured; the
// version the registry does not have (HTTP 404) stays NOT_MEASURED and named.
func TestProvisionFetchFillsTheRunCopyOnly(t *testing.T) {
	f := newDepsFixture(t)
	registry := filepath.Join(f.dir, "registry")
	mkPkg(t, registry, "acme/old", "0.1.0")
	t.Setenv("DOWNSTREAM_FAKE_REGISTRY", registry)
	o := f.options(f.base, f.cand)
	o.FetchDeps = true
	code, rep := f.run(o)
	if code != exitIncomplete || len(rep.Failures) > 0 {
		t.Fatalf("exit %d, want 4 with no instrument failure; failures %v", code, rep.Failures)
	}
	rows := rowStates(t, o.Out)
	if got := rows["pinned/main.ail"].State; got != statePassBoth {
		t.Errorf("pinned/main.ail: %q, want pass_both once acme/old is fetched (baseline %+v)", got, rows["pinned/main.ail"].Base)
	}
	if got := rows["gone/main.ail"].State; got != stateNotMeasured {
		t.Errorf("gone/main.ail: %q, want NOT_MEASURED", got)
	}
	cp := corpusProvision(t, rep, "deps")
	if got := pkgSource(cp, "acme/old"); got != "registry fetch" {
		t.Errorf("acme/old source %q, want registry fetch", got)
	}
	if rep.Provisioning.Mode != "fetch" || rep.Provisioning.Registry != registry {
		t.Errorf("mode %q registry %q, want fetch from %s", rep.Provisioning.Mode, rep.Provisioning.Registry, registry)
	}
	if !strings.Contains(strings.Join(cp.Unprovisioned, "\n"), "HTTP 404") {
		t.Errorf("acme/missing not reported as absent from the registry: %v", cp.Unprovisioned)
	}
	if _, err := os.Stat(filepath.Join(f.dir, "pkgroot", "acme", "old")); err == nil {
		t.Error("the fetch wrote into the operator's package root")
	}
	if _, err := os.Stat(filepath.Join(o.Out, "pkgroot", "acme", "old", "0.1.0")); err != nil {
		t.Errorf("the fetched package is not in the run's package root copy: %v", err)
	}
}

// Fetch mode with the registry unreachable is an instrument failure (exit 3),
// never a quiet NOT_MEASURED; and fetch mode cannot be combined with offline.
func TestProvisionFetchFailsLoudly(t *testing.T) {
	f := newDepsFixture(t)
	t.Setenv("DOWNSTREAM_FAKE_REGISTRY", "")
	o := f.options(f.base, f.cand)
	o.FetchDeps = true
	code, rep := f.run(o)
	if code != exitInstrument || !strings.Contains(strings.Join(rep.Failures, "\n"), "ailang install --no-bin acme/") {
		t.Fatalf("exit %d, want 3 naming the failed install; failures %v", code, rep.Failures)
	}

	f2 := newDepsFixture(t)
	o2 := f2.options(f2.base, f2.cand)
	o2.FetchDeps, o2.Offline = true, true
	code, rep = f2.run(o2)
	if code != exitInstrument || !strings.Contains(strings.Join(rep.Failures, "\n"), "FETCH_DEPS=1 needs the network") {
		t.Fatalf("fetch+offline: exit %d, want 3; failures %v", code, rep.Failures)
	}
}

// Mark's ruling (2026-11-10): 600 s per check, not 120 (240 with the retry).
func TestDefaultTimeoutIsSixHundred(t *testing.T) {
	if got := defaultOptions().Timeout; got != 600 {
		t.Errorf("default --timeout %d, want 600", got)
	}
}
