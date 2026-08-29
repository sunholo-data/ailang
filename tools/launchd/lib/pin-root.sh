#!/usr/bin/env bash
# pin-root.sh — make a launchd driver run COMMITTED code, and make failing to do so LOUD.
#
# THE CLASS THIS CLOSES (#558; measured twice — 2026-08-03 and 2026-08-12).
# launchd invokes each driver by absolute path into a shared, mutable clone that nothing
# keeps current, and everything the fire reads hangs off that one root:
#
#     REPO="${MISSION_WORKDIR:-$(cd "$(dirname "$0")/../.." && pwd)}"; cd "$REPO"
#
#   * the DRIVER itself         -> #556 retired qwen3.5 and the nightly ran it 24/24 two days
#                                  later; 564cc4640's lane fix was inert on V1 while both
#                                  sibling missions had it
#   * the SKILL (.claude/skills) -> stale skill, iteration 128
#   * the CHARTER (design_docs)  -> stale charter, iteration 129
#
# Three symptoms, one root. Patching any single artefact fixes one third of a bug, so this
# pins the ROOT: re-exec the driver out of a worktree pinned to a committed ref, which moves
# the script, the skill and the charter together in one move.
#
# WHY THE FAILURE PATH IS LOUD RATHER THAN SILENT. If the fetch fails and we quietly carry on
# from the working tree, we have rebuilt the exact defect 564cc4640 removed one layer up — a
# fallback whose only witness is a log nobody reads (Critical Principle 2). So a failed pin is
# NOT fatal (making network availability a hard dependency of every fire trades a rare silent
# staleness for a common loud outage) but it MUST be reported on the human channel by the
# caller. This file sets PIN_STATUS=STALE and returns 1; emitting is the caller's job, because
# only the caller knows its own early-exit points — a fire that never runs must never post.
#
# CONTRACT
#   source, never execute.  $0 stays the CALLER's path, which is what makes $0-relative
#   resolution below refer to the driver rather than to this file.
#
#   in    $0                          the driver's path (implicit)
#         AILANG_DRIVER_REF           ref to pin to           (default origin/dev)
#         AILANG_DRIVER_PIN=0         opt out entirely        (default on)
#         AILANG_DRIVER_FETCH_TIMEOUT bounded fetch, seconds  (default 120)
#         AILANG_DRIVER_PIN_DIR       override worktree path
#         MISSION_NAME                namespaces the worktree so missions never collide
#   out   PIN_STATUS  pinned | disabled | STALE
#         PIN_NOTE    one-line human summary, safe to log or post
#         PIN_DRIFT   how many commits the source clone is behind the ref ("?" if unknown)
#   exit  re-execs on success and NEVER RETURNS; returns 0 (already pinned / opted out) or
#         1 (STALE — caller continues on the working tree, loudly)
#
# CHICKEN AND EGG, stated rather than hidden: this block lives in the driver it protects, so it
# does nothing until the shared clone receives it once. That first reconcile is human (Principle
# 0 forbids unattended branch ops on a shared dirty tree). Afterwards it is self-maintaining.
#
# Portable to macOS bash 3.2.57 — no associative arrays, no ${v,,}, no GNU timeout.

PIN_STATUS="unknown"
PIN_NOTE=""
PIN_DRIFT="?"

