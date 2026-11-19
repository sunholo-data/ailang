#!/bin/bash
# driver:idle-fire-writes-full-block-report — ONE implementation, three entry points:
#   bare        the driver's MISSION IDLE PRE-CHECK compares this against the record;
#   --snapshot  the driver, once per fire, AFTER the overlap guard and BEFORE the controller
#               starts: the inputs as they stood when the controller began, keyed by
#               MISSION_FIRE_ID (<state>/mission-<name>-idle-fingerprint.pending);
#   --record    the controller, last step of a completed human-gated no-pick fire: PROMOTES
#               this fire's snapshot to the record. It never recomputes — an input that
#               changed DURING the fire (an attended ruling at minute 30) must wake the next
#               fire, not be absorbed into a record the controller never acted on.
# A failed read prints NO fingerprint and exits non-zero. Bash 3.2 + BSD tools.
set -uo pipefail
mode="read"
case "$#:${1:-}" in
  0:) ;;
  1:--snapshot) mode=snapshot ;;
  1:--record) mode=record ;;
  *) echo 'usage: idle-fingerprint.sh [--snapshot|--record]' >&2; exit 2 ;;
esac
record=0; [ "$mode" = read ] || record=1
name="${MISSION_NAME:-}"
case "$name" in
  ''|fleet|triage) echo 'idle-fingerprint: MISSION_NAME must name a product mission (not fleet/triage)' >&2; exit 2 ;;
esac
die() { echo "idle-fingerprint: $*" >&2; exit 1; }
repo="${MISSION_REPO:-}"
work="${MISSION_WORKDIR:-}"
doc="${MISSION_DOC:-}"
state_dir="${IDLE_STATE_DIR:-$HOME/.ailang/state}"
fp_file="$state_dir/mission-${name}-idle-fingerprint"
pend_file="$fp_file.pending"
fire="${MISSION_FIRE_ID:-}"
if [ "$mode" != read ]; then
  printf '%s' "$fire" | grep -Eq '^[A-Za-z0-9._-]+$' \
    || die "MISSION_FIRE_ID unset or malformed: --$mode is keyed to the driver's fire"
fi
if [ "$mode" = record ]; then
  [ -f "$pend_file" ] || die "no snapshot for this fire ($pend_file absent): the driver takes it before the controller starts"
  snap=$(head -1 "$pend_file") || die "cannot read $pend_file"
  case "$snap" in
    *" fire=$fire at="*) ;;
    *) die "the snapshot is not from this fire ($fire); not recording another fire's inputs" ;;
  esac
  printf '%s' "${snap%% *}" | grep -Eq '^[0-9a-f]{64}$' || die "snapshot has no fingerprint"
  printf '%s' "$snap" | grep -Eq ' at=[0-9]+$' || die "snapshot has no timestamp"
  mkdir -p "$state_dir" || die "cannot create $state_dir"
  if ! { printf '%s\n' "$snap" >"$fp_file.tmp.$$" && mv "$fp_file.tmp.$$" "$fp_file"; }; then
    rm -f "$fp_file.tmp.$$"; die "cannot write $fp_file"
  fi
  rm -f "$pend_file"
  printf '%s\n' "$snap"
  exit 0
fi
[ -n "$repo" ] && [ -n "$work" ] && [ -n "$doc" ] || die 'MISSION_REPO, MISSION_WORKDIR and MISSION_DOC are required'
for tool in git gh jq ailang; do command -v "$tool" >/dev/null 2>&1 || die "$tool not on PATH"; done
tmp=$(mktemp -d -t idle-fp) || die 'mktemp failed'
trap 'rm -rf "$tmp"; if [ "$record" -eq 1 ]; then rm -f "$fp_file.tmp.$$" "$pend_file.tmp.$$"; fi' EXIT

# A missing origin/HEAD (symbolic-ref rc=1) alone falls back to origin/dev. A bad
# repository or other read error is not a missing ref and must not silently fall back.
branch_ref=$(git -C "$work" symbolic-ref --quiet --short refs/remotes/origin/HEAD 2>"$tmp/err"); rc=$?
case "$rc" in
  0) case "$branch_ref" in origin/?*) branch=${branch_ref#origin/} ;; *) die "unexpected origin/HEAD: $branch_ref" ;; esac ;;
  1)
    git -C "$work" show-ref --verify --quiet refs/remotes/origin/HEAD; rc=$?
    [ "$rc" -eq 1 ] || die 'origin/HEAD exists but is not a symbolic ref'
    branch=dev ;;
  *) die "origin/HEAD read failed: $(tail -1 "$tmp/err")" ;;
