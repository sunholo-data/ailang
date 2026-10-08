#!/usr/bin/env bash
# mission_gate0_self_notices.sh — second, no-authority Gate-0 read: self-authored crash notices on the
# current AND previous bookkeeping issue, since the watermark. A hit grants nothing; it says a
# fire died.
#
# A comment is a crash notice iff author.login (lowercased) == --self (lowercased) AND the body
# starts at offset 0 with the literal '⚠️ Mission iteration **FAILED to complete** (rc=' followed
# by ASCII digits. Nothing after the digits is read (V3: the tail has drifted). No comment body
# is ever printed.
#
# This is a bash-3.2 script (the rig's /bin/bash is 3.2.57). No associative arrays, no ${x,,},
# no {n} intervals. The gh binary is the ONLY read seam (the stub in the suite shadows --gh-bin).
set -uo pipefail

GH_BIN="gh"
GH_TIMEOUT="5"
SKEW=300
TS_CTL="2026-09-08T06:19:00Z"
TS_ARM=""
ISSUE=""; PREV_ISSUE=""; PREV_PASSED=0; REPO=""; SELF=""; SINCE_ARG=""; CONTROL=""; DRIVER_SRC=""
WMFILES=()

usage() {
  cat <<'EOF'
Usage: mission_gate0_self_notices.sh --issue <n> [--prev-issue <m>] --repo <owner/name> --self <login>
                             ( --since <ISO8601> | --watermark-file <path> [--watermark-file <path> ...] )
                             --control <issue>:<count> | auto
                             [--driver-src <file>] [--gh-bin <path>] [--gh-timeout <secs>]

Second, no-authority Gate-0 read: self-authored crash notices on the current AND previous
bookkeeping issue, since the watermark. A hit grants nothing; it says a fire died.

--control auto resolves the control from a per-repo table keyed by --repo; an unknown repo
is a floor (no fallback).

Exit codes:
  0  no in-window crash notice, controls ok
  1  one or more in-window crash notices (a controller must look; run Gate 2's traces (a)-(c))
  2  any floor F0-F10 (usage, binary, timeout, gh failure, malformed, truncated, control,
     signature-source, watermark unavailable/invalid, timestamp parser unusable)
EOF
}

# ── bounded subprocess wrapper (bash 3.2; no timeout(1)/gtimeout on this rig) ──
run_bounded() {  # $1=timeout_s  $2=out_file  $3..=cmd ; returns 124 on timeout, child rc otherwise
  local t="$1" out="$2"; shift 2
  local child rc watchdog
  ( "$@" >"$out" 2>"$out.err" ) &
  child=$!
  ( /bin/sleep "$t"; kill -TERM "$child" 2>/dev/null; /bin/sleep 0.2; kill -KILL "$child" 2>/dev/null ) &
  watchdog=$!
  wait "$child" 2>/dev/null; rc=$?
  kill "$watchdog" 2>/dev/null
  wait "$watchdog" 2>/dev/null
  if [ "$rc" -ge 128 ]; then return 124; fi
  return "$rc"
}

SIG_TEXT="⚠️ Mission iteration **FAILED to complete** (rc="

# ── python3 comment parser (house pattern: verify_ail.sh / gate1_range_check.sh) ──
# Emits one `CRASH <createdAt> <rc> <url>` line per anchored self-authored notice (sorted by
# createdAt), then one `COUNT <comments> <n_self>` line. Exits 2 with a reason on F4.
parse_comments() {  # $1=file $2=issue $3=self_lc
  python3 - "$1" "$2" "$3" "$SIG_TEXT" <<'PY'
import json, sys
f, n, selfs, SIG = sys.argv[1], sys.argv[2], sys.argv[3].lower(), sys.argv[4]
try:
    with open(f) as fh:
        d = json.load(fh)
except Exception as e:
    sys.stderr.write("✗ malformed comments payload for issue #%s: could not parse JSON (%s)\n" % (n, e))
    sys.exit(2)
comms = d.get("comments")
if comms is None or isinstance(comms, bool) or not isinstance(comms, list):
    sys.stderr.write("✗ malformed comments payload for issue #%s: no comments list\n" % n)
    sys.exit(2)
rows = []
for i, c in enumerate(comms):
    if not isinstance(c, dict):
        sys.stderr.write("✗ malformed comments payload for issue #%s: comment %d not an object\n" % (n, i))
        sys.exit(2)
    a = c.get("author")
    login = a.get("login") if isinstance(a, dict) else None
    created = c.get("createdAt")
    body = c.get("body")
    if login is None or created is None or body is None:
        sys.stderr.write("✗ malformed comments payload for issue #%s: comment %d lacks author.login/createdAt/body\n" % (n, i))
        sys.exit(2)
    if not isinstance(login, str) or not isinstance(created, str) or not isinstance(body, str):
        sys.stderr.write("✗ malformed comments payload for issue #%s: comment %d field is not a string\n" % (n, i))
        sys.exit(2)
    rows.append((created, login, body, c.get("url")))
rows.sort(key=lambda r: r[0])
ncrash = 0
for created, login, body, url in rows:
    if login.lower() == selfs and body.startswith(SIG):
        tail = body[len(SIG):]
        j = 0
        while j < len(tail) and tail[j].isdigit():
            j += 1
        if j > 0:
            ncrash += 1
            rc = tail[:j]
            u = url if url else "-"
            sys.stdout.write("CRASH %s %s %s\n" % (created, rc, u))
nself = sum(1 for _, login, _, _ in rows if login.lower() == selfs)
sys.stdout.write("COUNT %d %d\n" % (len(rows), nself))
sys.exit(0)
PY
}

# ── python3 meta parser (F5 REST count) ──
parse_meta() {  # $1=file $2=issue
  python3 - "$1" "$2" <<'PY'
import json, sys
f, n = sys.argv[1], sys.argv[2]
try:
    with open(f) as fh:
        d = json.load(fh)
except Exception as e:
    sys.stderr.write("✗ malformed comments payload for issue #%s: could not parse meta JSON (%s)\n" % (n, e))
    sys.exit(2)
mc = d.get("comments")
if mc is None or isinstance(mc, bool) or not isinstance(mc, int):
    sys.stderr.write("✗ malformed comments payload for issue #%s: meta comments not an integer\n" % n)
    sys.exit(2)
print("META %d" % mc)
sys.exit(0)
PY
}

# ── D7a two-arm date helper (date invoked via PATH so a stub can shadow it) ──
ts_try() {  # $1=arm $2=value $3=outfmt -> stdout, rc from date
  case "$1" in
    bsd) TZ=UTC date -j -f '%Y-%m-%dT%H:%M:%SZ' "$2" "+$3" 2>/dev/null ;;
    gnu) date -u -d "$2" "+$3" 2>/dev/null ;;
  esac
}
ts_resolve_arm() {  # ONCE, before F9/F10; positive control on the parser itself (rule 3a)
  local a
  for a in bsd gnu; do
    if [ "$(ts_try "$a" "$TS_CTL" '%Y-%m-%dT%H:%M:%SZ')" = "$TS_CTL" ]; then TS_ARM="$a"; return 0; fi
  done
  echo "✗ timestamp parser unusable: neither BSD nor GNU date parsed the control value" >&2; exit 2
}
validate_ts() {  # $1=value $2=label(path or --since); prints nothing on ok; F10 message + exit 2 otherwise
  local v="$1" label="$2" canon epoch now
  case "$v" in
    [0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]T[0-9][0-9]:[0-9][0-9]:[0-9][0-9]Z) ;;
    *) echo "✗ watermark invalid: $label (not-ISO8601 '$v')" >&2; exit 2 ;;
  esac
  canon="$(ts_try "$TS_ARM" "$v" '%Y-%m-%dT%H:%M:%SZ')"
  if [ -z "$canon" ]; then echo "✗ watermark invalid: $label (not-canonical '$v')" >&2; exit 2; fi
  if [ "$canon" != "$v" ]; then echo "✗ watermark invalid: $label (not-canonical '$v' → '$canon')" >&2; exit 2; fi
  epoch="$(ts_try "$TS_ARM" "$v" '%s')"; now="$(date -u +%s)"
  if [ "$epoch" -gt $((now + SKEW)) ]; then echo "✗ watermark invalid: $label (future '$v' > now+${SKEW}s)" >&2; exit 2; fi
}

