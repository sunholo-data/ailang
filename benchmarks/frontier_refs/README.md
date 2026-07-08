# Frontier benchmark reference solutions

Verification artifacts for the 8 frontier-class stretch benchmarks authored for
[M-EVAL-FRONTIER-TIER](../../design_docs/planned/m-eval-frontier-tier.md)
(2026-07-08, Claude Fable 5). **These are NOT shipped to models** — they exist so
`expected_stdout` values can be re-derived and re-audited instead of trusted.

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