esac
git -C "$work" fetch -q origin "$branch" 2>"$tmp/err" || die "fetch failed: $(tail -1 "$tmp/err")"
tip=$(git -C "$work" rev-parse "origin/$branch" 2>"$tmp/err") || die "tip read failed: $(tail -1 "$tmp/err")"
# Keep the exclusions on their own lines: both the charter/log and verification are
# records. Reading ledger statuses separately lets an attended ruling still wake us.
head_sha=$(git -C "$work" log -1 --format=%H "origin/$branch" -- . \
  ":(exclude)design_docs/${name}-mission*" \
  ":(exclude)verification/${name}-iter*" \
  2>"$tmp/err") || die "head read failed: $(tail -1 "$tmp/err")"
printf '%s' "$head_sha" | grep -Eq '^[0-9a-f]{40,64}$' || die 'head read returned no commit'
printf 'head %s\n' "$head_sha" >"$tmp/head"
git -C "$work" show "origin/$branch:$doc" >"$tmp/charter" 2>"$tmp/err" || die "charter read failed: $(tail -1 "$tmp/err")"
# Same first/second-cell and D-row parsing as scripts/mission_decisions.sh. A
# completely absent block is zero rows; partial, repeated or reversed markers fail.
awk -F '|' '
  function trim(s) { gsub(/^[[:space:]]+|[[:space:]]+$/, "", s); return s }
  /<!-- decision-ledger:start -->/ { starts++; if (inside || ends) bad=1; inside=1; next }
  /<!-- decision-ledger:end -->/ { ends++; if (!inside) bad=1; inside=0; next }
  inside && $0 ~ /^\|[[:space:]]*D-/ {
    id=trim($2); status=trim($3)
    if (seen[id]++ || (status != "OPEN" && status != "RESOLVED")) bad=1
    printf "ledger %s:%s\n", id, status
  }
  END { if (bad || starts != ends || starts > 1) exit 1 }
' "$tmp/charter" >"$tmp/ledger" || die 'decision ledger malformed (markers/ID/status)'

[ -f "$state_dir/mission-${name}-gh-issue" ] || die 'bookkeeping gh-issue file absent'
bk=$(cat "$state_dir/mission-${name}-gh-issue") || die 'cannot read bookkeeping gh-issue file'
printf '%s' "$bk" | grep -Eq '^[0-9]+$' || die 'bookkeeping gh-issue is non-numeric'
# Set-but-empty is refused, as scripts/mission_directives.sh refuses it: a loop that has
# quietly stopped seeing its human must not read as a human who stopped commenting.
if [ "${MISSION_DIRECTIVE_AUTHORS+set}" = set ] && [ -z "$(printf '%s' "$MISSION_DIRECTIVE_AUTHORS" | tr -d ' ,')" ]; then
  die 'MISSION_DIRECTIVE_AUTHORS is set but empty'
fi
authors="${MISSION_DIRECTIVE_AUTHORS:-MarkEdmondson1234}"
gh api --paginate "repos/$repo/issues/$bk/comments?per_page=100" \
  >"$tmp/comments.json" 2>"$tmp/err" || die "gh api comments on #$bk failed: $(tail -1 "$tmp/err")"
# Same allowlist parsing and paginated jq as triage; report comments by the loop
# are not directive authors. Validate shape before interpolation can hide missing data.
jq -es 'length>0 and all(.[]; type=="array" and all(.[];
    (.id|type)=="number" and (.user.login|type)=="string" and (.updated_at|type)=="string"))' \
  "$tmp/comments.json" >/dev/null 2>"$tmp/err" || die "comments on #$bk unparseable: $(tail -1 "$tmp/err")"
