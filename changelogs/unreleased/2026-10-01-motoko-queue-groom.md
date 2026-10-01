### Changed — motoko mission queue groomed to product work

Every row in the motoko mission's queue was re-checked against motoko main
(`D-MOTOKO-GROOM-1`). The new order is 23 (KPI instrument), 22a (`context_limit`
resolves to 0), 24 (`ailang_tools` on a set with headroom), 27 (upstream hygiene:
12 carried commits down to what must stay ours), 20 (cloud executor in test,
parked on the release), then 25 (executor-lane trial). fmt (22b) and R3 (9) are
parked. Row 22 was split in two, and row 21 folded into row 27. The fork-era
loop-health rows 6t, 17/18 and 19, and the orphaned 6s probe, left as fleet
tickets. 6m and the 96,908-char eval teaching prompt moved to the AILANG backlog
in PROGRAM.md. A new guardrail: harness defects this loop hits go to fleet as
tickets and never become rows in this queue.
