// Command downstream measures a release's blast radius before it ships: it runs
// `ailang check` over every downstream corpus in corpora.yaml with a BASELINE and
// a CANDIDATE binary and reports only the difference — entries newly failing (or
// newly passing), grouped by corpus and normalised error code — then subtracts
// expected-breaks.yaml, so a deliberate breaking change fails the gate only on
// breaks nobody acknowledged. Design: design_docs/planned/v1_0_0/m-v1-downstream-compat-gate.md.
//
//	make downstream-check CANDIDATE=<binary> [BASELINE=latest|vX.Y.Z|<binary>] [TIER=release|ci]
//	     [CORPORA=a,b] [REF=default|pinned|worktree] [OUT=<dir>] [OFFLINE=1] [FETCH_DEPS=1]
//
// Exit codes (precedence 3 > 2 > 4; README.md has the full table):
//
//	0  PASS: every entry of every blocking corpus measured, no unacknowledged break
//	2  FAIL: an unacknowledged or over-max bucket in a blocking corpus, or an
//	   expected-breaks.yaml entry that matched nothing (a stale acknowledgement)
//	3  INSTRUMENT FAILURE: could not measure (missing corpus, fetch failure, binary
//	   that does not start, failed canary, bad configuration, a write where none may happen)
//	4  INCOMPLETE: a blocking corpus has NOT_MEASURED entries (the baseline reported
//	   unresolved dependencies, a timeout or a crash) beyond its allow_not_measured
//	64 usage: a flag that does not parse
//
// Isolation (design §6.3, enforced here, not by convention): each binary is copied
// into its own empty dir; every AILANG_*/XDG_* variable is dropped; HOME, compile
// cache, state and TMPDIR are per-arm scratch dirs; AILANG_PACKAGE_ROOT is the
// run's audited copy of the operator's package root (pkgroot.go), so a missing
// dependency is an error, never a download, and no other process can change it.
// Corpora are materialised into the run dir (snapshots of the remote ref through
// the operator's `gh`, or a copy of the local checkout in REF=worktree); the
// operator's checkouts are only read, with --no-optional-locks. Before the arms
// run, provision.go gives each corpus the dependencies its locks and manifests
// declare (`ailang lock` in a per-run copy of the snapshot; `ailang install` of
// missing registry versions only under --fetch-deps), so the arms themselves
// never download.
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
	Offline, NoNice, FetchDeps              bool
}

// defaultTimeout is the per-check deadline in seconds (retried once at 2x).
// RULED by Mark 2026-11-10: 600. At 240 s motoko's root package timed out in the
// baseline on a loaded machine and read as 22 false breaks; at 600 both arms agree.
const defaultTimeout = 600

func defaultOptions() Options {
	home, _ := os.UserHomeDir()
	return Options{
		Baseline: "latest", Tier: "release", Ref: refDefault,
		Manifest: "tools/downstream/corpora.yaml", Expected: "tools/downstream/expected-breaks.yaml",
		State:       filepath.Join(home, ".ailang", "state", "downstream"),
		PackageRoot: filepath.Join(home, ".ailang", "cache", "registry"),
		Jobs:        3, Timeout: defaultTimeout,
	}
}

