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
_mc_set_controller() {
  local requested="$1"
  MODEL_WHY="$2"
  case "$requested" in
    codex:*) CONTROLLER_PROVIDER=codex; MODEL="${requested#codex:}"; MISSION_ANTHROPIC_AVAILABLE=0 ;;
    pi:*) CONTROLLER_PROVIDER=pi; MODEL="${requested#pi:}"; MISSION_ANTHROPIC_AVAILABLE=0 ;;
    claude:*) CONTROLLER_PROVIDER=claude; MODEL="${requested#claude:}"; MISSION_ANTHROPIC_AVAILABLE=1 ;;
    *) CONTROLLER_PROVIDER=claude; MODEL="$requested"; MISSION_ANTHROPIC_AVAILABLE=1 ;;
  esac
  CONTROLLER_ID="${CONTROLLER_PROVIDER}:${MODEL}"
  export CONTROLLER_PROVIDER CONTROLLER_ID MODEL MODEL_WHY MISSION_ANTHROPIC_AVAILABLE
}
select_model() {
  # 1. absolute pin
  if [ -n "${MISSION_MODEL:-}" ]; then
    # A demoted pin has a SPENT bucket, so honouring it again just reruns the
    # failure. Fall through to probing and let the chain answer.
    if _mc_is_demoted "$(_mc_canon_id "$MISSION_MODEL")" || _mc_is_over_ration "$MISSION_MODEL"; then
      log "env pin $MISSION_MODEL unavailable (runtime limit or quota admission) — falling through to the chain"
    else
      _mc_set_controller "$MISSION_MODEL" "env pin"; return 0
    fi
  fi
  # 2. override file pin (optional expiry epoch)
  if [ -f "$OVERRIDE_FILE" ]; then
    local ov_model ov_until now
    read -r ov_model ov_until < "$OVERRIDE_FILE" 2>/dev/null || true
    now=$(date +%s)
    if [ -n "${ov_until:-}" ] && [ "$now" -ge "${ov_until:-0}" ]; then
      rm -f "$OVERRIDE_FILE"
      log "model override expired — resuming preference probing"
    elif [ -n "${ov_model:-}" ]; then
      if _mc_is_demoted "$(_mc_canon_id "$ov_model")" || _mc_is_over_ration "$ov_model"; then
        log "override pin $ov_model DEMOTED this fire (runtime bucket limit) — falling through to the chain"
      else
        _mc_set_controller "$ov_model" "override file"; return 0
      fi
    fi
  fi
  # 3. ordered preference probing.
  #
  # PROVIDER-DISPATCHED since 2026-09-05 (Mark, attended: "put astra ahead of each
  # fable instance, that falls back to fable"). This list used to be Anthropic-only —
  # every entry went to `_mc_probe`, the claude CLI probe — so a non-Anthropic model
  # could ONLY be expressed in CONTROLLER_FALLBACK, which is reached after EVERY
  # Anthropic candidate has failed. There was therefore no way to say
  # "opus, then astra, then fable": a codex entry could sit before opus (never) or
  # after fable (too late), but not BETWEEN them. That ordering is the whole ask.
  #
  # Bare and `claude:`-prefixed entries keep the exact 0/1/2 quota-vs-unusable
  # semantics they had; only the dispatch is new. _mc_set_controller already parses
  # every prefix, so a matched entry needs no special-casing beyond its probe.
  local m why rcode
  for m in $(printf '%s' "$PREFS" | tr ',' ' '); do
    if _mc_is_demoted "$(_mc_canon_id "$m")"; then
      log "controller candidate $m DEMOTED this fire (runtime bucket limit) — skipping"
      continue
    fi
    if _mc_is_over_ration "$m"; then
      log "controller candidate $m is OVER RATION (bucket $(_mc_rung_bucket "$m")) — skipping to a cheaper rung"
      continue
    fi
    case "$m" in
      codex:*)
        if _mc_probe_codex "${m#codex:}"; then
          _mc_set_controller "$m" "probe ok"; return 0
        fi
        log "controller preference $m unusable — falling through"
        ;;
      pi:*)
        _mc_probe_pi "${m#pi:}"
        rcode=$?
        if [ "$rcode" -eq 0 ]; then _mc_set_controller "$m" "probe ok"; return 0; fi
        log "controller preference $m probe failed (rc=$rcode within ${PROBE_TIMEOUT}s) — falling through"
        ;;
      *)
        _mc_probe "$m"; rcode=$?
        case "$rcode" in
          0) _mc_set_controller "$m" "probe ok"; return 0 ;;
          1) log "model $m quota-limited — falling through" ;;
          2) log "model $m unusable (auth/transient) — falling through" ;;
        esac
        ;;
    esac
  done
  # 4. cross-provider fallback CHAIN, walked in order (Mark 2026-08-31 — see the
  # CONTROLLER_FALLBACK comment above). Every rung is probe-gated; an unsupported
  # entry is skipped loudly rather than aborting the walk, so one typo cannot
  # disable the rungs behind it.
  log "all Anthropic controller candidates unavailable — walking fallback chain ($CONTROLLER_FALLBACK)"
  local fb
  for fb in $(printf '%s' "$CONTROLLER_FALLBACK" | tr ',' ' '); do
    if _mc_is_demoted "$(_mc_canon_id "$fb")"; then
      log "controller fallback rung $fb DEMOTED this fire (runtime bucket limit) — skipping"
      continue
    fi
    if _mc_is_over_ration "$fb"; then
      log "controller fallback rung $fb is OVER RATION (bucket $(_mc_rung_bucket "$fb")) — skipping to a cheaper rung"
      continue
    fi
    case "$fb" in
      codex:*)
        m="${fb#codex:}"
        if _mc_probe_codex "$m"; then
          _mc_set_controller "$fb" "Anthropic unavailable; subscription fallback"
          return 0
        fi
        ;;
      pi:*)
        m="${fb#pi:}"
        # Same probe shape as the role-lane pi loop: --no-tools keeps it ~1 reply
        # token, --no-session avoids polluting ~/.pi/sessions; rc is the verdict.
        # rc captured explicitly: after `if cmd; then...fi` falls through, $? is the
        # IF's status (0), not cmd's — logging it would report every failure as rc=0.
        _mc_probe_pi "$m"
        rcode=$?
        if [ "$rcode" -eq 0 ]; then
          _mc_set_controller "$fb" "Anthropic+codex unavailable; pi fallback rung"
          return 0
        fi
        log "pi controller rung '$m' probe failed (rc=$rcode within ${PROBE_TIMEOUT}s) — falling through"
        ;;
      *) log "unsupported CONTROLLER_FALLBACK entry '$fb' (expected codex:<model> or pi:<model>) — skipping" ;;
    esac
  done
  return 1
}
log(){ printf 'LOG %s\n' "$*"; }
_mc_load_ration(){ :; }
_mc_is_demoted(){ return 1; }
_mc_canon_id(){ printf '%s' "$1"; }
_mc_probe_pi(){ printf 'PROBE %s\n' "$1"; return 0; }
MISSION_MODEL=""; OVERRIDE_FILE=/tmp/fleet-iter14-executor-no-override
PREFS=""; PROBE_TIMEOUT=5
CONTROLLER_FALLBACK="pi:openrouter/z-ai/glm-5.3,pi:ollama/local"
MC_OVER_RATION=openrouter
select_model; printf 'RATION_BLOCK rc=%s selected=%s\n' "$?" "$CONTROLLER_ID"
MC_OVER_RATION=""
select_model; printf 'RATION_HEALTHY rc=%s selected=%s\n' "$?" "$CONTROLLER_ID"
RUNTIME_QUOTA_SIG="${MISSION_RUNTIME_QUOTA_SIG:-reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:}"
TRANSIENT_SIG="API Error: Overloaded|socket connection was closed|overloaded_error|API Error: 5[0-9][0-9]|API Error: Internal|API Error: Connection|API Error: Request timed out|Provider finish_reason: error"
for msg in '402: Insufficient credits' '429: quota exhausted' 'API Error: Overloaded' 'healthy response'; do
 printf '%s\n' "$msg" | grep -qE "$RUNTIME_QUOTA_SIG"; q=$?
 printf '%s\n' "$msg" | grep -qiE "$TRANSIENT_SIG"; t=$?
 printf 'CLASSIFY msg=%s runtime_match_rc=%s transient_match_rc=%s\n' "$msg" "$q" "$t"
