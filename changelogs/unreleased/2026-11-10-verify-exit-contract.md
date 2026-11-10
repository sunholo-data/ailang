### Breaking — `check --verify` / `ai-check` / `verify` fail when nothing was proved (M-V1-VERIFICATION-HONESTY M1)

`AILANG_Z3_PATH=/nonexistent ailang check --verify f.ail` exited 0 on a function that
violates its own `ensures`: with no solver it reported `"available": false` and nothing else.
All four spellings (`check --verify`, `ai-check`, `verify`, `verify --package`) now exit
through one function, `verification.ExitCode`, and agree on every case.

An **obligation** is a function with an `ensures` clause. A `requires`-only function is not one:
the verifier assumes a precondition while proving the function's own `ensures` and does not check
it at call sites, so there is nothing to prove. It stays in `results` (skipped,
`missing_postcondition`) but never moves the outcome or exit code, with or without Z3, and passes
`--strict`.

| Outcome | Exit | `outcome` |
|---|---|---|
| file does not type-check | 1 | `not_run` |
| no obligations (with or without Z3) | 0 | `no_obligations` |
| every obligation proved | 0 | `verified` |
| at least one proved + at least one skipped | 0, skipped functions on stderr and by name in `results`; 1 with `--strict` | `partial` |
| obligations exist, **none** proved, rest skipped | **1** | **`unproved`** (new) |
| any inconclusive (bounded recursion) | **1** (was 0); 0 with `--allow-inconclusive` unless nothing else was proved | `inconclusive` |
| any counterexample | 1 | `counterexample` |
| any solver error, `unknown` or timeout | 1 | `error` |
| obligations exist, Z3 missing | **1** (was 0 on `check --verify` / `ai-check`) | `unavailable` |

What you will now see:
- A file with contracts checked where Z3 is not found exits 1 with
  `Error: Z3 solver not found (<where it looked: AILANG_Z3_PATH=… or PATH and the usual paths>): N contract obligation(s) were not verified`.
- A file whose only contracts are skipped (effects, a higher-order parameter, a match on list
  patterns) exits 1 as `unproved`: `Error: no contract was proved: …`. Before this change such a
  file passed, except that `ai-check` reported a list-pattern contract as a verifier error.
- A recursive contract that is inconclusive under bounded unrolling exits 1. The examples that
  relied on it passing were changed to pass as documented: `recursive_verify.ail` and
  `per_function_depth_verify.ail` bound their inputs and declare `@verify(depth: N)` past the bound;
  `cross_module_functions_lib.ail` documents `--allow-inconclusive`.
- A contract on a function that matches on list patterns is a skip on every spelling (`ai-check`
  used to call it a verifier error, `verify` a skip).
- `verify` on a contract-free file without Z3 exits 0 (it refused up front).
- `available` always means "the solver was found", including on a failed check.
- Each command checks one file: `check --verify a.ail b.ail` used to verify `a.ail` and never look
  at `b.ail`; it is now a usage error (exit 2). Flags after the file (`check --verify f.ail --strict`)
  are honoured; they used to be dropped.
- `verify --json` on a file that does not compile prints a report with `outcome: not_run` and
  `compile_errors` (it printed nothing).

Migration: install Z3 wherever contracted code is checked, or point `AILANG_Z3_PATH` at it; pass
`--allow-inconclusive` where an inconclusive bounded-recursion result should pass; add `--strict`
where only a full proof should pass; run one file per invocation. `check --verify` and `ai-check`
gain `--strict` and `--allow-inconclusive`; `verify` gains `--allow-inconclusive`.

New JSON fields (additive; no key renamed or removed): `outcome`, `obligations` on the
`check --verify` / `ai-check` `verify` object; `outcome`, `obligations`, `available`,
`compile_errors` on `verify --json`. The legacy counters keep their meaning (`skipped` still counts
`requires`-only rows). `evidence_schema` stays `ailang.verification-evidence/v1`: it versions the
evidence block (numeric model, runtime-contract scope, reason codes), whose keys and meanings did
not change; the outcome is a separate, new key.

**Eval baseline boundary.** Banked `verify_ok` is not comparable across this change for rows whose
solution has a contract on a list-pattern function: those rows banked `verify_errors: 1` and
`verify_ok: false`, and now bank `verify_skipped: 1` and `verify_ok: true` (the harness's
`verify_ok` means "no violation found", and now also requires `check.passed`). Verified-success
(`isVerifiedSuccess`) is unaffected, since it already requires `skipped == 0`.

**Operational consequence, hosted `ailang_check`:** `docker/Dockerfile.executor-mcp` is
`distroless/static` with no z3, so `ai-check` there now exits 1 on contracted code. The tool
(`executor_tools/executor.ail`) handles that case explicitly: `ok` stays the type-check verdict,
`exit_code` stays ai-check's real status, two additive fields say what happened —
`verification` (the report's `verify.outcome`, on every `ailang_check` response) and
`verification_message` ("contract verification is not available on this hosted tool; type
checking passed", only when `verification` is `unavailable`) — stderr carries that message
instead of local install advice, and the call is logged as `ok_unverified` (a failed verification
is logged `verify_failed`; both were logged `ok`). Adding z3 to that image is a follow-up.
