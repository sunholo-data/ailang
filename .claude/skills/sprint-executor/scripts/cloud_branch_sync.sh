#!/bin/bash
# Publish/resume only the current coordinator task branch. Bash 3.2; no tool installs.
set -u
B='<unknown>'; SHA='<unknown>'
verdict() {
    printf 'cloud-branch-sync: %s branch=%s sha=%s\n' "$1" "$B" "$SHA"
    return 0
}
if [ "$#" -ne 1 ]; then
    echo 'usage: cloud_branch_sync.sh start|push' >&2
    exit 2
fi
case "$1" in start|push) command=$1 ;; *) echo 'usage: cloud_branch_sync.sh start|push' >&2; exit 2 ;; esac
B=$(git branch --show-current 2>/dev/null) || B='<unknown>'
[ -n "$B" ] || B='<detached>'
SHA=$(git rev-parse --verify HEAD 2>/dev/null) || SHA='<unknown>'
if [ -z "${AILANG_TASK_ID:-}" ]; then
    verdict 'skipped:no-task-id'; exit 0
fi
if [ "$B" != "coordinator/$AILANG_TASK_ID" ]; then
    verdict 'skipped:branch-mismatch'; exit 0
fi

# Local config/ref/branch operations are naturally finite. Network calls have a
# 30s default deadline plus at most 1s TERM grace. A separate process group lets
# us terminate Git's transport/credential children as well as Git, then reap Git.
# CLOUD_BRANCH_SYNC_TIMEOUT_S is a positive integer (tests use 1s).
seconds=${CLOUD_BRANCH_SYNC_TIMEOUT_S:-30}
case "$seconds" in ''|*[!0-9]*|0) verdict 'failed:timeout-config'; exit 0 ;; esac
if [ "${#seconds}" -gt 4 ]; then verdict 'failed:timeout-config'; exit 0; fi
seconds=$((10#$seconds))
if [ "$seconds" -le 0 ]; then verdict 'failed:timeout-config'; exit 0; fi
bounded_git() {
    local pid deadline rc
    # Bash monitor mode gives this asynchronous job its own process group, on
    # macOS and Linux, without requiring setsid, timeout, or a process-tree tool.
    set -m
    GIT_TERMINAL_PROMPT=0 GCM_INTERACTIVE=Never GIT_ASKPASS=/usr/bin/false \
        SSH_ASKPASS=/usr/bin/false GIT_SSH_COMMAND='ssh -oBatchMode=yes -oConnectTimeout=10' \
        git "$@" &
    pid=$!
    deadline=$(( $(date +%s) + seconds ))
    while kill -0 "$pid" 2>/dev/null; do
        if [ "$(date +%s)" -ge "$deadline" ]; then
            kill -TERM -- "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null
            sleep 1
            kill -KILL -- "-$pid" 2>/dev/null || kill -KILL "$pid" 2>/dev/null
            wait "$pid" 2>/dev/null
            set +m
            return 124
        fi
        sleep 0.1
    done
    wait "$pid"; rc=$?
    set +m
    return "$rc"
}

# Install before either subcommand's network work. '+' updates a fetch tracking
# ref; it never appears in a push refspec and does not permit force-pushing.
refspec="+refs/heads/${B}:refs/remotes/origin/${B}"
fetch_specs=$(git config --local --get-all remote.origin.fetch); config_rc=$?
if [ "$config_rc" -ne 0 ] && [ "$config_rc" -ne 1 ]; then
    verdict "failed:config-read rc=$config_rc"; exit 0
fi
if ! printf '%s\n' "$fetch_specs" | grep -Fxq "$refspec"; then
    if ! git config --local --add remote.origin.fetch "$refspec"; then
        verdict 'failed:config-add'; exit 0
    fi
fi
if [ "$command" = start ]; then
    probe=$(bounded_git ls-remote --exit-code origin "refs/heads/$B"); rc=$?
    if [ "$rc" -eq 2 ]; then
        verdict fresh; exit 0
    fi
    if [ "$rc" -ne 0 ]; then
        verdict "failed:probe rc=$rc"; exit 0
    fi
    bounded_git fetch origin; rc=$?
    if [ "$rc" -ne 0 ]; then
        verdict "failed:fetch rc=$rc"; exit 0
    fi
    git reset --keep "origin/$B"; rc=$?
    if [ "$rc" -ne 0 ]; then
        verdict "failed:reset-keep rc=$rc"; exit 0
    fi
    SHA=$(git rev-parse --verify HEAD 2>/dev/null) || SHA='<unknown>'
    verdict "resumed $SHA"; exit 0
fi
bounded_git push origin "HEAD:refs/heads/$B"; rc=$?
if [ "$rc" -ne 0 ]; then
    verdict "failed:push rc=$rc"; exit 0
fi
probe=$(bounded_git ls-remote --exit-code origin "refs/heads/$B"); rc=$?
observed=$(printf '%s\n' "$probe" | awk -v ref="refs/heads/$B" '$2 == ref {print $1}')
[ -n "$observed" ] || observed='<absent>'
if [ "$rc" -ne 0 ] || [ "$observed" != "$SHA" ]; then
    verdict "failed:verify rc=$rc observed=$observed"; exit 0
fi
tracking=$(git rev-parse --verify "refs/remotes/origin/$B" 2>/dev/null) || tracking='<absent>'
if [ "$tracking" != "$SHA" ]; then
    verdict "failed:verify-tracking observed=$observed tracking=$tracking"; exit 0
fi
verdict "pushed $SHA"
exit 0