done
RC=1; _mc_slot_last=gate-2
  case "$RC:$_mc_slot_last" in
    0:complete) _mc_slot_verdict="COMPLETED" ;;
    0:abort) _mc_slot_verdict="ABORTED" ;;
    0:fired|0:) _mc_slot_verdict="DIED-PRE-GATE-0" ;;
    0:gate-*) _mc_slot_verdict="REAPED at=$_mc_slot_last" ;;
    143:*|137:*) _mc_slot_verdict="KILLED at=${_mc_slot_last:-fired}" ;;
    *:*) _mc_slot_verdict="CRASHED at=${_mc_slot_last:-fired}" ;;
  esac
printf '402_WITH_HEARTBEAT verdict=%s\n' "$_mc_slot_verdict"
RC=0; _mc_slot_last=complete
  case "$RC:$_mc_slot_last" in
    0:complete) _mc_slot_verdict="COMPLETED" ;;
    0:abort) _mc_slot_verdict="ABORTED" ;;
    0:fired|0:) _mc_slot_verdict="DIED-PRE-GATE-0" ;;
    0:gate-*) _mc_slot_verdict="REAPED at=$_mc_slot_last" ;;
    143:*|137:*) _mc_slot_verdict="KILLED at=${_mc_slot_last:-fired}" ;;
    *:*) _mc_slot_verdict="CRASHED at=${_mc_slot_last:-fired}" ;;
  esac
printf 'HEALTHY_WITH_HEARTBEAT verdict=%s\n' "$_mc_slot_verdict"
