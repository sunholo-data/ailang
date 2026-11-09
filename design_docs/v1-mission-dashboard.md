# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History: `v1-mission.md` (STATUS) + `v1-mission-log.md`._

**Last update:** 2026-11-09 · ATTENDED re-cut (`D-64`) · **Goal distance:** **N = 7 open bar-v3 gate items** (unit changed from D-51's "design docs remaining", last N = 12 on 2026-09-14)
**Latest release:** v0.57.4. The V1 launchd loop has been PAUSED since 2026-09-08 (`~/.ailang/state/mission-control.disabled`) and stays paused; the attended session is the orchestrator and dispatches the work to subagents.

## The bar (v3, "1.0 = a promise we can keep")
| Gate | Open | What closes it |
|---|---|---|
| G1 Stability promise ratified | 1 | re-draft `stability.md` against today's surface → Mark ratifies |
| G2 Zero open P0s | 2 | #1443 (`pure func` + effect row), #1134 (IFC labels across modules) |
| G3 Packages in the promise | 1 | Packages tier section + one published package as the worked example |
| G4 Cloud exec MCP Stable | 2 | tool surface listed Stable; prod on a custom domain (docparse precedent) |
| G5 Simplification Phase 5 | 1 | metrics diff fragment for the v1.0.0 release notes |

Tiering (D-64): `ailang messages` **Experimental**; coordinator / missions / approvals / dashboards / eval tooling **Internal**.
Published at release, not gated: fleet-tier pass rates, cost-per-verified-success baseline.

## In flight (2026-11-09)
Subagents dispatched for G1+G3 (tier re-draft), G2 (#1443, #1134), G4 (domain mapping plan), and a stale-docs sweep.

## Parked on Mark
- G1 ratification (after the re-draft lands).
- G4 domain choice + apply (outward-facing; Terraform-only deploy).
