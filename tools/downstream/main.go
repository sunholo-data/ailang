// Command downstream measures a release's blast radius before it ships: it runs
// `ailang check` over every downstream corpus in corpora.yaml with a BASELINE and
// a CANDIDATE binary and reports only the difference — entries newly failing (or
// newly passing), grouped by corpus and normalised error code — then subtracts
// expected-breaks.yaml, so a deliberate breaking change fails the gate only on
// breaks nobody acknowledged. Design: design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md.
//
//	make downstream-check CANDIDATE=<binary> [BASELINE=latest|vX.Y.Z|<binary>] [TIER=release|ci]
//	     [CORPORA=a,b] [REF=default|pinned|worktree] [OUT=<dir>] [OFFLINE=1]
//
// Exit 0 = no unacknowledged break; 2 = an unacknowledged or over-max bucket in a
// blocking corpus; 3 = the instrument could not measure (missing corpus, fetch
// failure, binary that does not start, failed canary, a write where none may happen).
//
// Isolation (design §6.3, enforced here, not by convention): each binary is copied
// into its own empty dir; every AILANG_*/XDG_* variable is dropped; HOME, compile
// cache, state and TMPDIR are per-arm scratch dirs; AILANG_PACKAGE_ROOT is a
// read-only package root, so a missing dependency is an error, never a download.
// Corpora are materialised into the run dir (snapshots of the remote ref through
// the operator's `gh`, or a copy of the local checkout in REF=worktree); the
// operator's checkouts are only read, with --no-optional-locks.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Options are the command-line knobs (the make target passes them through).
type Options struct {
	Candidate, Baseline, Tier, Corpora, Ref string
	Manifest, Expected, Out, State          string
	PackageRoot, SelfRoot                   string
	Jobs, Timeout                           int
	Offline, NoNice                         bool
}

func defaultOptions() Options {
	home, _ := os.UserHomeDir()
	return Options{
		Baseline: "latest", Tier: "release", Ref: refDefault,
		Manifest: "tools/downstream/corpora.yaml", Expected: "tools/downstream/expected-breaks.yaml",
		State:       filepath.Join(home, ".ailang", "state", "downstream"),
		PackageRoot: filepath.Join(home, ".ailang", "cache", "registry"),
		Jobs:        3, Timeout: 120,
	}
}

func main() {
	o := defaultOptions()
	fs := flag.NewFlagSet("downstream", flag.ExitOnError)
	fs.StringVar(&o.Candidate, "candidate", "", "candidate ailang binary (required)")
	fs.StringVar(&o.Baseline, "baseline", o.Baseline, "baseline: latest, a release tag, or a binary path")
	fs.StringVar(&o.Tier, "tier", o.Tier, "release | ci")
	fs.StringVar(&o.Corpora, "corpora", "", "comma-separated corpus names (default: every corpus in the tier)")
	fs.StringVar(&o.Ref, "ref", o.Ref, "default | pinned | worktree")
	fs.StringVar(&o.Manifest, "manifest", o.Manifest, "corpora.yaml")
	fs.StringVar(&o.Expected, "expected", o.Expected, "expected-breaks.yaml")
	fs.StringVar(&o.Out, "out", "", "report dir (default <state>/runs/<timestamp>); must be new or empty")
	fs.StringVar(&o.State, "state", o.State, "cache dir for snapshots and release baselines")
	fs.StringVar(&o.PackageRoot, "package-root", o.PackageRoot, "read-only AILANG_PACKAGE_ROOT for dependencies")
	fs.StringVar(&o.SelfRoot, "self-root", "", "the ailang checkout for in-repo corpora (default: git toplevel of cwd)")
	fs.IntVar(&o.Jobs, "jobs", o.Jobs, "parallel checks per arm")
	fs.IntVar(&o.Timeout, "timeout", o.Timeout, "seconds per check (a timeout is retried once at 2x)")
	fs.BoolVar(&o.Offline, "offline", false, "no network: reuse the last snapshot per corpus and a cached baseline")
	fs.BoolVar(&o.NoNice, "no-nice", false, "do not run checks under nice -n 15")
	_ = fs.Parse(os.Args[1:])
	code, err := Run(o, ghRemote{}, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "downstream:", err)
	}
	os.Exit(code)
}

