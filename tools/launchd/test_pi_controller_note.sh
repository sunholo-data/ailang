#!/bin/bash
# shellcheck disable=SC1091,SC2015,SC2016,SC2034,SC2329
# SC1091: the sourced paths are dynamic (PI_NOTE_ROOT may be a scratch copy).
# SC2015: `ok … || bad …` is the reporting idiom of this suite, not an if/else.
# SC2016: backticks and ${POLICY_ENV} inside single quotes are literal text the suite greps for.
# SC2034: the globals (CONTROLLER_PROVIDER, MC_PI_SESSION_ID, HARD_TIMEOUT, …) are read by the
#         driver functions this suite sources and runs; shellcheck cannot see that.
# SC2329: _mc_stalled is called by the extracted _mc_run_once, not directly by this file.
# test_pi_controller_note.sh — the pi controller's session-protocol note (lib/pi-controller-note.sh)
# reaches the real pi launch in mission-control.sh's _mc_run_once: as --append-system-prompt AND as
# the prefix of -p, on the fresh and the resume path. claude and codex argv stay byte-unchanged, and
# the role runner (scripts/mission_pi_run.sh) does not get the note.
#
# Ticket gate0:session-protocol-deadlock:agent-policy-unset (stapledon fire 2026-09-29T12-53-33).
# Extraction, not duplication: _mc_run_once is awk'd out of the driver (PI_NOTE_DRIVER, default the
# real one), the expected note is read from the lib the driver sources (PI_NOTE_ROOT), and fake pi,
# claude and codex record their argv NUL-separated. No model, no inference. bash 3.2 only.
set -uo pipefail

# Hermetic: the codex arm forwards every MISSION_* / AILANG_DRIVER_* / CODEX_* in the caller's env and
# routes on AILANG_CODEX_RUNTIME, so a caller's rig env would change the argv under test. Clear them.
for _v in $(compgen -e); do
  case "$_v" in MISSION_*|AILANG_*|CONTROLLER_*|CODEX_*|GIT_CONFIG_*) unset "$_v" ;; esac
done

HERE="$(cd "$(dirname "$0")" && pwd)"
DRV="${PI_NOTE_DRIVER:-$HERE/mission-control.sh}"
ROOT_DIR="${PI_NOTE_ROOT:-$(cd "$HERE/../.." && pwd)}"

T="${TMPDIR:-/tmp}/pi-controller-note-$$"
rm -rf "$T"; mkdir -p "$T/x" "$T/bin" "$T/repo" "$T/home"
trap 'rm -rf "$T"' EXIT
X="$T/x"
PROMPT_BODY="PROMPT-BODY"
RESUME_BODY="RESUME-BODY"
SESSION_ID="note-session-$$"
pass=0; fail=0
ok()  { pass=$((pass + 1)); echo "  PASS: $1"; }
bad() { fail=$((fail + 1)); echo "  FAIL: $1"; }

# --- extraction (guarded: an empty extract would make every case vacuous) ------------------
awk '/^_mc_run_once\(\) \{/,/^\}$/' "$DRV" > "$X/run_once.sh"
{
  awk '/^_mc_pi_session_exists\(\) \{/,/^\}$/'      "$DRV"
  awk '/^_mc_claude_session_prepare\(\) \{/,/^\}$/' "$DRV"
  awk '/^_mc_uuid\(\) \{/,/^\}$/'                   "$DRV"
  awk '/^_mc_descendants\(\) \{/,/^\}$/'            "$DRV"
  awk '/^_mc_kill_tree\(\) \{/,/^\}$/'              "$DRV"
  /usr/bin/grep -E '^log\(\) \{' "$DRV"
  cat "$X/run_once.sh"
} > "$X/fns.sh"
for f in run_once fns; do
  [ -s "$X/$f.sh" ] || { echo "FAIL extraction: $f not found in $DRV"; exit 1; }
done
/usr/bin/grep -q '^_mc_run_once() {' "$X/run_once.sh" \
  || { echo "FAIL extraction: _mc_run_once boundary not found in $DRV"; exit 1; }
/usr/bin/grep -q '^log() {' "$X/fns.sh" \
  || { echo "FAIL extraction: log() not found in $DRV"; exit 1; }

