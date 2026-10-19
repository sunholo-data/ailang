#!/bin/bash
# Hermetic Bash 3.2/file:// regression suite. --mutations changes disposable copies only.
set -u
TEST_FILE="$(cd "$(dirname "$0")" && pwd)/$(basename "$0")"
if [ "${1:-}" != --isolated ]; then
    helper=${CLOUD_BRANCH_SYNC_HELPER:-$(dirname "$TEST_FILE")/cloud_branch_sync.sh}
    helper="$(cd "$(dirname "$helper")" && pwd)/$(basename "$helper")"
    real_git=$(command -v git) || exit 1
    root=$(mktemp -d "${TMPDIR:-/tmp}/cloud-branch-sync.XXXXXX") || exit 1
    trap 'rm -rf "$root"' EXIT
    mkdir -p "$root/home" "$root/tmp"
    # No inherited Git config, repository redirection, credentials, identities, hooks,
    # signing, task IDs or branch overrides. Never examine the caller's HOME or .git.
    env -i PATH="$PATH" HOME="$root/home" TMPDIR="$root/tmp" \
        GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null GIT_CONFIG_NOSYSTEM=1 \
        GIT_TERMINAL_PROMPT=0 GIT_ASKPASS=/usr/bin/false SSH_ASKPASS=/usr/bin/false \
        GIT_AUTHOR_NAME=Fixture GIT_AUTHOR_EMAIL=fixture@example.invalid \
        GIT_COMMITTER_NAME=Fixture GIT_COMMITTER_EMAIL=fixture@example.invalid \
        /bin/bash "$TEST_FILE" --isolated "$root" "$real_git" "$helper" \
        "${CLOUD_BRANCH_SYNC_ARM:-all}" "${1:-test}"
    exit $?
