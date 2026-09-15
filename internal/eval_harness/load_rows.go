package eval_harness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadOptions controls LoadRows. The zero value is the contract every
// rate-producing path gets WITHOUT opting in: valid rows only, one row per
// slot, the whole tree.
type LoadOptions struct {
	// IncludeInvalid keeps rows that are not measurements (Validity.Valid ==
	// false — dominantly the api_error backstop in validity.go). OFF by
	// default on purpose: if excluding non-measurements required opting in,
	// the next analysis written in a hurry would count harness crashes as
	// model failures again, which is exactly what eval-analyze did until
	// M-V1-SIMPLIFY-S3 (it had its own loader with no filter at all). Turn it
	// on only where the caller itself accounts for invalid rows out loud
	// (SummarizeRotation's InvalidExcluded) or is inspecting the harness.
	IncludeInvalid bool
	// KeepDuplicates disables the newest-wins collapse per
	// (model, id, lang, seed, mode, trial) slot. A cost tally must keep every
	// file — a re-run that was paid for is still spend — while any pass-rate
	// path must not, or a debug re-run pollutes the denominator.
	KeepDuplicates bool
	// SuiteLayout restricts the walk to exactly what MetricsLogger.Log writes
	// below a suite output dir: <dir>/*.json (legacy, no eval_mode),
	// <dir>/{standard,agent}/*.json and <dir>/{standard,agent}/<condition>/*.json.
	// Suite-level aggregators (cost tally, rotation summary) use it so a root
	// accumulator never pools the per-version bank dirs nested under it — a
	// depth bound cannot tell <dir>/v0.32.0/standard/ from <dir>/standard/z3/,
	// but the mode-directory name can. Off (the default) walks the whole tree,
	// which is what the analysis loaders want for a baseline or --merge dir.
	SuiteLayout bool
}

// suiteModeDirs are the only first-level directories MetricsLogger.Log creates
// under a suite output dir.
var suiteModeDirs = map[string]bool{EvalModeStandard: true, EvalModeAgent: true}

// LoadStats reports what LoadRows saw and what it dropped, so a caller can
// state a shrunken sample instead of silently shrinking it.
type LoadStats struct {
	// Files is the number of result JSON files considered (after the
	// summary.json / baseline.json name skips).
	Files int
	// Loaded is the number of rows returned.
	Loaded int
	// ParseErrors lists every file that could not be read, decoded, or lacked
	// a required field (id, lang, model), as "<basename>: <error>".
	ParseErrors []string
	// Duplicates counts rows dropped by the per-slot newest-wins collapse.
	Duplicates int
	// Invalid counts rows dropped by the validity filter, and InvalidBy breaks
	// that down by Validity.Reason.
	Invalid   int
	InvalidBy map[string]int
}

// resultFileSkips are the JSON files the suite writes NEXT TO result rows and
// that no loader may mistake for one.
var resultFileSkips = map[string]bool{
	"summary.json":  true,
	"baseline.json": true,
}

// LoadRows is THE loader for banked result rows (M-V1-SIMPLIFY-S3 M1). It
// replaced five walkers that each decoded RunMetrics JSON on their own —
// eval_analysis/loader.go (the only one with the validity filter),
// eval_analyzer/analyzer.go (no filter: harness errors counted as model
// failures), eval_harness/cost_tally.go, eval_harness/rotation_summary.go and
// cmd/ailang/eval_skip_existing.go — and every consumer now goes through here,
// so "what counts as a result row" has one definition.
//
// Every dir must exist (a missing dir is an error, not an empty result). A
// directory with no result files is NOT an error here: callers that need one
// (the analysis loader, eval-analyze) say so themselves with their own message.
// Rows come back newest-first. Files that fail to decode are reported in
// LoadStats.ParseErrors and skipped — a cost report or a rotation summary must
// never be the thing that fails a completed run — and it is again the caller's
// decision whether "some files failed AND nothing loaded" is fatal.
func LoadRows(dirs []string, opts LoadOptions) ([]RunMetrics, LoadStats, error) {
	if len(dirs) == 0 {
		return nil, LoadStats{}, fmt.Errorf("no directories provided")
	}
	var files []string
	seenPath := make(map[string]struct{})
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			return nil, LoadStats{}, fmt.Errorf("directory not found: %s", dir)
		}
		root := filepath.Clean(dir)
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip inaccessible entries
			}
			if d.IsDir() {
				if opts.SuiteLayout && path != root {
					rel, relErr := filepath.Rel(root, path)
					if relErr != nil {
						return filepath.SkipDir
					}
					parts := strings.Split(rel, string(filepath.Separator))
					// first level must be a mode dir; second level (a condition)
					// is the deepest the layout goes.
					if !suiteModeDirs[parts[0]] || len(parts) > 2 {
						return filepath.SkipDir
					}
				}
				return nil
			}
			if filepath.Ext(path) != ".json" || resultFileSkips[filepath.Base(path)] {
				return nil
			}
			// A --merge dir nested under the primary, or the same dir passed
			// twice, must load a file at most once.
			abs, absErr := filepath.Abs(path)
			if absErr != nil {
				abs = path
			}
			if _, dup := seenPath[abs]; dup {
				return nil
			}
			seenPath[abs] = struct{}{}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, LoadStats{}, fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}
	return LoadRowFiles(files, opts)
}

