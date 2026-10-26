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
# THE WATERMARK (#981). Reading NEVER writes it. The controller routes each directive first
# (ledger row, unpark/pick recorded, report draft acknowledges it) and only THEN runs
# `--mark <id>`. Two of Mark's answers were lost on #852: one sat below a watermark that a fire
# advanced before routing and then died; the other's createdAt EQUALLED the watermark, and the
# read was exclusive. So: the read is INCLUSIVE (>=), every directive carries its comment id,
# and the watermark file records which ids were processed so an inclusive re-read is a no-op.
#
# Watermark file (one per bookkeeping issue, path derived by the caller):
#   line 1   bare ISO8601 read boundary (inclusive). mission_gate0_self_notices.sh reads ONLY this.
#   line 2   "# mission-directives watermark v2 ..." (absent = LEGACY bare-timestamp file)
#   rest     "processed <comment-id> <createdAt>" for every processed id at/after line 1
# `--mark` re-fetches the issue and moves line 1 to the OLDEST allowlisted comment not yet
# processed (or the newest processed one when nothing is pending), so an out-of-order mark or a
# directive that landed mid-iteration can never be jumped. Written by tmp + rename: a fire that
# dies mid-write leaves the previous file intact.
#
# Usage:
#   scripts/mission_directives.sh --issue 635 --watermark-file <path>              # read
#   scripts/mission_directives.sh --issue 635 --watermark-file <path> --mark <id>  # after routing
#   scripts/mission_directives.sh --issue 635 [--repo owner/name] [--since ISO8601]  # raw read
#   MISSION_DIRECTIVE_AUTHORS="a,b" scripts/mission_directives.sh --issue 635 ...
#
# Exit codes: 0 = ran (zero or more directives on stdout), 1 = refused/error.
#
# shellcheck disable=SC2016  # the single-quoted strings are jq programs; $vars are jq's, not ours.
set -uo pipefail

RED='\033[0;31m'; YELLOW='\033[1;33m'; RESET='\033[0m'
die() { echo -e "${RED}✗ mission_directives: $*${RESET}" >&2; exit 1; }
warn() { echo -e "${YELLOW}⚠ mission_directives: $*${RESET}" >&2; }

ISSUE=""; REPO="${MISSION_REPO:-sunholo-data/ailang}"; SINCE=""; SINCE_SET=0; WMFILE=""; MARK=""
while [ $# -gt 0 ]; do
	case "$1" in
		-h|--help) sed -n '2,36p' "$0"; exit 0 ;;
		--issue|--repo|--since|--watermark-file|--mark)
			# bash 3.2: `shift 2` with one arg left fails WITHOUT shifting — an infinite loop.
			[ $# -ge 2 ] || die "$1 needs a value" ;;
		*) die "unknown argument: $1" ;;
	esac
	case "$1" in
		--issue) ISSUE="$2" ;;
		--repo)  REPO="$2" ;;
		--since) SINCE="$2"; SINCE_SET=1 ;;
		--watermark-file) WMFILE="$2" ;;
		--mark)  MARK="$2" ;;
	esac
	shift 2
done

[ -n "$ISSUE" ] || die "--issue is required"
case "$ISSUE" in ''|*[!0-9]*) die "--issue must be a number, got: $ISSUE" ;; esac
[ -n "$REPO" ] || die "no repo (pass --repo or set MISSION_REPO)"
if [ "$SINCE_SET" = 1 ] && [ -n "$WMFILE" ]; then
	die "--since and --watermark-file are exclusive — the file's line 1 IS the since"
fi
if [ -n "$MARK" ] && [ -z "$WMFILE" ]; then
	die "--mark needs --watermark-file (the record it updates); never hand-write the watermark"
fi

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

# ── the watermark record ─────────────────────────────────────────────────────────
# The read is a STRING compare on GitHub's fixed-width UTC form, so anything else on line 1
# is refused loudly rather than silently compared.
is_iso() { printf '%s\n' "$1" | grep -Eq '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'; }
EPOCH="1970-01-01T00:00:00Z"
WM_STATE="none"; PROCESSED_JSON='[]'
if [ -n "$WMFILE" ]; then
	if [ ! -e "$WMFILE" ]; then
		WM_STATE="absent"; SINCE="$EPOCH"
	else
		[ -r "$WMFILE" ] || die "watermark file unreadable: $WMFILE"
		SINCE=$(head -1 "$WMFILE" | tr -d '[:space:]')
		[ -n "$SINCE" ] || die "watermark file has an empty line 1: $WMFILE"
		if grep -q '^# mission-directives watermark v2' "$WMFILE"; then WM_STATE="v2"; else WM_STATE="legacy"; fi
		PROCESSED_JSON=$(awk '$1 == "processed" && NF >= 3 { print $2 }' "$WMFILE" | jq -R -s 'split("\n") | map(select(length > 0))') \
			|| die "could not parse processed ids in $WMFILE"
	fi
fi
SINCE="${SINCE:-$EPOCH}"
is_iso "$SINCE" || die "since/watermark is not a UTC ISO8601 timestamp (YYYY-MM-DDTHH:MM:SSZ): '$SINCE'"

# Build a jq allowlist array from the comma-separated env value, so the author
# comparison is data rather than an interpolated fragment of jq program text.
AUTHORS_JSON=$(printf '%s' "$AUTHORS" | jq -R 'split(",") | map(gsub("^\\s+|\\s+$";"")) | map(select(length > 0))')
[ "$(printf '%s' "$AUTHORS_JSON" | jq 'length')" -gt 0 ] || die "allowlist resolved to zero authors"