fi
root=$2; REAL_GIT=$3; HELPER=$4; selected=$5; mode=$6
export REAL_GIT
cd "$root" || exit 1
PASS=0; FAIL=0; B=coordinator/task-x
check() { "$@" || { echo "assertion failed: $*" >&2; return 1; }; }
eq() { [ "$1" = "$2" ] || { echo "expected [$2], observed [$1]" >&2; return 1; }; }
contains() { case "$1" in *"$2"*) return 0 ;; *) echo "missing [$2] in [$1]" >&2; return 1 ;; esac; }
git() { "$REAL_GIT" "$@"; }
head_sha() { git rev-parse HEAD; }
remote_sha() { git --git-dir="$remote" rev-parse "refs/heads/$B" 2>/dev/null || echo absent; }
tracking_sha() { git rev-parse --verify "refs/remotes/origin/$B" 2>/dev/null || echo absent; }
refs() { git --git-dir="$remote" for-each-ref --format='%(refname) %(objectname)'; }
config() { git config --local --list; }
# Observe HEAD/index/tracked/untracked bytes and status, not just the verdict.
state() {
    git rev-parse HEAD; git symbolic-ref -q HEAD || echo detached
    git ls-files --stage; git status --porcelain
    find . -type f ! -path './.git/*' | LC_ALL=C sort | while IFS= read -r f; do
        printf '%s ' "$f"; git hash-object "$f"
    done
    git hash-object .git/index
}
fixture_new() {
    fixture="$root/$1"; mkdir -p "$fixture"; remote="$fixture/origin.git"
    git -c init.templateDir= init --bare -q "$remote" || return 1
    git -c init.templateDir= init -q "$fixture/seed" || return 1
    cd "$fixture/seed" || return 1
    git symbolic-ref HEAD refs/heads/dev
    printf 'base\n' > work; printf '{"passes":false}\n' > sprint.json
    git add work sprint.json; git commit -qm base || return 1
    git remote add origin "file://$remote"
    git push -q origin HEAD:refs/heads/dev HEAD:refs/heads/main || return 1
    clone_task "$fixture/clone"
}
clone_task() {
    git -c init.templateDir= clone -q --depth 1 --branch dev "file://$remote" "$1" || return 1
    cd "$1" || return 1
    check test "$(git rev-parse --is-shallow-repository)" = true || return 1
    git checkout -qb "$B" || return 1
    eq "$(git config --get-all remote.origin.fetch)" '+refs/heads/dev:refs/remotes/origin/dev'
}
commit_work() {
    printf '%s\n' "$1" > work; printf '{"passes":"%s"}\n' "$1" > sprint.json
    git add work sprint.json; git commit -qm "$1"
}
invoke() {
    out=$(AILANG_TASK_ID=task-x /bin/bash "$HELPER" "$@" 2>"$fixture/helper.err"); rc=$?
    eq "$rc" 0 || return 1
    contains "$out" "branch=$B" || return 1
    contains "$out" "sha=$(head_sha)" || return 1
}
verdict() { contains "$out" "cloud-branch-sync: $1"; }
refspec_once() {
    eq "$(git config --get-all remote.origin.fetch | grep -Fxc "+refs/heads/$B:refs/remotes/origin/$B")" 1 || return 1
    eq "$(git config --get-all remote.origin.fetch | grep -Fxc '+refs/heads/dev:refs/remotes/origin/dev')" 1 || return 1
    eq "$(git config --get-all remote.origin.fetch | wc -l | tr -d ' ')" 2
}
published() { eq "$(remote_sha)" "$(head_sha)" && eq "$(tracking_sha)" "$(head_sha)"; }
# Step 5b, cmd/ailang/coordinator_cloud_task.go; branchNeedsPush in coordinator_cloud_branch.go.
# Keep this production decision verbatim, including the bare branch fallback and separate stderr.
wrapper_simulation() {
logOutput=$(git log "origin/${B}..HEAD" --oneline 2>"${fixture}/log.err")
logRC=$?
newBranch=false
lsRC=not-run
if [ "$logRC" -ne 0 ]; then
    git ls-remote --exit-code origin "$B" >"${fixture}/ls.out" 2>"${fixture}/ls.err"
    lsRC=$?
    if [ "$lsRC" -ne 0 ]; then
        newBranch=true
    fi
fi
# branchNeedsPush: newBranch || len(strings.TrimSpace(logOutput)) > 0
unpushed=false
if [ "$newBranch" = true ] || [ -n "$(printf '%s' "$logOutput" | tr -d '[:space:]')" ]; then
    unpushed=true
fi
    echo "WRAPPER logRC=$logRC lsRC=$lsRC newBranch=$newBranch unpushed=$unpushed HEAD=$(head_sha) tracking=$(tracking_sha) remote=$(remote_sha)"
}
# Every shim logs the real interception; production has no test bypass.
shim() {
    mkdir -p "$fixture/bin"
    cat > "$fixture/bin/git" <<'SHIM'
#!/bin/bash
printf '%s\n' "$*" >> "$SHIM_LOG"
case "$SHIM_KIND:$*" in
    fetch:fetch*) exit 71 ;;
    probe:ls-remote*) exit 72 ;;
    verify:ls-remote*) printf '%040d\t%s\n' 0 "refs/heads/coordinator/task-x"; exit 0 ;;
    missing:ls-remote*) exit 2 ;;
    config:config\ --local\ --add*) exit 73 ;;
    stall:ls-remote*)
        printf '%s\n' "$$" > "$SHIM_PID"
        printf 'prompt=%s askpass=%s ssh=%s\n' "${GIT_TERMINAL_PROMPT:-}" "${GIT_ASKPASS:-}" "${GIT_SSH_COMMAND:-}" > "$SHIM_ENV"
        exec sleep 20 ;;
