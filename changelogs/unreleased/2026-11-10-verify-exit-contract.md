### Breaking — `check --verify` / `ai-check` / `verify` fail when nothing was proved (M-V1-VERIFICATION-HONESTY M1)

`AILANG_Z3_PATH=/nonexistent ailang check --verify f.ail` exited 0 on a function that
violates its own `ensures`: with no solver it reported `"available": false` and nothing else.
All four spellings (`check --verify`, `ai-check`, `verify`, `verify --package`) now exit
through one function, `verification.ExitCode`, and agree on every case:

| Outcome | Exit | `outcome` |
|---|---|---|
| file does not type-check | 1 | `not_run` |
| no contracts (with or without Z3) | 0 | `no_obligations` |
| every contract proved | 0 | `verified` |
| proved + skipped | 0, skipped functions listed on stderr; 1 with `--strict` | `partial` |
| any inconclusive (bounded recursion) | **1** (was 0); 0 with `--allow-inconclusive` | `inconclusive` |
| any counterexample | 1 | `counterexample` |
| any solver error, `unknown` or timeout | 1 | `error` |
| contracts exist, Z3 missing | **1** (was 0 on `check --verify` / `ai-check`) | `unavailable` |

What you will now see:
- A file with contracts checked where Z3 is not installed exits 1 with
  `Error: Z3 solver not found: N contract obligation(s) were not verified` and the install
  command on stderr.
- A recursive function whose contract is inconclusive under bounded unrolling exits 1. Three
  in-repo examples do this: `examples/runnable/contracts/recursive_verify.ail`,
  `per_function_depth_verify.ail` and `cross_module_functions_lib.ail`.
- A skipped contract (effects, a higher-order parameter, an unsupported construct) still exits
  0, but stderr lists it: `1 contract(s) not verified (pass --strict to fail on these): g (Function "g" has effects)`.
- `ai-check` / `check --verify` used to report a contract whose body matches on list patterns
  as a verifier *error* (exit 1) while `verify` skipped it. Every spelling now skips it.
- `verify` on a contract-free file without Z3 now exits 0 (it refused up front).
- `available` now always means "the solver was found", including on a failed check (it read
  `false` there, which looked like "no solver").

Migration: install Z3 wherever contracted code is checked (`brew install z3`, `apt install z3`);
pass `--allow-inconclusive` where an inconclusive bounded-recursion result should pass; add
`--strict` where only a full proof should pass. `check --verify` and `ai-check` gain both
flags; `verify` gains `--allow-inconclusive`. New JSON fields (additive; no key renamed or
removed): `outcome`, `obligations` on the `check --verify` / `ai-check` `verify` object, and
`outcome`, `obligations`, `available` on `verify --json`.

**Operational consequence:** images that run `ai-check` without z3 now fail on contracted code
instead of passing silently. `docker/Dockerfile.executor-mcp` (the hosted `ailang_check`
executor tool) is `distroless/static` with no z3: after this release, contracted code sent to
`ailang_check` returns exit 1 with `outcome: unavailable` until that image ships z3. The
`agent-*` images inherit z3 from `docker/Dockerfile.agent-base` and are unaffected.
