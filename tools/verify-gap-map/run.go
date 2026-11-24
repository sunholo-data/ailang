package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// row is one verifier result, keyed by corpus, file and function.
type row struct {
	Corpus          string   `json:"corpus"`
	File            string   `json:"file"`
	Function        string   `json:"function"`
	Status          string   `json:"status"` // verify's status; skipped split into skip_mp / skipped
	ReasonCode      string   `json:"reason_code,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	RequiresCallees []string `json:"requires_callees,omitempty"`
	RequiresChecked []string `json:"requires_checked,omitempty"`
	Replayed        bool     `json:"witness_replayed,omitempty"`
}

func (r row) key() string { return r.Corpus + "|" + r.File + "|" + r.Function }

// fileFailure is a file the instrument could not measure (verify crashed, timed
// out, or printed no JSON). Reported on its own, never folded into a status.
type fileFailure struct {
	Corpus string `json:"corpus"`
	File   string `json:"file"`
	Detail string `json:"detail"`
}

type job struct {
	corpus gapCorpus
	file   string
}

type verifyOutput struct {
	Results []struct {
		Function        string   `json:"function"`
		Status          string   `json:"status"`
		ReasonCode      string   `json:"reason_code"`
		Reason          string   `json:"reason"`
		RequiresCallees []string `json:"requires_callees"`
		RequiresChecked []string `json:"requires_checked"`
		WitnessReplayed bool     `json:"witness_replayed"`
	} `json:"results"`
}

// runAll verifies every file with `ailang verify --json --relax-modules`, run
// from the corpus root as its authors run it, jobs at a time.
func runAll(bin string, jobs []job, workers int, timeout time.Duration, progress func(done, total int)) ([]row, []fileFailure) {
	var (
		mu       sync.Mutex
		rows     []row
		failures []fileFailure
		done     int
		wg       sync.WaitGroup
	)
	ch := make(chan job)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				rs, fail := runOne(bin, j, timeout)
				mu.Lock()
				rows = append(rows, rs...)
				if fail != nil {
					failures = append(failures, *fail)
				}
				done++
				if progress != nil {
					progress(done, len(jobs))
				}
				mu.Unlock()
			}
		}()
	}
	for _, j := range jobs {
		ch <- j
	}
	close(ch)
	wg.Wait()
	return rows, failures
}

func runOne(bin string, j job, timeout time.Duration) ([]row, *fileFailure) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "verify", "--json", "--relax-modules", j.file)
	cmd.Dir = j.corpus.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run() // verify exits non-zero on any non-verified result; the JSON is the answer
	fail := func(detail string) *fileFailure {
		return &fileFailure{Corpus: j.corpus.Name, File: j.file, Detail: detail}
	}
	if ctx.Err() != nil {
		return nil, fail(fmt.Sprintf("timed out after %s", timeout))
	}
	var out verifyOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return nil, fail(firstLine(stderr.String(), stdout.String()))
	}
	rows := make([]row, 0, len(out.Results))
	for _, r := range out.Results {
		st := r.Status
		if st == "skipped" && r.ReasonCode == "missing_postcondition" {
			st = "skip_mp"
		}
		rows = append(rows, row{
			Corpus: j.corpus.Name, File: filepath.ToSlash(j.file), Function: r.Function,
			Status: st, ReasonCode: r.ReasonCode, Reason: truncate(r.Reason, 240),
			RequiresCallees: r.RequiresCallees, RequiresChecked: r.RequiresChecked, Replayed: r.WitnessReplayed,
		})
	}
	return rows, nil
}

func firstLine(texts ...string) string {
	for _, t := range texts {
		for _, l := range strings.Split(t, "\n") {
			if l = strings.TrimSpace(l); l != "" {
				return truncate(l, 200)
			}
		}
	}
	return "no output"
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
