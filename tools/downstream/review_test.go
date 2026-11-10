package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tests for the PR #2208 review findings: each one reported PASS (or hung) without
// having measured.

// goFake returns the test binary as a stand-in compiler (see gofake_test.go).
func goFake(t *testing.T) string {
	t.Helper()
	needUnix(t)
	t.Setenv(goFakeEnv, "1")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

// Finding 1: the timeout must fire on a Go binary (which ignores SIGALRM) and take
// its whole process group with it, including a grandchild holding stdout open.
func TestTimeoutKillsGoBinaryAndItsGrandchild(t *testing.T) {
	bin := goFake(t)
	dir := t.TempDir()
	arm, err := PrepareArm("candidate", bin, "test", dir)
	if err != nil {
		t.Fatal(err)
	}
	corpus := filepath.Join(dir, "corpus")
	if err := os.MkdirAll(corpus, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corpus, "hang.ail"), []byte("-- HANG_CHILD\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &Runner{Jobs: 1, Timeout: 1, PackageRoot: dir, Nice: true, RunDir: dir}
	t0 := time.Now()
	res := r.runEntry(arm, fileEntry("c", corpus, "hang.ail"))
	if took := time.Since(t0); took > 15*time.Second {
		t.Fatalf("a 1 s timeout (retried at 2 s) took %s: the deadline did not fire on a Go binary", took)
	}
	if len(res.Codes) != 1 || res.Codes[0] != codeTimeout || !res.Retried {
		t.Fatalf("result %+v, want a retried TIMEOUT", res)
	}
	data, err := os.ReadFile(filepath.Join(arm.Dir, "state", "grandchild.pid"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for alive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if alive(pid) {
		killPid(pid)
		t.Fatalf("grandchild %d survived the timeout: the process group was not killed", pid)
	}
}

// Finding 1, end to end: a candidate-only hang of a real Go binary is timeout_cand.
func TestCandidateTimeoutIsItsOwnState(t *testing.T) {
	f := newFixture(t, "- name: app\n  repo: owner/app\n  tier: {ci: off, release: blocking}\n", "")
	cand := goFake(t)
	f.remote.repos["owner/app"] = map[string]string{"s/slow.ail": "module s/slow\n-- HANG\n"}
	o := f.options(f.base, cand)
	o.Timeout = 1
	t0 := time.Now()
	code, rep := f.run(o)
	if took := time.Since(t0); took > 20*time.Second {
		t.Fatalf("run took %s", took)
	}
	r := rowByRel(rep, "s/slow.ail")
	if code != 2 || r == nil || r.State != stateTimeoutCand || !r.Cand.Retried {
		t.Fatalf("exit %d row %+v, want 2 and a retried timeout_cand", code, r)
	}
}

const oneBlocking = "- name: app\n  repo: owner/app\n  tier: {ci: off, release: blocking}\n"

// Finding 2: a dependency that resolves in neither arm was never checked. It must
// not read as "unchanged" and the gate must not PASS on it.
func TestUnresolvedDepsInBothArmsAreNotMeasured(t *testing.T) {
	f := newFixture(t, oneBlocking, "")
	f.remote.repos["owner/app"] = map[string]string{
		"a/ok.ail":  "module a/ok\n",
		"a/dep.ail": "module a/dep\n-- DEPMISS\n",
	}
	code, rep := f.run(f.options(f.base, f.base))
	if r := rowByRel(rep, "a/dep.ail"); r == nil || r.State != stateNotMeasured || r.NotMeasured != reasonDeps {
		t.Errorf("dep row %+v, want NOT_MEASURED deps_unresolved", r)
	}
	if code != exitIncomplete {
		t.Fatalf("exit %d verdict %q, want 4 with an unmeasured entry in a blocking corpus", code, rep.Verdict)
	}
	if n := rep.Corpora[0].NotMeasured[reasonDeps]; n != 1 {
		t.Errorf("deps_unresolved counter = %d, want 1", n)
	}
	msg := strings.Join(rep.Incomplete, "\n")
	if !strings.Contains(msg, "app") || !strings.Contains(msg, "provision") || !strings.Contains(msg, "allow_not_measured") {
		t.Errorf("exit-4 message %q does not tell the operator what to fix or acknowledge", msg)
	}
	md, err := os.ReadFile(filepath.Join(f.dir, "out", "report.md"))
	if err != nil || !strings.Contains(string(md), "a/dep.ail") || !strings.Contains(string(md), "exit 4") {
		t.Errorf("report.md does not list the unmeasured entry: %v", err)
	}
}

// Finding 2: the allowance is explicit, counted and reviewed — within it passes,
// above it fails, and an allowance with nothing left to excuse is stale.
func TestNotMeasuredAllowance(t *testing.T) {
	allow := "  allow_not_measured: {max: %d, reason: \"deps not provisioned on the rig\"}\n"
	for _, tc := range []struct {
		name   string
		files  map[string]string
		max    int
		code   int
		status string
	}{
		{"within", map[string]string{"a/dep.ail": "-- DEPMISS\n"}, 1, exitPass, "within"},
		{"over", map[string]string{"a/dep.ail": "-- DEPMISS\n", "a/dep2.ail": "-- DEPMISS\n"}, 1, exitIncomplete, "OVER"},
		{"stale", map[string]string{"a/ok.ail": "module a/ok\n"}, 3, exitIncomplete, "STALE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, oneBlocking+fmt.Sprintf(allow, tc.max), "")
			f.remote.repos["owner/app"] = tc.files
			code, rep := f.run(f.options(f.base, f.base))
			if code != tc.code || rep.Corpora[0].AllowanceStatus != tc.status {
				t.Fatalf("exit %d status %q, want %d %q; %v", code, rep.Corpora[0].AllowanceStatus, tc.code, tc.status, rep.Incomplete)
			}
		})
	}
	// Advisory corpora report unmeasured entries but never gate on them.
	f := newFixture(t, "- name: app\n  repo: owner/app\n  tier: {ci: off, release: advisory}\n", "")
	f.remote.repos["owner/app"] = map[string]string{"a/dep.ail": "-- DEPMISS\n"}
	if code, rep := f.run(f.options(f.base, f.base)); code != exitPass || rep.Corpora[0].AllowanceStatus != "advisory" {
		t.Fatalf("advisory corpus: exit %d, want 0", code)
	}
}

func TestClassifyNotMeasured(t *testing.T) {
	e := Entry{Corpus: "c", Kind: "file", Rel: "x.ail"}
	fail := func(codes ...string) *Result { return &Result{Codes: codes} }
	for _, tc := range []struct {
		name       string
		b, c       *Result
		state, why string
		newCodes   string
	}{
		{"deps both", fail("DEP/unresolved"), fail("DEP/unresolved"), stateNotMeasured, reasonDeps, ""},
		{"timeout both", fail(codeTimeout), fail(codeTimeout), stateNotMeasured, reasonTimeout, ""},
		{"crash both", fail("CRASH/rc=2"), fail("CRASH/rc=2"), stateNotMeasured, reasonCrash, ""},
		{"no diagnostic", fail(codeNoDiagnostic), fail(codeNoDiagnostic), stateNotMeasured, reasonCrash, ""},
		{"base timeout, cand type error", fail(codeTimeout), fail("TYPE_ERROR/unify"), stateNotMeasured, reasonTimeout, "TYPE_ERROR/unify"},
		{"base deps, cand deps+type", fail("DEP/unresolved"), fail("DEP/unresolved", "TYPE_ERROR/unify"), stateNotMeasured, reasonDeps, "TYPE_ERROR/unify"},
		{"base deps, cand crash", fail("DEP/unresolved"), fail("CRASH/rc=139"), stateNotMeasured, reasonDeps, "CRASH/rc=139"},
		{"base deps, cand passes", fail("DEP/unresolved"), &Result{Passed: true}, stateNewlyPassing, "", ""},
		{"deps beside a real error is measured", fail("DEP/unresolved", "PAR001"), fail("DEP/unresolved", "PAR001"), stateFailSame, "", ""},
		{"base real, cand timeout", fail("PAR001"), fail(codeTimeout), stateTimeoutCand, "", codeTimeout},
	} {
		row := classify(e, tc.b, tc.c, expectation{})
		if row.State != tc.state || row.NotMeasured != tc.why || strings.Join(row.NewCodes, ",") != tc.newCodes {
			t.Errorf("%s: state %s reason %q new %v, want %s %q %q", tc.name, row.State, row.NotMeasured, row.NewCodes, tc.state, tc.why, tc.newCodes)
		}
	}
}

// Finding 5: an entry with a result from only one arm is an instrument failure.
func TestOneArmResultIsAnInstrumentFailure(t *testing.T) {
	e := Entry{Corpus: "app", Kind: "file", Rel: "a.ail", Cwd: t.TempDir()}
	rep := &Report{Corpora: []CorpusSummary{{Name: "app", Level: tierBlocking, States: map[string]int{}, EntryRules: map[string]int{}, NotMeasured: map[string]int{}}}}
	both := map[string]Result{e.key(): {Passed: true}}
	rep.score(map[string][]Entry{"app": {e}}, map[string]expectation{}, both, map[string]Result{})
	rep.verdict()
	if rep.ExitCode != exitInstrument || !strings.Contains(strings.Join(rep.Failures, "\n"), "only one arm") {
		t.Fatalf("exit %d failures %v, want 3 for a one-armed entry", rep.ExitCode, rep.Failures)
	}
}

// SHOULD-FIX: the package root is shared by every arm and must stay unwritten.
func TestPackageRootWriteIsAudited(t *testing.T) {
	f := newFixture(t, oneBlocking, "")
	f.remote.repos["owner/app"] = map[string]string{"a/w.ail": "module a/w\n-- WRITE_PKGROOT\n"}
	code, rep := f.run(f.options(f.base, f.cand))
	named := false
	for _, a := range rep.Audit {
		named = named || (strings.HasPrefix(a.Scope, "package root") && a.Path == "leaked.txt")
	}
	if code != exitInstrument || !named {
		t.Fatalf("exit %d audit %+v, want 3 naming the package-root write", code, rep.Audit)
	}
}

// Finding 2: a baseline timeout hides nothing — the candidate's real type error on
// that row is a new code.
func TestBaselineTimeoutDoesNotHideACandidateError(t *testing.T) {
	f := newFixture(t, oneBlocking, "")
	f.remote.repos["owner/app"] = map[string]string{"a/slow.ail": "module a/slow\n-- BASESLOW\n"}
	o := f.options(f.base, f.cand)
	o.Timeout = 1
	_, rep := f.run(o)
	found := false
	for _, b := range rep.Buckets {
		found = found || (b.Corpus == "app" && b.Code == "TYPE_ERROR/unify" && b.Status == "UNEXPECTED")
	}
	if !found {
		t.Fatalf("candidate type error behind a baseline timeout was not counted; buckets %+v rows %+v", rep.Buckets, rep.Rows)
	}
}

// Finding 3: exit_code in the examples manifest is a RUN-time exit code. Without an
// error_pattern it must not excuse a check failure.
func TestExamplesManifestRunTimeExitCodeExcusesNothing(t *testing.T) {
	manifest := "- name: ex\n  repo: owner/ex\n  examples_manifest: examples/manifest.json\n  tier: {ci: off, release: blocking}\n"
	f := newFixture(t, manifest, "")
	f.remote.repos["owner/ex"] = map[string]string{
		"examples/manifest.json":   `{"examples":[{"path":"exit_code.ail","expected":{"exit_code":42,"stdout":"Exiting with code 42\n"}}]}`,
		"examples/exit_code.ail":   "module exit_code\n-- PLANT\n",
		"examples/other/plain.ail": "module other/plain\n",
	}
	code, rep := f.run(f.options(f.base, f.cand))
	if r := rowByRel(rep, "examples/exit_code.ail"); r == nil || r.State != stateNewlyFailing {
		t.Errorf("row %+v, want NEWLY_FAILING (a run-time exit code is not a check expectation)", r)
	}
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

// Finding 3: an error_pattern that is not a valid regex is a configuration error,
// not a silent literal match.
func TestExamplesManifestInvalidRegexIsLoud(t *testing.T) {
	manifest := "- name: ex\n  repo: owner/ex\n  examples_manifest: examples/manifest.json\n  tier: {ci: off, release: blocking}\n"
	f := newFixture(t, manifest, "")
	f.remote.repos["owner/ex"] = map[string]string{
		"examples/manifest.json": `{"examples":[{"path":"neg/x.ail","expected":{"exit_code":1,"error_pattern":"unclosed (group"}}]}`,
		"examples/neg/x.ail":     "module neg/x\n",
	}
	code, rep := f.run(f.options(f.base, f.base))
	if code != 3 || !strings.Contains(strings.Join(rep.Failures, "\n"), "error_pattern") {
		t.Fatalf("exit %d failures %v, want 3 naming the bad error_pattern", code, rep.Failures)
	}
}

// Finding 4: two different errors with one code are not "the same failure".
func TestSameCodeDifferentErrorIsVisible(t *testing.T) {
	f := newFixture(t, oneBlocking, "")
	f.remote.repos["owner/app"] = map[string]string{
		"a/drift.ail": "module a/drift\n-- MSGDRIFT\n",
		"a/path.ail":  "module a/path\n-- ABSPATH\n",
	}
	code, rep := f.run(f.options(f.base, f.cand))
	if r := rowByRel(rep, "a/drift.ail"); r == nil || r.State != "fail_diag_changed" {
		t.Errorf("drift row %+v, want fail_diag_changed", r)
	}
	// The same error naming each arm's own scratch path is the same error.
	if r := rowByRel(rep, "a/path.ail"); r == nil || r.State != stateFailSame {
		t.Errorf("abs-path row %+v, want fail_same (volatile paths are normalised away)", r)
	}
	if code != 0 {
		t.Errorf("exit %d, want 0: a changed message is reported, not gated", code)
	}
}

// Finding 5: a corpus whose include globs match nothing is NOT CHECKED, never "0 breaks".
func TestNoEntriesDiscoveredIsLoud(t *testing.T) {
	f := newFixture(t, "- name: app\n  repo: owner/app\n  include: [\"nowhere/**/*.ail\"]\n  tier: {ci: off, release: blocking}\n", "")
	f.remote.repos["owner/app"] = map[string]string{"a/ok.ail": "module a/ok\n"}
	code, rep := f.run(f.options(f.base, f.base))
	if code != 3 || !strings.Contains(strings.Join(rep.Failures, "\n"), "no entries discovered") {
		t.Fatalf("exit %d failures %v, want 3 naming the empty corpus", code, rep.Failures)
	}
}

// Finding 5: both arms failing with different code sets counts the new codes.
func TestFailChangedCountsOnlyNewCodes(t *testing.T) {
	e := Entry{Corpus: "c", Kind: "file", Rel: "x.ail"}
	b := &Result{Codes: []string{"PAR001", "TYPE_ERROR/unify"}}
	c := &Result{Codes: []string{"MOD010", "TYPE_ERROR/unify"}}
	row := classify(e, b, c, expectation{})
	if row.State != stateFailChanged || strings.Join(row.NewCodes, ",") != "MOD010" {
		t.Fatalf("row %+v, want fail_changed with new code MOD010", row)
	}
}
