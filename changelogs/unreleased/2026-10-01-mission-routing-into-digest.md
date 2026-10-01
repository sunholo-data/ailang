### Changed — mission loops: model up/downgrades move from thread comments into the iteration digest

The driver no longer posts "🔁 Controller model" and "Executor/planner lane degraded" comments on
the mission bookkeeping thread. It exports `MISSION_ROUTING_NOTE` instead, and the controller's
Gate-5 digest carries it on the Cost row (`routing: …`, or "as configured"). The lane notice
still goes to `controlplane`. Fires that never reach an iteration (no usable controller, PAUSED)
still comment, because no digest follows them.
