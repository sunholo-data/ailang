package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The three controls the gate must pass before its numbers mean anything
// (design doc §7 M1/M3): null control, planted break, expected-break accounting.

const twoCorpora = `
- name: app
  repo: owner/app
  tier: {ci: advisory, release: blocking}
- name: side
  repo: owner/side
  tier: {ci: off, release: advisory}
`

func appFiles() map[string]string {
	return map[string]string{
		"src/a/ok.ail":     "module a/ok\nexport func x() -> int = 1\n",
		"src/a/also.ail":   "module a/also\nexport func y() -> int = 2\n",
		"src/a/broken.ail": "module a/broken\n-- BROKEN in both arms\n",
		"src/a/plant.ail":  "module a/plant\n-- PLANT: the candidate rejects this\n",
		"pkg/ailang.toml":  "[package]\nname = \"x/pkg\"\nailang = \">=0.50.0\"\n",
		"pkg/lib.ail":      "module lib\nexport func z() -> int = 3\n",
		"pkg2/ailang.toml": "[package]\nname = \"x/pkg2\"\n",
		"pkg2/bad.ail":     "module bad\n-- BROKEN package\n",
	}
}

func sideFiles() map[string]string {
	return map[string]string{"m/s.ail": "module m/s\n-- PLANT in an advisory corpus\n"}
}

// Control 1: the candidate is the baseline binary. Zero differences, exit 0.
func TestControlNullBaselineVsItself(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	code, rep := f.run(f.options(f.base, f.base))
	if code != 0 {
		t.Fatalf("exit %d, want 0; failures %v", code, rep.Failures)
	}
	if !rep.NullControl {
		t.Error("null_control not set although both arms are the same binary")
	}
	for st, n := range stateCounts(rep) {
		if st != statePassBoth && st != stateFailSame && n > 0 {
			t.Errorf("null control produced %d %s rows", n, st)
		}
	}
	if len(rep.Buckets) != 0 {
		t.Errorf("null control produced buckets %+v", rep.Buckets)
	}
	if got := stateCounts(rep)[statePassBoth] + stateCounts(rep)[stateFailSame]; got != 9 {
		t.Errorf("checked %d entries, want 9 (7 files + 2 packages)", got)
	}
}

// Control 2: one planted break is reported exactly once, with its code, and the
// file broken in both arms is fail_same, not a break.
func TestControlPlantedBreakReportedOnce(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	code, rep := f.run(f.options(f.base, f.cand))
	if code != 2 {
		t.Fatalf("exit %d, want 2 (unacknowledged break); failures %v", code, rep.Failures)
	}
	var blocking []Bucket
	for _, b := range rep.Buckets {
		if b.Blocking {
			blocking = append(blocking, b)
		}
	}
	if len(blocking) != 1 || blocking[0].Corpus != "app" || blocking[0].Code != "EFFECT/pure-with-row" || blocking[0].Count != 1 || blocking[0].Status != "UNEXPECTED" {
		t.Fatalf("blocking buckets = %+v, want exactly one app/EFFECT/pure-with-row x1 UNEXPECTED", blocking)
	}
	if r := rowByRel(rep, "src/a/plant.ail"); r == nil || r.State != stateNewlyFailing {
		t.Errorf("plant row = %+v, want NEWLY_FAILING", r)
	}
	if r := rowByRel(rep, "src/a/broken.ail"); r == nil || r.State != stateFailSame {
		t.Errorf("broken row = %+v, want fail_same", r)
	}
	if r := rowByRel(rep, "pkg2"); r == nil || r.Kind != "package" || r.State != stateFailSame {
		t.Errorf("failing package with a new candidate warning = %+v, want fail_same", r)
	}
	if n := stateCounts(rep)[stateNewlyFailing]; n != 2 {
		t.Errorf("NEWLY_FAILING = %d, want 2 (app plant + advisory side plant)", n)
	}
	for _, b := range rep.Buckets {
		if b.Corpus == "side" && b.Status != "advisory" {
			t.Errorf("advisory corpus bucket status %q, want advisory", b.Status)
		}
	}
	md, err := os.ReadFile(filepath.Join(f.dir, "out", "report.md"))
	if err != nil || !strings.Contains(string(md), "EFFECT/pure-with-row") || !strings.Contains(string(md), "clean snapshot") {
		t.Errorf("report.md lacks the bucket or the source state: %v", err)
	}
}

