# M-EVAL-GAP-FIXES: Eval-discovered AILANG and harness improvements

**Status**: Planned
**Target**: v0.15.0
**Priority**: P2 (Medium — quality-of-life, not blocking; close prompt + harness gaps surfaced by v0.14.2 baseline)
**Estimated**: 1.5–2 days (parser fix is the largest piece; the harness ticket is a single-file investigation; the prompt update is already shipped against v0.12.1.md and only needs to be folded into the next prompt revision)

## Context

The v0.14.2 baseline (1,178 runs across `extended_suite`, `harness_suite` incl. Pi, and `lang_harness_suite`) surfaced three concrete improvement areas. The eval-gap-finder breakdown by mode (0-shot / +Repair / agent) and per-harness made each one localisable.

Two of the three are **prompt + harness** issues that don't touch language semantics. One is a real parser limitation already noted in `.claude/rules/ailang-syntax.md` ("Match in blocks: Known parser bug — extract to helper function") that we should now tackle as a focused fix.

## Findings from v0.14.2

### 1. Match-block parser corner case (P2)

`.claude/rules/ailang-syntax.md` already lists this as a known bug. Models repeatedly write:

```ailang
func printAll(xs: [int]) -> () ! {IO} =
  match xs {
    [] => (),
    s :: rest => {
      println(show(s));
      printAll(rest)
    }
  }
```

This works. But the prompt's **Block Style vs Expression Style** rule (`{ }` → `;`, `=` → `in`) is bidirectional, and the most common LLM failure is writing semicolon-separated lets inside a `=`-bodied function:

```ailang
func bad() -> () ! {IO} =
  let x = parseInput();
  let y = transform(x);
  println(y)
-- → PAR_NO_PREFIX_PARSE: unexpected token: ;
-- → PAR015: bare assignment not supported (missing 'let' keyword)
```

The prompt has been updated (Rule #3 of "Required Program Structure" in `prompts/v0.12.1.md`) to make the bidirectional rule punchier, but the **underlying parser limitation** that the rules file flags as "Known parser bug — extract to helper function" should be characterised more precisely in this sprint:

- What exactly is rejected? Are nested `match { ... }` arms the trigger?
- Is the workaround "extract to helper" still required, or can the parser be relaxed to accept the pattern that the prompt rule says works?

### 2. `++` for strings keeps surfacing despite the v0.13.0 disambiguation (P3)

After v0.13.0's M-CONCAT-DISAMBIG (commit 99f76ec7), `++` is documented as list-only and the prompt teaches `${interp}`/`concat`/`join` for strings. Yet `contract_sorted_merge` × claude-opus-4-7 still produced:

```
type error in benchmark/solution (decl 6): ++ operator at benchmark/solution.ail:69:37:
  `++` is for lists only. For strings use "${expr}" interpolation, concat([parts]), or join(sep, parts).
```

The error message is excellent — but the model still emitted the wrong code. This is **not a prompt gap**; the prompt is clear. It's a model-knowledge-gap that no amount of prompt text will fix because models pattern-match from Haskell/Elixir habits.

**Action**: leave `contract_sorted_merge` in `stretch` as a deliberate "model knowledge regression guard". Don't fix it; it's catching a real pattern.

### 3. Pi harness regression on `config_file_parser` (P2 — first Pi failure observed)

In v0.14.2 cross-harness data, Pi achieved 97% AILANG / 0 API errors — the cleanest harness across all benchmarks. EXCEPT one: `config_file_parser` × AILANG × Pi failed in 100% of attempts where every other harness succeeded.

This is the first regression observed against Pi's otherwise-perfect record. Likely candidates:

- Pi's NDJSON event parser may mishandle the multi-line stdout that `config_file_parser` produces (model writes a config file and reads it back; each line goes through stdout).
- Pi's tool-call output stripping may interact with the benchmark's expected output format.

**Action**: capture one full Pi failure transcript on `config_file_parser` and inspect against the NDJSON parser tests in `internal/executor/pi/pi_test.go`. If reproducible, add a fixture and a fix. Do NOT block v0.15.0 release on this — it's one benchmark out of 33.

## Scope (M1–M3)

| Milestone | What | Files |
|---|---|---|
| **M1** | Fold prompt update (v0.12.1 Rule #3) into the next active prompt revision and add a regression test in `internal/eval_harness/langreg/ailang.go` exemplifying both block-style and expression-style let chains | `prompts/v0.12.x.md`, `internal/eval_harness/langreg/ailang.go` |
| **M2** | Pi `config_file_parser` reproducer: capture transcript, write a regression test in `pi_test.go`, file follow-up issue if root cause is non-trivial | `internal/executor/pi/pi.go`, `pi_test.go`, `pi/testdata/` |
| **M3** | Investigate and either fix or document the match-arm-block parser corner case. If fix is invasive, write a minimal reproducer + `examples/limitations/match_arm_block.ail` and update `docs/LIMITATIONS.md` | `internal/parser/`, `examples/limitations/`, `docs/LIMITATIONS.md` |

## Out of scope

- Any change to the `++` disambiguation. v0.13.0's behaviour is correct; the prompt is correct; the error message is correct. Models being wrong is not a language bug.
- Saturation rotation (effect_composition, effect_tracking_io_fs, expression_evaluator, higher_order_functions, merge_sort, state_machine_elevator). Tracked separately under benchmark-manager curation.
- New language features. This sprint only closes prompt + harness + parser gaps that the eval surfaced.

## Success criteria

1. **M1**: After the v0.13.x prompt revision, the v0.15.0 baseline's `polymorphic_ord_defaulting` and `config_file_parser` AILANG 0-shot rates rise by ≥30pp (current 17% / 60%).
2. **M2**: Pi `config_file_parser` × AILANG passes ≥1 of 6 attempts (currently 0/6) OR the failure mode is documented with a workaround.
3. **M3**: Either the parser accepts the documented match-arm-block pattern, OR `examples/limitations/match_arm_block.ail` exists with the documented helper-extraction workaround.

## Cost / risk

- **Low**: M1 is text edits. M2 is a single fixture + maybe one file change. M3 is the only one with parser risk; if the fix proves non-trivial, the design escapes to a follow-up doc.
- This is mainly **eval hygiene** — closing 0-shot prompt gaps and one harness regression, not new functionality.
