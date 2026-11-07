### Changed — Gate 3b allows a bounded 60-minute CI poll (2026-11-03)

Raise the mission-control Gate 3b CI poll deadline to 3600 seconds and its background Bash tool timeout to 3800000 milliseconds in both skill mirrors. CI has a median around 15 minutes, with a tail beyond 30 minutes including queue wait. Standing rule 6 names Gate 3b as an exception to its 30-minute default; expiry still parks the item for review. Proposed for decision D-FLEET-21 option A; not live until that ruling.

Decision: [D-FLEET-21 option A](../../design_docs/fleet-mission.md).