# _pin_bounded SECONDS CMD... — hard wall-clock cap; rc = CMD's rc, or 124 on expiry.
# Deliberately duplicates mission-control.sh's _mc_bounded rather than depending on it: this
# file has to be sourceable by drivers that define no such helper, and the rig has no GNU
# timeout (mission-control.sh:37). An unbounded fetch here would hang the fire before the
# driver's own watchdog exists — the failure mode _mc_bounded was written for.
_pin_bounded() {
  local secs="$1"; shift
  local out_f rc deadline pid
  out_f=$(mktemp -t pin_bounded) || { PIN_BOUNDED_OUT=""; return 125; }
  ( exec "$@" ) >"$out_f" 2>&1 &
  pid=$!
  deadline=$(( $(date +%s) + secs ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
      kill "$pid" 2>/dev/null; sleep 1; kill -9 "$pid" 2>/dev/null
      PIN_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
      return 124
    fi
    sleep 1
  done
  wait "$pid"; rc=$?
  PIN_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
  return "$rc"
}

_pin_stale() {  # $1 = reason -> STALE, caller reports and continues on the working tree
  PIN_STATUS="STALE"
  PIN_NOTE="$1"
  return 1
}

# pin_root_to_committed_ref "$@" — pass the driver's own args so the re-exec forwards them.
pin_root_to_committed_ref() {
  if [ -n "${AILANG_DRIVER_PINNED:-}" ]; then
    # we ARE the re-exec'd copy; the values below crossed the exec in the environment
    PIN_STATUS="pinned"
    PIN_DRIFT="${AILANG_DRIVER_DRIFT:-?}"
    PIN_NOTE="running committed ${AILANG_DRIVER_REF:-origin/dev} @ ${AILANG_DRIVER_PINNED} from ${MISSION_WORKDIR:-?} (source clone ${AILANG_DRIVER_SRC:-?} was ${PIN_DRIFT} behind)"
    return 0
  fi

  if [ "${AILANG_DRIVER_PIN:-1}" = "0" ]; then
    PIN_STATUS="disabled"
    PIN_NOTE="pin disabled via AILANG_DRIVER_PIN=0 — running the working tree as-is"
    return 0
  fi

  local ref src script wt target short drift fetch_s rc onboarding_key_count projects_len refreshed_helper
  ref="${AILANG_DRIVER_REF:-origin/dev}"
  fetch_s="${AILANG_DRIVER_FETCH_TIMEOUT:-120}"
  script=$(basename "$0")

  src=$(cd "$(dirname "$0")/../.." 2>/dev/null && pwd)
  if [ -z "$src" ]; then
    _pin_stale "cannot resolve the source clone from \$0=$0"; return 1
  fi
  if ! git -C "$src" rev-parse --git-dir >/dev/null 2>&1; then
    _pin_stale "source clone $src is not a git repository"; return 1
  fi

  _pin_bounded "$fetch_s" git -C "$src" fetch --quiet origin; rc=$?
  if [ "$rc" -eq 124 ]; then
    _pin_stale "git fetch origin exceeded ${fetch_s}s in $src"; return 1
  elif [ "$rc" -ne 0 ]; then
    _pin_stale "git fetch origin failed (rc=$rc) in $src: $(printf '%s' "${PIN_BOUNDED_OUT:-}" | tail -c 200 | tr '\n' ' ')"; return 1
  fi

  target=$(git -C "$src" rev-parse "$ref" 2>/dev/null)
  if [ -z "$target" ]; then
    _pin_stale "cannot resolve $ref in $src"; return 1
  fi

  # BOOTSTRAP THE GATE FROM THE REF BEFORE THE GATE DECIDES. This helper is initially sourced
  # from the mutable source clone, so without this hop an old onboarding predicate can refuse
  # the very pin that would replace it. Fetching above is not enough: shell functions already
  # loaded from the stale tree do not change when origin/dev moves.
  #
  # The marker is the full target oid, not a boolean. A nested caller that deliberately changes
  # AILANG_DRIVER_REF must refresh again, while the ordinary re-entry through the same blob is
  # exactly one hop. The temp file is removed before sourcing returns control to this function;
  # bash has parsed the sourced definitions by then. Failure stays loud and fail-closed.
  if [ "${AILANG_DRIVER_PIN_GATE_REFRESHED:-}" != "$target" ]; then
    refreshed_helper=$(mktemp -t pin_root_ref) || {
      _pin_stale "cannot create a temporary file for $ref's pin gate"; return 1
    }
    if ! git -C "$src" show "$target:tools/launchd/lib/pin-root.sh" >"$refreshed_helper" 2>/dev/null; then
      rm -f "$refreshed_helper"
      _pin_stale "$ref has no tools/launchd/lib/pin-root.sh — cannot refresh the pin gate"; return 1
    fi
    if ! /bin/bash -n "$refreshed_helper" 2>/dev/null; then
      rm -f "$refreshed_helper"
      _pin_stale "$ref has an invalid tools/launchd/lib/pin-root.sh — refusing to run an unverified pin gate"; return 1
    fi
    AILANG_DRIVER_PIN_GATE_REFRESHED="$target"
    export AILANG_DRIVER_PIN_GATE_REFRESHED
    . "$refreshed_helper"
    rc=$?
    rm -f "$refreshed_helper"
    if [ "$rc" -ne 0 ]; then
      _pin_stale "could not load $ref's tools/launchd/lib/pin-root.sh (rc=$rc)"; return 1
    fi
    pin_root_to_committed_ref "$@"
    return $?
  fi

  short=$(git -C "$src" rev-parse --short "$target" 2>/dev/null)
  drift=$(git -C "$src" rev-list --count "HEAD..$ref" 2>/dev/null)
  [ -n "$drift" ] || drift="?"
  PIN_DRIFT="$drift"

  wt="${AILANG_DRIVER_PIN_DIR:-$HOME/.ailang-driver-pin/${MISSION_NAME:-$(basename "$script" .sh)}}"

  # Refresh (or create) the pin worktree. It is a throwaway checkout with no user work in it,
  # kept outside every dev clone, so --force can never overwrite in-progress changes.
  #
  # Existence is decided by ASKING THE WORKTREE, not by string-matching `worktree list`. git
  # records the resolved realpath, so on any tree reached through a symlink — /var -> /private/var
  # on macOS being the everyday case — the recorded path never equals the one we computed, the
  # match silently fails, and we take the `add` branch against a directory that already exists.
  # That fails, which means the pin would have worked exactly ONCE and reported STALE on every
  # fire afterwards: a self-disabling fix, loud but useless. Caught by test 7, not by review.
  git -C "$src" worktree prune >/dev/null 2>&1
  if [ -e "$wt/.git" ] && git -C "$wt" rev-parse --git-dir >/dev/null 2>&1; then
    if ! git -C "$wt" checkout --quiet --detach --force "$target" 2>/dev/null; then
      _pin_stale "pin worktree checkout $short failed at $wt"; return 1
    fi
  elif [ -d "$wt" ]; then
    # Occupied but not a usable worktree. Deliberately NOT rm -rf'd: AILANG_DRIVER_PIN_DIR is
    # caller-supplied, and silently deleting a caller-named directory is a worse failure than
    # declining to pin. A human clears it; until then this is loud on every fire.
    _pin_stale "$wt exists but is not a usable git worktree — refusing to pin; remove it to re-enable"; return 1
  else
    mkdir -p "$(dirname "$wt")"
    if ! git -C "$src" worktree add --quiet --detach --force "$wt" "$target" 2>/dev/null; then
      _pin_stale "pin worktree add $short failed at $wt"; return 1
    fi
  fi

  # Refuse to exec into a driver the ref does not have — a rename on dev would otherwise turn
  # a pin into a silent no-run. Control for the whole worktree step in one assertion.
  if [ ! -f "$wt/tools/launchd/$script" ]; then
    _pin_stale "$ref has no tools/launchd/$script — refusing to re-exec into a missing driver"; return 1
  fi

  # ONBOARDING GATE — a checkout Claude Code has never seen makes headless `claude -p` block on
  # a trust dialog it cannot display, so EVERY model probe hangs to its timeout and the driver
  # refuses with "NO usable model in prefs" — indistinguishable from a quota outage. That cost
  # the motoko mission its entire first unattended fire (charter V22, commit 76ee4056c).
  # Staleness is the strictly smaller harm than a dead loop, so an unusable target means refuse.
  #
  # THE PREDICATE IS THE SOURCE REPO, NOT THE WORKTREE PATH. The first cut checked the pin
  # worktree's own entry and was measured WRONG on 2026-08-12: `claude -p` runs fine from
  # ~/.ailang-driver-pin/v1 while ~/.claude.json has NO entry for it at all — a git worktree
  # inherits its source clone's trust, having no separate identity to onboard. Checking the
  # worktree would have refused a demonstrably working target on every fire, leaving the pin
  # permanently off. The two measured cases discriminate cleanly:
  #
  #   ailang-motoko          fresh CLONE, source entry ABSENT   -> probes HANG   (iteration 1 lost)
  #   ~/.ailang-driver-pin/v1  WORKTREE of an onboarded clone,
  #                            own entry ABSENT                 -> probes WORK   (measured)
  #
  # So: accept when EITHER path is onboarded. That refuses exactly the motoko shape (nothing
  # onboarded anywhere) and accepts the worktree-of-a-good-clone shape. `$wt` is still checked
  # first because AILANG_DRIVER_PIN_DIR can point somewhere that is NOT a worktree of $src.
  #
  # Claude Code has used both `hasCompletedProjectOnboarding` (legacy) and
  # `hasTrustDialogAccepted` (current) for this state, so either true value is sufficient.
  #
  # Undeterminable is treated as un-onboarded, deliberately: pinning blind risks ~12 min of hung
  # probes and a dead loop, refusing costs staleness that is already reported. jq lives at
  # /usr/bin/jq, inside launchd's default PATH, so its absence means something is genuinely wrong.
  _pin_onboarded() {  # $1 = path -> echoes true/false
    jq -r --arg p "$1" '((.projects[$p].hasCompletedProjectOnboarding == true) or (.projects[$p].hasTrustDialogAccepted == true))' "$HOME/.claude.json" 2>/dev/null
  }
  if [ -n "${AILANG_DRIVER_SKIP_ONBOARD_CHECK:-}" ]; then
    :
  elif ! command -v jq >/dev/null 2>&1; then
    _pin_stale "cannot verify Claude Code onboarding for $wt (jq not found) — refusing to pin into a possibly-unusable checkout"; return 1
  else
    # THREE distinct refusals, because a diagnosis that cannot tell them apart sends the next
    # reader down the wrong path — which is the exact defect this gate was fixed for.
    #   (a) the file cannot be read as `.projects`-shaped  -> instrument unreadable
    #   (b) `.projects` is non-empty and NO entry carries either key -> Claude Code schema drift
    #   (c) `.projects` is readable and this path is simply not onboarded -> the human fix
    # (a) and (b) are NOT the same event: (b) means the gate needs a new key, (a) means the file
    # is missing/malformed. Both stay fail-closed; only the sentence changes.
    projects_len=$(jq -r 'if (.projects|type) == "object" then (.projects|length) else "NOTOBJ" end' "$HOME/.claude.json" 2>/dev/null)
    case "${projects_len:-}" in
      ''|*[!0-9]*)
        _pin_stale "cannot read a .projects object from $HOME/.claude.json (missing file, invalid JSON, or unexpected shape) — refusing to pin into an unverifiable checkout"; return 1 ;;
    esac
    onboarding_key_count=$(jq '[.projects[]? | select(has("hasCompletedProjectOnboarding") or has("hasTrustDialogAccepted"))] | length' "$HOME/.claude.json" 2>/dev/null)
    case "${onboarding_key_count:-}" in
      ''|*[!0-9]*)
        _pin_stale "could not count onboarding keys in $HOME/.claude.json — refusing to pin into an unverifiable checkout"; return 1 ;;
    esac
    if [ "$projects_len" -gt 0 ] && [ "$onboarding_key_count" -eq 0 ]; then
      _pin_stale "neither hasCompletedProjectOnboarding nor hasTrustDialogAccepted exists in ANY of the $projects_len project entries of $HOME/.claude.json — Claude Code schema drift; the onboarding gate needs a new key, not human onboarding"; return 1
    elif [ "$(_pin_onboarded "$wt")" != "true" ] && [ "$(_pin_onboarded "$src")" != "true" ]; then
      _pin_stale "neither $wt nor its source clone $src is onboarded in Claude Code — every model probe would hang there (charter V22). Run once, interactively: cd $src && claude"; return 1
    fi
  fi

  # MISSION_WORKDIR moves too, and that is the point: mission-control.sh:40 reads it AHEAD of
  # $0-relative resolution, so pinning only the script would leave motoko and World (whose env
  # files pin MISSION_WORKDIR — mission-motoko.env:8, mission-world.env:5) running a pinned
  # driver against a stale charter and skill. That half-fix reports green, which is worse than
  # no fix. Sprint worktrees are unaffected: the skill creates them by absolute path
  # (mission-control SKILL.md:1662), not relative to cwd.
  AILANG_DRIVER_PINNED="$short"
  AILANG_DRIVER_SRC="$src"
  AILANG_DRIVER_DRIFT="$drift"
  AILANG_DRIVER_REF="$ref"
  MISSION_WORKDIR="$wt"
  export AILANG_DRIVER_PINNED AILANG_DRIVER_SRC AILANG_DRIVER_DRIFT AILANG_DRIVER_REF MISSION_WORKDIR

  exec /bin/bash "$wt/tools/launchd/$script" "$@"
}
