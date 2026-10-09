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
#   CI_GATE=none <detail>     GitHub answered and there is no PR for the
#                             branch, or a PR older than EVAL_CI_NOCHECKS_MIN
#                             (default 15) has no checks: the caller runs the
#                             gates locally instead
#   CI_GATE=error <detail>    the PR or its checks could not be read. Never
#                             reported as "none": an auth failure that read as
#                             "no PR" sent the evaluator to local gates
#                             silently (cloud smoke test, 2026-10-09)
#
# Source: `gh` when it is authenticated; otherwise GitHub's REST API with curl
# and jq (unauthenticated works for a public repo, GH_TOKEN/GITHUB_TOKEN is used
# when set). The cloud executor withholds GITHUB_TOKEN from the agent, so the
# evaluator normally takes the REST path there.
#
# Env (tests set these): EVAL_GH, EVAL_CURL, EVAL_CI_REPO (owner/name; default
# from the origin remote), EVAL_CI_WAIT_MIN, EVAL_CI_POLL_SEC (default 30),
# EVAL_CI_NOCHECKS_MIN.
# bash 3.2-safe: the rig's /bin/bash is 3.2.57.
set -u

BRANCH="${1:-}"
GH="${EVAL_GH:-gh}"
CURL="${EVAL_CURL:-curl}"
API="https://api.github.com"
WAIT_MIN="${EVAL_CI_WAIT_MIN:-4}"
POLL_SEC="${EVAL_CI_POLL_SEC:-30}"
NOCHECKS_MIN="${EVAL_CI_NOCHECKS_MIN:-15}"

if [ -z "$BRANCH" ]; then
    echo "CI_GATE=none no branch given"
    exit 0
fi
USE_GH=false
if command -v "$GH" >/dev/null 2>&1 && "$GH" auth status >/dev/null 2>&1; then
    USE_GH=true
else
    if ! command -v "$CURL" >/dev/null 2>&1 || ! command -v jq >/dev/null 2>&1; then
        echo "CI_GATE=error gh is not authenticated and curl/jq are not available"
        exit 0
    fi
    REPO="${EVAL_CI_REPO:-$(git config --get remote.origin.url 2>/dev/null | sed -E 's#^.*github\.com[:/]##; s#\.git$##')}"
    if [ -z "$REPO" ]; then
        echo "CI_GATE=error cannot tell which GitHub repo this is (set EVAL_CI_REPO=owner/name)"
        exit 0
    fi
    TOKEN="${GH_TOKEN:-${GITHUB_TOKEN:-}}"
fi

# api PATH: GET from the REST API; exits non-zero on any HTTP or transport error.
api() {
    if [ -n "${TOKEN:-}" ]; then
        "$CURL" -fsS -m 30 -H "Authorization: Bearer $TOKEN" -H "Accept: application/vnd.github+json" "$API/$1"
    else
        "$CURL" -fsS -m 30 -H "Accept: application/vnd.github+json" "$API/$1"
    fi
}

# "<number> <age in seconds> <head sha>"; jq does the date arithmetic, because
# `date -d` (GNU) and `date -j -f` (BSD) do not agree.
if [ "$USE_GH" = true ]; then
    if ! PR_INFO=$("$GH" pr list --head "$BRANCH" --state all --json number,createdAt,headRefOid \
        --jq '.[0] | select(. != null) | "\(.number) \((now - (.createdAt | fromdate)) | floor) \(.headRefOid)"' 2>&1); then
        echo "CI_GATE=error gh pr list failed: $(printf '%s' "$PR_INFO" | head -1)"
        exit 0
    fi
else
    if ! PR_JSON=$(api "repos/$REPO/pulls?head=${REPO%%/*}:$BRANCH&state=all&per_page=1" 2>&1); then
        echo "CI_GATE=error GitHub API pulls lookup failed: $(printf '%s' "$PR_JSON" | head -1)"
        exit 0
    fi
    PR_INFO=$(printf '%s' "$PR_JSON" | jq -r '.[0] | select(. != null) | "\(.number) \((now - (.created_at | fromdate)) | floor) \(.head.sha)"')
fi
read -r PR PR_AGE PR_SHA <<EOF_PR
$PR_INFO
EOF_PR
case "${PR_AGE:-}" in ''|*[!0-9]*) PR_AGE=0 ;; esac
if [ -z "${PR:-}" ]; then
    echo "CI_GATE=none no pull request for $BRANCH"
    exit 0
fi

# check_lines: one "<bucket> <name>" line per check, buckets as gh names them
# (pass, fail, pending, skipping, cancel). Exits non-zero when unreadable.
check_lines() {
    if [ "$USE_GH" = true ]; then
        local out
        # gh exits 8 while checks are pending; only an empty answer with a
        # non-zero status is a failure to read.
        out=$("$GH" pr checks "$PR" --json bucket,name --jq '.[] | "\(.bucket) \(.name)"' 2>&1)
        local rc=$?
        if [ "$rc" -ne 0 ] && [ "$rc" -ne 8 ]; then
            printf '%s\n' "$out" >&2
            return 1
        fi
        printf '%s\n' "$out"
        return 0
    fi
    local runs statuses
    runs=$(api "repos/$REPO/commits/$PR_SHA/check-runs?per_page=100") || return 1
    statuses=$(api "repos/$REPO/commits/$PR_SHA/status") || return 1
    printf '%s' "$runs" | jq -r '.check_runs[] |
        (if .status != "completed" then "pending"
         elif .conclusion == "success" or .conclusion == "neutral" then "pass"
         elif .conclusion == "skipped" then "skipping"
         elif .conclusion == "cancelled" then "cancel"
         else "fail" end) + " " + .name'
    printf '%s' "$statuses" | jq -r '.statuses[] |
        (if .state == "pending" then "pending" elif .state == "success" then "pass" else "fail" end) + " " + .context'
}

start=$(date +%s)
deadline=$((start + WAIT_MIN * 60))

while :; do
    if ! checks=$(check_lines 2>/dev/null); then
        echo "CI_GATE=error could not read the checks of PR #$PR"
        exit 0
    fi
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
