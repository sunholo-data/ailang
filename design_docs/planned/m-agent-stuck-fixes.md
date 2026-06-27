# M-AGENT-STUCK-FIXES: deterministic fixes for the errors that loop the agent

**Status:** In Progress (2026-06-27)
**Owner:** agent-ergonomics loop
**Parent:** [M-AGENT-ERGONOMICS](m-agent-ergonomics.md) — "one suggestion engine, three surfaces"

## Motivation — from data, not assumptions

The `docx_reimplement` benchmark spirals to `max_steps` in ~2/3 of runs (N=5 rig study, motoko / qwen3.6). `tools/analyze_stuck.py` decomposes each `max_steps` session by the repeated (path/line-normalized) check failure the model never fixes. Two **deterministic, reproducible** bugs dominate — both give the agent *no actionable path*, so it edits blindly until the step budget is exhausted:

1. **PAR999 parser panic** — `interface conversion: ast.Expr is nil, not *ast.Lambda`.
   In run `213a5d7e` the model's `docx_parser.ail` hit this and looped **87 steps (18→105)** — the single largest stall in the study. Root cause: `parsePureLambda()` (`internal/parser/parser_expr.go:500`) does `p.parseLambda().(*ast.Lambda)` with **no nil guard**. Any malformed `pure func …` makes `parseLambda` return a nil `ast.Expr`; the assertion panics; `recover()` (parser.go:139 / parser_file.go:16) catches it and reports PAR999 with the raw Go message. The agent can't act on `interface conversion: ast.Expr is nil`, so it never converges.
   Reproduced: `pure func`, `pure func(`, `pure func()`, `pure func(x:int)`, … **all panic**.

2. **IMP010 `'show' not exported by std/string`** — run `950f1b2c`, ×4, steps 73→116.
   The model repeatedly writes `import std/string (show)`. But `show`/`print` are **auto-imported builtins**. IMP010's "not exported by" wording sends the model hunting for the wrong fix, and it loops.

These are distinct from the AILANG **dialect** errors in the same files (`;` in expressions = PAR017, bare assignment = PAR015, let-in misuse = PAR_NO_PREFIX) which **already carry actionable messages**. Those belong to a separate, longer "dialect ergonomics" track. This sprint fixes only the two bugs where the agent currently gets *nothing to act on*.

## Milestones

### M1 — Parser never panics on `pure func` (PAR999 → clean error)
- Guard the `.(*ast.Lambda)` assertion in `parsePureLambda`: when `parseLambda` returns nil (it has already recorded the real parse error), return nil instead of asserting.
- Systemic audit done: `grep -rnE 'p\.parse[A-Za-z]+\(\)\.\(\*ast\.' internal/parser` → **one** site. No siblings.
- Regression test: malformed `pure func` variants → a normal PAR error, never PAR999 / "parser panic".
- **Acceptance:** `ailang check` on any malformed `pure func` emits a real parse error, never a panic.

### M2 — IMP010 for auto-imported builtins → "it's auto-imported; remove the import"
- When IMP010 fires for a symbol that is an auto-imported builtin (`show`, `print`, …), append: "`show` is auto-imported — remove it from the import list."
- **Acceptance:** `import std/string (show)` → IMP010 with the auto-import hint.

## Verification
- **Deterministic:** reproduce each trigger → confirm the new message (no panic; helpful hint).
- **Loop re-measure:** post-fix N=5 docx study; the PAR999 / IMP010 entries should disappear from `analyze_stuck.py`'s repeated-error set, and the `max_steps` rate should drop if these were genuinely load-bearing.

## Out of scope (next loop iterations)
- Dialect ergonomics (`;` / `let` / `let-in` traps — messages already exist; re-measure whether they still stall after M1).
- The broader "model struggles to emit valid AILANG dialect" finding — prompt + structural work, tracked under M-AGENT-ERGONOMICS.
