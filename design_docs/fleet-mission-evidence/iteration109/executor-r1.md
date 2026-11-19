# Executor round 1 — recovered by the controller

The claude-haiku-5-5 executor hit its 1800 s alarm (rc 142, SIGALRM) after committing M1–M3 (`e68ae354a`, `be6fb2f59`, `7dbba8b4d`) and before writing its report. These are its on-disk working artifacts, verbatim.

## red-base.txt
```
FAIL: U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped
FAIL: U2 never-admit: $$, $PPID and pid 1 never listed; positive control listed
FAIL: U3 closure: child of a reparented ledger member is listed
FAIL: U4 report JSON: quote, backslash and non-ASCII command round-trip through jq
FAIL: U5 born during grace: TERM-child caught by post-grace refresh and KILLed
FAIL: R0 clean: rc 0 verdict ok; expected rc 0 ok with reaped_count 0
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count null (r1=66193 child=66197)
PASS: N2 control: born-during-run orphan in workdir survives (fixture fired)
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=67039)
FAIL: R3 stall: rc 12 verdict stream_dead (r3=68066)
PASS: N1 and N2 controls alive after R1, R2 and R3
passed=2 failed=9
/bin/bash tools/launchd/lib/suite-env.sh   1.19s user 1.92s system 4% cpu 1:09.51 total
rc=1
```

## m2-green.txt
```
PASS: U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped
PASS: U2 never-admit: $$, $PPID and pid 1 never listed; positive control listed
PASS: U3 closure: child of a reparented ledger member is listed
PASS: U4 report JSON: quote, backslash and non-ASCII command round-trip through jq
PASS: U5 born during grace: TERM-child caught by post-grace refresh and KILLed
PASS: R0 clean: rc 0 ok, reaped_count 0, reaped []
PASS: R1 ok + Godot shape: reparented tree reaped on ok; N1 alive; child command recorded
PASS: N2 control: born-during-run orphan in workdir survives (fixture fired)
PASS: R2 wall + TERM-proof: rc 13 wall_timeout; TERM+KILL row; tree gone; N1 alive
PASS: R3 stall: rc 12 stream_dead; orphan reaped after stub died; N1 alive
PASS: N1 and N2 controls alive after R1, R2 and R3
passed=11 failed=0
/bin/bash tools/launchd/lib/suite-env.sh   1.62s user 3.11s system 6% cpu 1:16.34 total
rc=0
```

## final-suites.txt
```
tools/launchd/test_mission_pi_run_orphan_reap.sh rc=0 passed=11 failed=0
scripts/test_mission_pi_run.sh rc=0 passed=36 failed=0
tools/launchd/test_mission_pi_run_commits.sh rc=0 passed=9 failed=0
tools/launchd/test_mission_pi_run_sandbox.sh rc=0 passed=13 failed=0
tools/launchd/test_mission_pi_run_provider_quota.sh rc=0 passed=29 failed=0
tools/launchd/test_mission_pi_run_tool_stall.sh rc=0 passed=16 failed=0
```

