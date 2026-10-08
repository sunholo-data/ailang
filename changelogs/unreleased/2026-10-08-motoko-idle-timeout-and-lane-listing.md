### Fixed — motoko executor enforces the idle timeout

The motoko executor never read `task.IdleTimeout`, so a model stream that stalled mid-response held
the run until the wall-clock bound. Measured 2026-10-08 on real `ailang_only` lane tasks: two runs
stalled (31 stream starts, 30 ends) and each waited out the full 40 minutes, where pi's executor
kills a silent run after its idle timeout. The executor now watches the growth of motoko's session
JSONL and stderr log, and kills a run that writes nothing for `task.IdleTimeout` (default 10
minutes, longer than pi's 3: tool calls and compaction write nothing meanwhile). The run is banked
as `idle timeout`, finish reason `timeout`.

### Added — the lane's `read` lists a directory

`ailang policy-tool`'s `read` op (behind `ailang_read` in both the pi and motoko `ailang_only`
lanes) now returns a sorted listing for a directory, with a trailing `/` on subdirectories and `.git`
omitted, instead of refusing. The lane has no shell and no other listing tool, and `ailang tree`
needs an `ailang.toml` at the root. In a monorepo, motoko once spent all 300 steps of a 3-line fix
writing its own directory lister. Paths outside the sandbox, including symlinks out, are still
refused.
