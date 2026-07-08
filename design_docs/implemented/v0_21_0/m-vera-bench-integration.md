# M-VERA-BENCH-INTEGRATION: Add AILANG as a target language in VeraBench

**Status**: Planned (future work from M-THREE-CAMPS)
**Target**: v0.23.0+
**Priority**: P2 — Strategic positioning (independent third-party benchmark validation)
**Estimated**: ~9–14 hours (revised — sourcing solutions from existing AILANG benchmarks/examples cuts the solution-writing phase from ~6h to ~3h)
**Dependencies**:
  - AILANG eval harness (`internal/eval_harness/`) — for generating reference solutions
  - VeraBench upstream: [aallan/vera-bench](https://github.com/aallan/vera-bench)
  - VeraBench fork: [sunholo-data/vera-bench](https://github.com/sunholo-data/vera-bench) (downstream development)

**Commissioning context**: The M-THREE-CAMPS sprint surfaced [VeraBench](https://github.com/aallan/vera-bench) — an independent third-party benchmark suite designed by the Vera author to test LLM code generation across multiple languages (Vera, Python, TypeScript, Aver). VeraBench published results showing Kimi K2.5 hits 100% on Vera vs 86% on Python vs 91% on TypeScript — strong evidence that language design beats training-data volume.

Adding AILANG as a VeraBench target language is a high-leverage move:
- **Independent benchmark suite** — we didn't design it; comparisons can't be gerrymandered
- **Published reference numbers** for Vera/Python/TypeScript to directly compare against
- **Same problem set** (50 problems across 5 difficulty tiers) — apples-to-apples
- **Adds AILANG to a research artifact with publication trajectory**

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Eval-suite work; no semantic change |
| A2: Replayability | +1 | VeraBench results are version-pinned + replayable |
| A5: Bounded Verification | +2 | Direct test of AILANG contracts on a verification-focused benchmark suite |
| A7: Machines First | +2 | The whole exercise is "how well does the LLM write AILANG against an independent benchmark" |
| A11: Structured Failure | +1 | VeraBench reports check@1 / verify@1 / run_correct@1 — structured outcome categories |
| (others) | 0 | N/A |

**Net Score: +6** → Proceed.

---

## VeraBench's Architecture (Reverse-Engineered)

VeraBench treats `vera` as a black-box CLI tool and runs LLM-generated solutions through a Python harness:

```
problems/tierN/VB_TX_NNN_name.json        ← language-neutral problem spec
  ├── id, tier, title
  ├── description (Vera-oriented prompt)
  ├── description_neutral (language-neutral version)
  ├── signature (Vera signature, used for Vera mode)
  ├── contracts.requires / ensures / effects (Vera-specific)
  ├── entry_point (function name; reused across all targets)
  └── test_cases: [{args: [...], expected: ...}, ...]   ← language-neutral

solutions/<lang>/VB_TX_NNN_name.<ext>      ← reference solution per language
  └── must export `entry_point` matching the JSON spec

vera_bench/baseline_runner.py              ← Python harness
  └── _build_<lang>_wrapper(problem, baseline_path) → str
      generates a script that imports the baseline, runs each test case,
      and emits {"passed": bool, "actual": str, "error": str?} JSON
```

For each target language, the harness:
1. Loads the problem JSON
2. Finds the matching reference solution
3. Builds a wrapper script (using the language's import/call syntax)
4. Executes the wrapper as a subprocess
5. Parses the JSON results
6. Reports `check@1` (compile/check pass), `run_correct@1` (output matches `expected`)

**Existing wrappers**: Python (`_build_python_wrapper`), TypeScript (`_build_typescript_wrapper`), and Aver (presumably `_build_aver_wrapper` in `vera_runner.py`).

---

## What Adding AILANG Requires

### 1. Reference solutions for all 50 problems (~3h with existing material, not 6h from scratch)

Write `solutions/ailang/VB_TX_NNN_*.ail` for each of the 50 problems. Each solution is a valid AILANG module exporting `entry_point` (function name from problem JSON) that implements the function per the `description` and `test_cases`.

**Existing AILANG material to source from** (cuts the work in half):

| VeraBench tier | What it tests | Existing AILANG sources |
|----------------|---------------|--------------------------|
| 1 (pure arithmetic) | absolute_value, clamp, signum, max_of_two, etc. | `examples/runnable/*.ail`, `benchmarks/numeric_modulo.yml`, `benchmarks/float_eq.yml` |
| 2 (string/array ops) | string manipulation, list operations | `benchmarks/balanced_parens.yml`, `benchmarks/run_length_encode.yml`, `benchmarks/list_comprehension.yml`, `benchmarks/fold_reduce.yml` |
| 3 (ADTs & match) | sum types, pattern matching | `benchmarks/adt_option.yml`, `benchmarks/exhaustive_pattern_matching.yml`, `benchmarks/expression_evaluator.yml`, `benchmarks/binary_tree_sum.yml` |
| 4 (recursion & termination) | structural recursion, decreasing arguments | `benchmarks/recursion_fibonacci.yml`, `benchmarks/merge_sort.yml`, `benchmarks/gcd_lcm.yml`, `benchmarks/red_black_tree.yml` |
| 5 (multi-function & effects) | effect propagation across functions | `benchmarks/effect_*.yml` (5 benchmarks), `benchmarks/contract_*.yml` (8 contract benchmarks with `requires`/`ensures`), `benchmarks/state_machine_*.yml` |

**Workflow per problem:**
1. Read VeraBench JSON → identify entry_point + signature + test_cases
2. Find the closest-matching existing AILANG benchmark/example
3. Adapt: rename function to match `entry_point`, adjust signature to match VeraBench's signature, ensure test cases pass
4. Write to `solutions/ailang/<VB_id>.ail`
5. Verify via `ailang run --entry <entry_point>` locally before commit

For the ~20 problems where no close AILANG analog exists, fall back to the original strategy: **use AILANG's eval harness to generate candidates** via claude-haiku-4-5 with the AILANG teaching prompt (cost: ~$0.10 total). Filter for passes, human-review for correctness, commit.

### 2. AILANG wrapper builder (~3h)

Add `_build_ailang_wrapper(problem, baseline_path) → str` to `vera_bench/baseline_runner.py`:

```python
def _build_ailang_wrapper(problem: dict, baseline_path: Path) -> str:
    """Build an AILANG wrapper script that runs test cases.

    AILANG modules export functions and are invoked via `ailang run --entry main`.
    The wrapper imports the baseline module and calls `entry_point` for each
    test case, building a JSON output that the harness can parse.
    """
    entry_point = problem["entry_point"]
    test_cases = problem.get("test_cases", [])

    # AILANG module syntax — see prompts/v0.16.0.md for details
    module_name = baseline_path.stem
    lines = [
        "module wrapper",
        f"import {module_name} ({entry_point})",
        "import std/json (encode, jarr, jobj, jbool, jstr)",
        "import std/io (println)",
        "",
        "export func main() -> () ! {IO} = {",
    ]
    for i, tc in enumerate(test_cases):
        args = tc.get("args", [])
        expected = tc.get("expected")
        # Build per-test-case lines that call entry_point and compare
        args_str = ", ".join(_lit(a) for a in args)
        lines.append(f"  -- test case {i}")
        lines.append(f"  let r{i} = {entry_point}({args_str});")
        # ... compare against expected ...
    lines.append("}")
    return "\n".join(lines)
```

Plus update `_EXT = {"python": ".py", "typescript": ".ts", "aver": ".av", "ailang": ".ail"}`.

### 3. `run_ailang_baseline` function (~2h)

Mirror `run_python_baseline` / `run_aver_baseline`:

```python
def run_ailang_baseline(
    problem: dict, solutions_dir: Path, work_dir: Path, timeout: int = 30,
) -> ProblemResult:
    """Run an AILANG baseline solution via `ailang run --entry main`."""
    # ... write wrapper, invoke `ailang run --entry main`, parse stdout, return ProblemResult
```

### 4. Test on a small subset, then full 50 (~3h)

- Add AILANG to `vera_bench/cli.py` language choices
- Run tier 1 (10 problems) end-to-end; fix wrapper issues
- Run all 50; compare results against published Vera/Python/TypeScript numbers

### 5. Upstream contribution (~2h)

Submit PR to `aallan/vera-bench` from `sunholo-data/vera-bench` fork:
- `solutions/ailang/` (50 files)
- `vera_bench/baseline_runner.py` updates
- README update with AILANG row in the results table

---

## Risk-Driven Considerations

### What if AILANG scores significantly worse than Vera?

That's a legitimate finding — Vera is designed around De Bruijn slot refs and mandatory contracts; some VeraBench problems may exercise those specifically. AILANG's HM type system handles names; would expect parity on contract-light problems, possible degradation on the verification-heavy tier 4+ tests.

If AILANG scores 60–80% range across the tiers, that's competitive with TypeScript's published numbers. If it scores 90%+, that's a major validation. If it scores <50%, that surfaces real AILANG gaps the talk should acknowledge.

### What if VeraBench's reference solutions assume Vera semantics?

Possible — e.g. tier 3 problems may rely on De Bruijn match arms. AILANG would need different idiomatic translations. The `description_neutral` field exists precisely to allow language-agnostic specs, but how well it's been validated across language targets is uncertain.

Mitigation: review each tier's problem set before writing solutions; flag any problems that can't be faithfully translated.

### What about contracts?

VeraBench reports `verify@1` (Vera's Z3-verified contracts pass) separately from `check@1` and `run_correct@1`. For Python/TypeScript these are baseline (no contracts). For AILANG: we DO have `ailang verify` with Z3 contracts — we could attempt true `verify@1` parity with Vera on the problems that have `contracts` field populated.

This is the **most distinctive ailang-as-verification-camp finding** we could publish. Saving it for Phase 2.

---

## Phase Plan

**Phase 1 — Wire AILANG into VeraBench (~12h)**:
- 50 reference solutions in `solutions/ailang/`
- `_build_ailang_wrapper` + `run_ailang_baseline` in `baseline_runner.py`
- End-to-end test on tier 1, then full 50
- Report AILANG row alongside published Vera/Python/TypeScript numbers
- Output: PR to aallan/vera-bench from sunholo-data fork

**Phase 2 — `verify@1` parity (~8h)**:
- Translate VeraBench's `contracts.requires/ensures` to AILANG's `requires`/`ensures` syntax for problems where it's possible
- Run `ailang verify` on each; report `verify@1` alongside Vera's
- This produces the verification-camp-vs-verification-camp data point
- Output: extended results section in VeraBench README; potentially a paper-worthy comparison

---

## Open Questions

1. **Should this be a PR to aallan/vera-bench or live only in sunholo-data/vera-bench?** PR is the most-cited outcome; fork-only is faster but less impactful. Default: develop in fork, send PR once results are clean.
2. **Which AILANG version to pin against?** Most recent stable. VeraBench has versioning conventions (e.g. "VeraBench v0.0.11 vs Vera v0.0.108") — we'd cite "VeraBench v0.0.11 vs AILANG v0.22.x".
3. **Which models to test?** VeraBench published Kimi K2.5 + GPT-4.1 + Claude Opus 4 + Claude Sonnet 4 + GPT-4o. We could test the same model set; or focus on Claude-family (most relevant to AILANG's ecosystem); or run our own (we've been using claude-haiku-4-5). Decision in Phase 1.
4. **What's the publication strategy?** PR + README row is the floor; talk-week the talk can cite "AILANG on VeraBench tier 1: X%" once data is in.

---

## Success Criteria

**Phase 1**:
- AILANG appears as a target language in VeraBench
- All 50 problems have AILANG reference solutions that pass VeraBench's grader on at least Claude Sonnet 4 / Claude Opus 4
- Published comparison: AILANG vs Vera vs Python vs TypeScript on all 50 problems
- PR open against aallan/vera-bench

**Phase 2**:
- `verify@1` reported for AILANG on the problems with non-trivial contracts
- Side-by-side: AILANG Z3 verification vs Vera Z3 verification, same problem set
- A blog post or paper-style writeup of the head-to-head verification comparison
