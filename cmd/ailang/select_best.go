package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/bestof"
)

// runSelectBest implements `ailang select-best [--caps IO] [--entry main] <c1.ail> <c2.ail> ...`.
//
// Best-of-N exact selector (M-AILANG-NATIVE-HARNESS probe #3 — the validated top pass-rate lever):
// given N candidate solutions (e.g. N independent agent samples at temp>0), verify each with
// `ailang check` + run and print the verified-best one. Ranking: runs > typechecks-only > neither
// (ties keep the model's order). Prints the selected path to stdout (machine-consumable); per-candidate
// verdicts go to stderr. This is the surface a best-of-N agent loop calls at finalization: generate N
// → `ailang select-best` → submit the winner. The exact in-loop verification a general harness lacks.
func runSelectBest() {
	fs := flag.NewFlagSet("select-best", flag.ExitOnError)
	caps := fs.String("caps", "IO", "Capabilities for running candidates (e.g. IO,FS)")
	entry := fs.String("entry", "main", "Entrypoint function")
	timeout := fs.Duration("timeout", 30*time.Second, "Per-candidate verify timeout")
	verifyContracts := fs.Bool("verify-contracts", false, "Also check ensures/requires contracts (rejects runs-but-wrong; the AILANG-native edge)")
	_ = fs.Parse(flag.Args()[1:])

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "%s: need at least one candidate file\n", red("Error"))
		fmt.Println("Usage: ailang select-best [--caps IO] [--entry main] <c1.ail> <c2.ail> ...")
		fmt.Println("Picks the candidate that typechecks + runs (runs > typechecks > neither); prints its path.")
		os.Exit(1)
	}

	v := bestof.AilangVerifier{Caps: *caps, Entry: *entry, Timeout: *timeout, VerifyContracts: *verifyContracts, RelaxModules: true}
	best, verdicts := bestof.SelectBest(files, v)
	for i, vd := range verdicts {
		status := "neither"
		if vd.Runs && vd.ContractsPass {
			status = "runs+contracts"
		} else if vd.Runs {
			status = "runs"
		} else if vd.TypeChecks {
			status = "typechecks"
		}
		marker := " "
		if i == best {
			marker = green("*")
		}
		fmt.Fprintf(os.Stderr, " %s [%d] %-10s %s\n", marker, i, status, files[i])
	}
	if best < 0 {
		os.Exit(1) // empty input
	}
	fmt.Println(files[best])
}
