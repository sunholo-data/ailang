# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*

**Iteration 309** · 2026-08-31 · [REFUTATION] · evaluator `sonnet` **74/100 PASS**
**dev:** GREEN at `c523a8c82`; standing **non-required** SonarCloud red, inherited
**Thread:** [#972](https://github.com/sunholo-data/ailang/issues/972) · **Parked on Mark: none** (ledger 50 rows, 0 OPEN)

## Headline
Mark's D-50 authorized `execute sprint`. The loop measured the authorized unit and **did not land it**:
`3500db0a7` cherry-picks rc=0 / 0 conflicts and would still make **39 of 104** routing cells
permanently undispatchable — `managed_agents` 13, empty provider 13, `eval`/`eval-go` 12, `pi-go` 1.
`ValidateExecutionRoute` runs *before* `checkVariantProviderAgreement` in `Dispatch`, so M1 becomes the
outer gate on landing. Three independent instruments agree on 39/0. **Deployed blast radius is ZERO**
(35 live agents, all inside the accept set) — a latent trap, not an outage.

## Next
1. `m-coordinator-config-route-preflight` — **gates the above.** `config diff` validates `local` only,
   in the DIFFERS branch only, so deploy-code-with-unchanged-config validates nothing.
2. `m-coordinator-child-env-opencode-retry-storm` — M1 **and** M1r as ONE commit. Evidence branch
   `mission/iter309-route-authority-parity` @ `8c8c29864`, UNMERGED.
3. `m-registry-interface-hash-blind-to-signatures` — PRODUCT, external report, REAL at HEAD:
   `InterfaceHash` hashes no signature data, so a **breaking** change cascades as `patch`.
4. `m-probe-discovery-arm-nondeterminism` (#975) — needs a CI runner, not this laptop.

## Routing / cost
controller `claude-opus-5` · designer ROTATION `claude:claude-fable-5` ⇄ `pi:ollama/deepseek-v4-flash`
(fable pin ACCEPTED) · planner `opus` (`fail-closed:planner-lane-field-missing`) ·
executor `codex:gpt-5.6-sol` · evaluator `sonnet`, own worktree. generator≠judge holds.
metered **$0.2377** (two quorum rounds) of $5; all other lanes quota buckets.

## Watch
- **Goal block is PROVISIONAL** (iter-309): charter had no countable finish line. Placeholder =
  open queue rows, **53 `[NEXT]` + 2 `[PARKED]`**. Mark to ratify or replace.
- The shared skill tells controllers to read a quorum key (`absent_reviewers`) the artifact **never
  writes**; absence really lives at `reviewers[].present`. Instance 1 of 2.
