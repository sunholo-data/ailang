# Iteration 71 — slot-verdict measurement (pi controllers since #1391, 2026-09-29T07:24Z)

Command: `for f in ~/.ailang/state/mission-*-slot-verdicts.log; do awk '$1>="2026-09-29T07:24"' $f | grep "controller=pi:"; done | awk '{print $2}' | sort | uniq -c`

```
   5 verdict=COMPLETED
   2 verdict=CRASHED_at=gate-3
   1 verdict=DIED-PRE-GATE-0
   1 verdict=KILLED_at=gate-5
 103 verdict=PAUSED-NO-CAPACITY_at=fired
   2 verdict=PAUSED-NO-CAPACITY_at=gate-1
   1 verdict=PAUSED-NO-CAPACITY_at=gate-2
  15 verdict=PAUSED-NO-CAPACITY_at=gate-3
   2 verdict=PAUSED-NO-CAPACITY_at=gate-3b
   1 verdict=REAPED_at=gate-3b
```

KILLED verdicts since then, all controllers:

```
2026-10-04T04:28:01Z verdict=KILLED_at=gate-5 rc=143 attempt=1/3 elapsed_s=5217 stamps=9 controller=pi:openrouter/z-ai/glm-5.3
2026-10-05T19:51:50Z verdict=KILLED_at=gate-3 rc=143 attempt=1/3 elapsed_s=3545 stamps=5 controller=claude:claude-opus-5-5
2026-10-13T02:01:55Z verdict=KILLED_at=gate-3 rc=143 attempt=1/3 elapsed_s=4064 stamps=5 controller=claude:claude-opus-5-5
2026-10-21T16:56:22Z verdict=KILLED_at=gate-3 rc=143 attempt=1/3 elapsed_s=5493 stamps=5 controller=claude:claude-opus-5-5
2026-10-21T20:16:04Z verdict=KILLED_at=gate-3 rc=143 attempt=1/3 elapsed_s=3075 stamps=5 controller=claude:claude-opus-5-5
```
