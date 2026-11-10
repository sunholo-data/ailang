# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`._

**Last update:** 2026-11-09 · ATTENDED re-cut (`D-64`) · **Goal distance:** **N = 1 open bar-v3 gate item** (was 7; G1+G3 closed by `D-65`, G4 closed on the prod domain, G2 closed by #2184 + #2185) — only G5 remains, at release (unit changed from D-51's "design docs remaining", last N = 12 on 2026-09-14)
**Latest release:** v0.57.4. The V1 launchd loop has been PAUSED since 2026-09-08 (`~/.ailang/state/mission-control.disabled`) and stays paused; the attended session is the orchestrator and dispatches the work to subagents.

## Roadmap accepted 2026-11-10 (`D-66`) — v1.0.0 is NOT the next tag
An independent readiness review found work bar v3 does not count: type-soundness holes, a prover/runtime
mismatch on integer overflow, a lossy `show`, and stdlib names that cannot change after the promise.
Mark accepted the roadmap: **v0.58.x is the last breaking window → soak ≥ 7 days → rc.1 → v1.0.0.**
All ten roadmap decisions are ruled (`D-66`, `D-67`). Order and the ten design docs: [planned/v1_0_0/ROADMAP.md](planned/v1_0_0/ROADMAP.md).
The N below still counts bar-v3 items only; re-cutting the unit to cover the roadmap is not yet ruled.

## The bar (v3, "1.0 = a promise we can keep")
| Gate | Open | What closes it |
|---|---|---|
| G1 Stability promise ratified | 0 ✓ | re-draft `stability.md` against today's surface → Mark ratifies |
| G2 Zero open P0s | 0 ✓ | #1443 (`pure func` + effect row), #1134 (IFC labels across modules) |
| G3 Packages in the promise | 0 ✓ | Packages tier section + one published package as the worked example |
| G4 Cloud exec MCP Stable | 0 ✓ | tool surface listed Stable; prod on a custom domain (docparse precedent) |
| G5 Simplification Phase 5 | 1 | metrics diff fragment for the v1.0.0 release notes |

Tiering (D-64): `ailang messages` **Experimental**; coordinator / missions / approvals / dashboards / eval tooling **Internal**.
Published at release, not gated: fleet-tier pass rates, cost-per-verified-success baseline.

## In flight (2026-11-09)
| Gate | PR | State |
|---|---|---|
| G1+G3 | #2182 stability re-draft | **CLOSED** — merged `71ec9bd36`, ratified by Mark (`D-65`) |
| G2 #1443 | #2184 `pure` ⇒ empty row (BREAKING) | **CLOSED** — merged `8d582b976`; teaching surfaces verified clean (active prompt v0.16.12) |
| G2 #1134 | #2185 cross-module IFC (BREAKING) | **CLOSED** — merged `bdbeebfea` (cache key v10; structural list/record/tuple label checks); IFC sites fixed: email-parse#43 merged, ✓ email 0.10.2 published; ✓ email-parse#44 merged, ✓ eparse 1.11.3 published — #2185 downstream DONE; follow-ups #2187, #2196 (P1) |
| G4 | multivac#12 issuer → `executor.ailang.sunholo.com` | **CLOSED** — promoted dev→test→prod `e752cc2`, metadata names the host, `ailang mcp check` auth/transport PASS. Not gated: 22 docs tools lack `title` + read-only hints (A7, directory readiness) |
| docs | #2181 runbook, #2183 stale sweep | MERGED (`d79a5d3be`, `5bda6c3eb`); charter links re-pointed after #2186's moves |

**Cache-key collision:** #2184 and #2185 both bump the compile-cache key v8→v9; the second to merge must go to v10.

**#2184 release order** (measured old-vs-new binary across downstream repos; merging is safe, RELEASING is not):
1. ✓ ailang-packages#183 merged `685490732` → ✓ published `sunholo/specstore` 0.3.1
2. ✓ email-parse#40 merged `d7b0669be` → ✓ published `sunholo/email` 0.10.1 → ✓ email-parse#41 → ✓ published eparse 1.11.2; ✓ daneel#417 merged `0d6e67025`
3. ✓ docparse#263 merged `7950010ee` (before docparse's `AILANG_VERSION` pin, now v0.57.1, moves past a #2184 release)
4. ✓ motoko fork#6 merged `597a6fbf9` (106 breaks → 0; its 2 red CI jobs pre-date it — base red since fork#4/#5: stale `CostRates` in `scripts/verify_skills_startup.ail:133`, hook-scope yield pin at 20 vs 21 extensions, 5 moved session.ail clock anchors) → fast-forward `~/dev/mk-20261027` + cloud `MOTOKO_COMMIT` — BEFORE the rig binary moves to a #2184 release
5. then #2184 merges and ships. Upstream arniwesth/motoko_agent has the same 36 sites — tell Arni.

**#2185 IFC sites (email-parse, owner decides, not auto-fixed):** `packages/email/api.ail:66` (`<arg> ⊔ <sqlsafe>` into `sqlEscape`), `packages/eparse/attachments.ail:604` (already-escaped value re-escaped). daneel: 0 IFC breaks.

## Parked on Mark
