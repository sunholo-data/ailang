// Command eval-elo derives ELO-style difficulty ratings for benchmarks and a
// capability rating for models from an eval baseline, by treating each trial as
// a game: a PASS is a win for the model against the benchmark. Ratings are fit by
// iterating the standard ELO update to convergence over the static result set
// (M-EVAL-RATING-EFFICIENCY, part 1). This surfaces which benchmarks are
// genuinely hard vs saturated — the input the frontier-tier demotion needs.
//
//	go run ./tools/eval-elo <baseline_dir> [--mode standard|agent|all]
//
// Difficulty bands (ELO): <1300 Trivial, 1300-1500 Easy, 1500-1700 Moderate,
// 1700-1900 Hard, >1900 Very hard.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type trial struct {
	model, bench string
	pass         bool
}

func main() {
	mode := "standard"
	var dir string
	for i := 0; i < len(os.Args[1:]); i++ {
		a := os.Args[1+i]
		switch {
		case a == "--mode" && i+1 < len(os.Args[1:]):
			mode = os.Args[1+i+1]
			i++
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
			os.Exit(2)
		default:
			dir = a
		}
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "usage: eval-elo <baseline_dir> [--mode standard|agent|all]")
		os.Exit(2)
	}

	var trials []trial
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
		trials = append(trials, trial{model: mod, bench: b, pass: co && ro && so})
		return nil
	})
	if len(trials) == 0 {
		fmt.Fprintf(os.Stderr, "no %s trials found in %s\n", mode, dir)
		os.Exit(1)
	}

	mRat := map[string]float64{}
	bRat := map[string]float64{}
	for _, t := range trials {
		if _, ok := mRat[t.model]; !ok {
			mRat[t.model] = 1500
		}
		if _, ok := bRat[t.bench]; !ok {
			bRat[t.bench] = 1500
		}
	}
	// Deterministic order; many epochs with decaying K converge the static fit.
	sort.Slice(trials, func(i, j int) bool {
		if trials[i].bench != trials[j].bench {
			return trials[i].bench < trials[j].bench
		}
		return trials[i].model < trials[j].model
	})
	const epochs = 400
	for e := 0; e < epochs; e++ {
		k := 4 + 28*math.Exp(-float64(e)/80.0) // 32 -> ~4
		for _, t := range trials {
			exp := 1.0 / (1.0 + math.Pow(10, (bRat[t.bench]-mRat[t.model])/400.0))
			act := 0.0
			if t.pass {
				act = 1.0
			}
			mRat[t.model] += k * (act - exp)
			bRat[t.bench] += k * (exp - act)
		}
	}

	band := func(r float64) string {
		switch {
		case r < 1300:
			return "Trivial"
		case r < 1500:
			return "Easy"
		case r < 1700:
			return "Moderate"
		case r < 1900:
			return "Hard"
		default:
			return "Very hard"
		}
	}
	// pass-rate per benchmark for context
	bp := map[string][2]int{}
	for _, t := range trials {
		v := bp[t.bench]
		v[1]++
		if t.pass {
			v[0]++
		}
		bp[t.bench] = v
	}

	type br struct {
		b string
		r float64
	}
	var benches []br
	for b, r := range bRat {
		benches = append(benches, br{b, r})
	}
	sort.Slice(benches, func(i, j int) bool { return benches[i].r > benches[j].r }) // hardest first

	fmt.Printf("ELO difficulty — %s mode, %d trials, %d benchmarks, %d models\n\n", mode, len(trials), len(bRat), len(mRat))
	fmt.Printf("%-32s %6s  %-9s %s\n", "benchmark", "ELO", "band", "pass")
	for _, x := range benches {
		v := bp[x.b]
		fmt.Printf("%-32s %6.0f  %-9s %d/%d\n", x.b, x.r, band(x.r), v[0], v[1])
	}

	type mr struct {
		m string
		r float64
	}
	var models []mr
	for m, r := range mRat {
		models = append(models, mr{m, r})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].r > models[j].r })
	fmt.Printf("\nModel capability (ELO):\n")
	for _, x := range models {
		fmt.Printf("  %6.0f  %s\n", x.r, x.m)
	}
}