esac
exec "$REAL_GIT" "$@"
SHIM
    chmod +x "$fixture/bin/git"
    export SHIM_LOG="$fixture/shim.log" SHIM_PID="$fixture/shim.pid" SHIM_ENV="$fixture/shim.env"
    : > "$SHIM_LOG"
}
with_shim() { PATH="$fixture/bin:$PATH" SHIM_KIND="$1" invoke "$2"; }
no_mutating_network() { ! grep -E '^(fetch|push|reset)' "$SHIM_LOG"; }
skip_case() {
    local branch=$1 id_mode=$2 command before after cfg allrefs trace
    fixture_new "skip-${branch//\//-}-$id_mode" || return 1
    if [ "$branch" = detached ]; then git checkout -q --detach
    else git checkout -qB "$branch"; fi
    commit_work unpublished || return 1
    before=$(state); cfg=$(config); allrefs=$(refs)
    for command in start push; do
        trace="$fixture/trace-$command"
        case "$id_mode" in
            unset) out=$(unset AILANG_TASK_ID; GIT_TRACE="$trace" /bin/bash "$HELPER" "$command" 2>"$fixture/skip.err"); rc=$? ;;
            empty) out=$(AILANG_TASK_ID= GIT_TRACE="$trace" /bin/bash "$HELPER" "$command" 2>"$fixture/skip.err"); rc=$? ;;
            *) out=$(AILANG_TASK_ID=task-x AILANG_PUSH_BRANCH="$branch" GIT_TRACE="$trace" /bin/bash "$HELPER" "$command" 2>"$fixture/skip.err"); rc=$? ;;
        esac
        # Observe writes even if a lying mutant prints a correct skipped verdict.
        eq "$(refs)" "$allrefs" || return 1
        eq "$(config)" "$cfg" || return 1
        after=$(state); eq "$after" "$before" || return 1
        eq "$rc" 0 && verdict skipped: || return 1
        if [ "$branch" = detached ]; then contains "$out" 'branch=<detached>' || return 1
        else contains "$out" "branch=$branch" || return 1; fi
        contains "$out" "sha=$(head_sha)" || return 1
        if [ -f "$trace" ]; then
            ! grep -E 'built-in: git (fetch|push|ls-remote|reset|config .*--add)' "$trace" || return 1
        fi
    done
}
A01() { skip_case dev set && skip_case main set; }
A02() { skip_case mission/iteration-49 set; }
A03() { skip_case coordinator/task-other set && skip_case coordinator/task-x-other set && skip_case coordinator/task-xy set; }
A04() { skip_case "$B" unset && skip_case "$B" empty; }
A05() { skip_case detached set; }
A06() {
    local command i
    for command in start push; do
        fixture_new "A06-$command" || return 1
        commit_work milestone || return 1
        for i in 1 2; do invoke "$command" && refspec_once || return 1; done
        if [ "$command" = push ]; then published || return 1; fi
    done
}
A07() {
    fixture_new A07 && commit_work milestone-1 && invoke push && verdict "pushed $(head_sha)" && published && refspec_once
}
A08() {
    fixture_new A08 && commit_work milestone-1 && invoke push && verdict pushed || return 1
    local first; first=$(head_sha)
    commit_work milestone-2 && wrapper_simulation || return 1
    eq "$logRC" 0 && contains "$logOutput" milestone-2 && eq "$lsRC" not-run && eq "$newBranch" false && eq "$unpushed" true && eq "$(remote_sha)" "$first" && eq "$(tracking_sha)" "$first"
}
A09() {
    fixture_new A09 && commit_work milestone-1 || return 1
    git push -q origin HEAD:"refs/heads/$B" || return 1
    local first; first=$(head_sha)
    commit_work milestone-2 && wrapper_simulation || return 1
    eq "$logRC" 128 && eq "$logOutput" '' && eq "$lsRC" 0 && eq "$newBranch" false && eq "$unpushed" false && eq "$(remote_sha)" "$first" && eq "$(tracking_sha)" absent && check test "$(head_sha)" != "$first"
}
A10() {
    fixture_new A10 && commit_work unpublished || return 1
    printf wrapper > wrapper.tmp
    local before allrefs; before=$(state); allrefs=$(refs)
    git ls-remote --exit-code origin "refs/heads/$B" >"$fixture/probe" 2>&1; eq "$?" 2 || return 1
    invoke start && verdict fresh && eq "$(state)" "$before" && eq "$(refs)" "$allrefs" && refspec_once
}
resume_fixture() {
    fixture_new "$1" && commit_work milestone-1 && invoke push && published || return 1
    first=$(head_sha)
    # Discard only this attempt's disposable clone, as a Cloud Run retry does.
    cd "$fixture"; rm -rf "$fixture/clone"
    clone_task "$fixture/retry" || return 1
    printf wrapper-bytes > wrapper.tmp
    invoke start && verdict "resumed $first" && eq "$(head_sha)" "$first" && eq "$(cat work)" milestone-1 && eq "$(cat sprint.json)" '{"passes":"milestone-1"}' && eq "$(cat wrapper.tmp)" wrapper-bytes
}
A11() { resume_fixture A11; }
A12() {
    resume_fixture A12 && commit_work milestone-2 && invoke push && verdict pushed && published && check git merge-base --is-ancestor "$first" HEAD
}
A13() {
    fixture_new A13 && commit_work refused || return 1
    mkdir -p "$remote/hooks"
    printf '#!/bin/bash\nexit 1\n' > "$remote/hooks/pre-receive"; chmod +x "$remote/hooks/pre-receive"
    local before allrefs; before=$(state); allrefs=$(refs)
    invoke push && verdict failed:push && eq "$(refs)" "$allrefs" && eq "$(state)" "$before"
}
A14() {
    fixture_new A14 || return 1
    local args before cfg allrefs; before=$(state); cfg=$(config); allrefs=$(refs)
    for args in missing unknown extra; do
        case "$args" in
            missing) out=$(/bin/bash "$HELPER" 2>&1); rc=$? ;;
            unknown) out=$(/bin/bash "$HELPER" nonsense 2>&1); rc=$? ;;
            extra) out=$(/bin/bash "$HELPER" push extra 2>&1); rc=$? ;;
        esac
        eq "$rc" 2 && contains "$out" usage && eq "$(state)" "$before" && eq "$(config)" "$cfg" && eq "$(refs)" "$allrefs" || return 1
    done
}
A15() {
    fixture_new A15 && commit_work milestone-1 && invoke push || return 1
    local original other before allrefs; original=$PWD
    clone_task "$fixture/other" || return 1
    AILANG_TASK_ID=task-x /bin/bash "$HELPER" start >"$fixture/other-start" || return 1
    commit_work competing && git push -q origin HEAD:"refs/heads/$B" || return 1
    other=$(head_sha); cd "$original" || return 1
    commit_work diverged || return 1
    before=$(state); allrefs=$(refs)
    invoke push && verdict failed:push && eq "$(remote_sha)" "$other" && eq "$(refs)" "$allrefs" && eq "$(state)" "$before"
}
A16() {
    fixture_new A16 && commit_work milestone-1 && invoke push || return 1
    clone_task "$fixture/retry" && shim || return 1
    local before allrefs; before=$(state); allrefs=$(refs)
    with_shim fetch start && verdict failed:fetch && contains "$(cat "$SHIM_LOG")" 'fetch origin' && eq "$(state)" "$before" && eq "$(refs)" "$allrefs"
}
A17() {
    fixture_new A17 && commit_work milestone-1 && invoke push || return 1
    clone_task "$fixture/retry" || return 1
    printf dirty-overlap > work; printf wrapper-bytes > wrapper.tmp
    local before allrefs; before=$(state); allrefs=$(refs)
    invoke start && verdict failed:reset && eq "$(refs)" "$allrefs" && eq "$(state)" "$before" && eq "$(cat work)" dirty-overlap && eq "$(cat wrapper.tmp)" wrapper-bytes
}
A18() {
    fixture_new A18 && commit_work local && shim || return 1
    local before allrefs; before=$(state); allrefs=$(refs)
    with_shim probe start && verdict failed:probe && no_mutating_network && eq "$(refs)" "$allrefs" && eq "$(state)" "$before"
}
A19() {
    local kind
    for kind in verify missing; do
        fixture_new "A19-$kind" && commit_work milestone && shim || return 1
        with_shim "$kind" push && verdict failed:verify && contains "$out" 'observed=' && published || return 1
        ! contains "$out" 'cloud-branch-sync: pushed' >/dev/null 2>&1 || return 1
        contains "$(cat "$SHIM_LOG")" "ls-remote --exit-code origin refs/heads/$B" || return 1
    done
}
A20() {
    fixture_new A20 && shim || return 1
    local began elapsed pid deadline watchdog helper_pid
    began=$(date +%s)
    # Independent outer bound kills even a mutant which removed the helper deadline.
    PATH="$fixture/bin:$PATH" SHIM_KIND=stall CLOUD_BRANCH_SYNC_TIMEOUT_S=1 AILANG_TASK_ID=task-x \
        /bin/bash "$HELPER" start >"$fixture/stall.out" 2>"$fixture/stall.err" &
    helper_pid=$!; deadline=$((began + 6)); watchdog=false
    while kill -0 "$helper_pid" 2>/dev/null; do
        if [ "$(date +%s)" -ge "$deadline" ]; then watchdog=true; kill -9 "$helper_pid" 2>/dev/null; break; fi
        sleep 0.1
    done
    wait "$helper_pid" 2>/dev/null; rc=$?
    elapsed=$(( $(date +%s) - began )); out=$(cat "$fixture/stall.out")
    if [ -f "$SHIM_PID" ]; then pid=$(cat "$SHIM_PID"); else echo 'shim did not run' >&2; return 1; fi
    if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null
        echo "stalled Git survived helper (pid=$pid)" >&2; return 1
    fi
    eq "$watchdog" false && eq "$rc" 0 && check test "$elapsed" -le 4 && verdict failed:probe && contains "$out" 'rc=124' && contains "$out" "branch=$B" && contains "$out" "sha=$(head_sha)" && contains "$(cat "$SHIM_ENV")" 'prompt=0 askpass=/usr/bin/false' && contains "$(cat "$SHIM_ENV")" BatchMode=yes
}
A21() {
    local command
    for command in start push; do
        fixture_new "A21-$command" && commit_work local && shim || return 1
        local before allrefs cfg; before=$(state); allrefs=$(refs); cfg=$(config)
        with_shim config "$command" && verdict failed:config && no_mutating_network && eq "$(refs)" "$allrefs" && eq "$(state)" "$before" && eq "$(config)" "$cfg" || return 1
        ! grep -E '^(ls-remote|fetch|push|reset)' "$SHIM_LOG" || return 1
    done
}
A22() {
    fixture_new A22 && commit_work milestone && shim || return 1
    local i
    for i in 1 2; do
        with_shim normal push && verdict "pushed $(head_sha)" && published && refspec_once || return 1
    done
    eq "$(grep -c '^ls-remote --exit-code' "$SHIM_LOG")" 2
}
run_arm() {
    if ( "$1" ); then echo "$1 PASS"; PASS=$((PASS+1))
    else echo "$1 FAIL"; FAIL=$((FAIL+1)); fi
}
# Mutation runner is below the arms so normal runs have no mutation dependencies.
if [ "$mode" = --mutations ]; then
    # Literal line replacement avoids sed portability/escaping differences. Assert
    # that each mutation really changed its disposable helper before executing it.
    mutate_line() {
        OLD_LINE="$2" NEW_LINE="$3" awk '
            $0 == ENVIRON["OLD_LINE"] {print ENVIRON["NEW_LINE"]; next}
            {print}
        ' "$HELPER" > "$1"
        ! cmp -s "$HELPER" "$1"
    }
    killed=0; survived=0
    for arm in A01 A02 A03 A04 A05 A06 A07 A08 A09 A10 A11 A12 A13 A14 A15 A16 A17 A18 A19 A20 A21 A22; do
        copy="$root/mutant-$arm.sh"
        case "$arm" in
            A01|A02|A03) old='if [ "$B" != "coordinator/$AILANG_TASK_ID" ]; then'; new='if false; then'; name=remove-exact-branch-gate ;;
            A04) old='if [ -z "${AILANG_TASK_ID:-}" ]; then'
                new='AILANG_TASK_ID=${AILANG_TASK_ID:-${B#coordinator/}}
