# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`._

**Last update:** 2026-11-09 · ATTENDED re-cut (`D-64`) · **Goal distance:** **N = 5 open bar-v3 gate items** (was 7; G1+G3 closed by `D-65`) (unit changed from D-51's "design docs remaining", last N = 12 on 2026-09-14)
**Latest release:** v0.57.4. The V1 launchd loop has been PAUSED since 2026-09-08 (`~/.ailang/state/mission-control.disabled`) and stays paused; the attended session is the orchestrator and dispatches the work to subagents.

## The bar (v3, "1.0 = a promise we can keep")
| Gate | Open | What closes it |
|---|---|---|
| G1 Stability promise ratified | 0 ✓ | re-draft `stability.md` against today's surface → Mark ratifies |
| G2 Zero open P0s | 2 | #1443 (`pure func` + effect row), #1134 (IFC labels across modules) |
| G3 Packages in the promise | 0 ✓ | Packages tier section + one published package as the worked example |
| G4 Cloud exec MCP Stable | 2 | tool surface listed Stable; prod on a custom domain (docparse precedent) |
| G5 Simplification Phase 5 | 1 | metrics diff fragment for the v1.0.0 release notes |

Tiering (D-64): `ailang messages` **Experimental**; coordinator / missions / approvals / dashboards / eval tooling **Internal**.
Published at release, not gated: fleet-tier pass rates, cost-per-verified-success baseline.

## In flight (2026-11-09)
| Gate | PR | State |
|---|---|---|
| G1+G3 | #2182 stability re-draft | **CLOSED** — merged `71ec9bd36`, ratified by Mark (`D-65`) |
| G2 #1443 | #2184 `pure` ⇒ empty row (BREAKING) | gated on the downstream order below |
| G2 #1134 | #2185 cross-module IFC (BREAKING) | hold release until email-parse's 2 IFC sites are resolved; follow-up #2187 (inferred rules, P1) |
| G4(b) | multivac#12 issuer → `executor.ailang.sunholo.com` | MERGED to multivac dev `e752cc2`; domain + cert LIVE; plan 0/4/0; awaiting test → prod branch promote |
| docs | #2181 runbook, #2183 stale sweep | MERGED (`d79a5d3be`, `5bda6c3eb`); charter links re-pointed after #2186's moves |

**Cache-key collision:** #2184 and #2185 both bump the compile-cache key v8→v9; the second to merge must go to v10.

**#2184 release order** (measured old-vs-new binary across downstream repos; merging is safe, RELEASING is not):
1. ✓ ailang-packages#183 merged `685490732` → publish `sunholo/specstore` 0.3.1
2. ✓ email-parse#40 merged `d7b0669be` → publish `sunholo/email` 0.10.1 → eparse dep bump → eparse 1.11.2; then daneel#417 out of draft
3. ✓ docparse#263 merged `7950010ee` (before docparse's `AILANG_VERSION` pin, now v0.57.1, moves past a #2184 release)
4. motoko fork#6 (106 breaks → 0) → fast-forward `~/dev/mk-20261027` + cloud `MOTOKO_COMMIT` — BEFORE the rig binary moves to a #2184 release
5. then #2184 merges and ships. Upstream arniwesth/motoko_agent has the same 36 sites — tell Arni.

**#2185 IFC sites (email-parse, owner decides, not auto-fixed):** `packages/email/api.ail:66` (`<arg> ⊔ <sqlsafe>` into `sqlEscape`), `packages/eparse/attachments.ail:604` (already-escaped value re-escaped). daneel: 0 IFC breaks.

## Parked on Mark
- G4: promote multivac `dev` → `test` → `prod` (carries only #12).
- Reject coordinator task-c0bf79fb `--no-retrigger` (PR #1790 already closed as superseded).
- Publishes: specstore 0.3.1, email 0.10.1, eparse 1.11.2.