// Run performs one comparison and writes the reports. The error is for failures
// before a report can be written; everything later is in the report and exit code.
func Run(o Options, remote Remote, log io.Writer) (int, error) {
	started := time.Now()
	if o.Candidate == "" {
		return 3, fmt.Errorf("CANDIDATE is required")
	}
	if o.Tier != "release" && o.Tier != "ci" {
		return 3, fmt.Errorf("TIER must be release or ci, got %q", o.Tier)
	}
	if o.Ref != refDefault && o.Ref != refPinned && o.Ref != refWorktree {
		return 3, fmt.Errorf("REF must be default, pinned or worktree, got %q", o.Ref)
	}
	corpora, err := LoadCorpora(o.Manifest)
	if err != nil {
		return 3, err
	}
	expected, err := LoadExpected(o.Expected, corpora)
	if err != nil {
		return 3, err
	}
	selected, err := selectCorpora(corpora, o.Tier, o.Corpora)
	if err != nil {
		return 3, err
	}
	if o.Out == "" {
		o.Out = filepath.Join(o.State, "runs", started.Format("20060102-150405"))
	}
	if err := freshDir(o.Out); err != nil {
		return 3, err
	}
	if o.SelfRoot == "" {
		if o.SelfRoot, err = gitOut(".", "rev-parse", "--show-toplevel"); err != nil {
			return 3, err
		}
	}
	rep := &Report{Started: started, Tier: o.Tier, RefMode: o.Ref, ArmWall: map[string]float64{}}
	fail := func(format string, a ...interface{}) { rep.Failures = append(rep.Failures, fmt.Sprintf(format, a...)) }
	finish := func(rows []Row) (int, error) {
		rep.WallSecs = time.Since(started).Seconds()
		rep.ExitCode, rep.Verdict = 0, "PASS"
		for _, b := range rep.Buckets {
			if b.Status == "UNEXPECTED" || b.Status == "OVER_MAX" {
				rep.ExitCode, rep.Verdict = 2, "FAIL: unacknowledged breaks"
			}
		}
		if len(rep.Failures) > 0 {
			rep.ExitCode, rep.Verdict = 3, "INSTRUMENT FAILURE: not measured"
		}
		if err := rep.WriteReports(o.Out, rows); err != nil {
			return 3, err
		}
		fmt.Fprintf(log, "downstream: %s (exit %d) — %s\n", rep.Verdict, rep.ExitCode, filepath.Join(o.Out, "report.md"))
		return rep.ExitCode, nil
	}

	// Preflight: every precondition of a meaningful measurement, loudly.
	if _, err := exec.LookPath("perl"); err != nil {
		fail("perl not found: it is the per-check timeout (macOS has no timeout(1)): %v", err)
	}
	if !o.NoNice {
		if _, err := exec.LookPath("nice"); err != nil {
			fail("nice not found: %v", err)
		}
	}
	for _, d := range systemStdlibDirs {
		if _, err := os.Stat(filepath.Join(d, "io.ail")); err == nil {
			fail("a system stdlib exists at %s; both arms would load it instead of their own", d)
		}
	}
	if fi, err := os.Stat(o.PackageRoot); err != nil || !fi.IsDir() {
		fail("package root %s is not a directory (it must exist; it is never written): %v", o.PackageRoot, err)
	}
	basePath, baseOrigin, err := ResolveBaseline(o.Baseline, o.State, o.Offline)
	if err != nil {
		fail("baseline: %v", err)
	}
	var base, cand *Arm
	if err == nil {
		if base, err = PrepareArm("baseline", basePath, baseOrigin, o.Out); err != nil {
			fail("%v", err)
		}
	}
	if cand, err = PrepareArm("candidate", o.Candidate, "path "+o.Candidate, o.Out); err != nil {
		fail("%v", err)
	}
	for _, a := range []*Arm{base, cand} {
		if a != nil {
			rep.Arms = append(rep.Arms, a)
		}
	}
	if len(rep.Failures) > 0 {
		return finish(nil)
	}
	rep.NullControl = base.SHA256 == cand.SHA256
	runner := &Runner{Jobs: o.Jobs, Timeout: o.Timeout, PackageRoot: o.PackageRoot, Nice: !o.NoNice}
	if msg := runCanary(runner, base, cand, filepath.Join(o.Out, "canary")); msg != "" {
		fail("canary: %s", msg)
		return finish(nil)
	}

	// Materialise and discover.
	mat := &Materialiser{StateDir: o.State, SelfRoot: o.SelfRoot, Offline: o.Offline, Remote: remote}
	var entries []Entry
	byCorpus := map[string][]Entry{}
	expectations := map[string]expectation{}
	var checked []Corpus
	var locals []string
	for _, c := range selected {
		sum := CorpusSummary{Name: c.Name, Level: c.tierOf(o.Tier), States: map[string]int{}, EntryRules: map[string]int{}}
		src, err := mat.Materialise(c, o.Ref, filepath.Join(o.Out, "corpora", c.Name))
		if src.Local != nil && c.Repo != selfRepo {
			// The audit covers sibling checkouts; the ailang checkout itself is
			// where the candidate is being built and edited, so it would only
			// report the operator's own work.
			locals = append(locals, src.Local.Path)
		}
		if err != nil {
			sum.NotChecked = err.Error()
			fail("NOT CHECKED %s: %v", c.Name, err)
			rep.Corpora = append(rep.Corpora, sum)
			continue
		}
		rep.Sources = append(rep.Sources, src)
		es, err := Discover(c, src.Root)
		if err == nil && len(es) == 0 {
			err = fmt.Errorf("no entries discovered (include/exclude match nothing?)")
		}
		var ex expectation
		if err == nil {
			ex, err = loadExpectation(c, src.Root)
		}
		if err != nil {
			sum.NotChecked = err.Error()
			fail("NOT CHECKED %s: %v", c.Name, err)
			rep.Corpora = append(rep.Corpora, sum)
			continue
		}
		rep.Pins = append(rep.Pins, findPins(c, src.Root)...)
		expectations[c.Name], byCorpus[c.Name] = ex, es
		entries = append(entries, es...)
		checked = append(checked, c)
		rep.Corpora = append(rep.Corpora, sum)
	}
	time.Sleep(20 * time.Millisecond)
	snapStamp := time.Now()

	// Both arms, one after the other.
	t0 := time.Now()
	baseRes := runner.RunArm(base, entries)
	rep.ArmWall["baseline"] = time.Since(t0).Seconds()
	t0 = time.Now()
	candRes := runner.RunArm(cand, entries)
	rep.ArmWall["candidate"] = time.Since(t0).Seconds()

	// Classify, gate, audit.
	var rows []Row
	blocking, inRun := map[string]bool{}, map[string]bool{}
	for _, c := range checked {
		inRun[c.Name] = true
		blocking[c.Name] = c.tierOf(o.Tier) == tierBlocking
	}
	for i := range rep.Corpora {
		sum := &rep.Corpora[i]
		for _, e := range byCorpus[sum.Name] {
			b, c := resultPtr(baseRes, e), resultPtr(candRes, e)
			row := classify(e, b, c, expectations[sum.Name])
			rows = append(rows, row)
			sum.States[row.State]++
			sum.EntryRules[e.Rule]++
			if e.Kind == "package" {
				sum.Packages++
			} else {
				sum.Files++
			}
			if _, err := os.Stat(filepath.Join(e.Cwd, "std", "io.ail")); err == nil {
				sum.CwdStdlibAt = append(sum.CwdStdlibAt, e.Rel)
			}
			if b != nil {
				sum.BaseSecs += b.Secs
				if !b.Passed {
					sum.BaseFail++
					if allDep(b.Codes) {
						sum.Unchecked++
					}
				}
			}
			if c != nil {
				sum.CandSecs += c.Secs
				if !c.Passed {
					for _, code := range c.Codes {
						rep.Uncoded[1]++
						if strings.HasPrefix(code, uncodedPrefix) {
							rep.Uncoded[0]++
						}
					}
				}
			}
			if row.State != statePassBoth {
				rep.Rows = append(rep.Rows, row)
			}
		}
	}
	if n := countState(rows, stateMissingArm); n > 0 {
		fail("%d entries have a result from only one arm", n)
	}
	rep.Buckets, rep.Expected, _ = Gate(rows, blocking, inRun, expected, cand.Version)
	for _, s := range rep.Sources {
		auditInto(rep, "snapshot "+s.Corpus, s.Root, snapStamp)
	}
	for _, l := range locals {
		auditInto(rep, "local checkout "+l, l, started)
	}
	return finish(rows)
}

