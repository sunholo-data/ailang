#!/bin/bash
# M-AGY-CLOUD Phase 0 spike — runs INSIDE a Linux container (Cloud Run job), never on
# the rig. design_docs/planned/m-agy-cloud-executor.md, P1/P2/P3/P5/P6/P7.
#
# Reads the cloud-only agy sign-in from Secret Manager into an isolated HOME, measures,
# and writes the token file back as a new secret version if agy rewrote it (the design's
# write-back contract, so the spike never leaves the secret stale). Prints hashes of the
# token, never its values.
#
# Env: SPIKE_PROJECT, SPIKE_SECRET (required); SPIKE_MODEL (default gemini-4-neon-low).
set -uo pipefail

: "${SPIKE_PROJECT:?}" "${SPIKE_SECRET:?}"
MODEL="${SPIKE_MODEL:-gemini-4-neon-low}"
AGY_URL="https://storage.googleapis.com/antigravity-public/antigravity-cli/1.5.15-5514532022517760/linux-x64/cli_linux_x64.tar.gz"
AGY_SHA512="af3f0be58b8475940d79f9abea19e762ecc3d265ea9f56cff8dd805e27ba299b44a305a79504a7ffe6ebdfa29a294c56edc4caed1875990bfaf2fef1508ec8b0"

