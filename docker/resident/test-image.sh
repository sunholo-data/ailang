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
for i in $(seq 1 90); do curl -sf localhost:8080/health >/tmp/health.json 2>/dev/null && break; sleep 1; done

have "health endpoint returns 200"        'curl -sf localhost:8080/health >/dev/null'
have "reports healthy"                    'grep -q "\"healthy\": true" /tmp/health.json'
have "herdr reported ok (probed, not assumed)" 'grep -q "\"ok\": true" /tmp/health.json'
have "herdr protocol reported"            'grep -q "\"protocol\"" /tmp/health.json'
have "registry has the pinned GLM model"  'grep -q "z-ai/glm-5.3-flash" /tmp/health.json'
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

echo "=== 5. A2A surface (Decision 2c) ==="
RPC() { curl -s -X POST localhost:8080/a2a -H 'content-type: application/json' -d "$1"; }
curl -s localhost:8080/.well-known/agent.json > /tmp/card.json 2>/dev/null

have "agent card served at /.well-known/agent.json" '[ -s /tmp/card.json ]'
have "card advertises the A2A invocation url"       'grep -q "\"url\".*\/a2a" /tmp/card.json'
have "card declares pushNotifications capability"   'grep -q "\"pushNotifications\": true" /tmp/card.json'
have "card declares a skill"                        'grep -q "\"skills\"" /tmp/card.json'
have "card lists the registered model"              'grep -q "z-ai/glm-5.3-flash" /tmp/card.json'
have "card also served at agent-card.json"          'curl -sf localhost:8080/.well-known/agent-card.json >/dev/null'

# The bespoke API this design deliberately does NOT ship.
have "NO bespoke /panes endpoint exists" '[ "$(curl -s -o /dev/null -w %{http_code} -X POST localhost:8080/panes)" = "404" ]'

RPC '{"jsonrpc":"2.0","id":1,"method":"nope/nope","params":{}}' > /tmp/r1.json
have "unknown method -> JSON-RPC -32601"  'grep -q -- "-32601" /tmp/r1.json'
RPC '{"id":2,"method":"message/send"}' > /tmp/r2.json
have "non-2.0 request -> -32600"          'grep -q -- "-32600" /tmp/r2.json'

# THE assertion that closes the silent-degradation hole: an unregistered model
# must be refused, because pi would otherwise run it at 16384/128000/no-reasoning.
RPC '{"jsonrpc":"2.0","id":3,"method":"message/send","params":{"metadata":{"model":"z-ai/not-registered"},"message":{"role":"user","messageId":"m1","kind":"message","parts":[{"kind":"text","text":"hi"}]}}}' > /tmp/r3.json
have "unregistered model REFUSED"                  'grep -q "not in the pi registry" /tmp/r3.json'
have "  ...and names the silent fallback"          'grep -q "16384" /tmp/r3.json'
have "  ...and lists what IS registered"           'grep -q "z-ai/glm-5.3-flash" /tmp/r3.json'

# push notification config round-trip (A2A's answer for disconnected clients)
RPC '{"jsonrpc":"2.0","id":4,"method":"tasks/pushNotificationConfig/set","params":{"taskId":"t-demo","pushNotificationConfig":{"url":"http://localhost:9/hook","token":"tok"}}}' >/dev/null
RPC '{"jsonrpc":"2.0","id":5,"method":"tasks/pushNotificationConfig/get","params":{"taskId":"t-demo"}}' > /tmp/r5.json
have "push notification config round-trips"        'grep -q "localhost:9/hook" /tmp/r5.json'

RPC '{"jsonrpc":"2.0","id":6,"method":"tasks/get","params":{"id":"nope"}}' > /tmp/r6.json
have "unknown task -> visible error, not empty ok"  'grep -q -- "-32001" /tmp/r6.json'

echo "=== 6. restart idempotence ==="
# The 7-day ceiling makes restarts routine, so a second boot must behave like
# the first rather than tripping over its own leftovers.
kill $BOOT 2>/dev/null; pkill -f "herdr server" 2>/dev/null; sleep 3
/usr/local/bin/boot.sh > /tmp/boot2.log 2>&1 &
for i in $(seq 1 90); do curl -sf localhost:8080/health >/tmp/health2.json 2>/dev/null && break; sleep 1; done
have "second boot reaches healthy"        'grep -q "\"healthy\": true" /tmp/health2.json'
have "second boot found the home writable" 'grep -q "agent home writable" /tmp/boot2.log'

cleanup; sleep 1
echo
echo "=== $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
