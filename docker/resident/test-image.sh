#!/bin/bash
# Acceptance tests for the resident image, run INSIDE it (v6.40.0 M1).
#
# Every assertion here corresponds to a trap that cost real time during the M0
# spike, or to a fail-closed guarantee the design depends on. Run by
# cloudbuild.yaml against the freshly built image.
set -uo pipefail
# Leave nothing running: a backgrounded health server or herdr would hold the
# CI step's stdout open and hang the build rather than failing it.
cleanup() { pkill -f "health.mjs" 2>/dev/null; pkill -f "herdr server" 2>/dev/null; }
trap cleanup EXIT
PASS=0; FAIL=0
ok()   { echo "  PASS: $*"; PASS=$((PASS+1)); }
bad()  { echo "  FAIL: $*"; FAIL=$((FAIL+1)); }
have() { if eval "$2"; then ok "$1"; else bad "$1"; fi; }

export AILANG_FS_SANDBOX=/workspace
export RESIDENT_AUDIENCE=https://test.invalid
export RESIDENT_ALLOWED_CALLERS=nobody@example.com
MODELS='{"providers":{"openrouter":{"baseUrl":"https://openrouter.ai/api/v1","api":"openai-completions","apiKey":"test-key","models":[{"id":"z-ai/glm-5.3-flash","maxTokens":32000,"contextWindow":1310720,"reasoning":true}]}}}'

echo "=== 1. image contents ==="
have "runs as uid 1000 (matches the gcsfuse uid=1000 mount)" '[ "$(id -u)" = "1000" ]'
have "herdr present"          'command -v herdr >/dev/null'
have "herdr is the pinned 0.8.2" 'herdr --version 2>&1 | grep -q "0.8.2"'
have "pi CLI present"          'command -v pi >/dev/null'
have "ailang binary present"   'command -v ailang >/dev/null'
have "node present"            'command -v node >/dev/null'

