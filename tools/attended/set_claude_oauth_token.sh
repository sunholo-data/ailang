#!/bin/bash
# Capture a long-lived Claude Code OAuth token into the file the fleet reads.
#
# WHY THIS EXISTS. `claude setup-token` PRINTS a token and stores it nowhere the
# mission fleet reads. Run it a hundred times and the fleet is no better off —
# measured 2026-09-22: CLAUDE_CODE_OAUTH_TOKEN was absent from secrets.env, every
# LaunchAgent plist, every shell rc file and the launchd domain, while the
# keychain's own access token had been EXPIRED since 2026-09-15.
#
# The consequence was not an outage, which is why it ran so long: `claude -p`
# kept working (Claude Code refreshes in memory), so only the QUOTA READ failed.
# internal/mission/anthropic_quota.go got HTTP 401, reported the bucket unknown,
# and `mission quota --over` blocks unknown by policy — so the fleet routed away
# from an Anthropic subscription sitting at ~89% free, and called it
# "over daily ration" in the log.
#
# This script closes the loop: generate, capture, verify, in one attended step.
set -euo pipefail

SECRETS="${AILANG_SECRETS_ENV:-$HOME/.config/ailang/secrets.env}"
VAR=CLAUDE_CODE_OAUTH_TOKEN

die() { printf '\n  ✗ %s\n\n' "$*" >&2; exit 1; }
ok()  { printf '  ✓ %s\n' "$*"; }

command -v claude >/dev/null 2>&1 || die "no \`claude\` on PATH"
# ATTEMPT THE OPEN, never `[ -r /dev/tty ]`. The test operator stats the path and
# returns true in a shell where opening the device fails with ENXIO — measured in
# an agent shell, and it is the same check-that-looks-like-verification this repo
# keeps paying for. The first version of THIS script shipped it.
has_tty() { ( : < /dev/tty ) 2>/dev/null; }
has_tty || die "no controlling terminal (opening /dev/tty failed).
    Run this from a real shell — \`claude setup-token\` is interactive."

mkdir -p "$(dirname "$SECRETS")"
[ -f "$SECRETS" ] || { ( umask 077; : > "$SECRETS" ); ok "created $SECRETS"; }
chmod 600 "$SECRETS"

printf '\nRunning `claude setup-token`. Complete the auth it asks for.\n'
printf 'The token is captured directly — it is never echoed to this terminal.\n\n'

tmp="$(mktemp)"; trap 'rm -f "$tmp"' EXIT
claude setup-token < /dev/tty > "$tmp" 2>/dev/tty || die "claude setup-token failed (rc=$?)"

tok="$(grep -oE '[A-Za-z0-9_-]{40,}' "$tmp" | tail -1 || true)"
if [ -z "$tok" ]; then
  # Fall back to a secure paste rather than failing. setup-token's output shape
  # is not a contract, and an operator who has just completed an auth flow
  # should not have to start again because a grep missed.
  printf '\n  ! could not read a token from the command output.\n'
  printf '    Paste it here (input is hidden): '
  read -rs tok < /dev/tty; printf '\n'
fi
[ -n "$tok" ] || die "no token supplied; $SECRETS is unchanged"

# VERIFY BEFORE WRITING. anthropic_quota.go prefers CLAUDE_CODE_OAUTH_TOKEN over
# the keychain, so a bad value here does not sit inert — it OVERRIDES a keychain
# path that may currently be working, and turns a healthy read into a 401. Test
# in a subshell first and leave the file untouched unless it is an improvement.
printf '\n  verifying the token against the usage endpoint before writing…\n'
probe="$(CLAUDE_CODE_OAUTH_TOKEN="$tok" ailang mission quota 2>&1 | grep -m1 'anthropic provider usage:' || true)"
case "$probe" in
  *"usage: unknown"*)
    printf '\n  ✗ the new token does not read the usage endpoint:\n      %s\n' "${probe#*usage: }"
    die "$SECRETS is UNCHANGED — the keychain path you have now is not made worse by this run"
    ;;
  "")
    die "could not probe the usage endpoint; $SECRETS is unchanged"
    ;;
esac
ok "token reads the usage endpoint: ${probe#*anthropic provider usage: }"

# Replace any existing line rather than appending a second: a duplicate export is
# a shadowing bug waiting to happen, and this fleet has already paid for a blanked
# credential shadowing a valid one.
if grep -q "^export ${VAR}=" "$SECRETS" 2>/dev/null; then
  new="$(mktemp)"; grep -v "^export ${VAR}=" "$SECRETS" > "$new"; ( umask 077; cat "$new" > "$SECRETS" ); rm -f "$new"
  ok "removed the previous $VAR line"
fi
( umask 077; printf 'export %s=%s\n' "$VAR" "$tok" >> "$SECRETS" )
chmod 600 "$SECRETS"
ok "wrote $VAR to $SECRETS (mode 0600, value not shown)"

printf '\nDone. The driver sources this file at mission-control.sh:119, and\n'
printf 'anthropic_quota.go prefers this variable over the keychain — so the fleet\n'
printf 'no longer depends on a keychain access token that expires every ~8 hours\n'
printf 'and is never written back by Claude Code.\n\n'
