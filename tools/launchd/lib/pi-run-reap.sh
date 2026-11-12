# pi-run-reap.sh — reap what a pi role run leaves behind. Sourced by scripts/mission_pi_run.sh.
# Plan: design_docs/planned/sprint-plan-pi-run-orphan-reap.md (ticket orphan:pi-runner:own-pgid-children-survive-role-exit).
#
# MECHANISM: an ancestry ledger. Every poll the runner calls _pr_ledger_update, which records
# (pid, birth) of every live descendant of the run's root, and of every descendant of a ledger
# member that is still alive with the same birth. At role exit _pr_reap signals that set on every
# outcome: TERM, grace, a ledger refresh (to catch children born during the grace), then KILL.
#
# WHY NOT PGID OR SESSION OR COMMAND LINE: pi's tools already setsid() (so a session reaches no tool
# descendant), a process can setpgrp away from its parent, and a command-line match would kill
# unrelated work on a shared rig. Membership here is proven by (pid, birth) plus ancestry only.
#
# BASH 3.2 SAFE: no declare -A, no mapfile, no ${v,,}, no BASHPID. Call from the main shell, never
# inside $( ). Hostile strings go through files and jq -R, never awk -v.
#
# RESIDUAL R1: a double-fork whose intermediate dies BETWEEN two polls is missed (the grandchild is
# reparented to pid 1 before any snapshot saw it attached).

# _pr_birth PID — normalized lstart (joins fields 1..5, so darwin's trailing blanks and %e padding
# do not matter). Prints nothing if PID is not alive.
_pr_birth() {
  LC_ALL=C ps -p "$1" -o lstart= 2>/dev/null | awk 'NF>=5{print $1" "$2" "$3" "$4" "$5}'
}

# _pr_members LEDGER ROOT ROOT_BIRTH — prints "pid<TAB>pgid<TAB>stat<TAB>birth" for every live member.
# Returns 1 and prints nothing when the snapshot is unusable (rule 4: a snapshot that does not list $$).
_pr_members() {
  local led=$1 root=$2 rootb=$3 snap
  snap=$(mktemp "${TMPDIR:-/tmp}/pi-reap.XXXXXX") || return 1
  if ! LC_ALL=C ps -A -o pid=,ppid=,pgid=,stat=,lstart= > "$snap" 2>/dev/null; then
    rm -f "$snap"; return 1
  fi
  if ! awk -v s="$$" '$1==s{f=1} END{exit !f}' "$snap"; then
    rm -f "$snap"; return 1
  fi
  awk -v self="$$" -v root="$root" -v rootb="$rootb" -v led="$led" '
    BEGIN { while ((getline l < led) > 0) { split(l, f, "\t"); wp[f[1] SUBSEP f[2]] = 1 }; close(led) }
    NF >= 9 { p=$1; pp[p]=$2; pg[p]=$3; st[p]=$4; b[p]=$5" "$6" "$7" "$8" "$9; ord[++n]=p }
    END {
      x=self
      while (x != "" && x+0 > 0 && !(x in bad)) { bad[x]=1; x=pp[x] }
      bad[1]=1; bad[0]=1
      if ((root in b) && b[root]==rootb && !(root in bad)) m[root]=1
      for (k in wp) {
        split(k, kk, SUBSEP)
        if ((kk[1] in b) && b[kk[1]]==kk[2] && !(kk[1] in bad)) m[kk[1]]=1
      }
      do {
        ch=0
        for (i=1;i<=n;i++) { p=ord[i]
          if (!(p in m) && !(p in bad) && (pp[p] in m)) { m[p]=1; ch=1 } }
      } while (ch)
      for (i=1;i<=n;i++) { p=ord[i]; if (p in m) printf "%s\t%s\t%s\t%s\n", p, pg[p], st[p], b[p] }
    }' "$snap"
  local rc=$?
  rm -f "$snap"
  return $rc
}

# _pr_ledger_update LEDGER ROOT ROOT_BIRTH — APPEND-ONLY: adds "pid<TAB>birth" for each member whose
# (pid, birth) is not already recorded. A failed snapshot leaves the ledger untouched.
_pr_ledger_update() {
  local led=$1 m
  m=$(_pr_members "$1" "$2" "$3") || return 0
  [ -z "$m" ] && return 0
  printf '%s\n' "$m" | awk -F'\t' -v led="$led" '
    BEGIN { while ((getline l < led) > 0) { split(l, f, "\t"); have[f[1] SUBSEP f[2]] = 1 }; close(led) }
    { k = $1 SUBSEP $4; if (!(k in have)) { print $1 "\t" $4; have[k] = 1 } }' >> "$led"
}

# _pr_reap LEDGER ROOT ROOT_BIRTH GRACE REPORT_TSV
# Writes REPORT_TSV rows "pid<TAB>pgid<TAB>signal<TAB>command", signal in {TERM, KILL, TERM+KILL}.
_pr_reap() {
  local led=$1 root=$2 rootb=$3 grace=$4 rep=$5 m cur p pg st b cmd sent=0
  _pr_ledger_update "$led" "$root" "$rootb"
  m=$(_pr_members "$led" "$root" "$rootb") || m=""
  : > "$rep"
  while IFS=$'\t' read -r p pg st b; do
    [ -n "$p" ] || continue
    case "$st" in *Z*) continue ;; esac
    [ "$(_pr_birth "$p")" = "$b" ] || continue
    cmd=$(LC_ALL=C ps -p "$p" -o command= 2>/dev/null | tr '\t\n' '  ' | cut -c1-200)
    if kill -TERM "$p" 2>/dev/null; then
      printf '%s\t%s\tTERM\t%s\n' "$p" "$pg" "$cmd" >> "$rep"
      sent=1
    fi
  done <<< "$m"
  [ "$sent" = 1 ] && sleep "$grace"
  _pr_ledger_update "$led" "$root" "$rootb"
  cur=$(_pr_members "$led" "$root" "$rootb") || cur=""
  while IFS=$'\t' read -r p pg st b; do
    [ -n "$p" ] || continue
    case "$st" in *Z*) continue ;; esac
    [ "$(_pr_birth "$p")" = "$b" ] || continue
    if awk -F'\t' -v p="$p" '$1==p{f=1} END{exit !f}' "$rep"; then
      if kill -KILL "$p" 2>/dev/null; then
        awk -F'\t' -v OFS='\t' -v p="$p" '$1==p && $3=="TERM"{$3="TERM+KILL"} {print}' "$rep" > "$rep.tmp" && mv "$rep.tmp" "$rep"
      fi
    else
      cmd=$(LC_ALL=C ps -p "$p" -o command= 2>/dev/null | tr '\t\n' '  ' | cut -c1-200)
      if kill -KILL "$p" 2>/dev/null; then
        printf '%s\t%s\tKILL\t%s\n' "$p" "$pg" "$cmd" >> "$rep"
      fi
    fi
  done <<< "$cur"
  return 0
}

# _pr_report_json REPORT_TSV — compact {"count":N,"list":[first 50 rows as objects]}. Built by jq -R -s
# so a hostile command string (quotes, backslashes, non-ASCII) cannot break the JSON.
_pr_report_json() {
  jq -R -s -c '
    split("\n") | map(select(length > 0) | split("\t")
      | {pid: (.[0] | tonumber), pgid: (.[1] | tonumber), signal: .[2], command: (.[3:] | join("\t"))}) as $all
    | {count: ($all | length), list: $all[0:50]}' "$1"
}