func main() {
	o := defaultOptions()
	// ContinueOnError: flag's own exit status (2) would read as "unacknowledged breaks".
	fs := flag.NewFlagSet("downstream", flag.ContinueOnError)
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
	fs.BoolVar(&o.Offline, "offline", false, "no network: reuse the last snapshot per corpus and a cached baseline (--offline=0|1)")
	fs.BoolVar(&o.NoNice, "no-nice", false, "do not run checks under nice -n 15")
	fs.BoolVar(&o.FetchDeps, "fetch-deps", false, "fetch registry versions missing from the package root (ailang install; network) into the run's copy (--fetch-deps=0|1)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: downstream --candidate <ailang binary> [flags]\n\n"+
			"exit 0 PASS · 2 FAIL (unacknowledged break, or an unused expected-breaks entry) · 3 INSTRUMENT FAILURE ·\n"+
			"4 INCOMPLETE (blocking corpus has NOT_MEASURED entries beyond allow_not_measured) · 64 usage\n\nflags:")
		fs.PrintDefaults()
	}
	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(exitPass)
		}
		os.Exit(exitUsage)
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "downstream: unexpected arguments %q\n", fs.Args())
		os.Exit(exitUsage)
	}
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
		return exitInstrument, fmt.Errorf("CANDIDATE is required")
	}
	if o.Tier != "release" && o.Tier != "ci" {
		return exitInstrument, fmt.Errorf("TIER must be release or ci, got %q", o.Tier)
	}
	if o.Ref != refDefault && o.Ref != refPinned && o.Ref != refWorktree {
		return exitInstrument, fmt.Errorf("REF must be default, pinned or worktree, got %q", o.Ref)
	}
	corpora, err := LoadCorpora(o.Manifest)
	if err != nil {
		return exitInstrument, err
	}
	expected, err := LoadExpected(o.Expected, corpora)
	if err != nil {
		return exitInstrument, err
	}
	selected, err := selectCorpora(corpora, o.Tier, o.Corpora)
	if err != nil {
		return exitInstrument, err
	}
	if o.Out == "" {
		o.Out = filepath.Join(o.State, "runs", started.Format("20060102-150405"))
	}
	if err := freshDir(o.Out); err != nil {
		return exitInstrument, err
	}
	if o.SelfRoot == "" {
		if o.SelfRoot, err = gitOut(".", "rev-parse", "--show-toplevel"); err != nil {
			return exitInstrument, err
		}
	}
	rep := &Report{Started: started, Tier: o.Tier, RefMode: o.Ref, ArmWall: map[string]float64{}}
	fail := func(format string, a ...interface{}) { rep.Failures = append(rep.Failures, fmt.Sprintf(format, a...)) }
	finish := func(rows []Row) (int, error) {
		rep.WallSecs = time.Since(started).Seconds()
		rep.verdict()
		if err := rep.WriteReports(o.Out, rows); err != nil {
			return exitInstrument, err
		}
		fmt.Fprintf(log, "downstream: %s (exit %d) — %s\n", rep.Verdict, rep.ExitCode, filepath.Join(o.Out, "report.md"))
		for _, msg := range rep.Incomplete {
			fmt.Fprintf(log, "downstream: NOT MEASURED %s\n", msg)
		}
		return rep.ExitCode, nil
	}

	// Preflight: every precondition of a meaningful measurement, loudly.
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
	if o.FetchDeps && o.Offline {
		fail("FETCH_DEPS=1 needs the network to reach the package registry; it cannot run with OFFLINE=1")
	}
	if fi, err := os.Stat(o.PackageRoot); err != nil || !fi.IsDir() {
		fail("package root %s is not a directory (it must exist; it is never written): %v", o.PackageRoot, err)
	}
	basePath, baseOrigin, err := ResolveBaseline(o.Baseline, o.State, o.Offline)
	if err != nil {
		fail("baseline: %v", err)
	}
	var base, cand, provArm *Arm
	if err == nil {
		if base, err = PrepareArm("baseline", basePath, baseOrigin, o.Out); err != nil {
			fail("%v", err)
		}
		// The provision arm is the baseline binary in its own scratch HOME: the
		// released `ailang lock` / `ailang install` provision both arms' input.
		if provArm, err = PrepareArm("provision", basePath, baseOrigin, o.Out); err != nil {
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
	pkgRoot, err := copyPackageRoot(o.PackageRoot, filepath.Join(o.Out, "pkgroot"))
	if err != nil {
		fail("%v", err)
		return finish(nil)
	}
	rep.PackageRoot = &pkgRoot
	runner := &Runner{Jobs: o.Jobs, Timeout: o.Timeout, PackageRoot: pkgRoot.Path, Nice: !o.NoNice, RunDir: o.Out}
	if msg := runCanary(runner, base, cand, filepath.Join(o.Out, "canary")); msg != "" {
		fail("canary: %s", msg)
		return finish(nil)
	}

	// Materialise every corpus, then stamp: from here on nothing may write a snapshot.
	mat := &Materialiser{StateDir: o.State, SelfRoot: o.SelfRoot, Offline: o.Offline, Remote: remote}
	type materialised struct {
		c   Corpus
		src Source
		sum CorpusSummary
		err error
	}
	var mats []materialised
	var locals []string
	for _, c := range selected {
		sum := CorpusSummary{Name: c.Name, Level: c.tierOf(o.Tier), States: map[string]int{}, EntryRules: map[string]int{}, NotMeasured: map[string]int{}}
		src, err := mat.Materialise(c, o.Ref, filepath.Join(o.Out, "corpora", c.Name))
		if src.Local != nil && c.Repo != selfRepo {
			// The audit covers sibling checkouts; the ailang checkout itself is
			// where the candidate is being built and edited, so it would only
			// report the operator's own work.
			locals = append(locals, src.Local.Path)
		}
		mats = append(mats, materialised{c, src, sum, err})
	}
	time.Sleep(20 * time.Millisecond)
	snapStamp := time.Now()

	// Provision each corpus's dependencies (provision.go), then discover its
	// entries in the tree the arms will read.
	prov := newProvisioner(provArm, pkgRoot.Path, o.Out, o.FetchDeps, o.Timeout, !o.NoNice)
	rep.Provisioning = prov.Report
	var entries []Entry
	byCorpus := map[string][]Entry{}
	expectations := map[string]expectation{}
	var checked []Corpus
	var trees []string
	for _, m := range mats {
		c, src, sum, err := m.c, m.src, m.sum, m.err
		var tree string
		if err == nil {
			rep.Sources = append(rep.Sources, src)
			if tree, err = prov.Corpus(c, src.Root); err == nil && tree != src.Root {
				trees = append(trees, tree)
			}
		}
		var es []Entry
		if err == nil {
			if es, err = Discover(c, tree); err == nil && len(es) == 0 {
				err = fmt.Errorf("no entries discovered (include/exclude match nothing?)")
			}
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
	provStamp := time.Now()

	// Both arms, one after the other.
	t0 := time.Now()
	baseRes := runner.RunArm(base, entries)
	rep.ArmWall["baseline"] = time.Since(t0).Seconds()
	t0 = time.Now()
	candRes := runner.RunArm(cand, entries)
	rep.ArmWall["candidate"] = time.Since(t0).Seconds()

	// Classify, gate, audit.
	blocking, inRun := map[string]bool{}, map[string]bool{}
	for _, c := range checked {
		inRun[c.Name] = true
		blocking[c.Name] = c.tierOf(o.Tier) == tierBlocking
	}
	rows := rep.score(byCorpus, expectations, baseRes, candRes)
	rep.gateNotMeasured(checked)
	rep.Buckets, rep.Expected, _ = Gate(rows, blocking, inRun, expected, cand.Version)
	for _, s := range rep.Sources {
		auditInto(rep, "snapshot "+s.Corpus, s.Root, snapStamp)
	}
	for _, l := range locals {
		auditInto(rep, "local checkout "+l, l, started)
	}
	// Both arms read the run's copy of the package root and the provisioned
	// trees; once provisioning is done they are never written (a write would be
	// one arm's output feeding the other's input).
	auditInto(rep, "package root copy", pkgRoot.Path, provStamp)
	for _, t := range trees {
		auditInto(rep, "provisioned tree "+filepath.Base(t), t, provStamp)
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
