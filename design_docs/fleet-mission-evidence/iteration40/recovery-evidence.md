# Fleet iteration 40 — recovery evidence and resume checklist

All four operator-requested roles were spawned through native Agent with explicit GPT-6.1 Sol pins and fresh contexts. Their responsibilities were read-only recovery, not generating another implementation. Controller and roles reported no provider token counts. Metered cost $0; subscription consumption unknown. Designer resolver refused over-ration Anthropic; planner/executor recipe models were preserved with native transport under the explicit operator request. Sonnet evaluator failed to spawn, then Gate 3's fresh-context Sol fallback supplied an independent agent judge. No Astra used.

## Measurements

- Base: `4749956296d226f06d92dd2458b36843a92d48a3`. Latest attributed fleet branch: `fleet/i39-unclassified` at `10fdc22223c06d660a5769ccd604e0f06037acc8`, containing records 36–39.
- Designer read latest charter/authority and independently checked CI: no new design warranted; D-FLEET-16/17/18 remain OPEN. No further stacked building under iteration 39's resume instruction.
- Planner independently checked all PR states, exact heads and bases, remote dev, and allowlisted directives: zero directives among 10 comments since `2026-10-13T00:00:23Z`; watermark unchanged.
- Executor read source-gate implementation before running `make check-git-exec`: make rc 2, script rc 1; missing baseline path `internal/executor/managed_agents/devapi_workspace.go (2)`. AST and regex totals 37, fixture 2. Pin remained clean.
- CI 38617113857, test job 116034928121: same baseline failure, 48 skipped steps. Build and Release 38617113836: Windows failure, macOS cancelled, Ubuntu success. Docs workflow 38617113983: MDX/acorn failure at env-vars.md:252:256, three skipped docs-build steps, deployment skipped.
- V1 kill switch exists. Offending Go file and git-exec baseline are outside fleet scope. Generated docs are allowed, but durable generator fix is outside scope.
- Controller re-read branch protection: required contexts test, lint, build, docs-gate, UI build gate. Exact-SHA check-runs count 20: test/docs-gate failure, lint success, build skipped, UI result absent. Logs carry known-present PASS lines and the exact baseline error; log-fetch rc 0.
- Independent evaluator report: [judge-report.md](judge-report.md), PASS 95 for disposition only.

Temporary raw logs remain on the rig: `/tmp/fleet-i40-dev-test-474995629.log`, `/tmp/fleet40-executor-source-gate-1792031981.log`, `/tmp/fleet40-executor-test-116034928121.log`, `/tmp/fleet40-executor-doclog-1792031981.log`. GitHub job IDs above are the durable raw evidence. No raw streams containing credentials/personal data were banked.

## Resume checklist (planner)

1. Re-read exact remote dev SHA and human directives. Require test and docs-gate SUCCESS at that SHA; a D17 ruling may authorize a repair, but never bypasses CI. Until then preserve all three PRs/tickets; do not stack further builds without a ruling.
2. Update #1824 against green dev within the approved scope. Require all expected checks PRESENT/completed on the resulting PR head; substantive changes need independent re-evaluation.
3. Merge #1824; capture actual full merge SHA. Run done-gates and expected CI at that merge SHA before resolving `quorum:invalid-absent-on-quoted-literals` and `quorum:reviewer-invalid-json-discarded`.
4. Retarget/update #1825 to dev only after #1824 lands. Repeat head checks, merge-SHA done-gates/CI; resolve `quorum:artifact-dir-cwd-relative` only then.
5. Retarget/update #1828 to dev only after #1825 lands. Repeat gates; resolve `driver:unclassified-state-unreachable` only then.
6. Each resolve uses the real passing merge SHA on origin/dev and verifies removal from ticket open. No message reads or acknowledgments close fleet tickets.

At each merge: confirm running surface and pushed reach; `make test-launchd-drivers` under a scrubbed environment; healthy/degraded dry-runs under idle armed sibling or synthetic HOME, pinned `AILANG_DRIVER_PINNED=<merge SHA>`; record exact outputs and confirm real pidfiles/state unchanged. Nothing reloaded mid-iteration. Enumerate expected workflows and pin all polling to full SHA; missing/skipped automerge alone is not a green. Prior PASS 88/90/91 and branch done-gates are historical evidence only.

## Record handling

The first mission-worktree helper launch produced a pending/incomplete `fleet-i40-record`. No writes, cleanup, lock removal or repair were attempted there. A distinct `fleet-i40-record-verified` helper invocation retained its parent through a bounded wait and returned `ready rc=0`, then a clean status, before any record writes. Records append to existing PR #1828, with no new stack layer. Main-checkout dirty files and iteration-39 `.snap/` remain untouched.
