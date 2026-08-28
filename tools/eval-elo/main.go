// Command eval-elo derives ELO-style difficulty ratings for benchmarks and a
// capability rating for models from an eval baseline, by treating each trial as
// a game: a PASS is a win for the model against the benchmark. The rating math
// lives in internal/eval_harness (FitFromTrials/Band); this tool loads a baseline,
// renders the result, and optionally persists to observatory.db.
//
// Ratings are MODE-SEPARATED: standard and agent are different difficulty regimes
// (agent saturates), so each mode is fitted and stored independently. --mode all
// processes both.
//
//	go run ./tools/eval-elo <baseline_dir> [--mode standard|agent|all] [--persist <observatory.db>]
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
	"github.com/sunholo-data/ailang/internal/observatory"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	mode := "standard"
	var dir, persist string
	for i := 0; i < len(os.Args[1:]); i++ {
		a := os.Args[1+i]
		switch {
		case a == "--mode" && i+1 < len(os.Args[1:]):
			mode = os.Args[1+i+1]
			i++
		case a == "--persist" && i+1 < len(os.Args[1:]):
			persist = os.Args[1+i+1]
			i++
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
			os.Exit(2)
		default:
			dir = a
		}
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "usage: eval-elo <baseline_dir> [--mode standard|agent|all] [--persist <observatory.db>]")
		os.Exit(2)
	}

	// Bucket trials by the mode segment of their path (standard|agent).
	// Each trial keeps its banked-file identity + metadata so the persist path
	// can append trial_history rows (M-EVAL-ROLLING-ELO M2).
	byMode := map[string][]trialMeta{}
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		sep := string(os.PathSeparator)
		var m string
		switch {
		case strings.Contains(p, sep+"standard"+sep):
			m = "standard"
		case strings.Contains(p, sep+"agent"+sep):
			m = "agent"
		default:
			return nil
		}
		raw, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		var rec map[string]any
		if json.Unmarshal(raw, &rec) != nil {
			return nil
		}
		// Measurement contract: quarantined rows (validity.valid == false) are
		// NOT measurements and must not reach the fit — an infrastructure 520
		// counted as a loss is exactly the sol-vs-terra contamination
		// (measured 2026-08-27). Mirrors eval_analysis.LoadResults' default.
		if v, ok := rec["validity"].(map[string]any); ok {
			if valid, ok := v["valid"].(bool); ok && !valid {
				return nil
			}
		}
		co, ok1 := rec["compile_ok"].(bool)
		ro, ok2 := rec["runtime_ok"].(bool)
		so, ok3 := rec["stdout_ok"].(bool)
		b, _ := rec["id"].(string)
		mod, _ := rec["model"].(string)
		if !(ok1 && ok2 && ok3) || b == "" || mod == "" {
			return nil
		}
		pv, _ := rec["prompt_version"].(string)
		byMode[m] = append(byMode[m], trialMeta{
			Trial:           eval_harness.Trial{Model: mod, Bench: b, Pass: co && ro && so},
			TrialID:         strings.TrimSuffix(filepath.Base(p), ".json"),
			PromptVersion:   pv,
			CompilerVersion: versionFromPath(p, sep),
		})
		return nil
	})

	modes := []string{mode}
	if mode == "all" {
		modes = []string{"standard", "agent"}
	}

	var db *sql.DB
	if persist != "" {
		var err error
		db, err = sql.Open("sqlite3", persist)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %s: %v\n", persist, err)
			os.Exit(1)
		}
		defer func() { _ = db.Close() }()
		if _, err := observatory.MigrateWithVersion(db); err != nil {
			fmt.Fprintf(os.Stderr, "migrate %s: %v\n", persist, err)
			os.Exit(1)
		}
	}

	for _, mo := range modes {
		metas := byMode[mo]
		if len(metas) == 0 {
			fmt.Fprintf(os.Stderr, "no %s trials found in %s\n", mo, dir)
			continue
		}
		renderAndMaybePersist(mo, metas, db, persist)
	}
}

// trialMeta carries a fit trial plus the banked-row identity and provenance
// needed for trial_history (M-EVAL-ROLLING-ELO M2).
type trialMeta struct {
	eval_harness.Trial
	TrialID         string
	PromptVersion   string
	CompilerVersion string
}

// versionFromPath extracts the banked version label: the path component
// immediately preceding the standard/agent mode segment — which is how
// --bank-by-version lays out result dirs (eval_results/.../<vX.Y.Z>/<mode>/...).
// A dir not banked by version yields that dir's own name (e.g. "v0.32.0" for
// baselines/v0.32.0/standard/...), which is the same authority.
func versionFromPath(p, sep string) string {
	parts := strings.Split(p, sep)
	for i, seg := range parts {
		if (seg == "standard" || seg == "agent") && i > 0 {
			return parts[i-1]
		}
	}
	return ""
}

