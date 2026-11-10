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
//	any inconclusive (bounded-unrolling sat)      -> 1  inconclusive  (see AllowInconclusive)
//	skipped only, NOTHING verified                -> 1  unproved
//	at least one verified + at least one skipped  -> 0  partial       (1 with Strict)
//	all obligations verified                      -> 0  verified
//
// An OBLIGATION is a function with at least one `ensures` clause: something
// the verifier is asked to prove about that function. A `requires`-only
// function is not one. The verifier assumes a precondition while proving the
// function's own ensures and does not check it at call sites, so a precondition
// alone gives it nothing to prove; such a function counts nowhere in this
// tally, with or without a solver, and passes --strict.
//
// A skip is a stated boundary of the decidable fragment (effects, a
// higher-order parameter, an unsupported construct). A run that verified at
// least one obligation and skipped others passes with the skips listed (and
// enumerated by name in the JSON results). A run whose obligations were ALL
// skipped proved nothing and fails as `unproved`. An inconclusive result is
// the solver's candidate counterexample and can be a real one, so it fails by
// default. There is ONE non-zero code; the JSON `outcome` discriminates.

// Outcome is the machine-readable verdict of one verification run. The values
// are the `verify.outcome` JSON field; new values may be added, existing ones
// are never renamed.
type Outcome string

const (
	OutcomeNotRun         Outcome = "not_run"
	OutcomeNoObligations  Outcome = "no_obligations"
	OutcomeVerified       Outcome = "verified"
	OutcomePartial        Outcome = "partial"
	OutcomeUnproved       Outcome = "unproved"
	OutcomeInconclusive   Outcome = "inconclusive"
	OutcomeCounterexample Outcome = "counterexample"
	OutcomeError          Outcome = "error"
	OutcomeUnavailable    Outcome = "unavailable"
)

// Tally is what a verification run measured, over OBLIGATIONS only (functions
// with an ensures clause; see the file comment). Obligations is counted from
// the compiled program, so it is known even when the solver is not; with a
// solver it equals Verified+Counterexample+Inconclusive+Skipped+Errors.
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
	case t.Skipped > 0 && t.Verified == 0:
		return OutcomeUnproved
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
		if !p.AllowInconclusive || p.Strict {
			return 1
		}
		// --allow-inconclusive accepts the inconclusive obligations and only
		// them: the rest of the run is judged as if they were absent, so
		// inconclusive + skipped with nothing verified is still unproved.
		rest := t
		rest.Inconclusive = 0
		if Classify(rest) == OutcomeUnproved {
			return 1
		}
		return 0
	default:
		return 1
	}
}