# ── argument parsing ──────────────────────────────────────────────────────────
while [ "$#" -gt 0 ]; do
  case "$1" in
    --issue) ISSUE="$2"; shift 2 ;;
    --prev-issue) PREV_ISSUE="$2"; PREV_PASSED=1; shift 2 ;;
    --repo) REPO="$2"; shift 2 ;;
    --self) SELF="$2"; shift 2 ;;
    --since) SINCE_ARG="$2"; shift 2 ;;
    --watermark-file) WMFILES+=("$2"); shift 2 ;;
    --control) CONTROL="$2"; shift 2 ;;
    --driver-src) DRIVER_SRC="$2"; shift 2 ;;
    --gh-bin) GH_BIN="$2"; shift 2 ;;
    --gh-timeout) GH_TIMEOUT="$2"; shift 2 ;;
    --help) usage; exit 0 ;;
    *) echo "✗ unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

f0_usage() {
  echo "✗ usage: --issue <digits>, --repo, --self and exactly one of --since | --watermark-file are required" >&2
  usage >&2; exit 2
}

# ── F0: mandatory arguments ───────────────────────────────────────────────────
[ -n "$ISSUE" ] || f0_usage
case "$ISSUE" in *[!0-9]*) f0_usage ;; esac
[ -n "$REPO" ] || f0_usage
[ -n "$SELF" ] || f0_usage
if [ "$PREV_PASSED" -eq 1 ]; then
  [ -n "$PREV_ISSUE" ] || f0_usage
  case "$PREV_ISSUE" in *[!0-9]*) f0_usage ;; esac