say() { printf '\n=== %s\n' "$*"; }
# The slim image has no Python and no jq: extract with grep/sed (agy writes the token
# file as JSON with "refresh_token":"..."), hash, never print.
rt_hash() { local v; v=$(grep -o '"refresh_token": *"[^"]*"' "$1" 2>/dev/null | sed 's/.*: *"//; s/"$//'); [ -n "$v" ] && printf '%s' "$v" | sha256sum | cut -c1-12 || echo none; }
expiry_of() { grep -o '"expiry": *"[^"]*"' "$1" 2>/dev/null | sed 's/.*: *"//; s/"$//'; }
# summarize_stream <ndjson>: event counts and the result line, without a JSON parser.
summarize_stream() {
  echo "events: $(grep -o '"event": *"[a-z_]*"' "$1" | sed 's/.*"\([a-z_]*\)"$/\1/' | sort | uniq -c | tr '\n' ' ')"
  echo "result: $(grep '"event": *"result"' "$1" | grep -o '"status": *"[A-Z_]*"\|"response": *"[^"]\{0,80\}\|"error": *"[^"]\{0,160\}\|"denied_actions": *\[[^]]*\]' | tr '\n' ' ')"
}
file_hash() { sha256sum "$1" 2>/dev/null | cut -c1-12 || echo none; }

say "env"
uname -a; . /etc/os-release && echo "$PRETTY_NAME"; id

say "install agy 1.5.15 (pinned, sha512-verified)"
BIN=/tmp/agybin; mkdir -p "$BIN"
curl -fsSL --retry 2 -o /tmp/agy.tgz "$AGY_URL" || { echo "download failed"; exit 1; }
got=$(sha512sum /tmp/agy.tgz | awk '{print $1}')
[ "$got" = "$AGY_SHA512" ] || { echo "SHA512 MISMATCH: $got"; exit 1; }
tar -xzf /tmp/agy.tgz -C "$BIN" antigravity && mv "$BIN/antigravity" "$BIN/agy" && chmod +x "$BIN/agy"
export AGY_CLI_DISABLE_AUTO_UPDATE=true
"$BIN/agy" --version </dev/null

H=/tmp/agyhome; mkdir -p "$H/.gemini/antigravity-cli"; chmod 700 "$H"
TOK="$H/.gemini/antigravity-cli/antigravity-oauth-token"
echo '{}' > "$H/.gemini/antigravity-cli/settings.json"
# run_agy <seconds> <args...>: isolated env, no stdin, hard deadline (124 = it hung).
run_agy() { local t="$1"; shift; timeout "$t" env -i PATH=/usr/bin:/bin HOME="$H" AGY_CLI_DISABLE_AUTO_UPDATE=true "$BIN/agy" "$@" </dev/null; }

say "load token from secret (P2)"
if gcloud secrets versions access latest --secret="$SPIKE_SECRET" --project="$SPIKE_PROJECT" > "$TOK" 2>/tmp/sm.err; then
  chmod 600 "$TOK"; echo "token loaded: file=$(file_hash "$TOK") refresh=$(rt_hash "$TOK") expiry=$(expiry_of "$TOK")"
  HAVE_TOKEN=1
else
  echo "no token version yet ($(tail -1 /tmp/sm.err)); running the unauthenticated checks only"
  rm -f "$TOK"; HAVE_TOKEN=0
fi
BASE_FILE=$(file_hash "$TOK"); BASE_RT=$(rt_hash "$TOK")

say "P7 /quota"
run_agy 60 --print-timeout 30s -p /quota > /tmp/p7.out 2>&1; echo "rc=$?"; head -8 /tmp/p7.out

say "P1 headless -p stream-json (no hang; schema)"
W=/tmp/ws1; mkdir -p "$W"; cd "$W"
start=$(date +%s)
run_agy 150 --model "$MODEL" --output-format stream-json --print-timeout 120s --disable-slash-commands -p 'reply with exactly: ok' > /tmp/p1.ndjson 2>/tmp/p1.err
rc=$?; echo "rc=$rc elapsed=$(( $(date +%s) - start ))s (124 = our timeout fired = HANG)"
summarize_stream /tmp/p1.ndjson
tail -3 /tmp/p1.err

say "P5 --sandbox + shell command inside Cloud Run"
W=/tmp/ws2; mkdir -p "$W"; cd "$W"
cat > "$H/.gemini/antigravity-cli/settings.json" <<JSON
{"permissions":{"allow":["command(*)","write_file(/tmp/agyhome/tmp)"],"deny":["command(git push)","command(rm -rf)","command(sudo)","command(curl)","command(wget)"]}}
JSON
run_agy 200 --model "$MODEL" --mode accept-edits --sandbox --output-format stream-json --print-timeout 180s --disable-slash-commands -p 'Use the shell to run exactly: echo spike-ok > out.txt && cat out.txt. Then reply done.' > /tmp/p5.ndjson 2>/tmp/p5.err
echo "rc=$? out.txt=$(cat "$W/out.txt" 2>/dev/null || echo MISSING)"
grep -o 'sandbox configuration error[^"]*' /tmp/p5.ndjson | head -2 || true
summarize_stream /tmp/p5.ndjson
echo "tool steps: $(grep -c '"step_type": *"tool"' /tmp/p5.ndjson) | commands: $(grep -o '"CommandLine": *"[^"]\{0,100\}' /tmp/p5.ndjson | sort -u | tr '\n' ' ')"
echo "tool outputs: $(grep -o '"output": *"[^"]\{0,120\}' /tmp/p5.ndjson | sort -u | head -4 | tr '\n' ' ')"
tail -3 /tmp/p5.err

say "P5b negative control: a write OUTSIDE the workspace and the granted paths"
W=/tmp/ws3; mkdir -p "$W"; cd "$W"; rm -f /var/tmp/outside.txt
run_agy 200 --model "$MODEL" --mode accept-edits --sandbox --output-format stream-json --print-timeout 180s --disable-slash-commands -p 'Use the shell to run exactly: echo escaped > /var/tmp/outside.txt ; echo rc=$? Then reply with the exact command output.' > /tmp/p5b.ndjson 2>/tmp/p5b.err
echo "rc=$? /var/tmp/outside.txt=$(cat /var/tmp/outside.txt 2>/dev/null || echo ABSENT) (ABSENT = sandbox enforced)"
echo "tool outputs: $(grep -o '"output": *"[^"]\{0,200\}' /tmp/p5b.ndjson | sort -u | head -3 | tr '\n' ' ')"
summarize_stream /tmp/p5b.ndjson

say "P6 leftover processes after agy exited"
# Only processes whose executable is the agy binary or something it spawned under its
# HOME; pid 1 (this script) and the scanner itself are excluded.
for p in /proc/[0-9]*; do pid=${p#/proc/}; { [ "$pid" = 1 ] || [ "$pid" = "$$" ] || [ "$pid" = "$BASHPID" ]; } && continue
  exe=$(readlink "$p/exe" 2>/dev/null); c=$(tr '\0' ' ' < "$p/cmdline" 2>/dev/null)
  case "$exe $c" in *"$BIN"*|*"$H"*|*language_server*) echo "LEFTOVER pid=$pid exe=$exe: ${c:0:160}";; esac
done; echo "(scan done)"

say "P3 token after the runs"
if [ "$HAVE_TOKEN" = 1 ]; then
  NOW_FILE=$(file_hash "$TOK"); NOW_RT=$(rt_hash "$TOK")
  echo "file: before=$BASE_FILE after=$NOW_FILE | refresh_token: before=$BASE_RT after=$NOW_RT"
  if [ "$NOW_FILE" != "$BASE_FILE" ]; then
    gcloud secrets versions add "$SPIKE_SECRET" --project="$SPIKE_PROJECT" --data-file="$TOK" --format='value(name)' | sed 's/^/written back: /'
  else
    echo "token file unchanged; no write-back"
  fi
fi
say "spike done"