raw=$(gh issue view "$ISSUE" --repo "$REPO" --json comments 2>/dev/null) \
	|| die "could not read issue #$ISSUE on $REPO (auth? issue exists?)"

# jq prelude shared by read and mark. ascii_downcase both sides: GitHub logins are
# case-insensitive, so a case-sensitive compare would drop a real directive from
# "markedmondson1234". The window is INCLUSIVE (>=): a comment whose createdAt equals the
# watermark must come back (#981); the processed ids make that re-read a no-op.
JQ_PRELUDE='
	($allow | map(ascii_downcase)) as $a
	| [ .comments[]
	    | select(.author.login != null)
	    | select(.author.login | ascii_downcase | IN($a[])) ] as $all
	| [ $all[] | select(.createdAt >= $since) ] as $win'
jqd() { printf '%s' "$raw" | jq -r --argjson allow "$AUTHORS_JSON" --arg since "$SINCE" --argjson seen "$PROCESSED_JSON" "$@"; }

# ── --mark <id>: record a ROUTED directive and move the watermark safely ────────────
if [ -n "$MARK" ]; then
	known=$(jqd --arg mark "$MARK" "$JQ_PRELUDE"' | [ $all[] | select(.id == $mark) ] | length') \
		|| die "could not evaluate the issue payload"
	[ "$known" -ge 1 ] || die "--mark $MARK: not an allowlisted comment on #$ISSUE — refusing (watermark unchanged)"
	# New boundary = the oldest allowlisted comment in the window that is still unprocessed;
	# when none is pending, the newest one. Only ids at/after the new boundary are kept: they
	# are the only ones an inclusive read can return again.
	content=$(jqd --arg mark "$MARK" "$JQ_PRELUDE"'
		| ($seen + [$mark] | unique) as $p
		| [ $win[] | select((.id | IN($p[])) | not) ] as $pending
		| (if ($pending | length) > 0 then ($pending | map(.createdAt) | min)
		   elif ($win | length) > 0 then ($win | map(.createdAt) | max)
		   else $since end) as $nw
		| $nw, "# mission-directives watermark v2 — line 1 is the inclusive read boundary; write only via mission_directives.sh --mark",
		  ( $win[] | select(.createdAt >= $nw) | select(.id | IN($p[])) | "processed \(.id) \(.createdAt)" )') \
		|| die "could not compute the new watermark"
	new_since=$(printf '%s\n' "$content" | head -1)
	is_iso "$new_since" || die "computed watermark is not ISO8601: '$new_since' — refusing to write"
	mkdir -p "$(dirname "$WMFILE")" || die "cannot create $(dirname "$WMFILE")"
	tmp=$(mktemp "$WMFILE.tmp.XXXXXX") || die "cannot create a temp file beside $WMFILE"
	if ! printf '%s\n' "$content" > "$tmp" || ! mv -f "$tmp" "$WMFILE"; then
		rm -f "$tmp"; die "could not write $WMFILE (previous record left intact)"
	fi
	pending=$(jqd --arg mark "$MARK" "$JQ_PRELUDE"' | ($seen + [$mark]) as $p | [ $win[] | select((.id | IN($p[])) | not) ] | length')
	echo "mission_directives: marked $MARK on #$ISSUE; watermark $SINCE -> $new_since; $pending directive(s) still pending"
	exit 0
fi

# ── read ─────────────────────────────────────────────────────────────────────────
jqd "$JQ_PRELUDE"'
	| $win[]
	| "\(.author.login) @ \(.createdAt) id=\(.id)"
	  + (if (.id | IN($seen[])) then " [ALREADY PROCESSED — context only, take no routing action]" else "" end)
	  + ":\n\(.body)\n---"'

# Report the filter on stderr so the caller can see what was enforced without it
# polluting the directive stream on stdout.
total=$(printf '%s' "$raw" | jq '.comments | length')
read -r kept new boundary <<EOF
$(jqd "$JQ_PRELUDE"' | [ ($win | length), ([ $win[] | select((.id | IN($seen[])) | not) ] | length),
	([ $win[] | select(.createdAt == $since) | select((.id | IN($seen[])) | not) ] | length) ] | @tsv')
EOF
case "$WM_STATE" in
	absent) warn "watermark file $WMFILE is absent — reading every allowlisted comment on #$ISSUE; the first --mark creates it" ;;
	legacy) warn "LEGACY watermark file $WMFILE (bare timestamp, no processed ids) — the boundary comment is re-presented; check the ledger by comment id before routing; the first --mark upgrades it" ;;
esac
if [ -n "$WMFILE" ] && [ "$SINCE" != "$EPOCH" ] && [ "${boundary:-0}" -gt 0 ]; then
	warn "BOUNDARY: $boundary directive(s) sit exactly at the watermark $SINCE and are NOT recorded as processed — a prior fire moved or held the watermark without marking them (the #981 signature). They are presented as NEW; if the ledger already cites the id, a fire died after routing: --mark it and route nothing."
fi
echo -e "${YELLOW}mission_directives: #$ISSUE — $kept directive(s) from [$AUTHORS] since $SINCE inclusive, ${new:-0} new, $(( ${kept:-0} - ${new:-0} )) already processed (of $total comments; the rest are public feedback, never directives)${RESET}" >&2
