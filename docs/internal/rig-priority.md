# Cooperative GPU priority

Implements the handoff requested in #1136. This is local rig coordination, not a
coordinator capability, deployment change, model unload, or release requirement.

Priority callers create a file named by their live PID under
`$RIG_LOCK_DIR.priority/` (a sibling of the lock directory). They retain that
reservation across all stages, remove it on exit, and still acquire the real lock
before using the GPU. Dead requester files are ignored/cleaned. Daneel waits up to
1200 seconds per acquisition, then records a retryable deferral.

New evals respect pending reservations. A running `eval-suite` drains all active
trials at its scheduler gate, releases the lock, and waits until priority work is
done before reacquiring. A shell wrapper exports its owner PID and blocks waiting
for the child; the child restores that same ownership before returning. Shell
cleanup only removes its own lock. The filler also checks at chunk boundaries.
Cancellation fails closed; no further trial runs without a successful handoff.
Lock age alone no longer authorizes stealing from a live/unknown holder.

The current in-flight inference is never interrupted. An old already-running
binary cannot gain this checkpoint retroactively; it must finish its existing
invocation before the new runner applies. A single long trial can still delay
priority work up to its existing timeout.

For local activation without a fleet release, the filler and language nightly can
use `~/.local/share/ailang/rig-priority/ailang` built with `make build`, with adjacent
`VERSION` copied from `std/VERSION`. A version mismatch refuses evaluation instead
of banking results under the wrong release. `AILANG_EVAL_BIN` is an explicit
override. The reproducible nightly keeps its existing source-pinned build path;
it gains the gate when its pinned source includes this commit.

Validation: riglock race tests cover admission, handoff, between-stage exclusion,
dead requesters and cancellation ownership. Eval-gate tests drain two active
trials before yielding and fail closed on handoff error. Daneel's
`tools/test-rig-priority.sh` exercises both actual shell implementations in isolated
processes, verifying eval → two task stages → eval with no overlap.