if [ -z "${AILANG_TASK_ID:-}" ]; then'; name=infer-task-from-branch ;;
            A05) old="[ -n \"\$B\" ] || B='<detached>'"
                new='[ -n "$B" ] || B="coordinator/${AILANG_TASK_ID:-}"'; name=infer-detached-destination ;;
            A06) old="if ! printf '%s\\n' \"\$fetch_specs\" | grep -Fxq \"\$refspec\"; then"; new='if true; then'; name=duplicate-refspec ;;
            A07|A12) old='bounded_git push origin "HEAD:refs/heads/$B"; rc=$?'; new='bounded_git push origin "HEAD:refs/heads/${B}-wrong"; rc=$?'; name=wrong-push-destination ;;
            A08|A09) old="if ! printf '%s\\n' \"\$fetch_specs\" | grep -Fxq \"\$refspec\"; then"; new='if false; then'; name=omit-tracking-refspec ;;
            A10) old='        verdict fresh; exit 0'; new='        verdict "failed:absent"; exit 0'; name=absent-is-failure ;;
            A11) old='    git reset --keep "origin/$B"; rc=$?'; new='    git reset --keep origin/dev; rc=$?'; name=reset-to-base ;;
            A13) old='    verdict "failed:push rc=$rc"; exit 0'; new='    verdict "pushed $SHA"; exit 0'; name=refused-push-is-success ;;
            A14) old='    exit 2'; new='    exit 0'; name=usage-returns-zero ;;
            A15) old='bounded_git push origin "HEAD:refs/heads/$B"; rc=$?'; new='bounded_git push --force origin "HEAD:refs/heads/$B"; rc=$?'; name=force-divergent-push ;;
            A16) old='        verdict "failed:fetch rc=$rc"; exit 0'; new='        verdict "resumed $SHA"; exit 0'; name=ignore-fetch-refusal ;;
            A17) old='    git reset --keep "origin/$B"; rc=$?'; new='    git reset --hard "origin/$B"; rc=$?'; name=destructive-reset ;;
            A18) old='        verdict "failed:probe rc=$rc"; exit 0'; new='        verdict fresh; exit 0'; name=probe-error-is-fresh ;;
            A19) old='if [ "$rc" -ne 0 ] || [ "$observed" != "$SHA" ]; then'; new='if false; then'; name=trust-push-without-verification ;;
            A20) old='        if [ "$(date +%s)" -ge "$deadline" ]; then'; new='        if false; then'; name=remove-network-deadline ;;
            A21) old="        verdict 'failed:config-add'; exit 0"; new='        : # ignore failed config'; name=ignore-config-refusal ;;
            A22) old='bounded_git push origin "HEAD:refs/heads/$B"; rc=$?'
                new='existing=$(bounded_git ls-remote --exit-code origin "refs/heads/$B")
