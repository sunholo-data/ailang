#!/bin/bash
# Measure CPU accounting for the three stall-watchdog shapes. No threshold lives here.
set -u

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
rusage_helper="$(dirname "$0")/lib/proc_rusage.py"
window=120
samples=6
shape=all
out=
selftest=0
declare -a shapes parents births roots

usage() {
  cat <<'HELP'
Usage: measure_stall_cpu.sh [--window SECS] [--samples N] [--shape drill|w1-git|w1-gh|w2|burn|all] [--out FILE] [--selftest] [--help]
  --window SECS  Seconds between samples (default 120)
  --samples N    Number of samples (default 6; five windows)
  --shape NAME   Default all; all four fixtures run concurrently, so wall time is about samples*window
  --out FILE     TSV destination (default: mktemp path printed on stderr)
  --selftest     Three samples six seconds apart, w1-git, w2 and burn
  --help         Show this help
measurement only — no threshold is applied or proposed by this script
CPU is read from proc_pid_rusage (ri_child_* = reaped children); ps -S columns are retained as a control because macOS ps -S does not fold reaped children.
HELP
}

positive_integer() {
  case "$1" in ''|*[!0-9]*) return 1 ;; esac
  [ "$1" -gt 0 ] 2>/dev/null
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --window|--samples|--shape|--out)
      [ "$#" -ge 2 ] || { echo "missing value for $1" >&2; exit 2; }
      case "$1" in
        --window) window=$2 ;;
        --samples) samples=$2 ;;
        --shape) shape=$2 ;;
        --out) out=$2 ;;
      esac
      shift 2 ;;
    --selftest) selftest=1; shift ;;
    --help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
done
positive_integer "$window" && positive_integer "$samples" && [ "$samples" -ge 2 ] || {
  echo "--window must be positive and --samples must be at least 2" >&2; exit 2;
}
case "$shape" in drill|w1-git|w1-gh|w2|burn|all) ;; *) echo "invalid --shape: $shape" >&2; exit 2 ;; esac
if [ "$selftest" -eq 1 ]; then window=6; samples=3; shape=selftest; fi

# BSD ps time is [days-][hours:]minutes:seconds[.centiseconds].
# Normalize all four displayed forms (and integer-second displays) to centiseconds.
parse_time() {
  awk -v value="$1" 'BEGIN {
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
    days=0; fraction=0
    if (index(value,"-")) {
      count=split(value,d,"-"); if (count!=2 || d[1]!~/^[0-9]+$/) exit 1
      days=d[1]+0; value=d[2]
    }
    if (index(value,".")) {
      count=split(value,f,"[.]"); if (count!=2 || f[2]!~/^[0-9]+$/ || length(f[2])>2) exit 1
      fraction=(length(f[2])==1 ? f[2]*10 : f[2]+0); value=f[1]
    }
    n=split(value,t,":")
    if (n!=2 && n!=3) exit 1
    for (i=1;i<=n;i++) if (t[i]!~/^[0-9]+$/) exit 1
    if (n==3) seconds=t[1]*3600+t[2]*60+t[3]
    else seconds=t[1]*60+t[2]
    printf "%.0f\n", ((days*86400)+seconds)*100+fraction
  }'
}

birth_of() {
  ps -p "$1" -o lstart= 2>/dev/null | awk '{$1=$1; print; exit}'
}

# Use one ps parent table per walk; child-first order also serves cleanup.
descendants() {
  ps -A -o pid=,ppid= 2>/dev/null | awk -v root="$1" '
    { parent[$1]=$2; present[$1]=1 }
    function visit(p,    c) {
      for (c in parent) if (parent[c]==p) visit(c)
      print p
    }
    END { if (present[root]) visit(root) }
  '
}

# All signalling passes through this identity gate. Never pass a process group.
kill_safe() {
  local pid="$1" birth="$2" signal="$3" now
  case "$pid" in ''|*[!0-9]*) return 1 ;; esac
  [ "$pid" -gt 1 ] 2>/dev/null || return 1
  [ "$pid" -ne "$$" ] && [ "$pid" -ne "${BASHPID:-$$}" ] || return 1
  [ -n "$birth" ] || return 1
  now=$(birth_of "$pid")
  [ -n "$now" ] && [ "$now" = "$birth" ] || return 1
  kill "-$signal" "$pid" 2>/dev/null
}

