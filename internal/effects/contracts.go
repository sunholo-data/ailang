package effects

import (
	"fmt"

	"github.com/sunholo-data/ailang/internal/ast"
)

// ContractMode controls how contract violations are handled
//
// Three modes support different use cases:
//   - Panic: Production safety - fail fast on any violation
//   - Report: Testing/analysis - collect all violations for inspection
//   - Off: Disabled - no runtime overhead from contract checks
type ContractMode int

const (
	ContractModePanic  ContractMode = iota // Panic on first violation (default)
	ContractModeReport                     // Collect all violations, continue execution
	ContractModeOff                        // Skip all contract checks (no overhead)
)

// String returns the string representation of ContractMode
func (m ContractMode) String() string {
	switch m {
	case ContractModePanic:
		return "panic"
	case ContractModeReport:
		return "report"
	case ContractModeOff:
		return "off"
	default:
		return "unknown"
	}
}

// ContractCheck represents the result of a single contract check
//
// This captures all information needed to report or analyze
// contract violations, including the contract kind, whether it
// passed, the function it was in, and source location.
type ContractCheck struct {
	Kind     ast.ContractKind // requires, ensures, or invariant
	Passed   bool             // Whether the contract held
	Message  string           // User-provided or auto-generated message
	Location string           // "file.ail:42" (auto-injected by compiler)
	Function string           // Function name where contract was checked
}

// ContractContext collects contract check results during execution
//
// The contract context is the host-side handler for contract verification.
// It accumulates contract check results during AILANG execution, with
// collection only available from the host (not from AILANG code).
//
// This implements the "write-only from AILANG" design pattern:
//   - AILANG code can trigger contract checks via generated code
//   - Only the host can call Collect(), Reset(), and configure Mode
//   - Prevents AILANG code from branching on its own contract state
//
// Thread-safety: ContractContext is designed for single-threaded use
// within one evaluation. Create a new context for each run.
type ContractContext struct {
	checks   []ContractCheck
	Mode     ContractMode // How to handle violations
	function string       // Current function being executed
}

// ContractOutput is the structured output from contract collection
type ContractOutput struct {
	Checks     []ContractCheck
	Mode       ContractMode
	TotalCount int
	PassCount  int
	FailCount  int
}

// NewContractContext creates a new contract context with default Panic mode
//
// The context starts empty in Panic mode. Call SetMode() to change
// how violations are handled before running AILANG code.
func NewContractContext() *ContractContext {
	return &ContractContext{
		checks:   make([]ContractCheck, 0),
		Mode:     ContractModePanic,
		function: "",
	}
}

// NewContractContextWithMode creates a contract context with specified mode
func NewContractContextWithMode(mode ContractMode) *ContractContext {
	return &ContractContext{
		checks:   make([]ContractCheck, 0),
		Mode:     mode,
		function: "",
	}
}

// SetMode changes how contract violations are handled
//
// Changing mode during execution is allowed but may lead to
// inconsistent behavior - prefer setting mode before evaluation.
func (c *ContractContext) SetMode(mode ContractMode) {
	c.Mode = mode
}

// SetFunction sets the current function for context in error messages
//
// Called by generated code when entering a function with contracts.
func (c *ContractContext) SetFunction(name string) {
	c.function = name
}

// CurrentFunction returns the current function name.
// Used by trace recording to annotate contract check events.
func (c *ContractContext) CurrentFunction() string {
	return c.function
}

// CheckRequires records a precondition check result
//
// This is called by generated code at function entry.
// In Panic mode, violations cause an immediate panic.
// In Report mode, violations are collected for later analysis.
// In Off mode, this is a no-op.
//
// Parameters:
//   - cond: The boolean result of evaluating the precondition
//   - msg: User-provided message or auto-generated predicate string
//   - location: Source location "file.ail:42" (injected by compiler)
//
// Returns error in Panic mode if check fails, nil otherwise.
func (c *ContractContext) CheckRequires(cond bool, msg, location string) error {
	return c.check(ast.RequiresKind, cond, msg, location)
}

// CheckEnsures records a postcondition check result
//
// This is called by generated code before function return.
// Behavior follows same pattern as CheckRequires.
func (c *ContractContext) CheckEnsures(cond bool, msg, location string) error {
	return c.check(ast.EnsuresKind, cond, msg, location)
}

// CheckInvariant records an invariant check result
//
// This is called by generated code at module/type boundaries.
// Behavior follows same pattern as CheckRequires.
func (c *ContractContext) CheckInvariant(cond bool, msg, location string) error {
	return c.check(ast.InvariantKind, cond, msg, location)
}

// check is the internal implementation for all contract checks
func (c *ContractContext) check(kind ast.ContractKind, cond bool, msg, location string) error {
	// Off mode: skip everything
	if c.Mode == ContractModeOff {
		return nil
	}

	check := ContractCheck{
		Kind:     kind,
		Passed:   cond,
		Message:  msg,
		Location: location,
		Function: c.function,
	}

	// Always record in Report mode
	if c.Mode == ContractModeReport {
		c.checks = append(c.checks, check)
		return nil
	}

	// Panic mode: fail fast on violation
	if !cond {
		return fmt.Errorf("contract violation: %s failed in %s at %s: %s",
			kind.String(), c.function, location, msg)
	}

	// Panic mode: record successful checks too (for analysis)
	c.checks = append(c.checks, check)
	return nil
}

// Collect returns all accumulated contract check results (HOST-ONLY)
//
// This method is NOT exposed to AILANG code. Only the host
// can collect contract output after AILANG execution completes.
func (c *ContractContext) Collect() ContractOutput {
	passCount := 0
	failCount := 0
	for _, check := range c.checks {
		if check.Passed {
			passCount++
		} else {
			failCount++
		}
	}

	return ContractOutput{
		Checks:     c.checks,
		Mode:       c.Mode,
		TotalCount: len(c.checks),
		PassCount:  passCount,
		FailCount:  failCount,
	}
}

// Reset clears the contract context for the next run
//
// Call this between runs to start fresh. Mode is preserved.
func (c *ContractContext) Reset() {
	c.checks = c.checks[:0]
	c.function = ""
}

// HasViolations returns true if any contract checks failed
func (c *ContractContext) HasViolations() bool {
	for _, check := range c.checks {
		if !check.Passed {
			return true
		}
	}
	return false
}

// Violations returns only the failed contract checks
func (c *ContractContext) Violations() []ContractCheck {
	var failed []ContractCheck
	for _, check := range c.checks {
		if !check.Passed {
			failed = append(failed, check)
		}
	}
	return failed
}

// ViolationsByKind returns failed checks filtered by contract kind
func (c *ContractContext) ViolationsByKind(kind ast.ContractKind) []ContractCheck {
	var failed []ContractCheck
	for _, check := range c.checks {
		if !check.Passed && check.Kind == kind {
			failed = append(failed, check)
		}
	}
	return failed
}
