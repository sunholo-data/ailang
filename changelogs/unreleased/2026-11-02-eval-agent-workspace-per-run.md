### Fixed — agent-mode trials no longer share a workspace and kill each other

`eval-suite --agent` named each run's workspace `<bench>_<model>_<YYYYMMDD_HHMMSS>_<pid>`: no
trial, no condition, one-second resolution. Trials of one benchmark that started in the same
second (every `--parallel` slot fills at once when a suite starts) shared one directory, overwrote
each other's solution file, and when the first finished its cleanup deleted the directory and
`ReapWorkspace` SIGKILLed the siblings. They banked as `api_error` "signal: killed" with empty code:
3 of 18 Haiku runs on 2026-11-02 (`cp_climb/step0`). Each run now gets an `os.MkdirTemp`
directory that keeps the `<bench>_<model>_` prefix wire-parity attribution matches on, plus the
trial and condition. Browser artifact dirs were keyed on the same name and are fixed with it.

Not affected: the Phase 0 CP cohort (all 123 Claude workspaces from 2026-10-28 held one session each).
