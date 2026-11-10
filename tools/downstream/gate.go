package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Exit codes. The order of precedence is 3 > 2 > 4 > 0: an instrument failure
// voids everything; a confirmed break is reported even when coverage is also
// short; an incomplete measurement is never a pass.
const (
	exitPass       = 0  // every entry of every blocking corpus measured; no unacknowledged break
	exitBreak      = 2  // unacknowledged or over-max bucket, or an expected-breaks entry that matched nothing
	exitInstrument = 3  // could not measure at all (corpus, binary, canary, write audit, config)
	exitIncomplete = 4  // a blocking corpus has NOT_MEASURED entries beyond its reviewed allowance
	exitUsage      = 64 // bad command-line flags
)

// score classifies every entry of every checked corpus and fills the per-corpus
// summaries. It returns all rows; rep.Rows keeps those that are not pass_both.
func (rep *Report) score(byCorpus map[string][]Entry, expectations map[string]expectation, baseRes, candRes map[string]Result) []Row {
	var rows []Row
	for i := range rep.Corpora {
		sum := &rep.Corpora[i]
		for _, e := range byCorpus[sum.Name] {
			b, c := resultPtr(baseRes, e), resultPtr(candRes, e)
			row := classify(e, b, c, expectations[sum.Name])
			rows = append(rows, row)
			sum.States[row.State]++
			sum.EntryRules[e.Rule]++
			if row.NotMeasured != "" {
				sum.NotMeasured[row.NotMeasured]++
			}
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
		rep.Failures = append(rep.Failures, fmt.Sprintf("%d entries have a result from only one arm", n))
	}
	return rows
}

// gateNotMeasured holds each blocking corpus to its allowance (zero unless the
// manifest names one, with a reason). An allowance with nothing left to excuse is
// stale and fails too: it would silently excuse the next coverage loss.
func (rep *Report) gateNotMeasured(corpora []Corpus) {
	allow := map[string]*Allowance{}
	for _, c := range corpora {
		allow[c.Name] = c.AllowNotMeasured
	}
	for i := range rep.Corpora {
		sum := &rep.Corpora[i]
		if sum.NotChecked != "" {
			continue
		}
		n := sum.States[stateNotMeasured]
		a := allow[sum.Name]
		sum.Allowance = a
		switch {
		case sum.Level != tierBlocking:
			if n > 0 {
				sum.AllowanceStatus = "advisory"
			}
		case n == 0 && a != nil:
			sum.AllowanceStatus = "STALE"
			rep.Incomplete = append(rep.Incomplete, fmt.Sprintf(
				"%s: allow_not_measured (max %d) excuses nothing now: every entry was measured; remove it from corpora.yaml", sum.Name, a.Max))
		case n == 0:
		case a == nil || n > a.Max:
			allowed := 0
			if a != nil {
				allowed = a.Max
			}
			sum.AllowanceStatus = "OVER"
			rep.Incomplete = append(rep.Incomplete, fmt.Sprintf(
				"%s (blocking): %d entries NOT MEASURED (%s), allowance %d. The baseline never reached a verdict on them, so "+
					"\"no break\" there is unproven. Fix the cause (deps_unresolved: read the UNPROVISIONED lines of the provisioning "+
					"section — FETCH_DEPS=1 fetches registry versions the package root lacks; timeout: raise --timeout or fix "+
					"the hang; crash: see the rows), or, after review, acknowledge "+
					"them with `allow_not_measured: {max: %d, reason: \"...\"}` on the corpus in corpora.yaml",
				sum.Name, n, reasonList(sum.NotMeasured), allowed, n))
		default:
			sum.AllowanceStatus = "within"
		}
	}
}

func reasonList(m map[string]int) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s %d", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// verdict sets the exit code and verdict from everything recorded so far.
func (rep *Report) verdict() {
	var why []string
	code := exitPass
	for _, b := range rep.Buckets {
		if b.Status == "UNEXPECTED" || b.Status == "OVER_MAX" {
			why, code = append(why, "unacknowledged breaks"), exitBreak
			break
		}
	}
	for _, e := range rep.Expected {
		if e.Status == "UNUSED" {
			why, code = append(why, "an expected-breaks entry matched nothing (remove it)"), exitBreak
			break
		}
	}
	if len(rep.Incomplete) > 0 {
		why = append(why, "blocking corpora have unmeasured entries beyond their allowance")
		if code == exitPass {
			code = exitIncomplete
		}
	}
	switch {
	case len(rep.Failures) > 0:
		rep.ExitCode, rep.Verdict = exitInstrument, "INSTRUMENT FAILURE: not measured"
	case code == exitBreak:
		rep.ExitCode, rep.Verdict = code, "FAIL: "+strings.Join(why, "; ")
	case code == exitIncomplete:
		rep.ExitCode, rep.Verdict = code, "INCOMPLETE: "+strings.Join(why, "; ")
	default:
		rep.ExitCode, rep.Verdict = exitPass, "PASS"
	}
}
