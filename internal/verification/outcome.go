package verification

// outcome.go — THE exit contract of every contract-verification command
// (M-V1-VERIFICATION-HONESTY M1, decision D1).
//
// `ailang check --verify`, its legacy spelling `ailang ai-check`, `ailang
// verify` and `ailang verify --package` all decide their process exit status
// here. Before this file they shared the verification loop (smt.Verify) but not
// the exit decision, and drifted: `check --verify` exited 0 when Z3 was missing
// (it had proved nothing), `verify` refused up front, and both passed an
// inconclusive obligation unless `verify --strict` was given.
//
// The rule is: exit non-zero whenever nothing was proved that should have been.
//
//	check failed                                  -> 1  not_run
//	no contract obligations (with or without Z3)  -> 0  no_obligations
//	obligations exist, solver unavailable         -> 1  unavailable
//	any counterexample                            -> 1  counterexample
//	any solver error, unknown or timeout          -> 1  error
//	any inconclusive (bounded-unrolling sat)      -> 1  inconclusive  (0 with AllowInconclusive)
//	verified + skipped only                       -> 0  partial       (1 with Strict)
//	all obligations verified                      -> 0  verified
//
// A skip is a stated boundary of the decidable fragment (effects, a
// higher-order parameter, an unsupported construct), so it passes by default —
// callers list it loudly on stderr. An inconclusive result is the solver's
// candidate counterexample and can be a real one, so it fails by default.
// There is ONE non-zero code; the JSON `outcome` discriminates.

// Outcome is the machine-readable verdict of one verification run. The values
// are the `verify.outcome` JSON field; new values may be added, existing ones
// are never renamed.
type Outcome string

const (
	OutcomeNotRun         Outcome = "not_run"
	OutcomeNoObligations  Outcome = "no_obligations"
	OutcomeVerified       Outcome = "verified"
	OutcomePartial        Outcome = "partial"
	OutcomeInconclusive   Outcome = "inconclusive"
	OutcomeCounterexample Outcome = "counterexample"
	OutcomeError          Outcome = "error"
	OutcomeUnavailable    Outcome = "unavailable"
)

// Tally is what a verification run measured. Obligations is the number of
// contract-bearing functions in the checked file; it is counted from the
// compiled program, so it is known even when the solver is not.
type Tally struct {
	CheckPassed     bool
	SolverAvailable bool
	Obligations     int
	Verified        int
	Counterexample  int
	Inconclusive    int
	Skipped         int
	Errors          int
}

// ExitPolicy holds the two flags that move a lane: Strict fails a partial
// run, AllowInconclusive passes an inconclusive one.
type ExitPolicy struct {
	Strict            bool
	AllowInconclusive bool
}

// Classify names the outcome of a run. Precedence is worst-first: a file with
// a counterexample and a skip is a counterexample.
func Classify(t Tally) Outcome {
	switch {
	case !t.CheckPassed:
		return OutcomeNotRun
	case !t.SolverAvailable && t.Obligations > 0:
		return OutcomeUnavailable
	case t.Counterexample > 0:
		return OutcomeCounterexample
	case t.Errors > 0:
		return OutcomeError
	case t.Inconclusive > 0:
		return OutcomeInconclusive
	case t.Skipped > 0:
		return OutcomePartial
	case t.Verified > 0:
		return OutcomeVerified
	default:
		return OutcomeNoObligations
	}
}

// ExitCode is the process exit status every verification command returns.
func ExitCode(t Tally, p ExitPolicy) int {
	switch Classify(t) {
	case OutcomeNoObligations, OutcomeVerified:
		return 0
	case OutcomePartial:
		if p.Strict {
			return 1
		}
		return 0
	case OutcomeInconclusive:
		// --strict outranks --allow-inconclusive: a strict gate passes nothing
		// that was not proved.
		if p.AllowInconclusive && !p.Strict {
			return 0
		}
		return 1
	default:
		return 1
	}
}
