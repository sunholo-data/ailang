package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Entry states (design doc §6.5). New codes count toward the gate's buckets from
// NEWLY_FAILING, fail_changed, timeout_cand and NOT_MEASURED rows, and only in
// blocking corpora. NOT_MEASURED rows are also counted on their own (see gate.go).
const (
	statePassBoth        = "pass_both"
	stateNewlyFailing    = "NEWLY_FAILING"
	stateNewlyPassing    = "newly_passing"
	stateFailSame        = "fail_same"
	stateFailDiagChanged = "fail_diag_changed" // same code set, different errors: reported, not gated
	stateFailChanged     = "fail_changed"
	stateExpectedFail    = "expected_fail"
	stateTimeoutCand     = "timeout_cand"
	stateNotMeasured     = "NOT_MEASURED" // the baseline never really checked it (deps, timeout, crash)
	stateMissingArm      = "MISSING_ARM"
)

// Reasons a baseline result is an infrastructure outcome rather than a verdict.
const (
	reasonDeps     = "deps_unresolved"
	reasonTimeout  = "timeout"
	reasonCrash    = "crash"
	reasonManifest = "invalid_manifest"  // the entry's own ailang.toml did not load
	reasonExport   = "unexported_import" // it imports a module a package does not export

)

// Row is one entry's outcome across both arms.
type Row struct {
	Entry
	State       string   `json:"state"`
	Base        *Result  `json:"baseline"`
	Cand        *Result  `json:"candidate"`
	NewCodes    []string `json:"new_codes,omitempty"`           // the codes that count toward buckets
	NotMeasured string   `json:"not_measured_reason,omitempty"` // why the baseline result is not a verdict
}

// expectation decides whether a candidate failure is an intended one.
type expectation struct {
	globs    globSet
	patterns map[string]*regexp.Regexp // corpus-relative path -> error_pattern (never nil)
}

