#!/bin/bash
# Ancestry-ledger reaper for pi role runs. Ticket orphan:pi-runner:own-pgid-children-survive-role-exit
# (fleet i109). Plan: design_docs/planned/sprint-plan-pi-run-orphan-reap.md.
#
# Unit arms U1-U5 drive tools/launchd/lib/pi-run-reap.sh directly. Runner arms R0-R3 and the
# controls N1/N2 drive scripts/mission_pi_run.sh with a stubbed `pi` on PATH (no model is called).
#
# SAFETY: no process is ever selected by command line, cwd, pgid or session. Every fixture writes
# its own pid into $FIXDIR, and the test records its birth time (normalized lstart) right after the
# pid file appears. Cleanup and every "gone" check go by pid plus that birth, never by name.
set -u

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SUT="$ROOT/scripts/mission_pi_run.sh"
LIB="$ROOT/tools/launchd/lib/pi-run-reap.sh"
TMP=$(mktemp -d)
FIXDIR="$TMP/fix"; mkdir -p "$FIXDIR"
FIX="$TMP/fix.pl"
export FIXDIR FIX

passed=0; failed=0
ok()  { echo "PASS: $1"; passed=$((passed+1)); }
bad() { echo "FAIL: $1"; failed=$((failed+1)); }

# Normalized birth: the SAME transform the library uses (raw `ps lstart` has trailing blanks on darwin).
birth_of() { LC_ALL=C ps -p "$1" -o lstart= 2>/dev/null | awk 'NF>=5{print $1" "$2" "$3" "$4" "$5}'; }
alive() {
  local s
  kill -0 "$1" 2>/dev/null || return 1
  s=$(ps -p "$1" -o stat= 2>/dev/null) || return 1
  case "$s" in *Z*) return 1 ;; esac
  return 0
}
# gone_within PID SECONDS — bounded by a date +%s deadline.
gone_within() {
  local deadline=$(( $(date +%s) + $2 ))
  while alive "$1"; do
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 0.2
  done
  return 0
}
# fixwait NAME — bounded (10 s) wait for the fixture's pid file; records its birth; prints the pid.
fixwait() {
  local f="$FIXDIR/$1.pid" deadline=$(( $(date +%s) + 10 )) p
  while [ ! -f "$f" ]; do
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 0.2
  done
  p=$(cat "$f")
  birth_of "$p" > "$FIXDIR/$1.birth"
  echo "$p"
}
cleanup() {
  local f p b
  for f in "$FIXDIR"/*.pid; do
    [ -f "$f" ] || continue
    p=$(cat "$f" 2>/dev/null)
    b=$(cat "${f%.pid}.birth" 2>/dev/null)
    if [ -n "$p" ] && [ -n "$b" ] && alive "$p" && [ "$(birth_of "$p")" = "$b" ]; then
      kill -KILL "$p" 2>/dev/null
    fi
  done
  rm -rf "$TMP"
}
trap cleanup EXIT

# The fixture. Modes (all write $FIXDIR/NAME.pid atomically before anything else):
#   own       setpgrp, then exec /bin/sleep 3127
#   super     own, plus a child that setpgrps into a third group and execs
#             perl -e 'sleep 3127' 'hostile"arg\back'   (pid file NAME.c.pid)
#   stubborn  setpgrp, ignores TERM, plus the same child as super; loops on select
#   termfork  on TERM, forks a child (pid file NAME.c.pid) that execs /bin/sleep; keeps running
cat > "$FIX" <<'PERL'
#!/usr/bin/perl
use strict; use warnings;
my ($mode, $name) = @ARGV;
my $dir = $ENV{FIXDIR} or die "FIXDIR unset\n";
sub writepid { my ($tag, $pid) = @_; my $f = "$dir/$tag.pid";
  open my $h, '>', "$f.tmp" or die "open: $!"; print $h "$pid\n"; close $h;
  rename "$f.tmp", $f or die "rename: $!"; }
sub hostile_child {
  my $c = fork(); die "fork: $!" unless defined $c;
  if ($c == 0) { setpgrp(0, 0); writepid("$name.c", $$);
    exec {'/usr/bin/perl'} 'perl', '-e', 'sleep 3127', "hostile\"arg\\back"; die "exec: $!"; }
}
if ($mode eq 'own') {
  setpgrp(0, 0); writepid($name, $$);
  exec '/bin/sleep', '3127'; die "exec: $!";
}
if ($mode eq 'super') {
  setpgrp(0, 0); hostile_child(); writepid($name, $$);
  exec '/bin/sleep', '3127'; die "exec: $!";
}
if ($mode eq 'stubborn') {
  setpgrp(0, 0); hostile_child(); $SIG{TERM} = 'IGNORE'; writepid($name, $$);
  select(undef, undef, undef, 1) while 1;
}
if ($mode eq 'termfork') {
  $SIG{TERM} = sub { my $c = fork(); if (defined $c && $c == 0) {
      writepid("$name.c", $$); exec '/bin/sleep', '3127'; exit 1; } };
  writepid($name, $$);
  select(undef, undef, undef, 1) while 1;
}
die "unknown mode $mode\n";
PERL

# Pre-existing unrelated fixtures (controls). N1 is started before the runner arms, with its cwd in
# r1.wt and the same command line as the orphans. N2 is born during R1, orphaned, cwd in r1.wt.
# Both must survive every reap.
export PATH
MISSION_PI_POLL_SECONDS=1; export MISSION_PI_POLL_SECONDS
unset MISSION_PI_STALL_SECONDS MISSION_PI_TOOL_STALL_SECONDS MISSION_PI_MAX_SECONDS

# ---------------------------------------------------------------- unit arms (U1-U5)
[ -f "$LIB" ] && . "$LIB"
LIBTMP="$TMP/lib"; mkdir -p "$LIBTMP"

# U1 identity: a live process in the ledger with a WRONG birth must not be admitted or reaped.
perl "$FIX" own u1 </dev/null >/dev/null 2>&1 &
perl "$FIX" own v1 </dev/null >/dev/null 2>&1 &
UP=$(fixwait u1); VP=$(fixwait v1)
L1="$LIBTMP/u1.ledger"
printf '%s\t%s\n' "$UP" "Mon Jan 1 00:00:00 2001" > "$L1"
printf '%s\t%s\n' "$VP" "$(cat "$FIXDIR/v1.birth")" >> "$L1"
M1=$(_pr_members "$L1" 999999 x 2>/dev/null)
if printf '%s\n' "$M1" | awk -F'\t' -v v="$VP" -v u="$UP" '$1==v{hv=1} $1==u{hu=1} END{exit !(hv && !hu)}' \
   && _pr_reap "$L1" 999999 x 10 "$LIBTMP/u1.tsv" >/dev/null 2>&1 \
   && gone_within "$VP" 10 && alive "$UP" \
   && [ "$(cut -f1 "$LIBTMP/u1.tsv" 2>/dev/null | sort -u)" = "$VP" ]; then
  ok "U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped"
else
  bad "U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped"
fi

# U2 never-admit: the test shell, its parent, pid 1 — all in the ledger with REAL births — must never
# be listed. Positive control W proves the listing works. Call _pr_members ONLY: a mutant that admits
# $$ would make _pr_reap TERM this suite.
perl "$FIX" own w2 </dev/null >/dev/null 2>&1 &
WP=$(fixwait w2)
L2="$LIBTMP/u2.ledger"
{
  printf '%s\t%s\n' "$$" "$(birth_of "$$")"
  printf '%s\t%s\n' "$PPID" "$(birth_of "$PPID")"
  printf '%s\t%s\n' 1 "$(birth_of 1)"
  printf '%s\t%s\n' "$WP" "$(cat "$FIXDIR/w2.birth")"
} > "$L2"
M2=$(_pr_members "$L2" 999999 x 2>/dev/null)
if printf '%s\n' "$M2" | awk -F'\t' -v w="$WP" -v s="$$" -v p="$PPID" '
     $1==w{hw=1} $1==s{hs=1} $1==p{hp=1} $1==1{h1=1}
     END{exit !(hw && !hs && !hp && !h1)}'; then
  ok "U2 never-admit: \$\$, \$PPID and pid 1 never listed; positive control listed"
else
  bad "U2 never-admit: \$\$, \$PPID and pid 1 never listed; positive control listed"
fi

# U3 closure through a reparented member: super fixture Y (its own child Z). Only Y is in the ledger,
# and ROOT is dead. Z must still be listed through Y.
perl "$FIX" super y3 </dev/null >/dev/null 2>&1 &
YP=$(fixwait y3); ZP=$(fixwait y3.c)
L3="$LIBTMP/u3.ledger"
printf '%s\t%s\n' "$YP" "$(cat "$FIXDIR/y3.birth")" > "$L3"
M3=$(_pr_members "$L3" 999999 x 2>/dev/null)
if [ -n "$ZP" ] && printf '%s\n' "$M3" | awk -F'\t' -v z="$ZP" '$1==z{h=1} END{exit !h}'; then
  ok "U3 closure: child of a reparented ledger member is listed"
else
  bad "U3 closure: child of a reparented ledger member is listed"
fi

# U4 report JSON: a command with a quote, a backslash and a non-ASCII byte must survive as valid JSON.
R4="$LIBTMP/u4.tsv"
printf '123\t456\tTERM\t%s\n' "hostile\"arg\\back $(printf '\303\251')" > "$R4"
if _pr_report_json "$R4" 2>/dev/null | jq -e '.count==1 and .list[0].pid==123 and .list[0].signal=="TERM" and (.list[0].command|test("hostile\"arg\\\\back"))' >/dev/null 2>&1; then
  ok "U4 report JSON: quote, backslash and non-ASCII command round-trip through jq"
else
  bad "U4 report JSON: quote, backslash and non-ASCII command round-trip through jq"
fi

# U5 born during grace: T ignores nothing, but on TERM forks C. C is born AFTER the TERM pass, so it
# must be caught by the post-grace ledger refresh and KILLed. T survives TERM, so it is TERM+KILL.
perl "$FIX" termfork t5 </dev/null >/dev/null 2>&1 &
TP=$(fixwait t5)
L5="$LIBTMP/u5.ledger"
printf '%s\t%s\n' "$TP" "$(cat "$FIXDIR/t5.birth")" > "$L5"
_pr_reap "$L5" 999999 x 10 "$LIBTMP/u5.tsv" >/dev/null 2>&1
CP=$(fixwait t5.c)
if [ -n "$CP" ] && gone_within "$TP" 10 && gone_within "$CP" 10 \
   && awk -F'\t' -v t="$TP" '$1==t && $3=="TERM+KILL"{h=1} END{exit !h}' "$LIBTMP/u5.tsv" \
   && awk -F'\t' -v c="$CP" '$1==c && $3=="KILL"{h=1} END{exit !h}' "$LIBTMP/u5.tsv"; then
  ok "U5 born during grace: TERM-child caught by post-grace refresh and KILLed"
else
  bad "U5 born during grace: TERM-child caught by post-grace refresh and KILLed"
fi

# ---------------------------------------------------------------- runner arms (R0-R3, N1, N2)
stub() { # stub <shell body run as pi>
  { echo '#!/bin/bash'; echo 'cat >/dev/null'; echo ': > "$PI_SANDBOX_READY_FILE"'; echo "$1"; } > "$TMP/bin/pi"
  chmod +x "$TMP/bin/pi"
}
repo() { # fresh git worktree with one base commit
  rm -rf "$1"; mkdir -p "$1"
  git -C "$1" init -q
  echo base > "$1/f.txt"
  git -C "$1" add f.txt
  git -C "$1" -c user.email=t@t -c user.name=t commit -qm base
}
field() { sed -n "s/.*\"$2\": \"\{0,1\}\([^\",}]*\)\"\{0,1\}.*/\1/p" "$1" 2>/dev/null; }
mkdir -p "$TMP/node_modules/@anthropic-ai/sandbox-runtime" "$TMP/bin" "$TMP/claude"
echo '{}' > "$TMP/node_modules/@anthropic-ai/sandbox-runtime/package.json"
MISSION_PI_SANDBOX_NODE_MODULES="$TMP/node_modules"
MISSION_PI_SANDBOX_STAGE_PARENT="$TMP"
MISSION_PI_CLAUDE_TMP_DIR="$TMP/claude"
export MISSION_PI_SANDBOX_NODE_MODULES MISSION_PI_SANDBOX_STAGE_PARENT MISSION_PI_CLAUDE_TMP_DIR
PATH="$TMP/bin:$PATH"; export PATH

# runner NAME MAX_SECONDS [extra runner args...] — runs the SUT in the already-created $TMP/NAME.wt.
runner() {
  local name=$1 max=$2; shift 2
  echo directive > "$TMP/$name.directive"
  "$SUT" --model m --directive "$TMP/$name.directive" --workdir "$TMP/$name.wt" \
    --out "$TMP/$name.ndjson" --max-seconds "$max" "$@" >"$TMP/$name.log" 2>&1
  RUN_RC=$?
  RUN_VF="$TMP/$name.ndjson.verdict.json"
}

# Control N1: started from the test shell, cwd in r1.wt, same command line as the orphans.
repo "$TMP/r1.wt"
( cd "$TMP/r1.wt" && perl "$FIX" own n1 </dev/null >/dev/null 2>&1 ) &
NP1=$(fixwait n1)
# Control N2: born DURING R1, orphaned by its subshell, cwd in r1.wt.
( sleep 3; cd "$TMP/r1.wt" && perl "$FIX" own n2 </dev/null >/dev/null 2>&1 ) &

# R0 clean: no backgrounded descendants. The new keys must exist, be empty, and not move rc/verdict.
repo "$TMP/r0.wt"
stub 'echo changed > f.txt; printf "{\"type\":\"agent_end\"}\n"'
runner r0 30
if [ "$RUN_RC" = 0 ] && [ "$(field "$RUN_VF" verdict)" = ok ] && jq -e . "$RUN_VF" >/dev/null 2>&1 \
   && jq -e '.reaped_count==0 and .reaped==[]' "$RUN_VF" >/dev/null 2>&1; then
  ok "R0 clean: rc 0 ok, reaped_count 0, reaped []"
else
  bad "R0 clean: rc $RUN_RC verdict $(field "$RUN_VF" verdict); expected rc 0 ok with reaped_count 0"
fi

# R1 ok + Godot shape: super fixture r1 backgrounded by pi, pi exits (r1 is reparented), then ok.
# The reap must run on the ok path, and must take r1 and its child, but not N1 or N2.
# r1.wt is the worktree N1 and N2 already sit in, so it is NOT re-created here.
stub 'perl "$FIX" super r1 </dev/null >/dev/null 2>&1 & sleep 12; echo changed > f.txt; printf "{\"type\":\"agent_end\"}\n"'
echo directive > "$TMP/r1.directive"
"$SUT" --model m --directive "$TMP/r1.directive" --workdir "$TMP/r1.wt" \
  --out "$TMP/r1.ndjson" --max-seconds 40 >"$TMP/r1.log" 2>&1
RUN_RC=$?
RUN_VF="$TMP/r1.ndjson.verdict.json"
R1P=$(fixwait r1); R1C=$(fixwait r1.c)
if [ "$RUN_RC" = 0 ] && [ "$(field "$RUN_VF" verdict)" = ok ] && jq -e . "$RUN_VF" >/dev/null 2>&1 \
   && jq -e --argjson a "[$R1P,$R1C]" '.reaped_count==2 and (([.reaped[].pid]|sort)==($a|sort))' "$RUN_VF" >/dev/null 2>&1 \
   && jq -r '.reaped[].command' "$RUN_VF" 2>/dev/null | grep -qF 'hostile"arg\back' \
   && gone_within "$R1P" 10 && gone_within "$R1C" 10 && alive "$NP1"; then
  ok "R1 ok + Godot shape: reparented tree reaped on ok; N1 alive; child command recorded"
else
  bad "R1 ok + Godot shape: rc $RUN_RC verdict $(field "$RUN_VF" verdict) reaped_count $(jq -r '.reaped_count' "$RUN_VF" 2>/dev/null) (r1=$R1P child=$R1C)"
fi
N2P=$(fixwait n2)
if [ -n "$N2P" ] && alive "$N2P"; then ok "N2 control: born-during-run orphan in workdir survives (fixture fired)"
else bad "N2 control: born-during-run orphan in workdir survives (pid='$N2P')"; fi

# R2 wall + TERM-proof: r2 ignores TERM, so only the reaper's KILL pass can take it. Row must be TERM+KILL.
repo "$TMP/r2.wt"
stub 'perl "$FIX" stubborn r2 </dev/null >/dev/null 2>&1 & sleep 60'
runner r2 12
R2P=$(fixwait r2); R2C=$(fixwait r2.c)
if [ "$RUN_RC" = 13 ] && [ "$(field "$RUN_VF" verdict)" = wall_timeout ] && jq -e . "$RUN_VF" >/dev/null 2>&1 \
   && jq -e --argjson p "$R2P" '.reaped[] | select(.pid==$p) | .signal=="TERM+KILL"' "$RUN_VF" >/dev/null 2>&1 \
   && gone_within "$R2P" 10 && gone_within "$R2C" 10 && alive "$NP1"; then
  ok "R2 wall + TERM-proof: rc 13 wall_timeout; TERM+KILL row; tree gone; N1 alive"
else
  bad "R2 wall + TERM-proof: rc $RUN_RC verdict $(field "$RUN_VF" verdict) (r2=$R2P)"
fi

# R3 stall: silent pi, orphan parked under the stub. The stall kill takes the stub, the orphan is
# reparented, and the ledger from earlier polls must still reach it.
repo "$TMP/r3.wt"
stub 'perl "$FIX" own r3 </dev/null >/dev/null 2>&1 & sleep 60'
runner r3 40 --stall-seconds 10
R3P=$(fixwait r3)
if [ "$RUN_RC" = 12 ] && [ "$(field "$RUN_VF" verdict)" = stream_dead ] && jq -e . "$RUN_VF" >/dev/null 2>&1 \
   && gone_within "$R3P" 10 && alive "$NP1"; then
  ok "R3 stall: rc 12 stream_dead; orphan reaped after stub died; N1 alive"
else
  bad "R3 stall: rc $RUN_RC verdict $(field "$RUN_VF" verdict) (r3=$R3P)"
fi

# Final control check: N1 and N2 still alive after every reap.
if alive "$NP1" && alive "$N2P"; then ok "N1 and N2 controls alive after R1, R2 and R3"
else bad "N1 and N2 controls alive after R1, R2 and R3"; fi

echo "passed=$passed failed=$failed"
[ "$failed" -eq 0 ]