cleanup() {
  local i pid b j
  local -a victims victim_births
  trap - EXIT INT TERM
  for ((i=0;i<${#parents[@]};i++)); do
    victims=(); victim_births=()
    # A reused fixture parent cannot authorize signalling any of its descendants.
    [ "$(birth_of "${parents[i]}")" = "${births[i]}" ] || continue
    for pid in $(descendants "${parents[i]}"); do
      b=$(birth_of "$pid")
      [ -n "$b" ] || continue
      victims[${#victims[@]}]="$pid"
      victim_births[${#victim_births[@]}]="$b"
    done
    for ((j=0;j<${#victims[@]};j++)); do
      kill_safe "${victims[j]}" "${victim_births[j]}" TERM || true
    done
    for ((j=0;j<${#victims[@]};j++)); do
      kill_safe "${victims[j]}" "${victim_births[j]}" KILL || true
    done
  done
  for ((i=0;i<${#roots[@]};i++)); do rm -rf -- "${roots[i]}"; done
}
trap 'exit 130' INT
trap 'exit 143' TERM
trap cleanup EXIT

# Do not launch persistent fixtures when the sandbox cannot identify processes.
if [ -z "$(birth_of "$$")" ] || ! ps -A -o pid=,ppid= 2>/dev/null | awk -v own="$$" '$1==own {seen=1} END {exit !seen}'; then
  echo "ps cannot observe the measuring process; no fixtures launched" >&2
  exit 2
fi

start_fixture() {
  local kind="$1" dir parent birth go_version
  dir=$(mktemp -d "${TMPDIR:-/tmp}/measure-stall.XXXXXXXX") || exit 2
  roots[${#roots[@]}]="$dir"
  case "$kind" in
    drill)
      go_version=$(go env GOVERSION 2>/dev/null) || { echo "go env GOVERSION failed" >&2; exit 2; }
      go_version=$(printf '%s\n' "$go_version" | sed -E 's/^go([0-9]+\.[0-9]+).*/\1/')
      case "$go_version" in *[!0-9.]*|'') echo "invalid Go version" >&2; exit 2 ;; esac
      printf 'module drill\n\ngo %s\n' "$go_version" > "$dir/go.mod"
      cat > "$dir/row_test.go" <<'GO'
package drill
import ("testing"; "time")
func TestRow(t *testing.T) { time.Sleep(3*time.Second); if Row < 0 { t.Fatal(Row) } }
GO
      # Keep the caller's default GOCACHE: its content-addressed writes are harmless.
      # A fresh GOCACHE would rebuild the standard library on row 1 and over-measure.
      TMPDIR="$dir" GOTELEMETRY=off GOTOOLCHAIN=local /bin/bash -c '
        dir=$1; n=0
        while :; do
          printf "package drill\nconst Row = %s\n" "$n" > "$dir/row.go"
          (cd "$dir" && go test -count=1 ./... >/dev/null 2>&1)
          n=$((n+1))
        done
      ' _ "$dir" & parent=$! ;;
    w1-git)
      GIT_OPTIONAL_LOCKS=0 /bin/bash -c '
        REPO_ROOT=$1
        until git -C "$REPO_ROOT" status --porcelain >/dev/null 2>&1 && false; do sleep 30; done
      ' _ "$REPO_ROOT" & parent=$! ;;
    w1-gh)
      GH_PROMPT_DISABLED=1 /bin/bash -c '
        until gh run list --repo sunholo-data/ailang --workflow CI --limit 1 --json status >/dev/null 2>&1 && false; do sleep 30; done
      ' & parent=$! ;;
    w2)
      mkfifo "$dir/blocked.fifo" || exit 2
      /bin/bash -c 'read -r line < "$1"' _ "$dir/blocked.fifo" & parent=$! ;;
    burn)
      /bin/bash -c 'while :; do perl -e '\''my $t=time; 1 while time-$t<2'\''; sleep 2; done' & parent=$! ;;
  esac
  birth=$(birth_of "$parent")
  [ -n "$birth" ] || { echo "cannot observe $kind parent PID $parent" >&2; exit 2; }
  shapes[${#shapes[@]}]="$kind"
  parents[${#parents[@]}]="$parent"
  births[${#births[@]}]="$birth"
}

# Snapshot values are globals so missing data can be reported rather than zeroed.
snapshot() {
  local root="$1" pid birth plain folded ru_output ru_pid ru_self ru_child ru_count j match
  local -a live_pids ps_plain ps_folded ru_seen
  snap_birth=; snap_parentS=; snap_treeS=0; snap_tree=0; snap_parentRU=; snap_treeRU=0; snap_note=ok; snap_vanished=0
  live_pids=(); ps_plain=(); ps_folded=(); ru_seen=()
  for pid in $(descendants "$root"); do
    birth=$(birth_of "$pid")
    plain=$(ps -p "$pid" -o time= 2>/dev/null)
    folded=$(ps -S -p "$pid" -o time= 2>/dev/null)
    if [ -z "$birth" ] || [ -z "$plain" ] || [ -z "$folded" ]; then
      if [ "$pid" != "$root" ]; then snap_vanished=$((snap_vanished+1)); continue; fi
      snap_note=missing-ps; continue
    fi
    live_pids[${#live_pids[@]}]="$pid"
    plain=$(parse_time "$plain") && folded=$(parse_time "$folded") || {
      snap_note=invalid-ps-time; continue;
    }
    ps_plain[${#live_pids[@]}-1]="$plain"
    ps_folded[${#live_pids[@]}-1]="$folded"
    if [ "$pid" = "$root" ]; then snap_birth="$birth"; snap_parentS="$folded"; fi
  done
  [ -n "$snap_parentS" ] || snap_note=missing-parent
  if [ "${#live_pids[@]}" -eq 0 ] || ! ru_output=$(python3 "$rusage_helper" "${live_pids[@]}"); then
    snap_note=rusage-unavailable
    return
  fi
  ru_count=0
  while IFS=$'\t' read -r ru_pid ru_self ru_child; do
    match=-1
    for ((j=0;j<${#live_pids[@]};j++)); do
      if [ "$ru_pid" = "${live_pids[j]}" ]; then match=$j; break; fi
    done
    if [ "$match" -lt 0 ] || [ "${ru_seen[match]:-0}" -eq 1 ]; then snap_note=rusage-unavailable; continue; fi
    ru_seen[match]=1
    ru_count=$((ru_count+1))
    if [ "$ru_self" = NA ] && [ "$ru_child" = NA ]; then
      if [ "$ru_pid" = "$root" ]; then snap_note=rusage-unavailable
      else snap_vanished=$((snap_vanished+1)); fi
      continue
    fi
    case "$ru_self:$ru_child" in
      *[!0-9:]*|:|*:|:*) snap_note=rusage-unavailable; continue ;;
    esac
    # A vanished child's CPU moves into its parent's ri_child_* once reaped.
    # Include ps and rusage values only for children present in both reads.
    snap_tree=$((snap_tree+${ps_plain[match]:-0}))
    # Live tree -S can double count a live child's already-reaped children.
    snap_treeS=$((snap_treeS+${ps_folded[match]:-0}))
    snap_treeRU=$((snap_treeRU+ru_self+ru_child))
    if [ "$ru_pid" = "$root" ]; then snap_parentRU=$((ru_self+ru_child)); fi
  done <<< "$ru_output"
  [ "$ru_count" -eq "${#live_pids[@]}" ] || snap_note=rusage-unavailable
  [ -n "$snap_parentRU" ] || snap_note=rusage-unavailable
  # ri_child_* counts only reaped children. A live child's reaped children stay
  # with that child until it is reaped, when its full total moves to its parent.
  # Thus summing (self + child) over the live tree does not double count.
  if [ "$snap_note" = ok ] && [ "$snap_vanished" -gt 0 ]; then snap_note="ok;vanished=$snap_vanished"; fi
}

valid_note() {
  case "$1" in
    ok) return 0 ;;
    ok\;vanished=*) positive_integer "${1#ok;vanished=}" ;;
    *) return 1 ;;
  esac
}

case "$shape" in
  all) for s in drill w1-git w1-gh w2; do start_fixture "$s"; done ;;
  selftest) for s in w1-git w2 burn; do start_fixture "$s"; done ;;
  *) start_fixture "$shape" ;;
esac
if [ -z "$out" ]; then
  out=$(mktemp "${TMPDIR:-/tmp}/measure-stall-cpu.XXXXXXXX.tsv") || exit 2
  echo "TSV: $out" >&2
fi
printf 'shape\twindow\tt_start\tt_end\telapsed_s\tpid\tlstart\tparentS_cs\ttreeS_cs\ttree_cs\tparentRU_cs\ttreeRU_cs\tnote\n' > "$out" || exit 2

declare -a old_birth old_parentS old_treeS old_tree old_parentRU old_treeRU old_note old_vanished old_time
deadline=$(($(date +%s)+samples*window+60))
check_deadline() {
  if [ "$(date +%s)" -gt "$deadline" ]; then
    echo "HARD DEADLINE EXCEEDED: measurement exceeded samples*window + 60 s" >&2
    exit 3
  fi
}
for ((sample=0;sample<samples;sample++)); do
  check_deadline
  for ((i=0;i<${#shapes[@]};i++)); do
    snapshot "${parents[i]}"
    now=$(date +%s)
    if [ "$sample" -gt 0 ]; then
      note=ok; delta_p=; delta_ts=; delta_t=; delta_pru=; delta_tru=
      if [ "$snap_note" = rusage-unavailable ] || [ "${old_note[i]}" = rusage-unavailable ]; then note=rusage-unavailable
      elif [ "${old_birth[i]}" != "$snap_birth" ]; then note=pid-reused
      else
        delta_pru=$((snap_parentRU-old_parentRU[i]))
        delta_tru=$((snap_treeRU-old_treeRU[i]))
        if ! valid_note "$snap_note" || ! valid_note "${old_note[i]}"; then note=missing-sample
        else
          delta_p=$((snap_parentS-old_parentS[i]))
          delta_ts=$((snap_treeS-old_treeS[i]))
          delta_t=$((snap_tree-old_tree[i]))
          vanished=$((snap_vanished+old_vanished[i]))
          if [ "$vanished" -gt 0 ]; then note="ok;vanished=$vanished"; fi
        fi
      fi
      printf '%s\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n' \
        "${shapes[i]}" "$sample" "${old_time[i]}" "$now" "$((now-old_time[i]))" \
        "${parents[i]}" "$snap_birth" "$delta_p" "$delta_ts" "$delta_t" "$delta_pru" "$delta_tru" "$note" >> "$out"
    fi
    old_birth[i]="$snap_birth"; old_parentS[i]="$snap_parentS"
    old_treeS[i]="$snap_treeS"; old_tree[i]="$snap_tree"
    old_parentRU[i]="$snap_parentRU"; old_treeRU[i]="$snap_treeRU"
    old_note[i]="$snap_note"; old_vanished[i]="$snap_vanished"; old_time[i]="$now"
  done
  check_deadline
  if [ "$sample" -lt "$((samples-1))" ]; then
    remaining=$((deadline-$(date +%s)))
    [ "$remaining" -gt 0 ] || { echo "HARD DEADLINE EXCEEDED: measurement exceeded samples*window + 60 s" >&2; exit 3; }
    sleep "$window"
  fi
done

for ((i=0;i<${#shapes[@]};i++)); do
  awk -F '\t' -v shape="${shapes[i]}" '
    $1==shape {
      if ($13!~/^ok(;vanished=[1-9][0-9]*)?$/ || $8=="" || $9=="" || $10=="" || $11=="" || $12=="") { bad++; next }
      if ($13 ~ /^ok;vanished=/) vanished_windows++
      n++; p[n]=$8/100; s[n]=$9/100; t[n]=$10/100; pr[n]=$11/100; tr[n]=$12/100
    }
    function report(label,a,    x,j,k,v) {
      for (j=1;j<=n;j++) for (k=j+1;k<=n;k++) if (a[k]<a[j]) {v=a[j];a[j]=a[k];a[k]=v}
      if (n) printf "  %s min=%.2f median=%.2f max=%.2f CPU-s\n",label,a[1],(n%2?a[(n+1)/2]:(a[n/2]+a[n/2+1])/2),a[n]
      else printf "  %s min=NA median=NA max=NA CPU-s\n",label
    }
    END {
      printf "SUMMARY %s windows=%d invalid=%d vanished_windows=%d\n",shape,n,bad,vanished_windows
      report("parentS",p); report("treeS",s); report("tree",t)
      report("parentRU",pr); report("treeRU",tr)
    }
  ' "$out"
done

if [ "$selftest" -eq 1 ]; then
  expected=$((1+3*(samples-1)))
  actual=$(awk 'END {print NR}' "$out")
  echo "INFO: ps -S parentS for burn: $(awk -F '\t' '$1=="burn" {printf "%s%s", (n++ ? "," : ""), $8}' "$out")"
  if [ "$actual" -eq "$expected" ]; then echo "PASS: TSV rows=$actual (header + six windows)"
  else echo "FAIL: TSV rows=$actual expected=$expected"; exit 1; fi
  if awk -F '\t' 'NR>1 {if ($13!~/^ok(;vanished=[1-9][0-9]*)?$/ || $11!~/^-?[0-9]+$/ || $12!~/^-?[0-9]+$/) exit 1; if ($1=="w2" && $11!=0) exit 1}' "$out"; then
    echo "PASS: numeric rusage deltas; w2 parentRU=0 in every window"
  else
    echo "FAIL: missing/non-numeric rusage delta or w2 parentRU nonzero"
    exit 1
  fi
  if awk -F '\t' '$1=="burn" && $11<100 {exit 1}' "$out"; then
    echo "PASS: burn parentRU >= 100 cs in every window"
  else
    echo "FAIL: rusage positive control did not fire"
    exit 1
  fi
fi
