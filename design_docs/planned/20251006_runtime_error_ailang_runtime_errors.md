# AILANG: Runtime Errors

**Discovered**: AI Eval Analysis - 2025-10-06
**Frequency**: 14 failures across 2 benchmark(s)
**Priority**: P1 (High Priority)
**Estimated**: 1040 LOC, 4 days
**Category**: runtime_error
**Impact**: high

## Problem Statement

AILANG benchmarks adt_option and fizzbuzz exhibit runtime_error failures dominated by two messages: (1) "builtin eq_Int expects Int arguments" and (2) "effect 'IO' requires capability, but none provided". In affected examples, equality over floats inside control flow (e.g., b == 0.0) triggers the wrong Eq dictionary at runtime, and IO printing fails due to missing capabilities in the benchmark runner.

Users expect statically type-checked AILANG programs to execute deterministically without runtime type mismatches, especially for basic comparisons like Float equality. They also expect the CLI/runner to either infer and provision required capabilities for entrypoints or provide actionable guidance that doesn't require tribal knowledge.



**Last Updated**: 2025-10-25 (merged 7 new failures)
## Evidence from AI Eval

**Affected Benchmarks**: adt_option, fizzbuzz

**Models Affected**: claude-sonnet-4-5, gpt5-mini

**Failure Rate**: 7/31 (22.6%)

### Example Failures


**Error 1:**
```
Error: execution failed: builtin eq_Int expects Int arguments

```

**Generated Code:**
```ailang
module benchmark/solution

import std/io (println)

type Option[a] = Some(a) | None

export func divide(a: float, b: float) -> Option[float] {
  if b == 0.0
  then None
  else Some(a / b)
}

export func printResult(result: Option[float]) -> () ! {IO} {
  match result {
    Some(v) => println("Result: " ++ show(v)),
    None => println("Error: Division by zero")
  }
}

export func main() -> () ! {IO} {
  let r1 = divide(10.0, 2.0);
  printResult(r1);
  let r2 = divide(10.0, 0.0);
  printResult(r2)
}
```

---

**Error 2:**
```
Error: execution failed: effect 'IO' requires capability, but none provided
Hint: Run with --caps IO

```

**Generated Code:**
```ailang
module benchmark/solution

import std/io (println)

type Option[a] = Some(a) | None

export func divide(a: float, b: float) -> Option[float] {
  if b == 0.0
  then None
  else Some(a / b)
}

export func printResult(result: Option[float]) -> () ! {IO} {
  match result {
    Some(v) => println("Result: " ++ show(v)),
    None => println("Error: Division by zero")
  }
}

export func main() -> () ! {IO} {
  let r1 = divide(10.0, 2.0);
  printResult(r1);
  let r2 = divide(10.0, 0.0);
  printResult(r2)
}
```


### Additional Examples (Latest Analysis)

**Error 1:**
```
Error: execution failed: builtin eq_Int expects Int arguments

```

**Generated Code:**
```ailang
module benchmark/solution

import std/io (println)

type Option[a] = Some(a) | None

export func divide(a: float, b: float) -> Option[float] {
  if b == 0.0
  then None
  else Some(a / b)
}

export func printResult(result: Option[float]) -> () ! {IO} {
  match result {
    Some(v) => println("Result: " ++ show(v)),
    None => println("Error: Division by zero")
  }
}

export func main() -> () ! {IO} {
  let r1 = divide(10.0, 2.0);
  printResult(r1);
  let r2 = divide(10.0, 0.0);
  printResult(r2)
}
```

---

**Error 2:**
```
Error: execution failed: effect 'IO' requires capability, but none provided
Hint: Run with --caps IO

```

**Generated Code:**
```ailang
module benchmark/solution

import std/io (println)

type Option[a] = Some(a) | None

export func divide(a: float, b: float) -> Option[float] {
  if b == 0.0
  then None
  else Some(a / b)
}

export func printResult(result: Option[float]) -> () ! {IO} {
  match result {
    Some(v) => println("Result: " ++ show(v)),
    None => println("Error: Division by zero")
  }
}

export func main() -> () ! {IO} {
  let r1 = divide(10.0, 2.0);
  printResult(r1);
  let r2 = divide(10.0, 0.0);
  printResult(r2)
}
```

