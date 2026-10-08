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

# VERIFY BEFORE WRITING — by running a model, which is all this token is for.
#
# It used to be verified against the usage endpoint and refused when that read failed. A
# setup-token ALWAYS fails that read (HTTP 403, measured 2026-09-22), so the check refused
# every token this script exists to store, and the loops stayed on the shared keychain
# login — the one a remote session's refresh rotates away (2026-10-07: 16 hours of "OAuth
# session expired" on every fire). The quota reader now reads usage with the login
# credential FIRST and tries this token only after it (anthropic_quota.go), so the token no
# longer has to read usage at all. What it must do is run inference without the keychain.
printf '  verifying it runs a model (one tiny haiku call, keychain not consulted)…\n'
probe="$(cd / && CLAUDE_CODE_OAUTH_TOKEN="$tok" claude -p --model claude-haiku-4-5-20251001 'Reply with the single word ok' </dev/null 2>&1 | head -c 400 || true)"
# The WHOLE reply must be "ok": a substring match passes on error text ("…token…").
case "$(printf '%s' "$probe" | tr -d '[:space:][:punct:]' | tr '[:upper:]' '[:lower:]')" in
  ok) ;;
  *) printf '      %s\n' "$probe"
     die "that token did not run a model — $SECRETS is UNCHANGED" ;;
esac
ok "runs a model on its own"

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
printf '\nThe driver sources that file on every fire. The loops now run on this token (valid for a\nyear) and no other session can rotate it; quota is still read with your login.\n\n'