// loadExpectation reads a corpus's expected_fail globs and its examples manifest.
//
// The manifest's expected.exit_code is the exit code of RUNNING the example
// (examples/runnable/exit_code.ail exits 42 and type-checks fine), so it cannot
// on its own say "this file fails `ailang check`". Only an entry with a non-zero
// exit_code AND an error_pattern is a check-time expectation, and only a failure
// whose message matches that pattern is excused. An error_pattern that does not
// compile is a configuration error. Paths are relative to the manifest's dir.
func loadExpectation(c Corpus, root string) (expectation, error) {
	ex := expectation{globs: newGlobSet(c.ExpectedFail), patterns: map[string]*regexp.Regexp{}}
	if c.ExamplesManifest == "" {
		return ex, nil
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.ExamplesManifest)))
	if err != nil {
		return ex, err
	}
	var m struct {
		Examples []struct {
			Path     string `json:"path"`
			Expected *struct {
				ExitCode     int    `json:"exit_code"`
				ErrorPattern string `json:"error_pattern"`
			} `json:"expected"`
		} `json:"examples"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return ex, err
	}
	base := filepath.ToSlash(filepath.Dir(c.ExamplesManifest))
	for _, e := range m.Examples {
		if e.Expected == nil || e.Expected.ExitCode == 0 || e.Expected.ErrorPattern == "" {
			continue // no check-time expectation: a run-time exit code excuses nothing
		}
		re, err := regexp.Compile(e.Expected.ErrorPattern)
		if err != nil {
			return ex, fmt.Errorf("%s: %s: error_pattern %q is not a valid regexp: %w", c.ExamplesManifest, e.Path, e.Expected.ErrorPattern, err)
		}
		ex.patterns[strings.TrimPrefix(base+"/"+e.Path, "./")] = re
	}
	return ex, nil
}

func (ex expectation) expectsFailure(rel string, cand *Result) bool {
	if ex.globs.match(rel) {
		return true
	}
	re, ok := ex.patterns[rel]
	return ok && re.MatchString(cand.Message)
}

func isTimeout(r *Result) bool { return len(r.Codes) == 1 && r.Codes[0] == codeTimeout }

// infraReason says why a failed result is an infrastructure outcome — the
// check never reached a verdict on the code — or "" when it is a real verdict.
// Every code must be infrastructure: a dependency failure beside a type error
// means the file was (partly) checked.
func infraReason(r *Result) string {
	if r.Passed {
		return ""
	}
	if len(r.Codes) == 0 {
		return reasonCrash
	}
	reason := ""
	for _, code := range r.Codes {
		var this string
		switch {
		case isDepCode(code):
			this = reasonDeps
		case code == codeUncheckedManifest:
			this = reasonManifest
		case code == codeUncheckedExport:
			this = reasonExport
		case code == codeTimeout:
			this = reasonTimeout
		case strings.HasPrefix(code, codeCrash), code == codeNoDiagnostic:
			this = reasonCrash
		default:
			return ""
		}
		if reason == "" {
			reason = this
		}
	}
	return reason
}

// environmental codes say nothing about the candidate's verdict on the code when
// the baseline did not reach one either; a crash or a real error does.
func environmental(code string) bool {
	return isDepCode(code) || code == codeTimeout || strings.HasPrefix(code, uncheckedPrefix)
}

// classify assigns a state to one entry.
func classify(e Entry, b, c *Result, ex expectation) Row {
	row := Row{Entry: e, Base: b, Cand: c}
	switch {
	case b == nil || c == nil:
		row.State = stateMissingArm
	case !c.Passed && ex.expectsFailure(e.Rel, c):
		row.State = stateExpectedFail
	case b.Passed && c.Passed:
		row.State = statePassBoth
	case infraReason(b) != "" && c.Passed:
		row.State = stateNewlyPassing
	case infraReason(b) != "":
		// The baseline never measured this entry, so "unchanged" would be a guess.
		// Whatever real error the candidate reports is new.
		row.State, row.NotMeasured = stateNotMeasured, infraReason(b)
		row.NewCodes = newCodes(b, c, environmental)
	case isTimeout(c):
		row.State, row.NewCodes = stateTimeoutCand, []string{codeTimeout}
	case b.Passed:
		row.State, row.NewCodes = stateNewlyFailing, c.Codes
	case c.Passed:
		row.State = stateNewlyPassing
	case strings.Join(b.Codes, ",") == strings.Join(c.Codes, ","):
		row.State = stateFailSame
		if strings.Join(b.Diags, "\x00") != strings.Join(c.Diags, "\x00") {
			row.State = stateFailDiagChanged
		}
	default:
		row.State = stateFailChanged
		row.NewCodes = newCodes(b, c, nil)
	}
	return row
}

// newCodes are the candidate's codes the baseline did not report, minus skip.
func newCodes(b, c *Result, skip func(string) bool) []string {
	had := map[string]bool{}
	for _, x := range b.Codes {
		had[x] = true
	}
	var out []string
	for _, x := range c.Codes {
		if !had[x] && (skip == nil || !skip(x)) {
			out = append(out, x)
		}
	}
	return out
}

// Bucket is the gate's unit: entries of one corpus newly failing with one code.
type Bucket struct {
	Corpus   string   `json:"corpus"`
	Code     string   `json:"code"`
	Count    int      `json:"count"`
	Blocking bool     `json:"blocking"`
	Max      int      `json:"acknowledged_max"` // sum of matching expected-breaks entries
	Status   string   `json:"status"`           // unexpected | acknowledged | over_max | advisory
	Examples []string `json:"examples"`
	Changes  []string `json:"acknowledged_by,omitempty"`
}

// ExpectedStatus reports what happened to each expected-breaks entry.
type ExpectedStatus struct {
	ExpectedBreak
	Status string `json:"status"` // used | UNUSED | not_in_run
	Count  int    `json:"count"`
	Stale  bool   `json:"stale"` // the candidate is newer than expires
}

// Gate groups counted rows into buckets and subtracts the expected-breaks file.
// It returns the buckets, every entry's status, and whether the gate fails.
func Gate(rows []Row, blocking map[string]bool, inRun map[string]bool, expected []ExpectedBreak, candVersion string) ([]Bucket, []ExpectedStatus, bool) {
	idx := map[string]*Bucket{}
	var order []string
	for _, r := range rows {
		for _, code := range r.NewCodes {
			k := r.Corpus + "\x00" + code
			b, ok := idx[k]
			if !ok {
				b = &Bucket{Corpus: r.Corpus, Code: code, Blocking: blocking[r.Corpus]}
				idx[k] = b
				order = append(order, k)
			}
			b.Count++
			if len(b.Examples) < 3 {
				b.Examples = append(b.Examples, r.Rel+" :: "+r.Cand.Msg)
			}
		}
	}
	statuses := make([]ExpectedStatus, len(expected))
	for i, e := range expected {
		statuses[i] = ExpectedStatus{ExpectedBreak: e, Status: "not_in_run", Stale: versionAfter(candVersion, e.Expires)}
		if !inRun[e.Corpus] {
			continue
		}
		statuses[i].Status = "UNUSED"
		if b, ok := idx[e.Corpus+"\x00"+e.Code]; ok {
			b.Max += e.Max
			b.Changes = append(b.Changes, e.Change)
			statuses[i].Status, statuses[i].Count = "used", b.Count
		}
	}
	fail := false
	var out []Bucket
	sort.Strings(order)
	for _, k := range order {
		b := idx[k]
		switch {
		case !b.Blocking:
			b.Status = "advisory"
		case b.Max == 0:
			b.Status, fail = "UNEXPECTED", true
		case b.Count > b.Max:
			b.Status, fail = "OVER_MAX", true
		default:
			b.Status = "acknowledged"
		}
		out = append(out, *b)
	}
	return out, statuses, fail
}

var semverRE = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)

// versionAfter reports whether version is strictly newer than expires (vX.Y.Z).
// An unparseable side yields false: staleness is advisory, never gating.
func versionAfter(version, expires string) bool {
	a, b := semverRE.FindStringSubmatch(version), semverRE.FindStringSubmatch(expires)
	if a == nil || b == nil {
		return false
	}
	for i := 1; i <= 3; i++ {
		x, _ := strconv.Atoi(a[i])
		y, _ := strconv.Atoi(b[i])
		if x != y {
			return x > y
		}
	}
	return false
}
