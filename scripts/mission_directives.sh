#!/usr/bin/env bash
#
# mission_directives.sh — fetch the human directives on a mission's bookkeeping
# issue, with the author allowlist enforced HERE, in code.
#
# WHY THIS EXISTS. The mission bookkeeping issue lives on a PUBLIC repo, so anyone
# can comment on it, and an allowlisted comment outranks the queue and can unpark
# work. Until now the allowlist was a `jq ... select(.author.login == "...")`
# written into the mission-control skill's prose — correct, but enforced only by
# the controller choosing to run that exact command. A model that paraphrases the
# pipeline, or that is talked into widening it by something it read, has nothing
# behind it. This script is that something.
#
# WHAT IT DOES NOT DO. It does not write the watermark — the skill does that after
# it has actually triaged, so a crash mid-triage cannot silently skip a directive.
#
# Usage:
#   scripts/mission_directives.sh --issue 635 [--repo owner/name] [--since ISO8601]
#   MISSION_DIRECTIVE_AUTHORS="a,b" scripts/mission_directives.sh --issue 635
#
# Exit codes: 0 = ran (zero or more directives on stdout), 1 = refused/error.
set -uo pipefail

RED='\033[0;31m'; YELLOW='\033[1;33m'; RESET='\033[0m'
die() { echo -e "${RED}✗ mission_directives: $*${RESET}" >&2; exit 1; }

ISSUE=""; REPO="${MISSION_REPO:-sunholo-data/ailang}"; SINCE=""
while [ $# -gt 0 ]; do
	case "$1" in
		--issue) ISSUE="${2:-}"; shift 2 ;;
		--repo)  REPO="${2:-}";  shift 2 ;;
		--since) SINCE="${2:-}"; shift 2 ;;
		-h|--help) sed -n '2,25p' "$0"; exit 0 ;;
		*) die "unknown argument: $1" ;;
	esac
done

[ -n "$ISSUE" ] || die "--issue is required"
case "$ISSUE" in ''|*[!0-9]*) die "--issue must be a number, got: $ISSUE" ;; esac
[ -n "$REPO" ] || die "no repo (pass --repo or set MISSION_REPO)"

# The directive principal. Defaults to the V1 human; missions override via their
# own env file. Set-but-EMPTY is a configuration error, not "trust nobody
# silently" — a loop that quietly stops seeing its human looks identical to a
# human who has stopped commenting.
if [ "${MISSION_DIRECTIVE_AUTHORS+set}" = "set" ] && [ -z "${MISSION_DIRECTIVE_AUTHORS}" ]; then
	die "MISSION_DIRECTIVE_AUTHORS is set but empty — refusing (unset it to use the default)"
fi
AUTHORS="${MISSION_DIRECTIVE_AUTHORS:-MarkEdmondson1234}"

command -v gh >/dev/null 2>&1 || die "gh CLI not found"
command -v jq >/dev/null 2>&1 || die "jq not found"

# SELF-DIRECTION GUARD. The account this loop pushes as must never be a directive
# principal: if it were, the loop could steer itself by commenting on its own
# bookkeeping issue, and every guard downstream would read that as a human
# instruction. This is the one failure that would be invisible in the report.
SELF=$(gh api user --jq .login 2>/dev/null || echo "")
if [ -n "$SELF" ]; then
	OLD_IFS="$IFS"; IFS=','
	for a in $AUTHORS; do
		a=$(printf '%s' "$a" | tr -d '[:space:]')
		# tr for lowercasing: the rig runs bash 3.2, which has no ${var,,}.
		if [ "$(printf '%s' "$a" | tr '[:upper:]' '[:lower:]')" = "$(printf '%s' "$SELF" | tr '[:upper:]' '[:lower:]')" ]; then
			IFS="$OLD_IFS"
			die "allowlist contains the authenticated account '$SELF' — that lets the loop direct itself. Refusing."
		fi
	done
	IFS="$OLD_IFS"
fi

SINCE="${SINCE:-1970-01-01T00:00:00Z}"

# Build a jq allowlist array from the comma-separated env value, so the author
# comparison is data rather than an interpolated fragment of jq program text.
AUTHORS_JSON=$(printf '%s' "$AUTHORS" | jq -R 'split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(length > 0))')
[ "$(printf '%s' "$AUTHORS_JSON" | jq 'length')" -gt 0 ] || die "allowlist resolved to zero authors"

raw=$(gh issue view "$ISSUE" --repo "$REPO" --json comments 2>/dev/null) \
	|| die "could not read issue #$ISSUE on $REPO (auth? issue exists?)"

# ascii_downcase both sides: GitHub logins are case-insensitive, so a
# case-sensitive compare would drop a real directive from "markedmondson1234".
printf '%s' "$raw" | jq -r \
	--argjson allow "$AUTHORS_JSON" \
	--arg since "$SINCE" '
	($allow | map(ascii_downcase)) as $a
	| [ .comments[]
	    | select(.author.login != null)
	    | select(.author.login | ascii_downcase | IN($a[]))
	    | select(.createdAt > $since) ]
	| .[] | "\(.author.login) @ \(.createdAt):\n\(.body)\n---"
	'

# Report the filter on stderr so the caller can see what was enforced without it
# polluting the directive stream on stdout.
total=$(printf '%s' "$raw" | jq '.comments | length')
kept=$(printf '%s' "$raw" | jq -r --argjson allow "$AUTHORS_JSON" --arg since "$SINCE" '
	($allow | map(ascii_downcase)) as $a
	| [ .comments[] | select(.author.login != null)
	    | select(.author.login | ascii_downcase | IN($a[]))
	    | select(.createdAt > $since) ] | length')
echo -e "${YELLOW}mission_directives: #$ISSUE — $kept directive(s) from [$AUTHORS] since $SINCE (of $total comments; the rest are public feedback, never directives)${RESET}" >&2