// Control 3: an expected-break entry suppresses exactly its bucket; an entry that
// matches nothing is UNUSED and fails the gate (a stale acknowledgement is how a
// later real break gets excused); a count above max fails.
func TestControlExpectedBreaks(t *testing.T) {
	used := `
- change: "#9999 planted"
  code: EFFECT/pure-with-row
  corpus: app
  max: 1
  expires: v0.58.0
`
	expected := used + `
- change: "#9998 never happens"
  code: TYPE_ERROR/unify
  corpus: app
  max: 4
`
	f0 := newFixture(t, twoCorpora, used)
	f0.remote.repos["owner/app"], f0.remote.repos["owner/side"] = appFiles(), sideFiles()
	if code, rep := f0.run(f0.options(f0.base, f0.cand)); code != 0 {
		t.Fatalf("exit %d, want 0 with the break acknowledged; buckets %+v failures %v", code, rep.Buckets, rep.Failures)
	}

	f := newFixture(t, twoCorpora, expected)
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	code, rep := f.run(f.options(f.base, f.cand))
	if code != 2 || !strings.Contains(rep.Verdict, "matched nothing") {
		t.Fatalf("exit %d verdict %q, want 2 for the unused entry", code, rep.Verdict)
	}
	got := map[string]string{}
	for _, e := range rep.Expected {
		got[e.Change] = e.Status
		if e.Change == "#9999 planted" && !e.Stale {
			t.Error("entry expiring v0.58.0 not stale against candidate v0.58.1")
		}
	}
	if got["#9999 planted"] != "used" || got["#9998 never happens"] != "UNUSED" {
		t.Errorf("expected-break statuses %v", got)
	}
	expected = used

	// Over max: a second planted file in the same bucket.
	files := appFiles()
	files["src/a/plant2.ail"] = "module a/plant2\n-- PLANT again\n"
	f2 := newFixture(t, twoCorpora, expected)
	f2.remote.repos["owner/app"], f2.remote.repos["owner/side"] = files, sideFiles()
	code, rep = f2.run(f2.options(f2.base, f2.cand))
	if code != 2 {
		t.Fatalf("exit %d, want 2 for 2 breaks over max 1", code)
	}
	for _, b := range rep.Buckets {
		if b.Corpus == "app" && (b.Status != "OVER_MAX" || b.Count != 2) {
			t.Errorf("bucket %+v, want OVER_MAX count 2", b)
		}
	}
}

// An examples-manifest entry with a non-zero exit code and a matching
// error_pattern is expected_fail, not a break (design §6.5, run A's two examples).
func TestExamplesManifestExpectedFail(t *testing.T) {
	manifest := `
- name: ex
  repo: owner/ex
  examples_manifest: examples/manifest.json
  tier: {ci: off, release: blocking}
`
	f := newFixture(t, manifest, "")
	f.remote.repos["owner/ex"] = map[string]string{
		"examples/manifest.json": `{"examples":[{"path":"neg/plant.ail","expected":{"exit_code":1,"error_pattern":"pure keyword contradicts"}},
			{"path":"neg/wrongpat.ail","expected":{"exit_code":1,"error_pattern":"information-flow"}}]}`,
		"examples/neg/plant.ail":    "module neg/plant\n-- PLANT\n",
		"examples/neg/wrongpat.ail": "module neg/wrongpat\n-- PLANT\n",
	}
	code, rep := f.run(f.options(f.base, f.cand))
	if r := rowByRel(rep, "examples/neg/plant.ail"); r == nil || r.State != stateExpectedFail {
		t.Errorf("plant row %+v, want expected_fail", r)
	}
	if r := rowByRel(rep, "examples/neg/wrongpat.ail"); r == nil || r.State != stateNewlyFailing {
		t.Errorf("a failure not matching error_pattern must stay NEWLY_FAILING, got %+v", r)
	}
	if code != 2 {
		t.Errorf("exit %d, want 2 (wrongpat is unexpected)", code)
	}
}

