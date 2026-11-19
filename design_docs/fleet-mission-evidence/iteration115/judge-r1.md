# Judge r1, iteration 115, D-FLEET-26 A. Score 88/100, PASS
## Measured
- Driver runs codex controllers via mc_codex_exec -> mc_codex_command (tools/launchd/lib/codex-auth-profile.sh): `ailang mission codex-exec` when AILANG_CODEX_RUNTIME=daemon (forced by mc_select_codex_profile), legacy `codex exec` only when unset. Output goes `>>"$LOG" 2>&1` (mission-control.sh ~2537). Retry grep (line ~2690): `tail -n +$((logpos+1)) "$LOG" | grep -qiE "$TRANSIENT_SIG"`, so -i and -E, new-attempt bytes only.
- Daemon error text: internal/executor/codex/daemon.go:344 `Codex daemon turn failed: <msg>`; printed by cmd/ailang/commands_platform.go:188 as `red("Error"): err`. Ran `ailang mission codex-exec --model zzz x >file 2>&1; od -c`: file ends `Error: Codex daemon turn failed: workspace routing discovery unauthorized (401)`, plain, no ANSI, column 0 (fatih/color is off on a non-TTY). So the `Error: Codex daemon turn failed: ` prefix is measured on the live path.
- Daemon never emits a {"type":"error"} event (grep of daemon.go), so no un-prefixed copy reaches the log in text mode.
- Suite: `/bin/bash tools/launchd/test_controller_capacity.sh` -> 21 cases passed, fail=0.
- Hygiene: grep -cE for column-0 stop text in the 4 touched files -> 0,0,0,0; positive control (printf piped to same grep) -> 1. Added diff lines starting with the text: none.
## Inferred (not measured)
- That the daemon's turn.error.message for a capacity stop is exactly `Selected model is at capacity. ...` (no JSON wrapping). Source is the ticket (raw exec rendering), not a daemon capture. The 2026-11-06 daemon 400 was JSON-wrapped, so a JSON-wrapped capacity message would be missed (fails safe: crash, as today).
## Mutations (scratch copies of the driver in /tmp/fleet_i115_judge/, via MC_CAPACITY_DRIVER)
| mutation | result |
|---|---|
| m1 drop the new alternative | killed (codex-capacity, codex-daemon-capacity) |
| m2 drop `^` anchor | killed (prose-codex-capacity) |
| m3 intended widening, but the edit left `): ` + ` model is at capacity` (double space) | INVALID mutation (broke the positives); not counted |
| m4 remove daemon alt | killed (codex-daemon-capacity) |
| m5 remove ERROR alt | killed (codex-capacity) |
| m6 widen prefix `Error: [A-Za-z ]*` | SURVIVED |
| m7 truncate literal to `Selected model is at` | SURVIVED |
| m8 widen tail to `: .*capacity` | SURVIVED |
| m9 allow leading spaces `^ *` | killed (prose indented quotes) |
| m10 unanchored literal | killed (prose) |
## Findings
- NON-BLOCKING 1: m6/m7/m8 survive: no arm pins the literal's tail or a different-prefix line, so a later widening of the prefix class or tail would pass. Ruling says "anchored literal", so a negative arm with e.g. `Error: Codex daemon turn failed: Selected model is at limit` and `ERROR: Selected model is at capacity` variants would pin it.
- NON-BLOCKING 2: the daemon capacity message text is inferred from the raw-exec ticket, not captured from the daemon. The changelog states the daemon rendering as fact ("prints `Error: Codex daemon turn failed: Selected model is at capacity...`"); only the prefix is measured.
- NON-BLOCKING 3: the fake is a pi controller, so the suite exercises the grep seam, not the codex spawn line; acceptable and stated in the test comment.
- NON-BLOCKING 4: anchor means a codex controller that cats a file and echoes a column-0 `ERROR: Selected model...` line would still retry (bounded to 3 attempts, no demotion), acceptable.
- No BLOCKING findings. Scope: anchored literal, no new class, same bounded retry; the 400 model_not_found daemon and exec forms stay crashes (tested).
## Strongest objection
The daemon-form positive arm asserts a message string nobody has seen the daemon produce; if the real daemon message differs (JSON wrapping, different punctuation), iteration 219's failure recurs silently while the suite stays green.
