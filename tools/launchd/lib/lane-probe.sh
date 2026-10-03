# shellcheck shell=bash
# lane-probe.sh — the model probes and the ration gate, sourced by the mission driver AND by
# mission-lane-check.sh (m-mission-light-profile Phase 1). Moved verbatim out of
# mission-control.sh on 2026-10-02: the probes were top-level functions in a script that runs on
# load, so nothing could reuse them without running the driver — and a second copy is how a
# readiness check drifts from what a fire actually runs. ONE copy, here.
#
# Callers provide: log(), and the env the ration gate reads (MISSION_* routing vars).

QUOTA_SIG="usage limit|rate.?limit|quota|exceeded|too many requests|weekly limit"
PROBE_TIMEOUT="${MISSION_PROBE_TIMEOUT:-120}"   # per-probe wall-clock cap, seconds

# _mc_bounded SECONDS CMD... — run CMD with a hard wall-clock cap.
# rc = CMD's rc, or 124 on expiry (mirrors GNU `timeout`, which this rig does not have).
# Combined stdout+stderr lands in $MC_BOUNDED_OUT.
#
# Why (2026-07-27): a model probe is a network call to a third party and CAN hang. Observed that
# day: `codex exec --model <unknown-model>` ran past 180s with no output. Both probes below used
# to be unbounded command substitutions, so one hung probe would burn the whole 6h fire before the
# driver's HARD_TIMEOUT reclaimed it — the exact failure class as mission-control Standing rule 6
# ("every wait is bounded"), which the loop enforces on itself but the driver did not.
_mc_bounded() {
  local secs="$1"; shift
  local out_f rc deadline pid
  out_f=$(mktemp -t mc_bounded) || { MC_BOUNDED_OUT=""; return 125; }
  ( exec "$@" ) >"$out_f" 2>&1 &
  pid=$!
  deadline=$(( $(date +%s) + secs ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
      kill "$pid" 2>/dev/null; sleep 2; kill -9 "$pid" 2>/dev/null
      MC_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
      return 124
    fi
    sleep 2
  done
  wait "$pid"; rc=$?
  MC_BOUNDED_OUT="$(cat "$out_f" 2>/dev/null)"; rm -f "$out_f"
  return "$rc"
}

# _mc_probe MODEL → 0 usable | 1 quota-limited | 2 unusable (auth/transient×2/timeout×2)
_mc_probe() {
  local m="$1" out rc
  # Ration gate, symmetric with _mc_probe_codex and _mc_probe_pi. Without it the
  # Anthropic ration was HALF-WALKED: _mc_set_controller consults the ration for the
  # controller (so an over-ration controller skips to a cheaper rung), but the ROLE
  # pre-flight calls this function directly, so the designer and evaluator kept probing
  # and spending Anthropic while the controller was yielding for exactly that reason.
  # Declared is not walked — the same shape as the role fallback chains that existed for
  # weeks with nothing reading them.
  if _mc_is_over_ration "$m"; then
    # AN UNREADABLE QUOTA IS NOT AN EXHAUSTED ONE — probe instead of refusing.
    #
    # Reading Anthropic subscription usage depends on a credential this fleet does
    # not control: it lives in a keychain item whose ACL names Claude Code, its
    # access token expires every ~8h and is refreshed IN MEMORY without write-back,
    # and a `claude setup-token` token authenticates but is FORBIDDEN from the usage
    # endpoint (HTTP 403, measured 2026-09-22). There is no configuration that makes
    # that read reliable, so a gate that REQUIRES it will keep failing.
    #
    # The probe on the very next line is the backstop and always was. On a
    # subscription it costs nothing metered, and it answers the only question the
    # gate actually needs answered: can this lane serve a request right now. If
    # Anthropic is genuinely exhausted the probe fails and the fallback proceeds
    # exactly as before; if it is healthy we keep the fleet's largest allocation.
    #
    # Scoped to UNREADABLE and to Anthropic deliberately. A measurably-over bucket
    # still refuses without probing — spending against a limit we know we passed is
    # what the ration exists to prevent — and the metered buckets (openrouter) keep
    # failing closed on unknown, because there the unknown protects money rather
    # than a subscription we have already paid for.
    if _mc_ration_unreadable anthropic; then
      log "anthropic:$m quota is UNREADABLE (not measured) — probing the lane instead of refusing it; an unreadable quota is not an exhausted one"
    else
      MC_BOUNDED_OUT="Anthropic quota admission blocked (over ration)"
      log "anthropic:$m quota admission blocked; skipping inference probe"
      return 75
    fi
  fi
  _mc_bounded "$PROBE_TIMEOUT" claude -p 'reply with exactly: ok' --model "$m"; rc=$?
  out="$MC_BOUNDED_OUT"
  [ "$rc" -eq 0 ] && return 0
  # Log the captured output tail on timeout: an EMPTY capture (claude -p is
  # silent until completion) means a hang/backoff loop, error text means an
  # actual failure — 2026-08-05's refusals were undiagnosable without this.
  [ "$rc" -eq 124 ] && log "model $m probe timed out after ${PROBE_TIMEOUT}s — captured output: '$(printf '%s' "$out" | tail -c 200 | tr '\n' ' ')'"
  if printf '%s' "$out" | grep -qiE "$QUOTA_SIG"; then return 1; fi
  # transient? retry once
  sleep 5
  _mc_bounded "$PROBE_TIMEOUT" claude -p 'reply with exactly: ok' --model "$m"; rc=$?
  out="$MC_BOUNDED_OUT"
  [ "$rc" -eq 0 ] && return 0
  [ "$rc" -eq 124 ] && log "model $m probe timed out after ${PROBE_TIMEOUT}s (retry) — captured output: '$(printf '%s' "$out" | tail -c 200 | tr '\n' ' ')'"
  printf '%s' "$out" | grep -qiE "$QUOTA_SIG" && return 1
  # rc=2 is where the fleet loses a lane, so it is the ONE outcome that must not
  # be silent. Until now the output was logged only on a timeout, so an rc=2
  # meant "anthropic unusable (rc=2)" and nothing else — no status, no message,
  # no way to tell an expired token from an overloaded API from a quota reply
  # whose wording QUOTA_SIG does not match.
  #
  # Measured 2026-09-06: that blindness cost a full week of codex. Anthropic
  # started returning rc=2 at 05:52, every role fell through to codex, and codex
  # went from ~13M tokens/day to 713M in eighteen hours — spending a weekly
  # bucket in under fifteen. The probe reproduced clean the next morning, so
  # whatever the cause was, it was transient and it is now unknowable.
  log "model $m UNUSABLE after 2 attempts (rc=$rc, no quota signature) — captured output: '$(printf '%s' "$out" | tail -c 300 | tr '\n' ' ')'"
  return 2
}

# _mc_probe_codex MODEL → 0 usable | non-zero unusable. The OpenAI API key is
# stripped above, so a pass proves the ChatGPT-subscription OAuth lane works.
_mc_probe_codex() {
  local m="$1" rc
  if _mc_is_over_ration "codex:$m"; then
    MC_BOUNDED_OUT="Codex quota admission blocked (over ration or observation unavailable)"
    log "codex:$m quota admission blocked; skipping inference probe"
    return 75
  fi
  _mc_bounded "$PROBE_TIMEOUT" codex exec --skip-git-repo-check --model "$m" 'reply with exactly: ok'
  rc=$?
  [ "$rc" -eq 124 ] && log "controller fallback codex:$m probe timed out after ${PROBE_TIMEOUT}s"
  [ "$rc" -ne 0 ] && log "controller fallback codex:$m probe failed (rc=$rc): $(printf '%s' "$MC_BOUNDED_OUT" | tail -3 | tr '\n' ' ')"
  return "$rc"
}

# All Pi role/controller probes share admission; local Ollama maps to no cloud bucket.
_mc_probe_pi() {
  local m="$1"
  if _mc_is_over_ration "pi:$m"; then
    MC_BOUNDED_OUT="Pi quota admission blocked (over ration or observation unavailable)"
    log "pi:$m quota admission blocked; skipping inference probe"
    return 75
  fi
  # --no-extensions: the probe asks whether the MODEL answers. Extension discovery loaded
  # the global and repo copies of the same tools and exited rc=1 in any checkout with
  # .pi/extensions (lib/pi-ext-args.sh), so the probe reported a working model as dead.
  _mc_bounded "$PROBE_TIMEOUT" pi --mode json --no-session --no-tools --no-extensions --model "$m" -p 'reply with exactly: ok'
}

# ---- the daily ration gate (M-QUOTA-RATIONING-ROUTING M4, D-1/D-4) ---------
#
# Routing has always asked "is this lane UP?" and never "can it AFFORD to be
# used?". A probe answers the first; only the ledger answers the second. A rung
# whose bucket is over its ration (Codex/Anthropic: the weekday pace, 20% per weekday
# with weekends spending the slack; others 10%/day) is skipped exactly like a failed probe,
# so the walk descends to a cheaper rung — and when nothing is left, the existing
# "NO usable controller" refusal takes over: it announces once per episode and
# spends zero tokens beyond probes, which IS the pause D-4 asks for.
#
# Codex and Ollama Cloud fail closed on unavailable accounting. Other providers
# retain their existing ledger policy. The quota command itself is bounded.
MC_OVER_RATION=""
MC_OVER_RATION_READ=0

_mc_load_ration() {
  [ "$MC_OVER_RATION_READ" -eq 1 ] && return 0
  MC_OVER_RATION_READ=1
  if ! command -v ailang >/dev/null 2>&1; then
    MC_OVER_RATION="codex ollama"
    log "ration gate: no ailang on PATH — blocking Codex and Ollama Cloud"
    return 0
  fi
  # rc is taken from ailang DIRECTLY, not through the pipe: a pipeline reports the
  # LAST command's status, so `ailang ... | tr` would report tr's 0 and hide the
  # failure on any shell where pipefail happens to be off.
  local _raw _rc
  # The bound must cover the reader's OWN budget, or the gate fails closed on a healthy account.
  # Measured 2026-10-01: once the keychain token went stale, the Anthropic read became 5s HTTP
  # (401) + up to 30s of `claude -p /usage`, so the whole command took 31-39s against a 15s
  # bound. All 7 fires after 2026-09-30 21:40 timed out and blocked Codex, with Codex at 51%
  # used of 68% allowed. 60s covers internal/mission/anthropic_usage_cli.go's 30s plus the
  # HTTP reads; it runs once per fire.
  _mc_bounded "${MISSION_QUOTA_TIMEOUT:-60}" ailang mission quota --over; _rc=$?
  _raw="$MC_BOUNDED_OUT"
  MC_OVER_RATION=$(printf '%s\n' "$_raw" | awk '/^(codex|ollama|anthropic|openrouter|opencode)$/' | tr '\n' ' ')
  # Keep the REASONS, not just the verdict. `--over` emits a bare bucket name as the
  # machine-readable block list and a `quota: <bucket> <state>: …` line as the human
  # one, and the driver used to read the first and discard the second — so a bucket
  # blocked because its quota could not be READ was reported to humans, in the GitHub
  # notice and the log, as "over daily ration".
  #
  # Measured 2026-09-22: Anthropic sat at ~89% FREE on the account while three World
  # iterations in a row were told `anthropic lane unusable (over daily ration)` and
  # descended codex -> ollama -> openrouter -> a hung pi. The gate was behaving as
  # designed (`--over` blocks unknown quota by policy); the SENTENCE was false, and it
  # is the sentence a human acts on.
  MC_RATION_REASONS=$(printf '%s\n' "$_raw" | awk '/^quota: /{sub(/^quota: /,""); print}')
  if [ "$_rc" -ne 0 ]; then
    # Say so. A silently unrationed fleet looks identical to a rationed one that
    # found nothing over — and that is the exact ambiguity this milestone exists
    # to remove. An `ailang` too old to know --over lands here.
    MC_OVER_RATION="codex ollama"
    log "ration gate: quota command failed (rc=$_rc) — blocking Codex and Ollama Cloud"
    return 0
  fi
  if [ -n "$MC_OVER_RATION" ]; then
    log "ration gate: blocked buckets (over ration or unknown quota):$MC_OVER_RATION"
  fi
  return 0
}

# _mc_ration_reason BUCKET → the human reason that bucket is blocked.
#
# Distinguishes the two states the old message conflated, because they have utterly
# different resume conditions: "over" clears when the window rolls; "unknown" never
# clears on its own and needs an operator. Falls back to the raw line rather than
# inventing a phrase.
_mc_ration_reason() {
  local b="$1" line
  line=$(printf '%s\n' "${MC_RATION_REASONS:-}" | grep -m1 "^${b} " 2>/dev/null)
  case "$line" in
    "${b} over"*)    printf 'over daily ration' ;;
    "${b} unknown"*) printf 'quota UNREADABLE (not measured — the lane may be fine; see `ailang mission quota`)' ;;
    "${b} stale"*)   printf 'quota observation STALE' ;;
    "")              printf 'blocked by the ration gate (no reason line captured)' ;;
    *)               printf '%s' "${line#"$b" }" ;;
  esac
}

