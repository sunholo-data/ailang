// Command eval-elo derives ELO-style difficulty ratings for benchmarks and a
// capability rating for models from an eval baseline, by treating each trial as
// a game: a PASS is a win for the model against the benchmark. The rating math
// lives in internal/eval_harness (FitFromTrials/Band); this tool just loads a
// baseline and renders the result (M-EVAL-RATING-EFFICIENCY).
//
//	go run ./tools/eval-elo <baseline_dir> [--mode standard|agent|all]
//
// Difficulty bands (ELO): <1300 Trivial, 1300-1500 Easy, 1500-1700 Moderate,
// 1700-1900 Hard, >1900 Very hard.
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

	var trials []eval_harness.Trial
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		if mode != "all" && !strings.Contains(p, string(os.PathSeparator)+mode+string(os.PathSeparator)) {
			return nil
		}
		raw, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			return nil
		}
		co, ok1 := m["compile_ok"].(bool)
		ro, ok2 := m["runtime_ok"].(bool)
		so, ok3 := m["stdout_ok"].(bool)
		b, _ := m["id"].(string)
		mod, _ := m["model"].(string)
		if !(ok1 && ok2 && ok3) || b == "" || mod == "" {
			return nil
		}
		trials = append(trials, eval_harness.Trial{Model: mod, Bench: b, Pass: co && ro && so})
		return nil
	})
	if len(trials) == 0 {
		fmt.Fprintf(os.Stderr, "no %s trials found in %s\n", mode, dir)
		os.Exit(1)
	}

	mRat, bRat := eval_harness.FitFromTrials(trials)

	// pass-rate per benchmark for context
	bp := map[string][2]int{}
	for _, t := range trials {
		v := bp[t.Bench]
		v[1]++
		if t.Pass {
			v[0]++
		}
		bp[t.Bench] = v
	}

	type kv struct {
		k string
		r float64
	}
	benches := make([]kv, 0, len(bRat))
	for b, r := range bRat {
		benches = append(benches, kv{b, r})
	}
	sort.Slice(benches, func(i, j int) bool { return benches[i].r > benches[j].r }) // hardest first

	fmt.Printf("ELO difficulty — %s mode, %d trials, %d benchmarks, %d models\n\n", mode, len(trials), len(bRat), len(mRat))
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
	fmt.Printf("\nModel capability (ELO):\n")
	for _, x := range models {
		fmt.Printf("  %6.0f  %s\n", x.r, x.k)
	}

	if persist != "" {
		modelTrials := map[string]int{}
		benchTrials := map[string]int{}
		for _, t := range trials {
			modelTrials[t.Model]++
			benchTrials[t.Bench]++
		}
		db, err := sql.Open("sqlite3", persist)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open %s: %v\n", persist, err)
			os.Exit(1)
		}
		defer func() { _ = db.Close() }()
		if _, err := observatory.MigrateWithVersion(db); err != nil {
			fmt.Fprintf(os.Stderr, "migrate %s: %v\n", persist, err)
			os.Exit(1)
		}
		if err := observatory.SaveRatings(context.Background(), db, mRat, bRat, modelTrials, benchTrials); err != nil {
			fmt.Fprintf(os.Stderr, "persist ratings: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\npersisted %d model + %d benchmark ratings → %s\n", len(mRat), len(bRat), persist)
	}
}
