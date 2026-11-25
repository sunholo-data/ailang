### Fixed — Mission base records reject the wrong repository (2026-11-25)

Mission gates call the pinned driver's base helper from the mission repository. When `MISSION_REPO`
is set, the helper checks the working repository's GitHub origin before reading a base ref or
appending a record. [Fleet iteration 138 plan](../../design_docs/fleet-mission-evidence/iteration138/plan.md)
