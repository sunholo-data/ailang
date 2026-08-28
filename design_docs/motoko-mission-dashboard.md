# Mission Dashboard — Motoko

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the mission log.*

**Last iteration**: 27 · 2026-08-28 · **LANDED** — first iteration since 23 to reach the queue head
(24, 25 and 26 were each preempted by a loop-health regression).

## What landed

Row **6h** — PR [#946](https://github.com/sunholo-data/ailang/pull/946), evaluator round 1
**PASS 94/100, ZERO blocking**. `openai.ChatStepResponse.Usage` was a **value type**, so an omitted
`usage` key unmarshalled to the zero struct and the guard issue #842 asked for was not merely missing,
it was **inexpressible**. Now `*ChatStepUsage`. Representation only — a 16-case differential harness
built by the judge shows byte-identical output vs the parent for every input including `usage:null`
and `usage:[]`. Policy (step 2) stays deferred on purpose. `internal/ai/ollama` parses through this
path, so the defect sat on our own eval rig.

## ⚠ The loop is still running stale code and cannot fix itself

This fire logged `DRIVER PIN FAILED` at **02:03:09** and executed the source clone at `e3ed9467f` —
now **205** commits behind `origin/dev` (172 one day ago, 152 the day before). Sprint work was done in
worktrees branched from `origin/dev`, so nothing shipped from the stale tree; the cost is that every
iteration pays a re-derivation tax and iteration 25's own fix has still never executed here.

**The predicate is sharper than "how far behind"** (measured this iteration, both clones, one command):
`git merge-base --is-ancestor ff0da7445 HEAD` — **NO** for motoko, **YES** for V1. V1 is *still 18
commits behind* and pins fine, so currency was never the condition; carrying the fix is. V1's last
`DRIVER PIN FAILED` was 2026-08-27 07:10 and it has fired cleanly since — it **recovered**. motoko
structurally cannot, because the tree that would have to read the fix is the one that is stale.

## Parked on Mark — one decision, and this is the sixth ask

**`D-MOTOKO-WORKDIR-2`** — grant *standing* authorization to reconcile the source clone to
`origin/dev` unattended when three predicates hold (0 ahead · no dirty file whose content differs from
origin · sha256-verified backup). Measured again this iteration: **0 ahead, 0 dirty in the clone and 0
across all nine worktrees**. One word: **yes** (standing) or **no** (keep asking).

## Queue

- **Next**: row **6i** — the production `run_lane` process-group kill is pinned by nothing
- then **6m** (new, filed this iteration): `cacheRead = usage.PromptTokensDetails.CachedTokens` has
  zero killers; `cache_usage_test.go` looks like coverage and reaches a different code path entirely
- then **6j** (`launchd drivers` arm 33 hangs on the runner), **6l** (pin bootstrap trap, blocked by
  the decision above), **7**, **8**
- **Parked, Phase-0 gated**: rows 10/11/12 — re-measured as a command this iteration, upstream
  `arniwesth/motoko_agent#154` still **OPEN** (control `#175` **MERGED**, negative control 404s)

## Loop posture

Cadence 12h · controller `claude:claude-opus-5` · executor `codex:gpt-5.6-sol` (probe rc=0, one run,
no fallback) · evaluator **sonnet** in its own worktree · no designer, no planner, no quorum.
Designer rotation pointer untouched at `claude:claude-fable-5`; **Fable unspent**.
Metered **$0.00** of $5 — every lane used was a quota bucket. No GPU, no `rig.lock`.

**dev CI**: one not-green, inherited and not ours — `SonarCloud Code Analysis`, non-required, first red
at `caea1f9e1` (V1's M-EVAL-ROLLING-ELO merge), conditions *64.2% coverage on new code* and
*B security rating on new code*. Handed to V1 with delivery asserted.
