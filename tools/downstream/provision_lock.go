package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The candidate's own `ailang lock` (review of #2220, must-fix 1).
//
// A corpus that commits no lock (daneel gitignores ailang.lock and its CI runs
// `ailang lock` with the deployed binary, failing hard) depends on the
// candidate's resolver and lock writer, not only its checker. So wherever the
// gate generates a lock, the candidate runs `ailang lock` too, in its own copy
// of the snapshot, and the two resolved sets (name, version, source) are
// compared. Each such dir becomes an entry of kind "lock":
//
//   - baseline and candidate resolve the same set: pass_both;
//   - the candidate's lock fails where the baseline's succeeded: NEWLY_FAILING
//     with LOCK/failed (a break: exit 2 unless acknowledged);
//   - both succeed with different sets: NEWLY_FAILING with
//     LOCK/resolution-differs. It gates: both arms' checks read the BASELINE's
//     lock, so a different candidate tree is a change the checks never saw;
//   - the baseline's lock fails: DEP/lock — the dir is NOT_MEASURED (or
//     newly_passing when the candidate's lock succeeds).

const (
	codeLockDep     = "DEP/lock"                // the baseline could not lock: not measured
	codeLockFailed  = "LOCK/failed"             // the candidate could not lock: a break
	codeLockDiffers = "LOCK/resolution-differs" // the candidate resolved another set: a break
)

// LockCheck is one generated lock, scored like any entry.
type LockCheck struct {
	Entry
	Base, Cand Result
}

// lockSet is a lock's resolved set as "name@version (source)".
func lockSet(dir string) ([]string, error) {
	lk, err := readLock(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range lk.Packages {
		out = append(out, fmt.Sprintf("%s@%s (%s)", e.Name, e.Version, e.Source))
	}
	sort.Strings(out)
	return out, nil
}

// setDiff lists what only a has, and what only b has.
func setDiff(a, b []string) (onlyA, onlyB []string) {
	in := func(xs []string) map[string]bool {
		m := map[string]bool{}
		for _, x := range xs {
			m[x] = true
		}
		return m
	}
	ma, mb := in(a), in(b)
	for _, x := range a {
		if !mb[x] {
			onlyA = append(onlyA, x)
		}
	}
	for _, x := range b {
		if !ma[x] {
			onlyB = append(onlyB, x)
		}
	}
	return
}

// candidateLocks runs the candidate's `ailang lock` in its own copy of the
// snapshot for every dir the baseline locked, and records one LockCheck each.
// An error is an instrument failure.
func (p *Provisioner) candidateLocks(c Corpus, snap, baseTree string, plans []lockPlan, cp *CorpusProvision) error {
	if p.CandArm == nil || len(plans) == 0 {
		return nil
	}
	tree := filepath.Join(p.RunDir, "provisioned-candidate", c.Name)
	if _, _, _, err := copyRegular(snap, tree, func(string) bool { return false }); err != nil {
		return fmt.Errorf("copy %s for the candidate's lock: %w", c.Name, err)
	}
	for i, pl := range plans {
		dir := filepath.Join(tree, pl.dir)
		if pl.unreadable {
			if err := os.Remove(filepath.Join(dir, lockFile)); err != nil {
				return err
			}
		}
		out, err := p.run(p.CandArm, dir, false, "lock")
		if err == errProvisionBudget {
			return err
		}
		lc := LockCheck{Entry: Entry{Corpus: c.Name, Kind: "lock", Rel: filepath.ToSlash(filepath.Join(pl.dir, lockFile)),
			Cwd: filepath.Join(baseTree, pl.dir), Arg: "lock", Rule: "lock"}}
		baseSet, berr := lockSet(filepath.Join(baseTree, pl.dir))
		if pl.failed != "" || berr != nil {
			lc.Base = Result{Codes: []string{codeLockDep}, RC: 1, Msg: firstLine("baseline `ailang lock`: " + pl.failed)}
		} else {
			lc.Base = Result{Passed: true}
		}
		var candSet []string
		if err == nil {
			candSet, err = lockSet(dir)
		}
		switch {
		case err != nil:
			code := codeLockFailed
			if !lc.Base.Passed {
				code = codeLockDep // both arms failed: still not measured, not a new break
			}
			msg := lastErrorLine(out, err)
			lc.Cand = Result{Codes: []string{code}, RC: 1, Msg: firstLine("candidate `ailang lock`: " + msg)}
			cp.Locks[plans[i].action].Candidate = "FAILED: " + msg
		case !lc.Base.Passed:
			lc.Cand = Result{Passed: true}
			cp.Locks[plans[i].action].Candidate = "succeeded where the baseline's lock failed"
		default:
			onlyB, onlyC := setDiff(baseSet, candSet)
			if len(onlyB)+len(onlyC) == 0 {
				lc.Cand = Result{Passed: true}
				cp.Locks[plans[i].action].Candidate = "same"
				break
			}
			msg := fmt.Sprintf("candidate resolved %s; baseline resolved %s (both arms were checked against the baseline's lock)",
				orNone(onlyC), orNone(onlyB))
			lc.Cand = Result{Codes: []string{codeLockDiffers}, RC: 1, Msg: firstLine(msg), Message: msg, Diags: []string{codeLockDiffers + " " + msg}}
			cp.Locks[plans[i].action].Candidate = "DIFFERS: " + msg
		}
		p.Checks = append(p.Checks, lc)
	}
	return nil
}

func orNone(xs []string) string {
	if len(xs) == 0 {
		return "nothing else"
	}
	return strings.Join(xs, ", ")
}