# --- fakes: pi, claude, codex record argv NUL-separated to $ARGV_OUT and exit 0 ----------------
cat > "$T/bin/fake" <<'STUB'
#!/bin/bash
printf '%s\0' "$@" > "$ARGV_OUT"
STUB
for n in pi claude codex; do cp "$T/bin/fake" "$T/bin/$n"; done
rm -f "$T/bin/fake"
chmod +x "$T/bin/"*

# read_argv FILE → ARGV (one array element per NUL-terminated argv entry)
read_argv() {
  ARGV=(); local a
  [ -s "$1" ] || return 0
  while IFS= read -r -d '' a; do ARGV+=("$a"); done < "$1"
}
# idx_of STRING → index of the first ARGV element equal to STRING, or -1
idx_of() {
  local i=0
  while [ "$i" -lt "${#ARGV[@]}" ]; do
    if [ "${ARGV[$i]}" = "$1" ]; then echo "$i"; return 0; fi
    i=$((i + 1))
  done
  echo -1
}
# any_has FRAGMENT → 0 if any ARGV element contains FRAGMENT
any_has() {
  local i=0
  while [ "$i" -lt "${#ARGV[@]}" ]; do
    case "${ARGV[$i]}" in *"$1"*) return 0 ;; esac
    i=$((i + 1))
  done
  return 1
}
# note_from ROOT → the note exactly as the driver's lib prints it (no trailing newline)
note_from() { ( . "$1/tools/launchd/lib/pi-controller-note.sh"; _mc_pi_controller_note ); }

# drive PROVIDER MODE ROOT NAME — one real _mc_run_once in its own subshell.
#   MODE fresh|resume (resume: a pi session file for SESSION_ID already exists). Argv -> $X/NAME/argv.bin
drive() {
  local prov="$1" mode="$2" root="$3" name="$4" lab="$X/$4" slug
  mkdir -p "$lab/state" "$lab/pi"
  if [ "$mode" = resume ]; then
    slug="$T/repo"; slug="${slug#/}"; slug=$(printf '%s' "$slug" | tr '/\\:' '---')
    mkdir -p "$lab/pi/sessions/--${slug}--"
    : > "$lab/pi/sessions/--${slug}--/2026-10-29T00-00-00_${SESSION_ID}.jsonl"
  fi
  export ARGV_OUT="$lab/argv.bin" PATH="$T/bin:/usr/bin:/bin" HOME="$T/home" \
         CODEX_HOME="$T/nocodex" PI_CODING_AGENT_DIR="$lab/pi"
  (
    set -uo pipefail
    . "$X/fns.sh"
    . "$root/tools/launchd/lib/pi-ext-args.sh"
    . "$root/tools/launchd/lib/codex-env-args.sh"
    . "$root/tools/launchd/lib/codex-auth-profile.sh"
    CONTROLLER_PROVIDER="$prov"; MODEL="model-x"; REPO="$T/repo"; PROMPT="$PROMPT_BODY"
    MC_DRIVER_ROOT="$root"; MC_PI_SESSION_ID="$SESSION_ID"; MC_PI_RESUME_PROMPT="$RESUME_BODY"
    LOG="$lab/driver.log"; PIDFILE="$lab/pid"; STATE_DIR="$lab/state"; _mc_slot_state="$lab/state"
    MISSION_NAME=notemission; attempt=1
    HARD_TIMEOUT=60; STALL_GRACE=60; STALL_INTERVAL=5; STALL_SAMPLES=3; STALL_CHILD_AGE=60
    _mc_stalled() { return 1; }
    : > "$LOG"
    _mc_run_once
    echo "rc=$?"
  ) > "$lab/out" 2>&1
}

# bounded NAME FN ARGS… — runs FN in the background and fails loudly past a 90 s deadline
bounded() {
  local name="$1"; shift
  ( "$@" ) > "$X/$name.case" 2>&1 &
  local pid=$! dl
  dl=$(( $(date +%s) + 90 ))
  while kill -0 "$pid" 2>/dev/null; do
    if [ "$(date +%s)" -ge "$dl" ]; then
      kill "$pid" 2>/dev/null
      bad "$name: case exceeded the 90s deadline"
      return 1
    fi
    sleep 1
  done
  wait "$pid" 2>/dev/null
  return 0
}

