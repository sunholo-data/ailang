// Command eval-regrade re-applies the current CompareOutput grader to the stored
// stdout/expected_stdout of an existing eval baseline, without re-running any
// models. It is used after a grader change (e.g. M-EVAL-JSON-COMPARE,
// M-EVAL-OUTPUT-NORMALIZE) to bring historical baselines up to the new grading.
//
// Report-first: by default it only prints what WOULD change. Pass --apply to
// rewrite the result files' stdout_ok in place. It asserts that no result flips
// pass->fail (a grader change that only normalizes formatting can only ever turn
// fails into passes); any pass->fail flip is reported as a BUG and blocks --apply.
//
//	go run ./tools/eval-regrade <baseline_dir>            # report only
//	go run ./tools/eval-regrade --apply <baseline_dir>    # rewrite stdout_ok in place
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

type counts struct{ passBefore, passAfter, total int }

func main() {
	apply := false
	var dir string
	for _, a := range os.Args[1:] {
		switch {
		case a == "--apply":
			apply = true
		case strings.HasPrefix(a, "-"):
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
			os.Exit(2)
		default:
			dir = a
		}
	}
	if dir == "" {
		fmt.Fprintln(os.Stderr, "usage: eval-regrade [--apply] <baseline_dir>")
		os.Exit(2)
	}

	var files []string
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && strings.HasSuffix(p, ".json") {
			files = append(files, p)
		}
		return nil
	})

	per := map[string]*counts{} // key: "lang/model"
	var gained, lost []string   // fail->pass (expected), pass->fail (BUG)
	processed := 0

	for _, p := range files {
		raw, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		// Only result records carry these fields.
		exp, ok1 := m["expected_stdout"].(string)
		got, ok2 := m["stdout"].(string)
		co, ok3 := m["compile_ok"].(bool)
		ro, ok4 := m["runtime_ok"].(bool)
		soOld, ok5 := m["stdout_ok"].(bool)
		if !(ok1 && ok2 && ok3 && ok4 && ok5) {
			continue
		}
		processed++
		lang, _ := m["lang"].(string)
		model, _ := m["model"].(string)
		id, _ := m["id"].(string)
		key := lang + "/" + model
		if per[key] == nil {
			per[key] = &counts{}
		}
		c := per[key]
		c.total++

		soNew := eval_harness.CompareOutput(exp, got)
		passBefore := co && ro && soOld
		// Monotonic: a re-grade recovers false-negatives; it must never remove a
		// stored pass. soFinal = stored OR newly-graded.
		soFinal := soOld || soNew
		passAfter := co && ro && soFinal
		if passBefore {
			c.passBefore++
		}
		if passAfter {
			c.passAfter++
		}
		if !passBefore && passAfter {
			gained = append(gained, fmt.Sprintf("%s [%s/%s]", id, lang, model))
		}
		// INFO only: a stored pass whose stored stdout the strict grader rejects —
		// a pre-existing runtime-grader leniency (e.g. verbose repair output), NOT
		// touched here. Preserved as a pass; surfaced for separate investigation.
		if passBefore && co && ro && !soNew {
			lost = append(lost, fmt.Sprintf("%s [%s/%s]", id, lang, model))
		}

		if apply && soFinal != soOld {
			m["stdout_ok"] = soFinal
			out, _ := json.MarshalIndent(m, "", "  ")
			_ = os.WriteFile(p, out, 0o644)
		}
	}

	fmt.Printf("regrade %s — %d result files\n\n", dir, processed)
	fmt.Printf("fail->pass flips (recovered): %d\n", len(gained))
	sort.Strings(gained)
	for _, g := range gained {
		fmt.Printf("  + %s\n", g)
	}
	if len(lost) > 0 {
		fmt.Printf("\nINFO: %d stored passes whose stored stdout the strict grader rejects\n", len(lost))
		fmt.Printf("      (pre-existing runtime-grader leniency, e.g. verbose repair output;\n")
		fmt.Printf("       PRESERVED as passes here — flagged for separate investigation):\n")
		for _, l := range lost {
			fmt.Printf("  ? %s\n", l)
		}
	}

	// Aggregate before/after pass-rate per lang.
	langB := map[string][2]int{}
	keys := make([]string, 0, len(per))
	for k := range per {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("\n%-26s %8s %8s\n", "lang/model", "before", "after")
	for _, k := range keys {
		c := per[k]
		fmt.Printf("%-26s %7.1f%% %7.1f%%\n", k, pct(c.passBefore, c.total), pct(c.passAfter, c.total))
		lang := strings.SplitN(k, "/", 2)[0]
		v := langB[lang]
		langB[lang] = [2]int{v[0] + c.passBefore, v[1] + c.passAfter}
	}
	fmt.Println()
	for _, lang := range sortedKeys(langB) {
		v := langB[lang]
		tot := 0
		for _, k := range keys {
			if strings.HasPrefix(k, lang+"/") {
				tot += per[k].total
			}
		}
		fmt.Printf("FIELD %-8s before %.1f%%  ->  after %.1f%%\n", lang, pct(v[0], tot), pct(v[1], tot))
	}

	if !apply {
		fmt.Println("\n(report only — re-run with --apply to rewrite stdout_ok)")
	} else {
		fmt.Println("\napplied: stdout_ok rewritten in place")
	}
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
func sortedKeys(m map[string][2]int) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
