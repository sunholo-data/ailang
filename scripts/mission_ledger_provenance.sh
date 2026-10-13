#!/usr/bin/env bash
# Read-only Gate 0 provenance, Bash 3.2/BSD. Git names are conventions, not authentication.
# Different git display names require --self <whole-name ERE>; no login/email lookup.
# GitHub-like committers conservatively imply ambiguity, not proof of a squash strategy.
set -uo pipefail
export LC_ALL=C
id=D-unknown; file=''; self='(sunholo-voight-kampff|Voight-Kampff \(bot\))'
creation=none; flip=none; seen_id=0; seen_file=0; seen_self=0
emit() { printf '%s verdict=%s flip=%s creation=%s\n' "$id" "$1" "$flip" "$creation"; }
fatal() { emit instrument-failure; printf 'ledger provenance: %s\n' "$1" >&2; exit 2; }
# Suppress raw git diagnostics: neither names nor addresses belong in this output.
read_git() { git "$@" 2>/dev/null; }
while [ "$#" -gt 0 ]; do
  [ "$#" -ge 2 ] || fatal 'missing option value'
  case "$1" in
    --id)
      [ "$seen_id" -eq 0 ] || fatal 'duplicate option'
      [[ "$2" =~ ^D-[A-Za-z0-9_-]+$ ]] || fatal 'invalid ID'
      id="$2"; seen_id=1 ;;
    --file) [ "$seen_file" -eq 0 ] || fatal 'duplicate option'; file="$2"; seen_file=1 ;;
    --self) [ "$seen_self" -eq 0 ] || fatal 'duplicate option'; self="$2"; seen_self=1 ;;
    *) fatal 'unknown option' ;;
  esac
  shift 2