# --- runs ------------------------------------------------------------------------------
SCR="$T/scratch"
bounded pi-fresh    drive pi fresh  "$ROOT_DIR" pi-fresh
bounded pi-resume   drive pi resume "$ROOT_DIR" pi-resume
bounded claude      drive claude fresh "$ROOT_DIR" claude
bounded codex       drive codex fresh  "$ROOT_DIR" codex
# T6b control: a benign rewording of the lib on a scratch root must not turn T1-T3 red.
mkdir -p "$SCR/tools/launchd/lib"
cp "$ROOT_DIR/tools/launchd/lib/"*.sh "$SCR/tools/launchd/lib/"
sed 's/Run every ailang command through bash\./Route every ailang command through bash./' \
  "$ROOT_DIR/tools/launchd/lib/pi-controller-note.sh" > "$SCR/tools/launchd/lib/pi-controller-note.sh"
if cmp -s "$ROOT_DIR/tools/launchd/lib/pi-controller-note.sh" "$SCR/tools/launchd/lib/pi-controller-note.sh"; then
  bad "T6b control: the benign rewording did not change the scratch lib (control is vacuous)"
fi
bounded pi-scratch drive pi fresh "$SCR" pi-scratch

# --- T1 / T2 / T3: pi argv on fresh and resume paths ------------------------------------------
NOTE=$(note_from "$ROOT_DIR")
P40=$(printf '%s' "$NOTE" | cut -c1-40)

# check_pi NAME NOTE WANT_P — the pi argv carries NOTE as --append-system-prompt and WANT_P as -p
check_pi() {
  local name="$1" note="$2" want_p="$3" i j pv
  read_argv "$X/$name/argv.bin"
  i=$(idx_of --append-system-prompt)
  if [ "$i" -ge 0 ] && [ "${ARGV[$((i + 1))]:-}" = "$note" ]; then
    ok "$name: --append-system-prompt is followed by the lib note, byte-equal"
  else
    bad "$name: --append-system-prompt missing or not followed by the lib note"
  fi
  j=$(idx_of -p)
  pv="${ARGV[$((j + 1))]:-}"
  if [ "$j" -ge 0 ] && [ "$pv" = "$want_p" ]; then
    ok "$name: -p value is note + blank line + prompt, exactly"
  else
    bad "$name: -p value is not note + blank line + expected prompt"
  fi
  if [ "$name" = pi-resume ]; then
    case "$pv" in *"$PROMPT_BODY"*) bad "$name: resume -p still contains the fresh PROMPT" ;; esac
  fi
}
check_pi pi-fresh "$NOTE" "$NOTE"$'\n\n'"$PROMPT_BODY"
check_pi pi-resume "$NOTE" "$NOTE"$'\n\n'"$RESUME_BODY"

# --- T1b: the driver calls the lib; it does not carry its own copy of the note ----------------
/usr/bin/grep -q '_mc_pi_controller_note' "$X/run_once.sh" \
  && ok "T1b: extracted _mc_run_once calls _mc_pi_controller_note" \
  || bad "T1b: extracted _mc_run_once does not call _mc_pi_controller_note"
if /usr/bin/grep -qF -- "$P40" "$DRV"; then
  bad "T1b: the driver carries a copy of the note (its first 40 bytes appear in $DRV)"
else
  ok "T1b: no copy of the note's first 40 bytes in the driver"
fi

# --- T4: claude argv unchanged by the note -----------------------------------------------------
read_argv "$X/claude/argv.bin"
if [ "$(idx_of --append-system-prompt)" -eq -1 ] && ! any_has "$P40"; then
  ok "T4: claude argv has no --append-system-prompt and no note text"
else
  bad "T4: claude argv carries the note"
fi
cj=$(idx_of -p)
if [ "$cj" -ge 0 ] && [ "${ARGV[$((cj + 1))]:-}" = "$PROMPT_BODY" ]; then
  ok "T4: claude -p value equals \$PROMPT byte-for-byte"
else
  bad "T4: claude -p value is not \$PROMPT"
fi