if [ $? -eq 0 ]; then verdict "failed:up-to-date"; exit 0; fi
bounded_git push origin "HEAD:refs/heads/$B"; rc=$?'; name=up-to-date-is-failure ;;
        esac
        if ! mutate_line "$copy" "$old" "$new"; then echo "MUTATION $arm ERROR unchanged"; exit 1; fi
        /bin/bash -n "$copy" || exit 1
        CLOUD_BRANCH_SYNC_HELPER="$copy" CLOUD_BRANCH_SYNC_ARM="$arm" \
            /bin/bash "$TEST_FILE" > "$root/mutation-$arm.log" 2>&1
        result=$?
        if [ "$result" -ne 0 ]; then
            echo "MUTATION $arm $name KILLED rc=$result"; killed=$((killed+1))
        else
            echo "MUTATION $arm $name SURVIVED rc=$result"; survived=$((survived+1))
        fi
        cat "$root/mutation-$arm.log"
    done
    # Strong adversary: print the exact skipped verdict while secretly publishing
    # the live branch. A01 must kill this by observed remote refs, not wording.
    copy="$root/mutant-A01-lying.sh"
    old="    verdict 'skipped:branch-mismatch'; exit 0"
    new=$(cat <<'LYING'
    if [ "$command" = push ]; then git push origin "HEAD:refs/heads/$B"; fi
    verdict 'skipped:branch-mismatch'; exit 0
LYING
)
    mutate_line "$copy" "$old" "$new" || exit 1
    CLOUD_BRANCH_SYNC_HELPER="$copy" CLOUD_BRANCH_SYNC_ARM=A01 \
        /bin/bash "$TEST_FILE" > "$root/mutation-A01-lying.log" 2>&1
    lying_rc=$?
    if [ "$lying_rc" -ne 0 ]; then echo "MUTATION A01 skipped-while-pushing KILLED rc=$lying_rc"
    else echo "MUTATION A01 skipped-while-pushing SURVIVED rc=$lying_rc"; fi
    cat "$root/mutation-A01-lying.log"
    # A09 deliberately never calls the helper: a helper-only mutant MUST survive
    # this negative control. Independently mutate a disposable simulator copy to
    # show that A09 rejects a corrected/always-true interpretation of Step 5b.
    sim="$root/simulator-mutant.sh"
    sed 's/^unpushed=false$/unpushed=true/' "$TEST_FILE" > "$sim"
    CLOUD_BRANCH_SYNC_HELPER="$HELPER" CLOUD_BRANCH_SYNC_ARM=A09 \
        /bin/bash "$sim" > "$root/mutation-A09-simulator.log" 2>&1
    sim_rc=$?
    if [ "$sim_rc" -ne 0 ]; then echo "MUTATION A09 simulator-always-true KILLED rc=$sim_rc"
    else echo "MUTATION A09 simulator-always-true SURVIVED rc=$sim_rc"; fi
    cat "$root/mutation-A09-simulator.log"
    echo "Helper mutations: $killed KILLED, $survived SURVIVED; simulator rc=$sim_rc"
    [ "$killed" -eq 21 ] && [ "$survived" -eq 1 ] && [ "$sim_rc" -ne 0 ] && [ "$lying_rc" -ne 0 ]
    exit $?
fi
for arm in A01 A02 A03 A04 A05 A06 A07 A08 A09 A10 A11 A12 A13 A14 A15 A16 A17 A18 A19 A20 A21 A22; do
    if [ "$selected" = all ] || [ "$selected" = "$arm" ]; then run_arm "$arm"; fi
done
echo "Arms: $PASS PASS, $FAIL FAIL"
[ "$PASS" -gt 0 ] && [ "$FAIL" -eq 0 ]
