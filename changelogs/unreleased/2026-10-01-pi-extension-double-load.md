### Fixed — mission pi lanes crashed on duplicate extensions in checkouts with `.pi/extensions`

`ailang pi install` copies the repo's extensions into the global `~/.pi/agent/extensions`, and pi
discovers both, so every tool registered twice and pi exited rc=1. Fleet lost its pi lanes on 12
fires from 2026-09-30, and the docs evaluator chain fell through to opus. The model probe now runs
with `--no-extensions`; the pi controller and `scripts/mission_pi_run.sh` load the repo's own
extensions once, explicitly (`tools/launchd/lib/pi-ext-args.sh`). Checkouts without
`.pi/extensions` (world, stapledon) are unchanged. Settings packages are not loaded in the
affected checkouts.