---

---


## Root Cause Analysis

Equality failures arise from a gap between the type system and runtime dictionary-passing: the Eq dictionary selected for expressions like b == 0.0 is incorrectly specialized to Int at elaboration/codegen time. Contributing factors: missing or incomplete Eq Float instance in the instance registry and too-early defaulting of class constraints before final unification of numeric literal types. Additionally, decimal literals are currently treated through generic Num defaulting rather than first-class Float literals, increasing ambiguity.

Capability failures are environmental: the benchmark harness invokes modules with IO effects without passing --caps IO. While the runtime correctly enforces capability security, the lack of automatic capability inference or explicit preflight reporting leads to avoidable failures in standard examples that print.

## Proposed Solution

Address the equality issue by (a) adding complete Float instances/builtins (Eq, Ord, Show) and (b) fixing instance resolution and defaulting: delay defaulting until after constraint solving; never default a resolved concrete type; and prefer literal-kind-driven typing (distinct Int vs Float literals) to avoid ambiguous Eq selections. Concretely, introduce native Float literals at the parser/AST level and wire eq_Float through the type class dictionary and codegen paths.

For capabilities, implement entrypoint effect preflight and optional auto-cap provisioning. The runtime will statically extract the effect row of the chosen entrypoint, list required capabilities, and either (1) fail fast with a precise message and a suggested --caps command or (2) if --auto-caps is enabled (or AILANG_AUTO_CAPS=1 in CI), automatically grant only those caps. Update the benchmark harness to use --auto-caps to maintain security defaults elsewhere.

This fits AILANG’s architecture: type class fixes live in the type checker and instance registry with dictionary-passing already in place; numeric literal typing changes stay local to parser/AST + inference; runtime changes extend the capability manager and CLI without weakening the security model.

### Implementation Approach

- Add Float instances and builtins: Implement Eq Float, Ord Float, Show Float with runtime builtins (eq_Float, cmp_Float, show_Float) and register in the instance table.
- Literal typing overhaul: Lexer/parser to distinguish IntLit vs FloatLit tokens; AST nodes Carry literal kind; type inference assigns concrete Int/Float without relying on Num defaulting for explicit decimal literals.
- Constraint/defaulting fix: In the type class solver, postpone defaulting until after generalization; forbid defaulting when a type variable is unified with a concrete type; ensure dictionaries for (==) are picked from the final monotype.
- Dictionary selection audit: Ensure codegen passes the correct Eq dictionary for both operands’ unified type; add an assert in elaboration to reject mixed-type equality before runtime.
- Improved runtime error messages: On builtin equality mismatch, include observed types and expected type from dictionary for faster diagnosis; add TC_ERR codes.
- Entrypoint effect preflight: At module load/run, extract effect row of entrypoint; compute capability set; print a one-line hint or JSON manifest; exit code distinct for missing caps.
- --auto-caps flag and CI env var: Implement CLI flag and env var to automatically grant only inferred caps; logs the granted set; off by default.
- Benchmark harness update: Enable --auto-caps for evaluation runs; add guards to disallow Net/FS unless benchmarks require them.
- Docs and examples: Update examples to showcase Float equality and capability preflight; add a short section in README on auto-caps and preflight output.
- Migration guardrails: Add a linter rule to flag use of (==) on types without Eq instances at compile-time, preventing latent runtime mismatches.

## Technical Design

### API Changes

- CLI: new --auto-caps flag; preflight output on missing caps with distinct exit code. No language-level API change for users.

### Type System Changes

- Distinct Int vs Float literal typing at parse/AST level.
- Defaulting deferred until after constraint solving; no defaulting for concrete types.
- Ensure Eq instance resolution uses final monotypes.

### Runtime Changes

- New builtins for Float Eq/Ord/Show and improved error messages.
- Entrypoint effect preflight and optional auto-cap provisioning in CLI/runtime.

## Implementation Plan


1. **1. Float Builtins & Instances (~120 LOC, 0.5d) - Implement eq_Float, cmp_Float, show_Float; register Eq/Ord/Show Float in typeclasses/instances.go and runtime/builtins.** (~TBD LOC, TBD)
   

