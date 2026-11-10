package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Entry states (design doc §6.5). Only NEWLY_FAILING, fail_changed (its new codes)
// and timeout_cand count toward the gate, and only in blocking corpora.
const (
	statePassBoth     = "pass_both"
	stateNewlyFailing = "NEWLY_FAILING"
	stateNewlyPassing = "newly_passing"
	stateFailSame     = "fail_same"
	stateFailChanged  = "fail_changed"
	stateExpectedFail = "expected_fail"
	stateTimeoutCand  = "timeout_cand"
	stateTimeoutBase  = "timeout_base"
	stateMissingArm   = "MISSING_ARM"
)

// Row is one entry's outcome across both arms.
type Row struct {
	Entry
	State    string   `json:"state"`
	Base     *Result  `json:"baseline"`
	Cand     *Result  `json:"candidate"`
	NewCodes []string `json:"new_codes,omitempty"` // the codes that count toward buckets
}

// expectation decides whether a candidate failure is an intended one.
type expectation struct {
	globs    globSet
	patterns map[string]*regexp.Regexp // corpus-relative path -> error_pattern (nil = any)
}

// loadExpectation reads a corpus's expected_fail globs and its examples manifest
// (entries with a non-zero expected.exit_code; paths are relative to the manifest's dir).
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
		if e.Expected == nil || e.Expected.ExitCode == 0 {
			continue
		}
		var re *regexp.Regexp
		if e.Expected.ErrorPattern != "" {
			if re, err = regexp.Compile(e.Expected.ErrorPattern); err != nil {
				re = regexp.MustCompile(regexp.QuoteMeta(e.Expected.ErrorPattern))
			}
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
	if !ok {
		return false
	}
	return re == nil || re.MatchString(cand.Message)
}

func isTimeout(r *Result) bool { return len(r.Codes) == 1 && r.Codes[0] == codeTimeout }

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
	case isTimeout(c) && !isTimeout(b):
		row.State, row.NewCodes = stateTimeoutCand, []string{codeTimeout}
	case isTimeout(b) && !isTimeout(c):
		row.State = stateTimeoutBase
	case b.Passed:
		row.State, row.NewCodes = stateNewlyFailing, c.Codes
	case c.Passed:
		row.State = stateNewlyPassing
	case strings.Join(b.Codes, ",") == strings.Join(c.Codes, ","):
		row.State = stateFailSame
	default:
		row.State = stateFailChanged
		had := map[string]bool{}
		for _, x := range b.Codes {
			had[x] = true
		}
		for _, x := range c.Codes {
			if !had[x] {
				row.NewCodes = append(row.NewCodes, x)
			}
		}
	}
	return row
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
