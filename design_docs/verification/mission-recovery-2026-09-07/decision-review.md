# Blocked decisions — attended review, 2026-09-07

Status: proposed rulings for Mark, not resolutions. Deployment authorization does not answer
these product/security/process choices. None was acknowledged or marked resolved by this review.

Sources: merged V1, Motoko and Docs charter ledgers; World working charter; current weekly
threads V1 #1072, World #129, Motoko #1078 and Docs #979. There are 12 items: 10 in the inspected
merged/working ledgers, plus D-60 and D-WORLD-35 carried by pending records. Avoid silently losing
those two because a record PR has not merged. Docs' weekly issue title is still August 31.

| Mission / decision | Recommended ruling | Why / condition |
|---|---|---|
| V1 D-55 — cache trust scope | C: ship accidental-corruption correctness with explicit separate adversarial-hardening work | Fix silent wrong-program execution without claiming a hash proves compiler origin. Keep untrusted cache decoding outside the accepted guarantee. |
| V1 D-56 — author/reviewer collision | Exclude the author’s entire origin vendor | The offered A substitutes another OpenAI model and therefore does not satisfy provider independence. Route remaining/replacement vendors explicitly; never silently count an absent judge as a pass. |
| V1 D-57 — cache directory names | A: retain limited readable prefix plus hash | Treat it as a hint, not injective identity; stamp validation remains required. Smaller bounded change than another naming redesign. |
| V1 D-58 — Pi content comparison | A: one scoped designer revision and quorum | Settle failure ordering, bounded comparisons and alias handling; no gate bypass. |
| V1 D-59 — capped shell candidate | A: fresh scoped iteration only after SMT prerequisite is green | Repair the temp residue and obtain a new independent evaluation; do not grant a fourth round to the old sequence. |
| V1 D-60 — notification recovery | B: bound direct/drain/GitHub calls and repair their tests | Resolve the live timeout exposure with the observation-test failure; preserve bounded design/plan/evaluation. Decision is in unmerged #1073. |
| World D-32 — exposed credential | Rotate/revoke and replace after confirming the owning service/account | Explicit account action required; no secret value needs to be posted to a thread or review document. Not part of pin deployment. |
| World D-33 — prose-lint redesign | DEFER for this week's run window | Keep the item parked and use ready work rather than adding another unproven prose inference rule. |
| World D-35 — docs-only record merges | A: allow if application gate passes and other reds are demonstrated inherited on the exact comparison | Scoped to record-only PRs; no blanket permission for code changes or bypass of required checks. This keeps decision ledgers current. |
| Motoko D-MOTOKO-CARVEOUT-1 | B: overrule the carve-out, finish the residual audit and run a fresh quorum | An unperformed requested audit is not a wording correction. Keep generator/judge separation. |
| Docs D-4 — docs-search design exception | A, conditional on verifying every specified correction before planning | Scoped one-time approval; no general widening of the carve-out and no implementation approval bypass. |
| Docs D-5 — eval-input design wording | A: correct both claims and proceed to planning | Record bypass-only skip reporting and unconfirmed docx attribution accurately; no fifth wording-only quorum needed for this scoped ruling. |

## First batch presented for approval

D-56 provider independence separately; D-60 B, Motoko B and World D-35 A as a reliability batch.
Remaining decisions stay open pending Mark's answers. Use scripts/mission_answer.sh with actual
attended provenance when answers arrive, then validate the relevant ledger and publish it to the
ref each loop reads. A bot comment alone is not an allowlisted human directive.

## Useful week of evidence

Count accepted product outcomes separately from harness repairs and parked decisions. Record
provider/role/actual wire route, stage and failure class, elapsed time, metered/estimated spend,
retries and manual intervention. Keep the current cadence while World is the single pin canary.
A successful preflight or an execution_completed receipt is not accepted product progress.