## mut-batchA.txt
```
== mutant 1 diff vs repo:
375d374
<   _pr_ledger_update "$REAP_LEDGER" "$RUNNER_PID" "$RUNNER_BIRTH"
mutant 1 suite rc=1
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 0 (r1=29655 child=29660)
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=30957)
FAIL: R3 stall: rc 12 verdict stream_dead (r3=31930)
passed=8 failed=3
== mutant 2 diff vs repo:
430c430
< _pr_reap "$REAP_LEDGER" "$RUNNER_PID" "$RUNNER_BIRTH" 2 "$STAGE/reaped.tsv"
---
> if [ "$OUTCOME" != finished ]; then _pr_reap "$REAP_LEDGER" "$RUNNER_PID" "$RUNNER_BIRTH" 2 "$STAGE/reaped.tsv"; fi
mutant 2 suite rc=1
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 0 (r1=34278 child=34284)
passed=10 failed=1
== mutant 3 diff vs repo:
(runner identical)
46c46
<         if ((kk[1] in b) && b[kk[1]]==kk[2] && !(kk[1] in bad)) m[kk[1]]=1
---
>         if ((kk[1] in b) && !(kk[1] in bad)) m[kk[1]]=1
mutant 3 suite rc=1
FAIL: U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped
passed=10 failed=1
== mutant 4 diff vs repo:
(runner identical)
41,42c41,42
<       while (x != "" && x+0 > 0 && !(x in bad)) { bad[x]=1; x=pp[x] }
<       bad[1]=1; bad[0]=1
---
mutant 4 suite rc=1
FAIL: U2 never-admit: $$, $PPID and pid 1 never listed; positive control listed
passed=10 failed=1
== mutant 5 diff vs repo:
(runner identical)
44c44
<       for (k in wp) {
---
>       for (k in wp) if (0) {
mutant 5 suite rc=1
FAIL: U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped
FAIL: U2 never-admit: $$, $PPID and pid 1 never listed; positive control listed
FAIL: U3 closure: child of a reparented ledger member is listed
FAIL: U5 born during grace: TERM-child caught by post-grace refresh and KILLed
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 0 (r1=49817 child=49822)
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=54441)
FAIL: R3 stall: rc 12 verdict stream_dead (r3=60267)
passed=4 failed=7
```

## mut-batchB.txt
```
== mutant 6 diff vs repo:
(runner identical)
96c96
<       if kill -KILL "$p" 2>/dev/null; then
---
>       if false; then
mutant 6 suite rc=1
FAIL: U5 born during grace: TERM-child caught by post-grace refresh and KILLed
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=79852)
passed=9 failed=2
== mutant 7 diff vs repo:
(runner identical)
89d88
<   _pr_ledger_update "$led" "$root" "$rootb"
mutant 7 suite rc=0
passed=11 failed=0
== mutant 8 diff vs repo:
(runner identical)
76c76
<   m=$(_pr_members "$led" "$root" "$rootb") || m=""
---
>   m=$(LC_ALL=C ps -A -o pid=,pgid=,stat=,lstart=,command= | awk '/sleep 3127/{p=$1; pg=$2; st=$3; b=$4" "$5" "$6" "$7" "$8; printf "%s\t%s\t%s\t%s\n", p, pg, st, b}')
mutant 8 suite rc=1
FAIL: U1 identity: wrong-birth member not admitted or reaped; right-birth member reaped
FAIL: U5 born during grace: TERM-child caught by post-grace refresh and KILLed
FAIL: R0 clean: rc 0 verdict ok; expected rc 0 ok with reaped_count 0
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 3 (r1=2568 child=2574)
FAIL: N2 control: born-during-run orphan in workdir survives (pid='2370')
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=3485)
FAIL: R3 stall: rc 12 verdict stream_dead (r3=4549)
FAIL: N1 and N2 controls alive after R1, R2 and R3
passed=3 failed=8
== mutant 9 diff vs repo:
(runner identical)
116a117
> _pr_report_json() { local n; n=$(wc -l < "$1" | tr -d ' '); printf '{"count":%s,"list":[{"command":"%s"}]}\n' "$n" "$(cut -f4 "$1" | head -1)"; }
mutant 9 suite rc=1
FAIL: U4 report JSON: quote, backslash and non-ASCII command round-trip through jq
FAIL: R0 clean: rc 0 verdict ok; expected rc 0 ok with reaped_count 0
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 2 (r1=6799 child=6805)
FAIL: R2 wall + TERM-proof: rc 13 verdict wall_timeout (r2=7718)
passed=7 failed=4
== mutant 10 diff vs repo:
435c435
< REAPED_COUNT=$(printf '%s' "$REAPED_JSON" | jq -r .count 2>/dev/null) || REAPED_COUNT=0
---
> REAPED_COUNT=$(wc -l < "$REAP_LEDGER" | tr -d " ")
mutant 10 suite rc=1
FAIL: R1 ok + Godot shape: rc 0 verdict ok reaped_count 6 (r1=10731 child=10737)
passed=10 failed=1
```