fi
if { [ -n "$SINCE_ARG" ] && [ "${#WMFILES[@]}" -gt 0 ]; } || \
   { [ -z "$SINCE_ARG" ] && [ "${#WMFILES[@]}" -eq 0 ]; }; then
  f0_usage
fi

# ── F6: control required ─────────────────────────────────────────────────────
if [ -z "$CONTROL" ]; then
  echo "✗ control required: --control <issue>:<count>" >&2; exit 2
fi
if [ "$CONTROL" = "auto" ]; then
  case "$REPO" in
    sunholo-data/ailang) CONTROL="852:10" ;;
    sunholo-data/ailang-world) CONTROL="107:4" ;;
    sunholo-data/stapledons-godot) CONTROL="4:5" ;;
    *) echo "✗ control required: no control for repo $REPO (pass --control <issue>:<count>)" >&2; exit 2 ;;
  esac
fi
CTRL_ISSUE="${CONTROL%%:*}"
CTRL_EXPECT="${CONTROL#*:}"

# ── F8: driver-source signature control (pre-network) ─────────────────────────
SIGNATURE_LINE="signature-source: unchecked"
if [ -n "$DRIVER_SRC" ]; then
  if /usr/bin/grep -qF "$SIG_TEXT" "$DRIVER_SRC" 2>/dev/null; then
    SIGNATURE_LINE="signature-source: $DRIVER_SRC ok"
  else
    echo "✗ signature not found in driver source: $DRIVER_SRC" >&2; exit 2
  fi
fi

# ── timestamp-parser control (pre-network) ───────────────────────────────────
ts_resolve_arm

# ── watermark disposition + F9/F10 + since ───────────────────────────────────
echo "gate0_self_notices: repo=$REPO self=$SELF issues=$ISSUE${PREV_ISSUE:+,$PREV_ISSUE}"
SELF_LC="$(printf '%s' "$SELF" | tr '[:upper:]' '[:lower:]')"
READABLE=0; GIVEN=0; WMIN_LIST=""; WMPATHS=""
if [ -n "$SINCE_ARG" ]; then
  validate_ts "$SINCE_ARG" "--since"
  SINCE="$SINCE_ARG"
  WSMARK="watermarks literal"
else
  for wf in "${WMFILES[@]}"; do
    GIVEN=$((GIVEN+1))
    if [ -n "$WMPATHS" ]; then WMPATHS="$WMPATHS $wf"; else WMPATHS="$wf"; fi
    if [ ! -e "$wf" ]; then
      echo "watermark: $wf ABSENT — DEGRADED: single-watermark read (iter-52 rule wants two)"
      continue
    fi
    if [ ! -r "$wf" ]; then
      echo "✗ watermark invalid: $wf (unreadable)" >&2; exit 2
    fi
    v=""
    IFS= read -r v < "$wf" || [ -n "$v" ]
    if [ -z "$v" ]; then
      echo "✗ watermark invalid: $wf (empty)" >&2; exit 2
    fi
    validate_ts "$v" "$wf"
    echo "watermark: $wf = $v ok"
    READABLE=$((READABLE+1))
    WMIN_LIST="${WMIN_LIST:+$WMIN_LIST$'\n'}$v"
  done
  if [ "$READABLE" -eq 0 ]; then
    echo "✗ watermark channel unavailable: no readable watermark among $GIVEN file(s): $WMPATHS" >&2; exit 2
  fi
  SINCE="$(printf '%s\n' "$WMIN_LIST" | /usr/bin/sort | head -1)"
  WSMARK="watermarks $READABLE/$GIVEN"