done
[ "$seen_id" -eq 1 ] && [ -n "$file" ] || fatal 'require --id and --file'
[ -n "$self" ] || fatal 'empty self pattern'
printf '\n' | grep -Ei "^($self)$" >/dev/null 2>&1; pattern_rc=$?
[ "$pattern_rc" -le 1 ] || fatal 'invalid self pattern'
[ -f "$file" ] && [ -r "$file" ] || fatal 'file is not readable'
root="$(read_git rev-parse --show-toplevel)" || fatal 'git history unavailable'
prefix="$(read_git rev-parse --show-prefix)" || fatal 'git history unavailable'
case "$file" in
  "$root"/*) file="${file#"$root"/}" ;;
  /*) fatal 'file is outside repository' ;;
  *) file="$prefix$file" ;;
esac
cd "$root" || fatal 'repository unavailable'
# Resolve ./ components using Git's tracked path, preserving spaces and literal pathspecs.
file="$(read_git ls-files --full-name --error-unmatch -- ":(literal)$file")" || fatal 'git history unavailable'
[ -n "$file" ] && [ "$(printf '%s\n' "$file" | wc -l | tr -d ' ')" = 1 ] || fatal 'require one tracked file'
pathspec=":(literal)$file"
head="$(read_git rev-parse --verify HEAD)" || fatal 'git history unavailable'
shallow="$(read_git rev-parse --is-shallow-repository)" || fatal 'git history unavailable'
[ "$shallow" = false ] || fatal 'shallow history'
# status catches staged, unstaged and conflicted target changes, without a git write.
dirty="$(read_git --no-optional-locks status --porcelain --untracked-files=no -- "$pathspec")" || fatal 'git history unavailable'
[ -z "$dirty" ] || fatal 'uncommitted target changes'
row_re="^[[:space:]]*\|[[:space:]]*$id[[:space:]]*\|"
resolved_re="$row_re[[:space:]]*RESOLVED([[:space:]][^|]*)?[[:space:]]*\|"
# Snapshot absence is distinct from an object/read failure. Historical duplicate rows fail.
rows_at() {
  local revision="$1" exists content count
  exists="$(read_git ls-tree -r --name-only "$revision" -- "$pathspec")" || return 2
  if [ -z "$exists" ]; then printf ''; return; fi
  content="$(read_git show "$revision:$file")" || return 2
  count="$(printf '%s\n' "$content" | grep -Ec "$row_re")"
  [ "$count" -le 1 ] || return 2
  printf '%s\n' "$content" | grep -E "$row_re" || :
}
current="$(rows_at "$head")" || fatal 'git history unavailable'
[ -n "$current" ] || fatal 'missing current row'
# No rename following: whole-file additions must remain visible to -G.
# Refuse renamed paths instead of claiming the new path is the original creation.
renames="$(read_git log --full-history --format= --name-status --find-renames "$head" -- "$pathspec")" || fatal 'git history unavailable'
if printf '%s\n' "$renames" | grep -Eq '^R[0-9]+[[:space:]]'; then fatal 'renamed charter history'; fi
history="$(read_git log --full-history -m --no-renames --reverse --topo-order --format=%H -G"$row_re" "$head" -- "$pathspec")" || fatal 'git history unavailable'
for revision in $history; do
  parents="$(read_git rev-list --parents -n 1 "$revision")" || fatal 'git history unavailable'
  parents="${parents#* }"
  [ "$parents" != "$revision" ] || parents=''
  after="$(rows_at "$revision")" || fatal 'git history unavailable'
  before=''
  case "$parents" in *' '*)
    # An inherited row/resolution is already established on a parent. Only a new
    # merge introduction is ambiguous; ordinary later merges do not replace the flip.
    for parent in $parents; do
      parent_row="$(rows_at "$parent")" || fatal 'git history unavailable'
      if printf '%s\n' "$parent_row" | grep -Eq "$resolved_re"; then
        before="$parent_row"; break
      fi
      [ -z "$parent_row" ] || before="$parent_row"
    done
    if [ -n "$after" ] && [ -z "$before" ]; then fatal 'ambiguous merge creation'; fi
    if printf '%s\n' "$after" | grep -Eq "$resolved_re" && \
       ! printf '%s\n' "$before" | grep -Eq "$resolved_re"; then
      fatal 'ambiguous merge resolution'
    fi
    continue ;;
  esac
  if [ -n "$parents" ]; then before="$(rows_at "$parents")" || fatal 'git history unavailable'; fi
  patch="$(read_git diff-tree --root --no-commit-id --no-ext-diff --no-textconv --no-renames -r -p "$revision" -- "$pathspec")" || fatal 'git history unavailable'
  if [ -n "$after" ] && [ -z "$before" ]; then
    moved="$(read_git diff-tree --root --no-commit-id --name-status --find-renames -r "$revision")" || fatal 'git history unavailable'
    if printf '%s\n' "$moved" | awk -F '\t' -v path="$file" '$1 ~ /^R[0-9]+$/ && ($2 == path || $3 == path) { found=1 } END { exit !found }'; then
      fatal 'renamed charter history'
    fi
    [ "$creation" = none ] || fatal 'reused ID or interrupted row history'
    printf '%s\n' "$patch" | grep -Eq "^\+${row_re#^}" || fatal 'creation addition unavailable'
    creation="$revision"
  fi
  # LATEST verified non-RESOLVED -> RESOLVED addition: the transition that produced the
  # current resolution, so a reopen + fleet re-resolve is not hidden behind an earlier
  # attended flip. Removal and resolved-to-resolved reword are never transitions.
  if printf '%s\n' "$after" | grep -Eq "$resolved_re" && \
     ! printf '%s\n' "$before" | grep -Eq "$resolved_re" && \
     printf '%s\n' "$patch" | grep -Eq "^\+${resolved_re#^}"; then
    flip="$revision"
  fi
done
[ "$creation" != none ] || fatal 'creation history unavailable'
if [ "$flip" = none ]; then emit never-resolved; exit 0; fi
committer="$(read_git show -s --format=%cn "$flip")" || fatal 'git history unavailable'
if printf '%s\n' "$committer" | grep -Eiq '^(GitHub|GitHub\[bot\]|web-flow)$'; then
  emit undetermined:squash-merge; exit 0
fi
author="$(read_git show -s --format=%an "$flip")" || fatal 'git history unavailable'
if printf '%s\n' "$author" | grep -Eiq "^($self)$"; then emit fleet; else emit attended; fi
