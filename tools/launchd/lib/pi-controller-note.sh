# shellcheck shell=bash
# pi-controller-note.sh — sourced by the mission driver's pi controller branch (mission-control.sh).
#
# Ticket gate0:session-protocol-deadlock:agent-policy-unset (stapledon fire 2026-09-29T12-53-33).
# A pi-driven controller tried to satisfy the session-protocol inbox step with ailang_cli, which can
# never satisfy it (AILANG_AGENT_POLICY is unset on purpose), and died at Gate 0 after 5 refusals.
# This note says how the step IS satisfied. It changes no extension, gate or policy.
#
# Why it goes in BOTH the system prompt and the -p prompt: the ollama/glm-5.3:cloud route's
# system-role handling is unmeasured (ailang-exec.ts:322-324 records a route that dropped it).
# Why role spawns (scripts/mission_pi_run.sh) do NOT get it: they are stages that skip the inbox
# step (#1393), and in a policy-bearing role lane the note would be false.
#
# _mc_pi_controller_note → prints the note with no trailing newline. Pure ASCII, no globs.
# SC2016: the backticks in the note are literal text the model reads, not command substitutions.
# shellcheck disable=SC2016
_mc_pi_controller_note() {
  printf '%s' 'PI CONTROLLER SESSION PROTOCOL (from the mission driver). This session runs behind a session-protocol gate: until session_protocol_ack succeeds, bash runs only single read-only commands, and a chained command (any `;`, `&&`, `||` or `|`) is refused if any part of it is not on the read-only list. Do the protocol first, in this order: (1) read CLAUDE.md with the read tool; (2) run `ailang messages list --unread` as ONE plain bash call on its own, with nothing chained to it; (3) call session_protocol_ack. The ailang_cli and ailang_run tools are NOT granted in a controller session (AILANG_AGENT_POLICY is unset on purpose): every call refuses, and none of them counts toward the protocol. Run every ailang command through bash.'
}