# _mc_reset_hint → the reset credits held in reserve, one line per bucket that reports
# any, or nothing. `ailang mission quota --over` appends a "; N Codex reset credit(s) in
# reserve ... attended only: <command>" clause to a blocked bucket's reason, and this lifts
# it out so the notices a human reads when quota runs dry say what is still in hand.
# Spending a credit is an ATTENDED decision (Mark, 2026-09-24): the driver only reports it.
_mc_reset_hint() {
  printf '%s\n' "${MC_RATION_REASONS:-}" | awk -F'; ' '{for (i = 2; i <= NF; i++) if ($i ~ /reset credit/) print $i}'
}

# _mc_ration_unreadable BUCKET → true when the bucket is blocked because its quota
# could not be READ, as opposed to measurably exceeding it.
#
# The distinction decides whether a probe is worth making. "Over" is a measurement:
# probing anyway spends against a limit we know we have passed. "Unknown" is the
# ABSENCE of a measurement, and blocking on it means refusing a lane that may be
# entirely healthy — which is what happened for a week.
_mc_ration_unreadable() {
  case "$(_mc_ration_reason "$1")" in
    *UNREADABLE*) return 0 ;;
    *) return 1 ;;
  esac
}

# _mc_rung_bucket ENTRY → canonical bucket, mirroring observatory.CanonicalQuotaBucket.
# An entry we cannot classify returns EMPTY and is therefore never rationed —
# attaching an unknown rung to the nearest real bucket is how a ration ends up
# measuring the wrong thing and saying nothing.
_mc_rung_bucket() {
  case "$1" in
    codex:*) printf 'codex' ;;
    pi:openrouter/*) printf 'openrouter' ;;
    pi:ollama/*:cloud|pi:ollama/*-cloud) printf 'ollama' ;;
    pi:ollama/*) printf '' ;; # Local models do not consume the cloud subscription.
    claude:*) printf 'anthropic' ;;
    pi:*) printf '' ;;
    *) printf 'anthropic' ;;
  esac
}

_mc_is_over_ration() {
  local b
  _mc_load_ration
  [ -z "$MC_OVER_RATION" ] && return 1
  b=$(_mc_rung_bucket "$1")
  [ -z "$b" ] && return 1
  case " $MC_OVER_RATION " in *" $b "*) return 0 ;; esac
  return 1
}
