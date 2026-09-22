#!/bin/bash
# Store a long-lived Claude Code OAuth token where the mission fleet reads it.
#
# YOU run `claude setup-token` yourself, plainly, in your own terminal. This
# script does NOT invoke it — an earlier version did, redirecting stdin so it
# could capture the output, and that redirect crashed the command:
#
#   EINVAL: invalid argument, kqueue   (Bun v1.4.3)
#
# The lesson is cheap and worth keeping: a capture wrapper that changes how a
# program is invoked is not observing that program, it is running a different
# one. So this script only does the parts that were actually missing — verify,
# and store without clobbering.
#
# WHY IT IS NEEDED AT ALL. `claude setup-token` PRINTS a token and stores it
# nowhere the fleet reads. Measured 2026-09-22: CLAUDE_CODE_OAUTH_TOKEN was
# absent from secrets.env, every LaunchAgent plist, every shell rc file and the
# launchd domain — so many prior runs had all gone nowhere, and the fleet was
# silently falling back to the `Claude Code-credentials` keychain item whose
# access token lives ~8 HOURS and which Claude Code refreshes in memory without
# writing back. Inference kept working; only the quota READ went stale.
set -euo pipefail

SECRETS="${AILANG_SECRETS_ENV:-$HOME/.config/ailang/secrets.env}"
VAR=CLAUDE_CODE_OAUTH_TOKEN

die() { printf '\n  ✗ %s\n\n' "$*" >&2; exit 1; }
ok()  { printf '  ✓ %s\n' "$*"; }

printf '\nStep 1 — in THIS terminal, run:\n\n    claude setup-token\n\n'
printf 'Step 2 — paste the token below. Input is hidden and never echoed.\n\n'
printf '  token: '
read -rs tok
printf '\n\n'
[ -n "${tok:-}" ] || die "nothing pasted; $SECRETS is unchanged"

# VERIFY BEFORE WRITING. anthropic_quota.go prefers this variable over the
# keychain, so a bad value does not sit inert — it OVERRIDES a keychain path that
# may currently be working and turns a healthy read into a 401.
printf '  verifying against the usage endpoint…\n'
probe="$(CLAUDE_CODE_OAUTH_TOKEN="$tok" ailang mission quota 2>&1 | grep -m1 'anthropic provider usage:' || true)"
[ -n "$probe" ] || die "could not probe the usage endpoint; $SECRETS is unchanged"
case "$probe" in
  *"usage: unknown"*)
    printf '      %s\n' "${probe#*usage: }"
    die "that token does not read the usage endpoint — $SECRETS is UNCHANGED,
    so whatever is working today is not made worse by this run" ;;
esac
ok "reads the usage endpoint: ${probe#*anthropic provider usage: }"

mkdir -p "$(dirname "$SECRETS")"
[ -f "$SECRETS" ] || ( umask 077; : > "$SECRETS" )
# Replace, never append a second line: a duplicate export is a shadowing bug, and
# this fleet has already paid for a blanked credential shadowing a valid one.
if grep -q "^export ${VAR}=" "$SECRETS" 2>/dev/null; then
  t="$(mktemp)"; grep -v "^export ${VAR}=" "$SECRETS" > "$t"; ( umask 077; cat "$t" > "$SECRETS" ); rm -f "$t"
  ok "replaced the previous $VAR line"
fi
( umask 077; printf 'export %s=%s\n' "$VAR" "$tok" >> "$SECRETS" )
chmod 600 "$SECRETS"
ok "written to $SECRETS (mode 0600, value not shown)"
printf '\nThe driver sources that file at mission-control.sh:119. Done — no 8-hour expiry.\n\n'
