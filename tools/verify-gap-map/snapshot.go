package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// snapshot is one banked measurement.
type snapshot struct {
	Taken    string            `json:"taken"` // RFC 3339
	Ailang   string            `json:"ailang"`
	Corpora  []corpusInfo      `json:"corpora"`
	Totals   map[string]int    `json:"totals"` // status → count
	D1       d1Radius          `json:"d1"`
	Rows     []row             `json:"rows"`
	Files    map[string]string `json:"files"` // corpus|file → content hash
	Failures []fileFailure     `json:"failures,omitempty"`
}

type corpusInfo struct {
	Name  string `json:"name"`
	Root  string `json:"root,omitempty"`
	Head  string `json:"head,omitempty"`
	Dirty bool   `json:"dirty,omitempty"`
	Files int    `json:"files"`
	Rows  int    `json:"rows"`
	Note  string `json:"note,omitempty"` // why it was not measured
}

// d1Radius is the population decision D1 could change: results whose function
// reaches a callee with a requires the verifier assumes rather than checks.
type d1Radius struct {
	Results  int            `json:"results"`
	ByStatus map[string]int `json:"by_status"`
}

func totals(rows []row) map[string]int {
	t := map[string]int{}
	for _, r := range rows {
		t[r.Status]++
	}
	return t
}

func d1(rows []row) d1Radius {
	d := d1Radius{ByStatus: map[string]int{}}
	for _, r := range rows {
		if len(r.RequiresCallees) > 0 {
			d.Results++
			d.ByStatus[r.Status]++
		}
	}
	return d
}

var (
	quoted   = regexp.MustCompile(`"[^"]*"`)
	digits   = regexp.MustCompile(`\d+`)
	calleeRe = regexp.MustCompile(`calls "([^"]+)" whose signature uses an unencodable type "([^"]+)"`)
	applyRe  = regexp.MustCompile(`unsupported application ([A-Za-z0-9_$./]+)`)
)

// normaliseReason folds names and numbers out of a reason so that the same
// limitation hit by different functions lands in one bucket.
func normaliseReason(reason string) string {
	s := quoted.ReplaceAllString(reason, `"_"`)
	s = digits.ReplaceAllString(s, "N")
	if i := strings.Index(s, "\n"); i >= 0 {
		s = s[:i]
	}
	return truncate(s, 140)
}

// attribution names what a skip or error is blamed on: an unencodable callee
// signature (callee, type) or an unsupported builtin application.
type attribution struct {
	Callees  map[string]int
	Types    map[string]int
	Builtins map[string]int
}

func attribute(rows []row) attribution {
	a := attribution{Callees: map[string]int{}, Types: map[string]int{}, Builtins: map[string]int{}}
	for _, r := range rows {
		if r.Status != "skipped" && r.Status != "error" {
			continue
		}
		if m := calleeRe.FindStringSubmatch(r.Reason); m != nil {
			a.Callees[m[1]]++
			a.Types[m[2]]++
		}
		if m := applyRe.FindStringSubmatch(r.Reason); m != nil {
			a.Builtins[m[1]]++
		}
	}
	return a
}

// reasonBuckets counts normalised reasons for rows with the given statuses.
func reasonBuckets(rows []row, statuses ...string) map[string]int {
	want := map[string]bool{}
	for _, s := range statuses {
		want[s] = true
	}
	b := map[string]int{}
	for _, r := range rows {
		if want[r.Status] {
			b[normaliseReason(r.Reason)]++
		}
	}
	return b
}

type kv struct {
	Key   string
	Count int
}

// top returns the n largest entries, ties broken by key.
func top(m map[string]int, n int) []kv {
	out := make([]kv, 0, len(m))
	for k, v := range m {
		out = append(out, kv{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// transition is one result whose status changed between snapshots.
type transition struct {
	Key, From, To string
	SourceChanged bool // the file's content differs: not attributable to the verifier
}

// regression reports whether a transition in unchanged source makes the
// verifier's answer worse: a proof lost, or a new error/unknown.
func (t transition) regression() bool {
	if t.SourceChanged {
		return false
	}
	return t.From == "verified" || t.To == "error" || t.To == "unknown"
}

// diff compares two snapshots result by result.
func diff(prev, cur *snapshot) (trans []transition, added, removed int) {
	before := map[string]row{}
	for _, r := range prev.Rows {
		before[r.key()] = r
	}
	seen := map[string]bool{}
	for _, r := range cur.Rows {
		k := r.key()
		seen[k] = true
		p, ok := before[k]
		if !ok {
			added++
			continue
		}
		if p.Status != r.Status {
			fk := r.Corpus + "|" + r.File
			trans = append(trans, transition{Key: k, From: p.Status, To: r.Status, SourceChanged: prev.Files[fk] != cur.Files[fk]})
		}
	}
	for k := range before {
		if !seen[k] {
			removed++
		}
	}
	sort.Slice(trans, func(i, j int) bool { return trans[i].Key < trans[j].Key })
	return trans, added, removed
}

// latestSnapshot returns the newest banked snapshot in dir, or nil.
func latestSnapshot(dir string) (*snapshot, string, error) {
	names, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(names) == 0 {
		return nil, "", nil
	}
	sort.Strings(names)
	return readSnapshot(names[len(names)-1])
}

func readSnapshot(path string) (*snapshot, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var s snapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, "", err
	}
	return &s, path, nil
}
