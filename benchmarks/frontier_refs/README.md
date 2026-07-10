# Frontier benchmark reference solutions

Verification artifacts for the frontier-class stretch benchmarks authored for
[M-EVAL-FRONTIER-TIER](../../design_docs/planned/v0_29_0/m-eval-frontier-tier.md)
(2026-07-08, Claude Fable 5; wave 2 added after the sonnet probe showed wave 1
sat at the top of stretch). **These are NOT shipped to models** — they exist so
`expected_stdout` values can be re-derived and re-audited instead of trusted.

Wave 2 (delegation-proof difficulty — the model must count/construct/derive at
generation time): `emit_exact_bytes`, `digitless_constants` (both graded via
`source_constraints`), `commonmark_emphasis` (every expected output
cross-validated against the `cmark` reference binary on NOVEL vectors),
`binary_strings_1e18` (n=10^18 makes O(n) DP impossible in-budget in BOTH
languages; refs run <1s via the O(log n) route).

Wave 3 (self-reference — designed frontier logic-error traps, exploiting the
iterative-authoring vs one-shot asymmetry): `quine` (graded `grading: quine`:
stdout must equal the submitted source; the AILANG ref was built by scripted
fixpoint construction — run `python3`-style iteration is exactly what one-shot
models don't get) and `emit_exact_bytes_varied` (512 exact bytes, no 3+ char
runs — uniform-padding tricks banned, so the byte-count must be done over
varied text). NOTE: the `quine` refs are verified by running them and
diffing stdout against their own file content, not against a YAML
expected_stdout (a quine has none).

Each benchmark has:

- `<id>.py` — the Python reference that **computed** the benchmark's
  `expected_stdout`. Several also contain divergence checks proving that
  plausible-wrong implementations (classic LFU bugs, greedy non-backtracking
  resolver, `fnmatch` delegation, peek-instead-of-pop rollback) produce
  DIFFERENT output — the property that makes the benchmark discriminate.
  Run with `python3 <id>.py`; divergence checks print to stderr.
- `<id>.ail` — a hand-verified AILANG solution producing byte-identical output
  (`ailang run --caps IO <id>.ail`), proving the task is achievable in AILANG
  under the 30s execution budget.

## Re-verify everything

```bash
for f in *.ail; do
  echo "== $f"; AILANG_RELAX_MODULES=1 ailang run --caps IO "$f" 2>/dev/null | grep -v "^→\|^✓"
done
for f in *.py; do
  echo "== $f"; python3 "$f" 2>/dev/null
done
```

Outputs must match the `expected_stdout` in the sibling `benchmarks/<id>.yml`.
Re-run this after any output-normalization or grader change (M-EVAL-OUTPUT-NORMALIZE)
before touching the YAML files.

## Maintenance notes

- `stream_lcg_topk` uses N=2000 deliberately: the AILANG reference runs ~2s;
  N=5000 ran ~13s, too close to the harness's 30s budget for less-optimal
  model solutions (would fail on perf, not logic — guardrail violation).
- `bytecode_vm_trace.ail` deliberately matches `nth`'s Option with the `None`
  arm FIRST and without importing `std/option` — the exact shape that used to
  silently take the None arm for every value (#323, fixed: an uppercase
  identifier in pattern position now always elaborates as a constructor
  pattern). Keep it that way; it doubles as an end-to-end regression check.
- The AILANG solutions dodge known parser footguns (nested `match` in arms is
  parenthesized; record updates `{r | f: v}` are parenthesized after `then`).
  Models won't know to do this — which is part of what the AILANG side of the
  benchmark measures.