// LoadRowFiles is LoadRows over an explicit file list — the entry point for a
// caller that already resolved a glob (--skip-existing). Same decode, ordering,
// dedup and validity contract as LoadRows.
func LoadRowFiles(files []string, opts LoadOptions) ([]RunMetrics, LoadStats, error) {
	stats := LoadStats{Files: len(files), InvalidBy: map[string]int{}}
	rows := make([]RunMetrics, 0, len(files))
	for _, path := range files {
		m, err := loadRowFile(path)
		if err != nil {
			stats.ParseErrors = append(stats.ParseErrors, fmt.Sprintf("%s: %v", filepath.Base(path), err))
			continue
		}
		rows = append(rows, m)
	}

	// Newest first; a stable sort with the id as tie-break keeps the order —
	// and therefore which duplicate survives — deterministic across platforms.
	sort.SliceStable(rows, func(i, j int) bool {
		if !rows[i].Timestamp.Equal(rows[j].Timestamp) {
			return rows[i].Timestamp.After(rows[j].Timestamp)
		}
		return rows[i].ID < rows[j].ID
	})

	if !opts.KeepDuplicates {
		// Keep only the latest row per (model, id, lang, seed, mode, trial).
		// Mode MUST be in the key: a model run in BOTH standard and agent
		// suites has a legitimately distinct result per mode for the same
		// (benchmark, lang, seed); without it one is silently dropped and the
		// model vanishes from that mode's leaderboard. Trial MUST be in the
		// key: both trials share the rest, so omitting it made an 84-row arm
		// load as 42 and every multi-trial rate was computed from half the
		// data (2026-07-29). Legacy rows carry Trial=0 and behave as before.
		type slot struct {
			Model, ID, Lang string
			Seed            int64
			Mode            string
			Trial           int
		}
		seen := make(map[slot]struct{}, len(rows))
		kept := rows[:0]
		for _, r := range rows {
			mode := EvalModeStandard // empty/legacy eval_mode == standard
			if r.EvalMode == EvalModeAgent {
				mode = EvalModeAgent
			}
			k := slot{r.Model, r.ID, r.Lang, r.Seed, mode, r.Trial}
			if _, dup := seen[k]; dup {
				stats.Duplicates++
				continue
			}
			seen[k] = struct{}{}
			kept = append(kept, r)
		}
		rows = kept
	}

	if !opts.IncludeInvalid {
		kept := rows[:0]
		for _, r := range rows {
			if !r.IsValid() {
				stats.Invalid++
				stats.InvalidBy[r.InvalidReason()]++
				continue
			}
			kept = append(kept, r)
		}
		rows = kept
	}
	stats.Loaded = len(rows)
	return rows, stats, nil
}

// loadRowFile decodes one banked row. A row without id, lang and model is not
// a result (it is a manifest, a summary, or a truncated write) and is rejected
// rather than counted as a phantom run.
func loadRowFile(path string) (RunMetrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunMetrics{}, fmt.Errorf("failed to read file: %w", err)
	}
	var m RunMetrics
	if err := json.Unmarshal(data, &m); err != nil {
		return RunMetrics{}, fmt.Errorf("failed to parse JSON: %w", err)
	}
	switch {
	case m.ID == "":
		return RunMetrics{}, fmt.Errorf("missing required field: id")
	case m.Lang == "":
		return RunMetrics{}, fmt.Errorf("missing required field: lang")
	case m.Model == "":
		return RunMetrics{}, fmt.Errorf("missing required field: model")
	}
	return m, nil
}
