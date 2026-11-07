### Fixed — codex "Selected model is at capacity" now retries instead of crashing the iteration (2026-11-03)

When a codex controller exited rc=1 after `ERROR: Selected model is at capacity. Please try a
different model.`, the mission driver recorded the slot as CRASHED. That happened to World iteration
219 (2026-10-02), which had already finished its work. `TRANSIENT_SIG` in
`tools/launchd/mission-control.sh` now carries the line-anchored `^ERROR: Selected model is at
capacity`, so this gets the same bounded same-model retry with backoff as Anthropic overloads. The
anchor is there so that prose quoting the string mid-line does not trigger a retry. New cases
`codex-capacity-is-transient`, `prose-codex-capacity-not-transient` and
`codex-other-error-not-transient` in `tools/launchd/test_controller_capacity.sh`; the last keeps
every other column-0 codex `ERROR:` a crash. Decision D-FLEET-26 (option A).
