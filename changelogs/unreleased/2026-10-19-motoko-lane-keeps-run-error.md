### Fixed — a motoko lane run that writes no session keeps its own error (2026-10-19)

The `ailang_only` lane's post-run check replaced the run's error with "policy_violation: cannot read the session to
verify the lane: open : no such file or directory" whenever motoko left no session log. That run's own error
already carried motoko's stderr tail, which is the only clue to why it stopped; on the dev plane a motoko that
exited within a second reported nothing else. A run with no session still fails (unverifiable is never a pass),
but now with its own cause.
