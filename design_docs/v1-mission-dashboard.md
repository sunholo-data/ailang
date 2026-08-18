# Mission Dashboard — V1
> Snapshot only; history lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.
> Namespaced at iteration 216: the shared `mission-dashboard.md` was one literal that every
> mission overwrote (frictions at iterations 212–215). Motoko's snapshot stays in its own file.

**Updated**: 2026-08-18 ~09:00 local (iteration 221)

## Now
- **v0.33.1** · `dev` **GREEN** on the required set. `origin/dev` **`e0be952be`**.
- **The five-iteration SonarCloud red got opened, and it was hiding a real hole.** One failed
  condition: 78.3% new-code coverage vs an 80% threshold. Its 44 uncovered lines decide the gate
  (denominator is only 203 lines) — and **nine of them are a capability gate**.
- **`9504393d0`'s capability fix had zero tests.** It closed a real bypass: prelude `println` wrote
  to stdout via `fmt` while `import std/io (println)` went through `RequireCap`, so `--caps` was
  evadable by *not* importing `std/io`. Neutering the gate at `5a3a59126`: mutant LANDED + BUILDS,
  and `./internal/... ./cmd/ailang/...` returned **rc=0, 100 packages ok**. The hole re-opened in
  full without redding one test.
- Pinned in `#770` → **`e0be952be`** (21 checks, zero not-green, 4/4 required). **No production
  code changed.** Swept all four production `registerBuiltins` sites, not the one that was found.
- **9 arms, 6 drills**, every inverse `-skip` rc=0. M6 — making `NewCoreEvaluatorWithRegistry` pass
  nil — reds *only* the arms the site enumeration created, which is what the sweep bought.
- Denied arms assert **nothing reached stdout** and the capability asked for was exactly `IO`; "an
  error came back" is satisfied by any failure and would have been a hollow pin.
- **Platform narrowing closed**: the ubuntu `test` log names all four new tests (12/12/16/4).
- **All three open nightly alarms closed** (`#769`, `#709`, `#649`): none has *ever* passed both
  trials (0/16, 0/24, 0/16 nights, 201-full-pass control) — capability gaps, not regressions.
- **Sweep orphans: still 3 of 15 dispositioned** — this iteration's pick was the red, not an orphan.

## Next
1. **`#687`** closes the mission-infra sweep lane — `⚠ Binary may be stale` is an mtime heuristic
   over CWD-relative dirs, so it mis-fires in every fresh worktree, including this loop's own.
2. **12 remaining sweep orphans** — then the language/stdlib group (`#688`, `#689`, `#662`,
   `#646`, `#644`), each ghost-disciplined at HEAD.
3. **`[world-DEMAND]`** — `ailang#764`: `serveapi` is an API seam but not a *dependency* seam
   (486 non-stdlib packages in its closure). Blocks World's item 5. Needs a design doc + quorum.

## Loop
- Controller `claude:claude-opus-5`, inline. No designer / planner / executor / evaluator / quorum /
  GPU lane fired — mechanical, well-specified test work. **metered $0.00**.
- Billing CLEAN; gh `sunholo-voight-kampff`; running skill byte-identical to origin.
- ⚠ **Codex quota dry until 2026-08-20 05:34** — V1 remains on a single controller lane.

## Parked on Mark (issue rotates weekly — see `~/.ailang/state/mission-gh-issue`, now `#745`)
- **11 OPEN ledger rows**: `D-1`, `D-2`, `D-8`–`D-14`, `D-COV-1`, `D-18`.
  Generate the current list with `scripts/mission_decisions.sh --open` — never quote a range.
- **`D-COV-1` just got cheaper to answer.** It forbids a coverage sprint until you choose LOCALITY
  or EXECUTION. This iteration shows the *gate* is a usable finder even under that park: it named
  44 lines, nine of which were an untested security guard. Worth deciding.
- **`D-18` still has a live cost**: two missions share this repo with no claim protocol, so a red
  blocking both gets fixed twice (`#758`/`#759`, 3m49s apart). A: claim file under
  `~/.ailang/state/` · B: a `[claimed]` marker on the tracking issue · C: accept it as cheap.
- Nothing new parked this iteration.
