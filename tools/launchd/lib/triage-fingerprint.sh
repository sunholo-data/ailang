#!/bin/bash
# triage-fingerprint.sh — the triage mission's queue fingerprint (M-TRIAGE-MISSION Part B).
#
# ONE implementation, two callers, so the reader and the writer cannot drift:
#   * the driver's TRIAGE IDLE PRE-CHECK runs it bare (under _mc_bounded) and compares the
#     hash with the recorded one — equal means nothing changed since the last digest, so the
#     fire exits before any probe (charter clause 5, "idle is free");
#   * the controller runs it with --record as the LAST step of an iteration, after its digest
#     is posted (gate-3-triage.md, "Recording the fingerprint"). Nothing else writes the file.
#
# Fingerprint = sha256 over three sorted sections:
#   pr     <number>:<headRefOid>        every open PR (a new push changes it; an attestation
#                                        comment does not)
#   card   <approval id>                every pending coordinator approval card
#   issue  <number>:<updatedAt>         every open issue EXCEPT the current bookkeeping thread
#                                        (the digest comments on it every iteration, so
#                                        including it would make every fire look changed)
#   directive <comment id>:<updatedAt>  a comment ON the bookkeeping thread by a directive
#                                        author (MISSION_DIRECTIVE_AUTHORS, the same default
#                                        as scripts/mission_directives.sh). Without it Mark's
#                                        answer to a D-row woke nothing: the thread is excluded
#                                        above, so a reply sat unread until some unrelated PR
#                                        moved (Mark, 28 Oct 2026, answering by email). The
#                                        loop's own digest comments are never an author's, so
#                                        they still do not move the hash.
#
# Output (stdout, ONE line, last line): "<sha256> prs=N cards=N issues=N".
# Any read that fails or returns something unparseable exits non-zero with the reason on
# stderr and prints NO fingerprint: a failed read is neither "unchanged" nor "changed".
#
# Bash 3.2 + BSD tools (the rig). Needs gh, jq (/usr/bin/jq ships with macOS) and ailang.
set -u

repo="${MISSION_REPO:-sunholo-data/ailang}"
state_dir="${TRIAGE_STATE_DIR:-$HOME/.ailang/state}"
fp_file="$state_dir/mission-triage-queue-fingerprint"
record=0
case "${1:-}" in
  --record) record=1 ;;
  "") ;;
  *) echo "usage: triage-fingerprint.sh [--record]" >&2; exit 2 ;;
esac

die() { echo "triage-fingerprint: $*" >&2; exit 1; }
command -v jq >/dev/null 2>&1 || die "jq not on PATH"
command -v gh >/dev/null 2>&1 || die "gh not on PATH"
command -v ailang >/dev/null 2>&1 || die "ailang not on PATH"

tmp=$(mktemp -d -t triage-fp) || die "mktemp failed"
trap 'rm -rf "$tmp"' EXIT

# --limit is far above the measured queue (33 PRs / 70 issues); gh truncates SILENTLY at the
# limit, so a hit on it is treated as a failed read rather than a short list.
LIMIT=1000

gh pr list -R "$repo" --state open --limit "$LIMIT" --json number,headRefOid \
  >"$tmp/prs.json" 2>"$tmp/err" || die "gh pr list failed: $(tail -1 "$tmp/err")"
jq -r 'if type=="array" then .[] | "pr \(.number):\(.headRefOid)" else error("not an array") end' \
  "$tmp/prs.json" >"$tmp/prs" 2>"$tmp/err" || die "gh pr list output unparseable: $(tail -1 "$tmp/err")"

gh issue list -R "$repo" --state open --limit "$LIMIT" --json number,updatedAt \
  >"$tmp/issues.json" 2>"$tmp/err" || die "gh issue list failed: $(tail -1 "$tmp/err")"
bk=""
[ -f "$state_dir/mission-triage-gh-issue" ] && bk=$(tr -dc '0-9' <"$state_dir/mission-triage-gh-issue")
jq -r --arg bk "$bk" 'if type=="array" then .[] | select((.number|tostring) != $bk)
    | "issue \(.number):\(.updatedAt)" else error("not an array") end' \
  "$tmp/issues.json" >"$tmp/issues" 2>"$tmp/err" || die "gh issue list output unparseable: $(tail -1 "$tmp/err")"

: >"$tmp/directives"
if [ -n "$bk" ]; then
  authors="${MISSION_DIRECTIVE_AUTHORS:-MarkEdmondson1234}"
  gh api --paginate "repos/$repo/issues/$bk/comments?per_page=100" \
    >"$tmp/comments.json" 2>"$tmp/err" || die "gh api comments on #$bk failed: $(tail -1 "$tmp/err")"
  jq -rs --arg a "$authors" '($a | split(",") | map(gsub("^ +| +$"; ""))) as $who
      | if all(type=="array") then add // [] | .[] | select(.user.login as $l | $who | index($l))
      | "directive \(.id):\(.updated_at)" else error("not an array") end' \
    "$tmp/comments.json" >"$tmp/directives" 2>"$tmp/err" || die "comments on #$bk unparseable: $(tail -1 "$tmp/err")"
fi

# stdout only: the binary prints staleness and MOD010 warnings on stderr.
ailang coordinator approvals --remote gcp --json >"$tmp/cards.json" 2>"$tmp/err" \
  || die "coordinator approvals failed: $(grep -v '^ *$' "$tmp/err" | tail -1)"
jq -r 'if (.pending|type)=="array" then .pending[] | "card \(.id)" else error("no .pending array") end' \
  "$tmp/cards.json" >"$tmp/cards" 2>"$tmp/err" || die "approvals JSON unparseable: $(tail -1 "$tmp/err")"

n_pr=$(grep -c . "$tmp/prs"); n_is=$(grep -c . "$tmp/issues"); n_cd=$(grep -c . "$tmp/cards")
raw_pr=$(jq length "$tmp/prs.json"); raw_is=$(jq length "$tmp/issues.json")
[ "$raw_pr" -lt "$LIMIT" ] && [ "$raw_is" -lt "$LIMIT" ] || die "a list hit --limit $LIMIT (gh truncates silently)"

hash=$(LC_ALL=C sort "$tmp/prs" "$tmp/cards" "$tmp/issues" "$tmp/directives" | shasum -a 256 | awk '{print $1}')
printf '%s' "$hash" | grep -Eq '^[0-9a-f]{64}$' || die "hash failed"
line="$hash prs=$n_pr cards=$n_cd issues=$n_is"

if [ "$record" -eq 1 ]; then
  mkdir -p "$state_dir" || die "cannot create $state_dir"
  printf '%s\n' "$line" >"$fp_file.tmp.$$" && mv "$fp_file.tmp.$$" "$fp_file" || die "cannot write $fp_file"
fi
printf '%s\n' "$line"
