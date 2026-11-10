### Fixed — the end-of-session auto-push no longer times out behind its own pre-push gates (2026-11-10)

`scripts/hooks/push_dev_on_stop.sh` gave `git push` 25 seconds, sized for a bare push. The push runs
the `pre-push` hook, which builds the tree and runs `make ci-quick` whenever the push touches code,
examples, changelogs or docs. On 2026-11-10 that hook began resolving again after six weeks as a
dangling symlink, and every auto-push that day was then killed mid-gate (twelve in a row) and logged
as "auth, network, or a race", so commits sat unpushed in the shared checkout: the failure the
auto-push exists to prevent.

- The bound is now 300 s, overridable with `AILANG_AUTOPUSH_TIMEOUT` (seconds; a non-numeric value
  falls back to 300).
- The three outcomes are reported apart, on screen and in `~/.ailang/state/autopush.log`:
  `TIMED OUT` (our own bound fired; nothing was refused), `REFUSED` (the pre-push gate said no; its
  own message is shown) and `FAILED` (anything else, with the last lines of git's output).
