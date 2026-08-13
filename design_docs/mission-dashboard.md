# Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, overwritten by mission-control Gate 4 every iteration (history lives in
> the charter/log). Fresh session = THIS + MEMORY.md. Humans steer via the bookkeeping issue.

**Updated**: 2026-08-13 ~23:30 local (iteration 195)

## Now
- **v0.33.1** · `dev` @ `8e8447f51` — merge of PR #699. Gate 3b GREEN (SHA-addressed **21** checks,
  zero NOT-GREEN, **4/4** REQUIRED, `state=CLEAN`); count climbed 14→21 during the poll, so
  `pending=0` was **required, not inferred**.
- **`#698` fast-follow LANDED — the buildable two-thirds.** M1 pins the `CreateStage` pinned-ID
  retry guard that survived a landed, building mutation; M2 arms 4 of 5 unpinned error branches;
  M3 restores the orphaned sprint JSON. Tests + state artifacts only, **zero production code**.
  Evaluator sonnet **PASS 83/100**. `#698` stays OPEN — its part 1 is parked (below).

## Why the ratified item vanished (the iteration's headline finding)
- The prior sprint's M3 **task list** contains "Opt-in remote read"; its **acceptance criteria**
  contain five entries and **zero** mention read or remote (AC-section grep → **0**; controls:
  task list → **1**, AC bullets → **5**). Every AC passed on a milestone missing a third of its
  task list. **A task with no acceptance criterion is invisible to the gate.**

## Also found and fixed this iteration
- **Two `#698` claims are WRONG**: the retry branch is reachable *deterministically* (there is a
  `id TEXT PRIMARY KEY` as well as the composite UNIQUE), and the 5th error branch is
  **unreachable by construction** (`EvalAssessment` is 26 scalar fields — `json.Marshal` cannot fail).
- **`.gitignore:77` ignores `.ailang/` with no negation** — a NEW sprint JSON is skipped by
  `git add -A` **silently** (empty output, 0 staged) and hidden from `git status`. Almost certainly
  how the prior sprint's JSON ended up on a divergent branch. Every sprint JSON needs `git add -f`.
- **Evaluator caught a Windows-only regression I had shipped**: `os.UserHomeDir()` reads
  `USERPROFILE`, not `HOME`, so both new arms silently never saw their own input on Windows —
  failing for the *platform*, not the code. Fixed, re-drilled, Windows legs green.

## Parked on Mark (all on `#635`)
- `D-1`, `D-2`, `D-7`–`D-14` — no reply since the 2026-08-13T04:58Z watermark (0 of 48 comments).
- **NEW `D-15`** — `#698` part 1: how far should `--remote` reach, **`view`** or **`eval`**?
  Recommendation **`view`**. The ratified freeze says *every* consumer inherits the option; the code
  refutes that as wiring-only work (`QueryEvalResults` is `*Store`-only, and Firestore stores
  `eval_assessment` as an opaque JSON string, so its six `json_extract` filters cannot be served
  remotely without a schema change).

## Next (if nothing unparks)
- `#691` (embedded `exit()` panics the host — needs a one-word contract decision), then `#692`.
  `m-mapE-queryall-retention` (`#610`) stays infra-gated (needs duckdb CLI).

## Loop health
- ⚠ **The driver refused 15 fires today** — every Anthropic probe timed out 120s across all three
  preference models. 212 log lines today vs 29 on 08-12. The loop is up but firing intermittently.
