### Fixed — mission slot UNCLASSIFIED verdict

Unknown heartbeat labels at rc=0 now emit `UNCLASSIFIED at=<label>` and use the existing
bounded, episode-gated phase-2 notice. `REAPED` matches only gate-0/1/2/3/3b/4/5;
invalid gate-like labels are UNCLASSIFIED. Non-zero CRASHED/KILLED verdicts and the
PAUSED-NO-CAPACITY override retain their existing behavior (M-FLEET-SLOT-UNCLASSIFIED).