2. **2. Lexer/Parser Literal Kinds (~150 LOC, 0.5d) - Distinguish IntLit vs FloatLit in lexer.go; propagate to parser.go and AST nodes.** (~TBD LOC, TBD)
   

3. **3. Inference for Literals (~90 LOC, 0.5d) - Assign concrete types for literals in infer_literals.go; remove reliance on Num defaulting for FloatLit.** (~TBD LOC, TBD)
   

4. **4. Constraint Solver Defaulting Fix (~200 LOC, 1d) - Delay defaulting until post-unification; prohibit defaulting of concrete types; refine instance lookup in infer_classes.go.** (~TBD LOC, TBD)
   

5. **5. Codegen Dictionary Wiring Audit (~100 LOC, 0.5d) - Ensure equality uses correct dictionary; add compile-time assertion for operand types; files: codegen/elab_classes.go.** (~TBD LOC, TBD)
   

6. **6. Runtime Error Enrichment (~80 LOC, 0.25d) - Improve error strings with observed types; add error codes; files: runtime/builtins/errors.go.** (~TBD LOC, TBD)
   

7. **7. Entrypoint Effect Preflight (~180 LOC, 1d) - Extract effect rows; compute caps; print plan; files: runtime/module_runtime.go, cli/run.go.** (~TBD LOC, TBD)
   

8. **8. --auto-caps Flag & Env (~90 LOC, 0.25d) - Implement flag/env handling; secure defaults; files: cli/flags.go, cli/run.go.** (~TBD LOC, TBD)
   

9. **9. Benchmark Harness Update (~70 LOC, 0.25d) - Enable auto-caps in evaluator; restrict caps to required; files: tools/bench_runner.go.** (~TBD LOC, TBD)
   

10. **10. Docs & Examples (~60 LOC, 0.25d) - Update README/examples; add Float equality sample; files: docs/, examples/.** (~TBD LOC, TBD)
   


## Testing Strategy

### Unit Tests



### Integration Tests

- Re-run adt_option examples: divide/printResult path; assert no runtime eq_Int error; output matches expected strings.
- Re-run fizzbuzz: equality against 0 for Int uses eq_Int; with IO auto-caps, program prints correctly.
- Module with multiple effects (IO, FS) and only IO requested: preflight lists both; auto-caps grants both if enabled; without, fails with guidance.
- Cross-module import with Float equality inside nested functions: ensure correct dictionary is passed through module boundaries.

### New Benchmarks

- float_eq_control_flow.ail: If/else on Float equality driving ADT return; asserts no runtime errors; checks printed output.
- caps_preflight_report: Entry-only main with declared effects; verify CLI output and exit code; CI uses --auto-caps to run to completion.

## Success Criteria


- [ ] Eq Float instance available and selected for Float operands; no eq_Int mismatch at runtime.

- [ ] Decimal literals parsed as Float and typed as Float without defaulting.

- [ ] Mixed-type equality is rejected at compile-time with clear diagnostics.

- [ ] Bench harness runs IO programs without manual --caps via --auto-caps, preserving secure defaults.

- [ ] adt_option and fizzbuzz benchmarks pass end-to-end with zero runtime_error incidents.

- [ ] Preflight capability report shows accurate caps for any entrypoint.

- [ ] Error messages include expected/observed types and error codes.

- [ ] All new unit/integration tests pass in CI.


## References

- **Similar Features**: See design_docs/implemented/ for reference implementations
- **Design Docs**: CLAUDE.md, README.md, design_docs/planned/v0_4_0_net_enhancements.md
- **AILANG Architecture**: See CLAUDE.md, README.md

## Estimated Impact

**Before Fix**:
- AI success rate: 72.7%%
- Token efficiency: Higher retries due to runtime errors; additional tokens spent on re-executions and debugging messages.

**After Fix** (projected):
- AI success rate: 78–81% (projected +5–8% absolute, eliminating ~22.6% of current failures)%
- Token efficiency: Fewer reruns and shorter error traces; projected 5–10% reduction in tokens per successful benchmark run due to first-try success and clearer preflight guidance.

---

*Generated by ailang eval-analyze on 2025-10-06 12:04:47*
*Model: gpt5*