jq -rs --arg a "$authors" '($a | split(",") | map(gsub("^ +| +$"; ""))) as $who
    | if all(type=="array") then add // [] | .[] | select(.user.login as $l | $who | index($l))
    | "directive \(.id):\(.updated_at)" else error("not an array") end' \
  "$tmp/comments.json" >"$tmp/directives" 2>"$tmp/err" || die "comments on #$bk unparseable: $(tail -1 "$tmp/err")"

# stdout only: warnings go to stderr. The CLI marshals []InboxMessage, whose nil
# empty slice is JSON null (cmd/ailang/messages_crud.go, internal/messaging/inbox.go).
# --limit truncates silently: hitting it is a failed read, never a short queue.
# The store is PINNED here, as the driver's own sends pin it: the controller's zsh tool shell
# gets AILANG_STORAGE_MESSAGING=gcp from ~/.zshenv but the launchd bash driver does not, so an
# inherited store would make the reader (driver, local SQLite) and the writer (--record, gcp)
# hash different inboxes and the pre-check could never match.
LIMIT=1000
env AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT="${AILANG_MESSAGES_PROJECT:-ailang-multivac}" \
  ailang messages list --unread --inbox "mission-$name" --json --limit "$LIMIT" \
  >"$tmp/inbox.json" 2>"$tmp/err" || die "inbox read failed: $(tail -1 "$tmp/err")"
jq -ers 'if length==1 and (.[0]==null or (.[0]|type)=="array") then .[0] // []
    | if all(.[]; (.message_id|type)=="string" and (.message_id|length)>0) then .
      else error("missing message_id") end
    else error("not an inbox array/null") end | length' \
  "$tmp/inbox.json" >"$tmp/inbox-count" 2>"$tmp/err" || die "inbox JSON unparseable: $(tail -1 "$tmp/err")"
n_in=$(cat "$tmp/inbox-count")
[ "$n_in" -lt "$LIMIT" ] || die "inbox list hit --limit $LIMIT (truncates silently)"
jq -r '. // [] | .[] | "inbox \(.message_id)"' "$tmp/inbox.json" >"$tmp/inbox" || die 'inbox IDs unparseable'

# Check the TIP, even when it is a record-only commit. Only red conclusions wake;
# pending -> success does not. Paginated API objects can be consecutive JSON values.
gh api "repos/$repo/commits/$tip/check-runs" --paginate \
  >"$tmp/checks.json" 2>"$tmp/err" || die "gh check-runs failed: $(tail -1 "$tmp/err")"
jq -rs 'if length>0 and all(.[]; type=="object" and (.total_count|type)=="number"
      and (.check_runs|type)=="array" and all(.check_runs[];
        (.name|type)=="string" and has("conclusion") and (.conclusion==null or (.conclusion|type)=="string")))
    then .[] | .check_runs[]
      | select(.conclusion as $c | ["failure", "timed_out", "cancelled", "action_required", "startup_failure"] | index($c))
      | "red \(.name):\(.conclusion)"
    else error("not check-runs objects") end' \
  "$tmp/checks.json" >"$tmp/red" 2>"$tmp/err" || die "check-runs unparseable: $(tail -1 "$tmp/err")"
n_ledger=$(grep -c . "$tmp/ledger"); n_open=$(grep -c ':OPEN$' "$tmp/ledger")
n_dir=$(grep -c . "$tmp/directives"); n_red=$(grep -c . "$tmp/red")
hash=$(LC_ALL=C sort "$tmp/head" "$tmp/ledger" "$tmp/directives" "$tmp/inbox" "$tmp/red" | shasum -a 256 | awk '{print $1}') || die 'hash failed'
printf '%s' "$hash" | grep -Eq '^[0-9a-f]{64}$' || die 'hash failed'
line="$hash head=${head_sha:0:7} ledger=$n_ledger open=$n_open directives=$n_dir inbox=$n_in red=$n_red"
if [ "$mode" = snapshot ]; then
  # at= stays LAST: the driver's expiry parse anchors on it.
  mkdir -p "$state_dir" || die "cannot create $state_dir"
  if ! { printf '%s fire=%s at=%s\n' "$line" "$fire" "$(date +%s)" >"$pend_file.tmp.$$" \
    && mv "$pend_file.tmp.$$" "$pend_file"; }; then die "cannot write $pend_file"; fi
fi
printf '%s\n' "$line"