func renderAndMaybePersist(mode string, metas []trialMeta, db *sql.DB, persist string) {
	trials := make([]eval_harness.Trial, len(metas))
	for i, m := range metas {
		trials[i] = m.Trial
	}
	// Placement fit (M-EVAL-ROLLING-ELO M1): standard mode is anchored — the
	// frozen panel pins the scale so ratings are comparable across corpora.
	// Agent-mode difficulty is a different scale with no anchor yet; it stays
	// unanchored (documented limitation, revisit when an agent anchor exists).
	var mRat, bRat map[string]float64
	if mode == "standard" {
		mRat, bRat = eval_harness.FitFromTrialsAnchored(trials, eval_harness.AnchorPanelV1, nil)
	} else {
		mRat, bRat = eval_harness.FitFromTrials(trials)
	}

	bp := map[string][2]int{}
	modelTrials := map[string]int{}
	benchTrials := map[string]int{}
	for _, t := range trials {
		v := bp[t.Bench]
		v[1]++
		if t.Pass {
			v[0]++
		}
		bp[t.Bench] = v
		modelTrials[t.Model]++
		benchTrials[t.Bench]++
	}

	type kv struct {
		k string
		r float64
	}
	benches := make([]kv, 0, len(bRat))
	for b, r := range bRat {
		benches = append(benches, kv{b, r})
	}
	sort.Slice(benches, func(i, j int) bool { return benches[i].r > benches[j].r })

	fmt.Printf("=== ELO difficulty — %s mode, %d trials, %d benchmarks, %d models ===\n", mode, len(trials), len(bRat), len(mRat))
	fmt.Printf("%-32s %6s  %-9s %s\n", "benchmark", "ELO", "band", "pass")
	for _, x := range benches {
		v := bp[x.k]
		fmt.Printf("%-32s %6.0f  %-9s %d/%d\n", x.k, x.r, eval_harness.Band(x.r), v[0], v[1])
	}

	models := make([]kv, 0, len(mRat))
	for m, r := range mRat {
		models = append(models, kv{m, r})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].r > models[j].r })
	fmt.Printf("\nModel capability (%s ELO):\n", mode)
	for _, x := range models {
		fmt.Printf("  %6.0f  %s\n", x.r, x.k)
	}
	fmt.Println()

	if db != nil {
		ctx := context.Background()
		// trial_history (M-EVAL-ROLLING-ELO M2): "before" ratings are the
		// stored values PRIOR to this persist overwriting them (fit seed for
		// entities never stored). INSERT OR IGNORE on the trial_id PK makes a
		// re-persist of the same corpus a no-op by schema constraint.
		prevModel := map[string]float64{}
		prevBench := map[string]float64{}
		if rows, err := observatory.LoadModelRatings(ctx, db, mode); err == nil {
			for _, r := range rows {
				prevModel[r.ModelID] = r.Rating
			}
		}
		if rows, err := observatory.LoadBenchmarkRatings(ctx, db, mode); err == nil {
			for _, r := range rows {
				prevBench[r.BenchmarkID] = r.Rating
			}
		}
		before := func(m map[string]float64, k string) float64 {
			if v, ok := m[k]; ok {
				return v
			}
			return eval_harness.DefaultInitialRating
		}
		now := time.Now().UTC()
		entries := make([]observatory.TrialHistoryEntry, 0, len(metas))
		for _, t := range metas {
			outcome := 0
			if t.Pass {
				outcome = 1
			}
			entries = append(entries, observatory.TrialHistoryEntry{
				TrialID:           t.TrialID,
				BenchmarkID:       t.Bench,
				ModelID:           t.Model,
				Mode:              mode,
				Outcome:           outcome,
				PromptVersion:     t.PromptVersion,
				CompilerVersion:   t.CompilerVersion,
				BenchRatingBefore: before(prevBench, t.Bench),
				ModelRatingBefore: before(prevModel, t.Model),
				BenchRatingAfter:  bRat[t.Bench],
				ModelRatingAfter:  mRat[t.Model],
				RecordedAt:        now,
			})
		}
		if err := observatory.AppendTrialHistory(ctx, db, entries); err != nil {
			fmt.Fprintf(os.Stderr, "append trial_history (%s): %v\n", mode, err)
			os.Exit(1)
		}
		if err := observatory.SaveRatings(ctx, db, mode, mRat, bRat, modelTrials, benchTrials); err != nil {
			fmt.Fprintf(os.Stderr, "persist %s ratings: %v\n", mode, err)
			os.Exit(1)
		}
		fmt.Printf("persisted %d model + %d benchmark %s ratings + %d trial_history rows → %s\n\n", len(mRat), len(bRat), mode, len(entries), persist)
	}
}