// A corpus that cannot be fetched is NOT CHECKED and the run exits 3 — never a
// silent skip that reads as "0 breaks".
func TestUnreachableCorpusIsLoud(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/side"] = sideFiles()
	f.remote.fail["owner/app"] = true
	code, rep := f.run(f.options(f.base, f.base))
	if code != 3 {
		t.Fatalf("exit %d, want 3", code)
	}
	if !strings.Contains(strings.Join(rep.Failures, "\n"), "NOT CHECKED app") {
		t.Errorf("failures %v do not name the unchecked corpus", rep.Failures)
	}
}

func TestWorktreeModeMissingCheckoutIsLoud(t *testing.T) {
	f := newFixture(t, "- name: app\n  repo: owner/app\n  local: /nonexistent/checkout\n  tier: {ci: off, release: blocking}\n", "")
	o := f.options(f.base, f.base)
	o.Ref = refWorktree
	code, rep := f.run(o)
	if code != 3 || !strings.Contains(strings.Join(rep.Failures, "\n"), "unreadable checkout") {
		t.Fatalf("exit %d failures %v, want 3 naming the checkout", code, rep.Failures)
	}
}

func TestBinaryThatDoesNotStartIsLoud(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	dead := filepath.Join(f.dir, "dead", "ailang")
	if err := os.MkdirAll(filepath.Dir(dead), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dead, []byte("#!/bin/bash\necho 'dyld: missing symbol' >&2\nexit 134\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, cand := range map[string]string{"crashing": dead, "missing": filepath.Join(f.dir, "nope")} {
		o := f.options(f.base, cand)
		o.Out = filepath.Join(f.dir, "out-"+name)
		if code, rep := f.run(o); code != 3 || len(rep.Failures) == 0 {
			t.Errorf("%s candidate: exit %d, want 3", name, code)
		}
	}
}

// The canary catches a broken timeout wrapper: a binary that answers --version but
// never produces check JSON must not read as "0 flips".
func TestCanaryCatchesBlindInstrument(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	blind := filepath.Join(f.dir, "blind", "ailang")
	if err := os.MkdirAll(filepath.Dir(blind), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blind, []byte("#!/bin/bash\n[ \"$1\" = --version ] && { echo v; exit 0; }\nexit 127\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, rep := f.run(f.options(blind, blind))
	if code != 3 || !strings.Contains(strings.Join(rep.Failures, "\n"), "canary") {
		t.Fatalf("exit %d failures %v, want 3 from the canary", code, rep.Failures)
	}
}

// Isolation: the child sees no operator AILANG_* variable, a scratch HOME, and the
// read-only package root; a write into a corpus is an instrument failure.
func TestIsolationEnvAndWriteAudit(t *testing.T) {
	t.Setenv("AILANG_LEAK_PROBE", "operator-value")
	f := newFixture(t, twoCorpora, "")
	files := appFiles()
	files["src/a/writer.ail"] = "module a/writer\n-- WRITE_CWD\n"
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = files, sideFiles()
	code, rep := f.run(f.options(f.base, f.cand))
	named := false
	for _, a := range rep.Audit {
		named = named || (a.Scope == "snapshot app" && a.Path == "src/leaked.txt")
	}
	if code != 3 || !named {
		t.Fatalf("exit %d audit %+v, want 3 naming leaked.txt", code, rep.Audit)
	}
	env, err := os.ReadFile(filepath.Join(f.dir, "out", "arms", "candidate", "state", "env.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(env)
	if strings.Contains(s, "AILANG_LEAK_PROBE") {
		t.Error("operator AILANG_* variable leaked into the check environment")
	}
	for _, want := range []string{
		"HOME=" + filepath.Join(f.dir, "out", "arms", "candidate", "home"),
		"AILANG_CACHE_DIR=" + filepath.Join(f.dir, "out", "arms", "candidate", "cache"),
		"AILANG_PACKAGE_ROOT=" + filepath.Join(f.dir, "out", "pkgroot"),
		"AILANG_TRACE=off",
	} {
		if !strings.Contains(s, want+"\n") {
			t.Errorf("child env lacks %s", want)
		}
	}
}

// The second run reuses the cached snapshot and REF=default resolves the ref again.
func TestSnapshotCacheAndOffline(t *testing.T) {
	f := newFixture(t, twoCorpora, "")
	f.remote.repos["owner/app"], f.remote.repos["owner/side"] = appFiles(), sideFiles()
	if code, _ := f.run(f.options(f.base, f.base)); code != 0 {
		t.Fatalf("first run exit %d", code)
	}
	f.remote.calls = nil
	o := f.options(f.base, f.base)
	o.Out, o.Offline = filepath.Join(f.dir, "out2"), true
	code, rep := f.run(o)
	if code != 0 || len(f.remote.calls) != 0 {
		t.Fatalf("offline run exit %d, remote calls %v (want none)", code, f.remote.calls)
	}
	if !rep.Sources[0].Cached {
		t.Error("offline source not marked cached")
	}
}

// A sibling checkout that changes while the gate runs is somebody else's work on a
// shared machine, not a write by the gate: the arms read snapshots. It is listed in
// the report and must not turn a clean measurement into exit 3 — which it did on the
// first release-tier run after the corpus fixes (0 newly failing, 0 not measured,
// verdict "INSTRUMENT FAILURE") because another session saved files in a checkout.
func TestLocalCheckoutChangeIsReportedNotGated(t *testing.T) {
	needUnix(t)
	local := filepath.Join(t.TempDir(), "sibling")
	if err := os.MkdirAll(local, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = local
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	manifest := "\n- name: app\n  repo: owner/app\n  local: " + local + "\n  tier: {ci: advisory, release: blocking}\n"
	f := newFixture(t, manifest, "")
	edited := filepath.Join(local, "edited-by-someone-else.txt")
	f.remote.repos["owner/app"] = map[string]string{
		"src/a/ok.ail":        "module a/ok\nexport func x() -> int = 1\n",
		"src/a/bystander.ail": "module a/bystander\n-- TOUCH_FILE=" + edited + "\n",
	}
	code, rep := f.run(f.options(f.base, f.cand))
	if rep == nil {
		t.Fatalf("no report (exit %d)", code)
	}
	if _, err := os.Stat(edited); err != nil {
		t.Fatalf("control did not fire: the sibling checkout was never touched: %v", err)
	}
	if code != 0 || len(rep.Failures) != 0 || len(rep.Audit) != 0 {
		t.Fatalf("exit %d failures %v audit %+v; a bystander's edit must not gate", code, rep.Failures, rep.Audit)
	}
	named := false
	for _, c := range rep.LocalChanges {
		named = named || (strings.Contains(c.Scope, local) && c.Path == "edited-by-someone-else.txt")
	}
	if !named {
		t.Fatalf("local change not reported: %+v", rep.LocalChanges)
	}
	md, err := os.ReadFile(filepath.Join(f.dir, "out", "report.md"))
	if err != nil || !strings.Contains(string(md), "reported, not gated") || !strings.Contains(string(md), "edited-by-someone-else.txt") {
		t.Fatalf("report.md does not show the local change (err %v)", err)
	}
}