fi

if [ -n "$SINCE_ARG" ]; then
  echo "since: $SINCE (--since literal)"
else
  echo "since: $SINCE (older of $READABLE readable watermark(s) of $GIVEN)"
fi

# ── per-issue reads (F1-F5), the seam is --gh-bin ─────────────────────────────
LATEST_PARSED=""
read_issue() {  # $1=issue ; F1-F5 ; sets LATEST_PARSED on success; returns 2 on floor
  local n="$1" out po poer prc rrc meta mo moer mrc meta_n mcnt cline ccom
  out="$(mktemp)"
  run_bounded "$GH_TIMEOUT" "$out" "$GH_BIN" issue view "$n" --repo "$REPO" --json comments
  rrc=$?
  if [ "$rrc" -eq 124 ]; then echo "✗ gh timed out after ${GH_TIMEOUT}s for issue #$n" >&2; rm -f "$out" "$out.err"; return 2; fi
  if [ "$rrc" -eq 127 ]; then echo "✗ gh binary unreachable: $GH_BIN" >&2; rm -f "$out" "$out.err"; return 2; fi
  if [ "$rrc" -ne 0 ]; then echo "✗ gh failed (rc=$rrc) for issue #$n" >&2; rm -f "$out" "$out.err"; return 2; fi
  po="$(mktemp)"; poer="$(mktemp)"
  parse_comments "$out" "$n" "$SELF_LC" > "$po" 2> "$poer"; prc=$?
  rm -f "$out" "$out.err"
  if [ "$prc" -ne 0 ]; then cat "$poer" >&2; rm -f "$po" "$poer"; return 2; fi
  rm -f "$poer"
  parsed="$(cat "$po")"; rm -f "$po"

  meta="$(mktemp)"
  run_bounded "$GH_TIMEOUT" "$meta" "$GH_BIN" api "repos/$REPO/issues/$n"
  rrc=$?
  if [ "$rrc" -eq 124 ]; then echo "✗ gh timed out after ${GH_TIMEOUT}s for issue #$n" >&2; rm -f "$meta" "$meta.err"; return 2; fi
  if [ "$rrc" -eq 127 ]; then echo "✗ gh binary unreachable: $GH_BIN" >&2; rm -f "$meta" "$meta.err"; return 2; fi
  if [ "$rrc" -ne 0 ]; then echo "✗ gh failed (rc=$rrc) for issue #$n" >&2; rm -f "$meta" "$meta.err"; return 2; fi
  mo="$(mktemp)"; moer="$(mktemp)"
  parse_meta "$meta" "$n" > "$mo" 2> "$moer"; mrc=$?
  rm -f "$meta" "$meta.err"
  if [ "$mrc" -ne 0 ]; then cat "$moer" >&2; rm -f "$mo" "$moer"; return 2; fi
  rm -f "$moer"
  meta_n="$(cat "$mo")"; rm -f "$mo"
  mcnt="${meta_n#META }"
  cline="$(printf '%s\n' "$parsed" | /usr/bin/grep '^COUNT')"
  ccom="$(printf '%s' "$cline" | /usr/bin/awk '{print $2}')"
  if [ "$ccom" -ne "$mcnt" ]; then echo "✗ comment array truncated for issue #$n: got $ccom of $mcnt" >&2; return 2; fi
  LATEST_PARSED="$parsed"
  return 0
}

READ_N=(); READ_P=(); READ_LIST=""
for iss in "$ISSUE" "$PREV_ISSUE" "$CTRL_ISSUE"; do
  [ -n "$iss" ] || continue
  dup=0
  for x in $READ_LIST; do [ "$x" = "$iss" ] && dup=1; done
  [ "$dup" -eq 1 ] && continue
  if ! read_issue "$iss"; then
    # floor already printed to stderr
    exit 2
  fi
  READ_N+=("$iss"); READ_P+=("$LATEST_PARSED")
  READ_LIST="$READ_LIST $iss"
done

