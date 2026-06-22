package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sunholo-data/ailang/internal/astedit"
	"github.com/sunholo-data/ailang/internal/bestof"
)

// runSelectBest implements `ailang select-best [flags] <c1.ail> <c2.ail> ...`.
//
// Best-of-N exact selector (the validated top pass-rate lever): verify each candidate with
// `ailang check` + run and print the verified-best one. Ranking: runs+contracts > runs >
// typechecks-only > neither (ties keep the model's order). Prints the selected ORIGINAL path
// to stdout; per-candidate verdicts go to stderr.
//
// R1 contract-glue (--contract-spec): the benchmark provides requires/ensures the model usually
// omits. Inject the PROVIDED spec into each candidate, then --verify-contracts becomes a
// reference-free oracle that rejects runs-but-WRONG candidates — the moat a general harness can't run.
func runSelectBest() {
	fs := flag.NewFlagSet("select-best", flag.ExitOnError)
	caps := fs.String("caps", "IO", "Capabilities for running candidates (e.g. IO,FS)")
	entry := fs.String("entry", "main", "Entrypoint function")
	timeout := fs.Duration("timeout", 30*time.Second, "Per-candidate verify timeout")
	verifyContracts := fs.Bool("verify-contracts", false, "Also check ensures/requires contracts (rejects runs-but-wrong; the AILANG-native edge)")
	verifyZ3 := fs.Bool("verify-z3", false, "Also run `ailang verify` (Z3 SMT): statically PROVES contracts for ALL inputs; ranked above runtime contracts (needs z3 installed)")
	contractSpec := fs.String("contract-spec", "", "Path to a contract spec (requires/ensures clauses) INJECTED into each candidate before verifying (R1 moat). Implies --verify-contracts.")
	contractFunc := fs.String("contract-func", "", "Function name to inject --contract-spec into (required with --contract-spec)")
	_ = fs.Parse(flag.Args()[1:])

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "%s: need at least one candidate file\n", red("Error"))
		fmt.Println("Usage: ailang select-best [--caps IO] [--entry main] [--verify-contracts] [--contract-spec FILE --contract-func NAME] <c1.ail> ...")
		fmt.Println("Picks the candidate that typechecks + runs (runs+contracts > runs > typechecks > neither); prints its path.")
		os.Exit(1)
	}

	// Preserve original paths: verification may run against injected temp copies, but we report
	// and return the candidate the caller actually passed.
	origFiles := append([]string{}, files...)

	if *contractSpec != "" {
		if *contractFunc == "" {
			fmt.Fprintf(os.Stderr, "%s: --contract-func is required with --contract-spec\n", red("Error"))
			os.Exit(1)
		}
		specBytes, err := os.ReadFile(*contractSpec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: reading --contract-spec: %v\n", red("Error"), err)
			os.Exit(1)
		}
		spec := string(specBytes)
		*verifyContracts = true
		for i, f := range files {
			src, err := os.ReadFile(f)
			if err != nil {
				continue // leave original; the verifier will mark it
			}
			injected, err := astedit.InjectContract(string(src), f, *contractFunc, spec)
			if err != nil {
				// function not found / unsupported form → verify the candidate as-is.
				fmt.Fprintf(os.Stderr, "   [%d] inject-skip: %v\n", i, err)
				continue
			}
			tmp, err := os.CreateTemp("", "selectbest-*.ail")
			if err != nil {
				continue
			}
			_, _ = tmp.WriteString(injected)
			_ = tmp.Close()
			defer os.Remove(tmp.Name())
			files[i] = tmp.Name()
		}
	}

	v := bestof.AilangVerifier{Caps: *caps, Entry: *entry, Timeout: *timeout, VerifyContracts: *verifyContracts, VerifyZ3: *verifyZ3, RelaxModules: true}
	best, verdicts := bestof.SelectBest(files, v)
	for i, vd := range verdicts {
		status := "neither"
		if vd.Runs && vd.Verifies {
			status = "z3-verified"
		} else if vd.Runs && vd.ContractsPass {
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
		fmt.Fprintf(os.Stderr, " %s [%d] %-14s %s\n", marker, i, status, origFiles[i])
	}
	if best < 0 {
		os.Exit(1) // empty input
	}
	fmt.Println(origFiles[best])
}
