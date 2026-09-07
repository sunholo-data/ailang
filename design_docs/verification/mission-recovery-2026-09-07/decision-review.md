# Blocked decisions — attended review, 2026-09-07

Status: RESOLVED under Mark's explicit attended delegation on 2026-09-07: "please make the rulings so we are all unblocked". The recommendations below were recorded with their conditions in the live ledgers. No inbox acknowledgements were made.

Sources: merged V1, Motoko and Docs charter ledgers; World working charter; current weekly
threads V1 #1072, World #129, Motoko #1078 and Docs #979. The initial review found 12 items. Publication reconciliation found a thirteenth, D-WORLD-34, in the older pending World record #127. All three pending-record decisions (D-60, D-WORLD-34 and D-WORLD-35) were restored before resolution. Docs' weekly issue title is still August 31.

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
| World D-34 — obsolete fleet diagnostic | A: retire the obsolete local diagnostic and fixtures under DE-FORK | Shared fleet verification belongs to AILANG; preserve World application verification and complete the scoped CI-repair design and gates. |
| World D-35 — docs-only record merges | A: allow if application gate passes and other reds are demonstrated inherited on the exact comparison | Scoped to record-only PRs; no blanket permission for code changes or bypass of required checks. This keeps decision ledgers current. |
| Motoko D-MOTOKO-CARVEOUT-1 | B: overrule the carve-out, finish the residual audit and run a fresh quorum | An unperformed requested audit is not a wording correction. Keep generator/judge separation. |
| Docs D-4 — docs-search design exception | A, conditional on verifying every specified correction before planning | Scoped one-time approval; no general widening of the carve-out and no implementation approval bypass. |
| Docs D-5 — eval-input design wording | A: correct both claims and proceed to planning | Record bypass-only skip reporting and unconfirmed docx attribution accurately; no fifth wording-only quorum needed for this scoped ruling. |

## Published attended rulings

- AILANG `dev`: `ccd2b4d36bc2ec8640c27e097a39c2c72879e885` (V1 six, Motoko one, Docs two).
- World `dev`: `565d0b257dc6f28fcd25479fccbfd334b1e420ea` (four).
- All four ledger validators passed; all four `--open` listings were empty after recording.
- World was idle and clean; its work checkout was fast-forwarded to the published ledger.
  The other loops fetch the default driver ref at their next fire; already-running iterations
  may retain their starting snapshot. No active iteration was restarted.
- GitHub accepted the AILANG documentation commit via the account's branch-rule bypass,
  reporting four required checks expected. This is recorded, not represented as passing CI.
- Credential rotation is authorized but not performed by this ledger edit. Conditional design,
  audit, prerequisite-test and independent-evaluation requirements remain real work.
- World still pins `f26d64666`; its full canary remains pending. The fleet has not adopted
  durable role-run, artifact acceptance or quota reservations merely because decisions closed.

## Useful week of evidence

Count accepted product outcomes separately from harness repairs and parked decisions. Record
provider/role/actual wire route, stage and failure class, elapsed time, metered/estimated spend,
retries and manual intervention. Keep the current cadence while World is the single pin canary.
A successful preflight or an execution_completed receipt is not accepted product progress.
