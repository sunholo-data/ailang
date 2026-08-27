# Mission Dashboard — V1

_Snapshot after iteration 293 (2026-08-27). Overwritten each iteration; history lives in the charter STATUS block and the mission log._

## Latest
- **Release**: v0.34.0 · `origin/dev` @ `caea1f9e1`
- **In flight**: PR [#940](https://github.com/sunholo-data/ailang/pull/940) — the served teaching prompt now identifies its own version, and the writer stops propagating the stale header
- **Evaluator**: `sonnet` **PASS 97/100, zero blocking** — it out-drilled the controller, enumerating all 8 refusal branches and finding the joint mutant that silently ships a mislabelled prompt

## What iteration 293 found
- **`ailang prompt` was serving the RIGHT bytes under the WRONG name.** The external report ("serves v0.16.2 on a v0.34.0 binary") was wrong on mechanism — `active` is `v0.16.6` and the served bytes match it exactly. The defect is that the file's own line 1 said `v0.16.2`.
- **Systemic, with a named mechanism**: 15 of 54 prompt files misidentify themselves because `create_prompt_version.sh` copies the base file and never rewrites the header. 14 of the 15 are FROZEN and must stay as they are; only `v0.16.6` was editable.
- **The freeze gate's hole is wider than the queue row said** (planner measurement): a mutable version's `.md` can be stale, diverged, or absent in BOTH copies at rc=0 — and `//go:embed all:prompts` means a missing prompt compiles clean and fails only for users of a released binary.
- **dev CI red was infrastructure, not code**: a `sum.golang.org` HTTP/2 stream error in `Get dependencies`. Re-run on the byte-identical tree went green. No revert, no fix-forward.

## Next picks
1. `m-prompt-freeze-mirror-all-versions` — the EXTEND decision, design already written by iteration 293's planner
2. `m-string-charat-totality` — `charAt` panics where every sibling accessor returns `Option`; needs a breaking-change call
3. `m-dx-papercuts-docs-verify-parser` · `m-prompt-teaching-gaps-yaml` · `m-std-smt` (needs a design doc + quorum)

## Loop health
- **Routing**: controller `opus` · designer NOT SPAWNED (doc existed — **Fable diet UNSPENT**) · planner `opus` (`fail-closed:planner-lane-field-missing`) · executor `codex:gpt-5.6-sol` · evaluator `sonnet`, own worktree
- **metered = $0.00** of $5
- **Known lane constraint (new)**: `.agents/` is read-only under codex `--sandbox workspace-write`, so a milestone touching it cannot run in that lane
- **PATH binary drifts by design** — it was 39 commits stale this fire; build to a scratch dir with ldflags, never `make quick-install` mid-run

## Parked on Mark
- **`D-42`** is the only OPEN ledger row (standing authorisation for a local↔origin reconcile). Not exercised this iteration — local dev == origin/dev exactly.
- **No decisions are being asked this iteration.**

## Standing
- Non-required `SonarCloud` is red on new-code coverage (63.3%, needs 80%) and a B security rating. **Not attributable to any one merge** — the new-code period spans 2404 issues. Filed with two candidate dispositions.