# --- T5: codex argv unchanged by the note ------------------------------------------------------
read_argv "$X/codex/argv.bin"
n=${#ARGV[@]}
if [ "$n" -gt 0 ] && [ "${ARGV[$((n - 1))]}" = "$PROMPT_BODY" ] && ! any_has "$P40"; then
  ok "T5: codex last argv element equals \$PROMPT and no element carries the note"
else
  bad "T5: codex argv carries the note, or its last element is not \$PROMPT"
fi

# --- T6: note content and shape ----------------------------------------------------------------
note_from "$ROOT_DIR" > "$X/note.raw"
for needle in '`ailang messages list --unread`' 'bash' 'ailang_cli' 'ailang_run' \
              'AILANG_AGENT_POLICY' 'CLAUDE.md' 'session_protocol_ack'; do
  /usr/bin/grep -qF -- "$needle" "$X/note.raw" && ok "T6: note names $needle" || bad "T6: note does not name $needle"
done
cmd=$(sed -n 's/.*`\(ailang messages list[^`]*\)`.*/\1/p' "$X/note.raw")
case "$cmd" in
  '') bad "T6: inbox command not found in backticks" ;;
  *';'*|*'&'*|*'|'*) bad "T6: inbox command is chained: [$cmd]" ;;
  *) ok "T6: inbox command is one plain call: [$cmd]" ;;
esac
bytes=$(wc -c < "$X/note.raw" | tr -d ' ')
[ "$bytes" -le 900 ] && ok "T6: note is $bytes bytes (<= 900)" || bad "T6: note is $bytes bytes (> 900)"
LC_ALL=C /usr/bin/grep -q '[^ -~]' "$X/note.raw" && bad "T6: note is not pure ASCII" || ok "T6: note is pure ASCII"
[ -n "$(tail -c1 "$X/note.raw")" ] && ok "T6: note has no trailing newline" || bad "T6: note ends in a newline"

# --- T6b: positive control — the scratch rewording still satisfies T1-T3 -------------------------
SNOTE=$(note_from "$SCR")
check_pi_scratch() {
  local i pv
  read_argv "$X/pi-scratch/argv.bin"
  i=$(idx_of --append-system-prompt)
  [ "$i" -ge 0 ] && [ "${ARGV[$((i + 1))]:-}" = "$SNOTE" ] \
    && ok "T6b: scratch rewording flows through --append-system-prompt" \
    || bad "T6b: scratch rewording does not reach --append-system-prompt (test may hard-code its note)"
  i=$(idx_of -p); pv="${ARGV[$((i + 1))]:-}"
  [ "$i" -ge 0 ] && [ "$pv" = "$SNOTE"$'\n\n'"$PROMPT_BODY" ] \
    && ok "T6b: scratch rewording flows through -p" \
    || bad "T6b: scratch rewording does not reach -p"
}
check_pi_scratch

# --- T7: the role runner gets no note and keeps its stage marker -------------------------------
/usr/bin/grep -q 'pi-controller-note' "$ROOT_DIR/scripts/mission_pi_run.sh" \
  && bad "T7: scripts/mission_pi_run.sh references pi-controller-note" \
  || ok "T7: scripts/mission_pi_run.sh does not reference pi-controller-note"
/usr/bin/grep -q 'AILANG_MISSION_STAGE=1' "$ROOT_DIR/scripts/mission_pi_run.sh" \
  && ok "T7: scripts/mission_pi_run.sh still sets AILANG_MISSION_STAGE=1" \
  || bad "T7: scripts/mission_pi_run.sh no longer sets AILANG_MISSION_STAGE=1"

# --- T8: drift tripwire on what the note claims ------------------------------------------------
GATE="$ROOT_DIR/.pi/extensions/session-protocol-gate.ts"
EXEC="$ROOT_DIR/.pi/extensions/ailang-exec.ts"
for s in 'part.name === "bash"' 'cmd.includes("ailang messages")' '^ailang (check|messages'; do
  if /usr/bin/grep -qF -- "$s" "$GATE" 2>/dev/null; then
    ok "T8: $(basename "$GATE") still contains [$s]"
  else
    bad "T8: $GATE no longer contains [$s]; the note must be revisited"
  fi
done
# The refusal is built from a constant (POLICY_ENV), so the literal "AILANG_AGENT_POLICY is unset"
# is not in the source. Check the constant and the refusal template that uses it.
if /usr/bin/grep -qF -- 'POLICY_ENV = "AILANG_AGENT_POLICY"' "$EXEC" 2>/dev/null \
  && /usr/bin/grep -qF -- '(${POLICY_ENV} is unset)' "$EXEC" 2>/dev/null; then
  ok "T8: $(basename "$EXEC") still names AILANG_AGENT_POLICY as POLICY_ENV and refuses when it is unset"
else
  bad "T8: $EXEC no longer names AILANG_AGENT_POLICY as POLICY_ENV with an unset refusal; the note must be revisited"
fi

echo "pi-controller-note: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