# ── classify --issue and --prev-issue (scoped; control never classified) ──────
ISSUE_ARR=(); COMMENTS_ARR=(); SELF_ARR=(); CRASH_ARR=()
CRASH_TEXT=""; IN_COUNT=0; NEWEST_AT=""; NEWEST_RC=""; NEWEST_ISSUE=""
emit_issue() {  # $1=issue $2=parsed
  local n="$1" parsed="$2" line at rc url yv count nself ncrash=0
  while IFS= read -r line; do
    case "$line" in
      CRASH\ *)
        set -- $line
        at="$2"; rc="$3"; url="${4}"
        if [ "$at" \> "$SINCE" ]; then yv=yes; else yv=no; fi
        if [ "$yv" = yes ]; then
          IN_COUNT=$((IN_COUNT+1))
          if [ -z "$NEWEST_AT" ] || [ "$at" \> "$NEWEST_AT" ]; then NEWEST_AT="$at"; NEWEST_RC="$rc"; NEWEST_ISSUE="$n"; fi
        fi
        CRASH_TEXT="${CRASH_TEXT}crash: issue=$n at=$at rc=$rc in-window=$yv url=$url"$'\n'
        ncrash=$((ncrash+1))
        ;;
      COUNT\ *)
        set -- $line
        count="$2"; nself="$3"
        ;;
    esac
  done <<< "$parsed"
  ISSUE_ARR+=("$n"); COMMENTS_ARR+=("$count"); SELF_ARR+=("$nself"); CRASH_ARR+=("$ncrash")
}
emit_for() {  # $1=issue ; skips if already emitted
  local i d
  for d in $EMITTED; do [ "$d" = "$1" ] && return 0; done
  for i in "${!READ_N[@]}"; do
    if [ "${READ_N[$i]}" = "$1" ]; then
      emit_issue "$1" "${READ_P[$i]}"
      EMITTED="$EMITTED $1"
      return 0
    fi
  done
  return 1
}
EMITTED=""
emit_for "$ISSUE"
if [ -n "$PREV_ISSUE" ]; then
  emit_for "$PREV_ISSUE" || { echo "✗ internal: --prev-issue $PREV_ISSUE was not read" >&2; exit 2; }
fi

# ── control all-time crash count on the control issue ────────────────────────
CTRL_GOT=0
for i in "${!READ_N[@]}"; do
  if [ "${READ_N[$i]}" = "$CTRL_ISSUE" ]; then
    CTRL_GOT="$(printf '%s\n' "${READ_P[$i]}" | /usr/bin/grep -c '^CRASH')"
  fi
done

# ── table (crash:, issue:), then F7 control, signature, verdict ──────────────
printf '%s' "$CRASH_TEXT"
for i in "${!ISSUE_ARR[@]}"; do
  echo "issue: ${ISSUE_ARR[$i]} comments=${COMMENTS_ARR[$i]} self=${SELF_ARR[$i]} crash=${CRASH_ARR[$i]} other=$((SELF_ARR[$i]-CRASH_ARR[$i]))"
done

if [ "$CTRL_GOT" -ne "$CTRL_EXPECT" ]; then
  echo "✗ CONTROL MISMATCH: issue $CTRL_ISSUE expect=$CTRL_EXPECT got=$CTRL_GOT"
  exit 2
fi
echo "control: issue=$CTRL_ISSUE expect=$CTRL_EXPECT got=$CTRL_GOT ok"
echo "$SIGNATURE_LINE"

VERDICT_SUF="mission_gate0_self_notices.sh, control $CTRL_ISSUE:$CTRL_EXPECT, $WSMARK"
if [ "$IN_COUNT" -gt 0 ]; then
  echo "verdict: $IN_COUNT crash notice(s) since $SINCE on #$ISSUE${PREV_ISSUE:+,#$PREV_ISSUE} — A FIRE DIED (newest rc=$NEWEST_RC at $NEWEST_AT on #$NEWEST_ISSUE): run Gate 2's died-mid-flight traces (a) open PRs (b) worktrees (c) uncommitted state BEFORE picking, and credit the orphan in the log; \"the queue is untouched\" describes the charter, not the work [$VERDICT_SUF]"
  exit 1
fi
echo "verdict: 0 crash notice(s) since $SINCE on #$ISSUE${PREV_ISSUE:+,#$PREV_ISSUE} — no death signal from the driver; Gate 2's traces (a)-(c) still run [$VERDICT_SUF]"
exit 0
