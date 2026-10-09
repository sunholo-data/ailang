#!/bin/bash
# ci_gate.sh BRANCH — read the CI verdict of BRANCH's pull request instead of
# re-running the suite.
#
# Why: the cloud evaluator ran `make test` itself. It is silent for longer than
# the executor's 5-minute idle timeout, so every cloud evaluation on 2026-09-29,
# 10-01 and 10-02 was killed mid-run ("pi idle for 5m0s mid-generation") and
# none returned a verdict. CI has already run the full suite, lint and
# verify-examples on the PR's head, so re-running them proves nothing new.
#
# One call waits at most EVAL_CI_WAIT_MIN (default 4) minutes, then reports
# pending: CI takes 20-25 minutes, and no single tool call may block that long
# (Claude Code's Bash tool stops a command at 10 minutes; pi kills an agent with
# no output for 5). The evaluator calls this again until it reports pass or
# fail; each return is a tool result, which is also what keeps pi's idle timer
# from firing. It prints a status line per poll, then exactly one result line:
#   CI_GATE=pass <detail>     every check finished green
#   CI_GATE=fail <detail>     a check failed or was cancelled, or every check
#                             was skipped (a PR whose checks all skip usually
#                             conflicts with its base)
#   CI_GATE=pending <detail>  checks still running: call again
#   CI_GATE=none <detail>     no PR, no gh, or a PR older than
#                             EVAL_CI_NOCHECKS_MIN (default 15) with no checks:
#                             the caller runs the gates locally instead
#
# Env (tests set these): EVAL_GH (gh binary), EVAL_CI_WAIT_MIN,
# EVAL_CI_POLL_SEC (default 30), EVAL_CI_NOCHECKS_MIN.
# bash 3.2-safe: the rig's /bin/bash is 3.2.57.
set -u

BRANCH="${1:-}"
GH="${EVAL_GH:-gh}"
WAIT_MIN="${EVAL_CI_WAIT_MIN:-4}"
POLL_SEC="${EVAL_CI_POLL_SEC:-30}"
NOCHECKS_MIN="${EVAL_CI_NOCHECKS_MIN:-15}"

if [ -z "$BRANCH" ]; then
    echo "CI_GATE=none no branch given"
    exit 0
fi
if ! command -v "$GH" >/dev/null 2>&1; then
    echo "CI_GATE=none gh is not available"
    exit 0
fi

# "<number> <age in seconds>": gh's jq does the date arithmetic, because
# `date -d` (GNU) and `date -j -f` (BSD) do not agree.
PR_INFO=$("$GH" pr list --head "$BRANCH" --state all --json number,createdAt \
    --jq '.[0] | select(. != null) | "\(.number) \((now - (.createdAt | fromdate)) | floor)"' 2>/dev/null)
PR="${PR_INFO%% *}"
PR_AGE="${PR_INFO##* }"
case "$PR_AGE" in ''|*[!0-9]*) PR_AGE=0 ;; esac
if [ -z "$PR" ]; then
    echo "CI_GATE=none no pull request for $BRANCH"
    exit 0
fi

start=$(date +%s)
deadline=$((start + WAIT_MIN * 60))

while :; do
    checks=$("$GH" pr checks "$PR" --json bucket,name --jq '.[] | "\(.bucket) \(.name)"' 2>/dev/null)
    total=$(printf '%s\n' "$checks" | grep -c .)
    pending=$(printf '%s\n' "$checks" | grep -c '^pending ')
    skipping=$(printf '%s\n' "$checks" | grep -c '^skipping ')
    failed=$(printf '%s\n' "$checks" | grep -E '^(fail|cancel) ' | cut -d' ' -f2- | tr '\n' ';')
    echo "ci-gate: PR #$PR: $total checks, $pending pending ($(date -u +%H:%M:%SZ))"

    now=$(date +%s)
    if [ "$total" -eq 0 ]; then
        if [ $((PR_AGE + now - start)) -ge $((NOCHECKS_MIN * 60)) ]; then
            echo "CI_GATE=none PR #$PR has reported no CI checks ${NOCHECKS_MIN}m after it opened"
            exit 0
        fi
    elif [ -n "$failed" ]; then
        echo "CI_GATE=fail PR #$PR: failed checks: ${failed%;}"
        exit 0
    elif [ "$pending" -eq 0 ]; then
        if [ "$skipping" -eq "$total" ]; then
            echo "CI_GATE=fail PR #$PR: every check was skipped (the PR likely conflicts with its base)"
            exit 0
        fi
        echo "CI_GATE=pass PR #$PR: $((total - skipping)) checks green"
        exit 0
    fi

    if [ "$now" -ge "$deadline" ]; then
        echo "CI_GATE=pending PR #$PR: $pending of $total checks still running; call again"
        exit 0
    fi
    sleep "$POLL_SEC"
done