echo "=== 2. fail-closed guarantees ==="
# No registry => pi would silently run every model at 16384/128000/no-reasoning.
out=$(env -u MODELS_JSON RESIDENT_PORT=8091 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "refuses to start with no MODELS_JSON" 'echo "$out" | grep -q "MODELS_JSON is unset"'
have "  ...and says why (silent fallback named)" 'echo "$out" | grep -q "16384"'

out=$(MODELS_JSON="$MODELS" PI_HOME=/agent-home/.pi AGENT_HOME=/agent-home RESIDENT_PORT=8092 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "refuses PI_HOME inside AGENT_HOME (key on a GCS bucket)" 'echo "$out" | grep -q "Refusing to start"'

out=$(MODELS_JSON='{"providers":{}}' RESIDENT_PORT=8093 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "refuses an empty registry" 'echo "$out" | grep -q "declares no models"'

out=$(MODELS_JSON='not json' RESIDENT_PORT=8094 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "refuses invalid JSON" 'echo "$out" | grep -q "not valid JSON"'

# AILANG treats an unset sandbox as NO sandbox, so an unset variable must be
# refused rather than silently running effects unconfined.
out=$(env -u AILANG_FS_SANDBOX MODELS_JSON="$MODELS" RESIDENT_PORT=8095 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "refuses an unset AILANG_FS_SANDBOX"     'echo "$out" | grep -q "AILANG_FS_SANDBOX is unset"'
have "  ...and names the silent-unconfined risk" 'echo "$out" | grep -q "NO sandbox"'
have "ailang sandbox-check REJECTS an escape"  '! AILANG_FS_SANDBOX=/workspace ailang sandbox-check /etc/passwd >/dev/null 2>&1'
have "ailang sandbox-check ALLOWS inside root" 'AILANG_FS_SANDBOX=/workspace ailang sandbox-check /workspace/x.txt >/dev/null 2>&1'

echo "=== 3. happy path ==="
export MODELS_JSON="$MODELS" RESIDENT_PORT=8080 AGENT_HOME=/tmp/fake-home
mkdir -p /tmp/fake-home
/usr/local/bin/boot.sh > /tmp/boot.log 2>&1 &
BOOT=$!
# Readiness is /livez, the one unauthenticated route. The health BODY is behind
# auth now, so its facts are asserted from the boot log and the registry file
# instead — same facts, without weakening auth to observe them.
for i in $(seq 1 90); do [ "$(curl -s localhost:8080/livez 2>/dev/null)" = "ok" ] && break; sleep 1; done

have "livez answers once serving"         '[ "$(curl -s localhost:8080/livez)" = "ok" ]'
# herdr is OFF by default since it left the task path: starting it cost a
# process and ~memory for nothing. Assert the default rather than its presence.
have "herdr is not started by default"    'grep -q "herdr: not started" /tmp/boot.log'
have "pi extensions are off by default"   'grep -q "pi extensions: disabled" /tmp/boot.log'
have "boot reported the model registry"   'grep -q "model registry: 1 models" /tmp/boot.log'
have "registry has the pinned GLM model"  'grep -q "z-ai/glm-5.3-flash" /home/ailang/.pi/agent/models.json'
# Comments-stripped: boot.sh documents the setsid trap, and the comment
# explaining it must not itself trip the check for it.
have "boot does not USE setsid (comments excluded)" '! grep -vE "^[[:space:]]*#" /usr/local/bin/boot.sh | grep -q setsid'
have "registry written to local disk"     '[ -f /home/ailang/.pi/agent/models.json ]'
have "registry mode is 0600 (holds the key)" '[ "$(stat -c %a /home/ailang/.pi/agent/models.json)" = "600" ]'
have "readiness probed via api snapshot"  'grep -q "api snapshot" /usr/local/bin/boot.sh'
have "herdr socket at the explicit path"  '[ -S "${HERDR_SOCKET_PATH}" ]'
have "agent home probe cleaned up"        '[ ! -f /tmp/fake-home/.boot-probe ]'
have "boot reported the sandbox live"     'grep -q "effect sandbox live" /tmp/boot.log'

# The registry carries a placeholder rather than a second copy of the key.
PLACEHOLDER='{"providers":{"openrouter":{"apiKey":"${OPENROUTER_API_KEY}","models":[{"id":"z-ai/glm-5.3-flash","maxTokens":32000,"contextWindow":1310720}]}}}'
out=$(MODELS_JSON="$PLACEHOLDER" env -u OPENROUTER_API_KEY RESIDENT_PORT=8096 timeout 25 /usr/local/bin/boot.sh 2>&1)
have "placeholder without a key is refused"   'echo "$out" | grep -q "OPENROUTER_API_KEY is unset"'
have "  ...and names the consequence"         'echo "$out" | grep -q "literal placeholder"'

MODELS_JSON="$PLACEHOLDER" OPENROUTER_API_KEY='sk-t3st/w1th+specials&chars' RESIDENT_PORT=8097 timeout 30 /usr/local/bin/boot.sh >/tmp/sub.log 2>&1 &
sleep 12
have "key substituted into the registry"      'grep -q "sk-t3st/w1th+specials&chars" /home/ailang/.pi/agent/models.json'
have "  ...placeholder fully replaced"        '! grep -q "OPENROUTER_API_KEY" /home/ailang/.pi/agent/models.json'
have "  ...and boot said so"                  'grep -q "provider key substituted" /tmp/sub.log'
# Regression: awk/sed treat & in the replacement as the matched text, so a key
# containing & was silently turned back into the placeholder. The fixture key
# above contains & precisely to keep that fixed.
have "  ...& in the key survives verbatim"    'grep -q "specials&chars" /home/ailang/.pi/agent/models.json'
pkill -f "server.mjs" 2>/dev/null; pkill -f "herdr server" 2>/dev/null; sleep 2

echo "=== 4. program allowlist (Decision 6) ==="
# Default-deny: no manifest means nothing runs. An agent that can reason but not
# act is degraded; one that runs anything because a file was missing is an
# incident.
out=$(PROGRAM_ALLOWLIST_FILE=/nonexistent resident-run anything 2>&1); rc=$?
have "no manifest -> denies (default-deny)"   '[ "$rc" = "2" ]'
have "  ...and says why"                      'echo "$out" | grep -q "Default-deny"'

cat > /tmp/allow.json <<'JSON'
{"programs":{"ok":{"path":"/tmp/ok.ail","caps":["IO"]},"badcap":{"path":"/tmp/ok.ail","caps":["IO","Nope"]},"needsfs":{"path":"/tmp/ok.ail","caps":["FS"]}}}
JSON
out=$(PROGRAM_ALLOWLIST_FILE=/tmp/allow.json resident-run not-listed 2>&1); rc=$?
have "unlisted program refused"               '[ "$rc" = "2" ]'
have "  ...and lists what IS allowed"         'echo "$out" | grep -q "Allowed: ok"'

out=$(PROGRAM_ALLOWLIST_FILE=/tmp/allow.json resident-run badcap 2>&1)
have "unknown capability refused, not dropped" 'echo "$out" | grep -q "unknown capabilities: Nope"'

out=$(PROGRAM_ALLOWLIST_FILE=/tmp/allow.json env -u AILANG_FS_SANDBOX resident-run needsfs 2>&1)
have "FS entry refused when sandbox unset"    'echo "$out" | grep -q "NO sandbox"'

echo "" > /tmp/ok.ail
out=$(PROGRAM_ALLOWLIST_FILE=/tmp/allow.json AILANG_FS_SANDBOX=/workspace resident-run ok 2>&1)
have "allowed program passes ONLY its caps"   'echo "$out" | grep -q -- "--caps IO /tmp/ok.ail"'
have "  ...and does not grant the union"      '! echo "$out" | grep -qE -- "--caps [A-Za-z,]*FS"'

echo "=== 5. public-ingress authorisation (Preview edge does not enforce invoker) ==="
have "/livez is public and reveals nothing"     '[ "$(curl -s localhost:8080/livez)" = "ok" ]'
have "/health requires a token"                 '[ "$(curl -s -o /dev/null -w %{http_code} localhost:8080/health)" = "401" ]'
have "agent card requires a token"              '[ "$(curl -s -o /dev/null -w %{http_code} localhost:8080/.well-known/agent.json)" = "401" ]'
have "A2A JSON-RPC requires a token"            '[ "$(curl -s -o /dev/null -w %{http_code} -X POST localhost:8080/a2a -d "{}")" = "401" ]'
have "a garbage bearer token is refused"        '[ "$(curl -s -o /dev/null -w %{http_code} -H "Authorization: Bearer not.a.jwt" localhost:8080/health)" = "401" ]'
# An unsigned token with the right claims must NOT pass — signature is checked
# before any claim is trusted.
FORGED=$(node -e '
const h=Buffer.from(JSON.stringify({alg:"RS256",kid:"x"})).toString("base64url");
const p=Buffer.from(JSON.stringify({iss:"https://accounts.google.com",aud:"https://test.invalid",email:"nobody@example.com",email_verified:true,exp:Math.floor(Date.now()/1000)+3600})).toString("base64url");
console.log(h+"."+p+".AAAA");')
have "a FORGED token with correct claims is refused" '[ "$(curl -s -o /dev/null -w %{http_code} -H "Authorization: Bearer $FORGED" localhost:8080/health)" = "401" ]'
# What matters is that the refusal discloses nothing about WHY: a caller must
# not be able to distinguish "bad signature" from "wrong audience" from "not on
# the allowlist" and probe its way in.
curl -s -H "Authorization: Bearer $FORGED" localhost:8080/health > /tmp/r_forged.txt
curl -s -H "Authorization: Bearer not.a.jwt" localhost:8080/health > /tmp/r_junk.txt
curl -s localhost:8080/health > /tmp/r_none.txt
have "refusal body is identical for every failure" 'diff -q /tmp/r_forged.txt /tmp/r_junk.txt >/dev/null && diff -q /tmp/r_junk.txt /tmp/r_none.txt >/dev/null'
have "  ...and leaks no reason"                    '! grep -qiE "signature|audience|allowlist|expired|verified" /tmp/r_forged.txt'

echo "=== 6. A2A surface (Decision 2c) ==="
# Driven against the modules directly. The HTTP routes now require a verified
# Google ID token, which cannot be minted inside the test container — and
# weakening auth to make tests pass would defeat the point of having it.
cd /usr/local/bin
A2A() { node --input-type=module -e "$1" 2>&1; }

out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
try { a2a.assertModelRegistered("z-ai/not-registered"); console.log("NO THROW"); }
catch (e) { console.log(e.message); }')
have "unregistered model REFUSED"              'echo "$out" | grep -q "not in the pi registry"'
have "  ...and names the silent fallback"      'echo "$out" | grep -q "16384"'
have "  ...and lists what IS registered"       'echo "$out" | grep -q "z-ai/glm-5.3-flash"'

out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
const m = a2a.assertModelRegistered("openrouter/z-ai/glm-5.3-flash");
console.log(JSON.stringify(m));')
have "registered model accepted with its limits" 'echo "$out" | grep -q "1310720"'

out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
console.log(JSON.stringify(a2a.agentCard("https://example.invalid")));')
have "agent card advertises /a2a"              'echo "$out" | grep -q "https://example.invalid/a2a"'
have "card declares pushNotifications"         'echo "$out" | grep -q "\"pushNotifications\":true"'
have "card declares a skill"                   'echo "$out" | grep -q "coding-agent"'
have "card lists the registered model"         'echo "$out" | grep -q "z-ai/glm-5.3-flash"'

out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
a2a.setPushConfig("t1", {url:"https://hook.invalid", token:"x"});
console.log(JSON.stringify(a2a.getPushConfig("t1")));')
have "push notification config round-trips"    'echo "$out" | grep -q "hook.invalid"'

# The bespoke API this design deliberately does not ship. Asserted against the
# source now that every route is behind auth.
have "NO bespoke /panes route in the server"   '! grep -qE "\"/panes\"|/panes/" /usr/local/bin/server.mjs'
have "A2A JSON-RPC route is present"           'grep -q "/a2a" /usr/local/bin/server.mjs'

echo "=== 6b. the executor runs pi HEADLESS ==="
# THE 2026-09-03 BUG. pi spawned, produced no NDJSON, never exited and never
# errored, so the A2A task sat at `submitted` and the instance looked healthy.
# Cause: Node's spawn defaults stdin to an open pipe, where Go's exec.Cmd (which
# internal/executor/pi/pi.go relies on) leaves it nil and gets /dev/null. These
# three assertions are the regression guard, cheapest first.
have "pi answers --version headless with stdin closed" \
     'timeout 20 pi --version </dev/null >/dev/null 2>&1'
have "the executor gives pi /dev/null on stdin, not a pipe" \
     'grep -q "stdio: \[\"ignore\"" /usr/local/bin/lib/pi.mjs'
have "boot proves pi runs headless"        'grep -q "pi headless check" /tmp/boot.log'

# A hang must become a NAMED failure quickly, not a 15-minute silence. Proven
# against a stub `pi` that never speaks: without the TTFT timer this assertion
# hangs the build, which is precisely the behaviour being guarded.
STUB=$(mktemp -d)
printf '#!/bin/sh\nsleep 300\n' > "$STUB/pi"; chmod +x "$STUB/pi"
out=$(PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", ttftMs: 2000 })
  .then(() => console.log("RESOLVED"))
  .catch((e) => console.log("REJECTED: " + e.message));' 2>&1)
have "a silent pi fails fast on the TTFT timer" 'echo "$out" | grep -q "no output within"'
rm -rf "$STUB"

echo "=== 7. restart idempotence ==="
# The 7-day ceiling makes restarts routine, so a second boot must behave like
# the first rather than tripping over its own leftovers.
kill $BOOT 2>/dev/null; pkill -f "herdr server" 2>/dev/null; sleep 3
/usr/local/bin/boot.sh > /tmp/boot2.log 2>&1 &
for i in $(seq 1 90); do [ "$(curl -s localhost:8080/livez 2>/dev/null)" = "ok" ] && break; sleep 1; done
have "second boot serves again"           '[ "$(curl -s localhost:8080/livez)" = "ok" ]'
have "second boot found the home writable" 'grep -q "agent home writable" /tmp/boot2.log'

cleanup; sleep 1
echo
echo "=== $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