func resultPtr(m map[string]Result, e Entry) *Result {
	r, ok := m[e.key()]
	if !ok {
		return nil
	}
	return &r
}

func allDep(codes []string) bool {
	for _, c := range codes {
		if !isDepCode(c) {
			return false
		}
	}
	return len(codes) > 0
}

func countState(rows []Row, st string) int {
	n := 0
	for _, r := range rows {
		if r.State == st {
			n++
		}
	}
	return n
}

func auditInto(rep *Report, scope, root string, stamp time.Time) {
	paths, err := writtenSince(root, stamp, 20)
	if err != nil {
		rep.Failures = append(rep.Failures, fmt.Sprintf("write audit of %s failed: %v", scope, err))
		return
	}
	for _, p := range paths {
		rep.Audit = append(rep.Audit, AuditFinding{Scope: scope, Path: p})
	}
	if len(paths) > 0 {
		rep.Failures = append(rep.Failures, fmt.Sprintf("write audit: %d path(s) under %s changed during the run", len(paths), scope))
	}
}

func selectCorpora(all []Corpus, tier, names string) ([]Corpus, error) {
	want := map[string]bool{}
	for _, n := range strings.Split(names, ",") {
		if n = strings.TrimSpace(n); n != "" {
			want[n] = true
		}
	}
	narrowed := len(want) > 0
	var out []Corpus
	for _, c := range all {
		if narrowed {
			if want[c.Name] {
				if c.tierOf(tier) == tierOff {
					return nil, fmt.Errorf("corpus %s is off in tier %s", c.Name, tier)
				}
				out = append(out, c)
				delete(want, c.Name)
			}
			continue
		}
		if c.tierOf(tier) != tierOff {
			out = append(out, c)
		}
	}
	for n := range want {
		return nil, fmt.Errorf("CORPORA names unknown corpus %q", n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no corpora selected for tier %s", tier)
	}
	return out, nil
}

func freshDir(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(ents) > 0 {
		return fmt.Errorf("report dir %s is not empty", dir)
	}
	return nil
}

// runCanary is the planted positive control run every time (design §6.3: a
// missing timeout binary once exited 127 in both arms and reported "0 flips").
// A valid file must pass and a broken one must fail with a parsed diagnostic, in
// both arms; anything else means the instrument cannot see.
func runCanary(r *Runner, base, cand *Arm, dir string) string {
	files := map[string]string{
		"canary/ok.ail":     "module canary/ok\n\nexport func answer() -> int = 42\n",
		"canary/broken.ail": "module canary/broken\n\nexport func answer() -> int = \"not an int\"\n",
	}
	var es []Entry
	for rel, src := range files {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err.Error()
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			return err.Error()
		}
		es = append(es, fileEntry("_canary", dir, rel))
	}
	for _, a := range []*Arm{base, cand} {
		res := r.RunArm(a, es)
		for _, e := range es {
			got := res[e.key()]
			wantPass := strings.HasSuffix(e.Rel, "ok.ail")
			if got.Passed != wantPass {
				return fmt.Sprintf("%s arm: %s passed=%v, want %v (codes %v, rc %d, %s)", a.Name, e.Rel, got.Passed, wantPass, got.Codes, got.RC, got.Msg)
			}
			if !wantPass && (len(got.Codes) == 0 || strings.HasPrefix(got.Codes[0], codeCrash) || got.Codes[0] == codeTimeout) {
				return fmt.Sprintf("%s arm: %s did not fail with a parsed diagnostic (codes %v, rc %d, %s)", a.Name, e.Rel, got.Codes, got.RC, got.Msg)
			}
		}
	}
	return ""
}

// findPins greps each pin_files "path:regex" spec, plus every ailang.toml's
// `ailang =` floor, in the materialised snapshot.
func findPins(c Corpus, root string) []PinHit {
	var out []PinHit
	grep := func(rel string, match func(string) bool) {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return
		}
		for i, line := range strings.Split(string(data), "\n") {
			if match(line) {
				out = append(out, PinHit{Corpus: c.Name, Where: fmt.Sprintf("%s:%d", rel, i+1), Text: firstLine(strings.TrimSpace(line))})
			}
		}
	}
	for _, spec := range c.PinFiles {
		rel, re, _ := splitPinSpec(spec)
		grep(rel, re.MatchString)
	}
	es, _ := Discover(Corpus{Name: c.Name, Modes: []string{"package"}, Exclude: c.Exclude}, root)
	for _, e := range es {
		grep(filepath.ToSlash(filepath.Join(e.Rel, "ailang.toml")), func(l string) bool {
			return strings.HasPrefix(strings.TrimSpace(l), "ailang =") || strings.HasPrefix(strings.TrimSpace(l), "ailang=")
		})
	}
	return out
}
