#!/bin/bash
# Acceptance tests for the resident image, run INSIDE it (v6.40.0 M1).
#
# Every assertion here corresponds to a trap that cost real time during the M0
# spike, or to a fail-closed guarantee the design depends on.
#
# Run against the freshly built image by three pipelines: this directory's own
# docker/resident/cloudbuild.yaml (standalone manual build), and the repo-root
# cloudbuild-dev.yaml (push -> dev) and cloudbuild-release.yaml (v* tag -> test).
# In dev the step carries allowFailure; in the release pipeline it is a hard gate.
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
# /livez answers from boot section 2, long before boot FINISHES — it is bound
# early on purpose so a slow start cannot be killed by the startup probe. So
# assertions about the boot log must wait for the end-of-boot marker, or they
# race the very steps they are checking.
for i in $(seq 1 60); do grep -q "resident agent ready" /tmp/boot.log && break; sleep 1; done

have "livez answers once serving"         '[ "$(curl -s localhost:8080/livez)" = "ok" ]'
have "boot ran to completion"             'grep -q "resident agent ready" /tmp/boot.log'
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
# The herdr socket assertion that stood here is retired: herdr is no longer
# started by default (D2), so a socket at the explicit path would mean
# something had gone wrong rather than right. HERDR_SOCKET_PATH still matters
# when herdr IS enabled, so the env is asserted instead of the socket.
have "herdr socket path is set explicitly, not inherited from HOME" \
     '[ "$HERDR_SOCKET_PATH" = "/home/ailang/.herdr/herdr.sock" ]'
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

echo "=== 6c. session persistence (M10) ==="
# The design's whole premise. Until now the host persisted and the agent did
# not, and nothing in this suite could tell the difference.
have "boot probed pi for --session-id"      'grep -qE "session persistence: (ENABLED|DISABLED)" /tmp/boot.log'
have "capabilities written for the server to read" '[ -f /home/ailang/.resident/capabilities.json ]'
have "the pinned pi supports --session-id"  'grep -q "session persistence: ENABLED" /tmp/boot.log'

# The executor must actually USE it. A stub pi records its argv, so this
# asserts the command line rather than trusting the code to mean what it says.
STUB=$(mktemp -d)
printf '#!/bin/sh\necho "$@" > /tmp/pi-argv.txt\n' > "$STUB/pi"; chmod +x "$STUB/pi"
PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", sessionId: "ctx-test", ttftMs: 5000 }).catch(() => {});
' >/dev/null 2>&1
have "a session run passes --session-id"    'grep -q -- "--session-id ctx-test" /tmp/pi-argv.txt'
have "  ...and drops --no-session"          '! grep -q -- "--no-session" /tmp/pi-argv.txt'
# THE CORRUPTION TRAP: gcsfuse has no POSIX locking and pi rewrites the session
# file all through a run, so --session-dir must point at LOCAL disk. Staging to
# the mount happens after the run, not during it.
have "sessions live on local disk, not the GCS mount" \
     '! grep -qE -- "--session-dir[ =]*/agent-home" /tmp/pi-argv.txt'

PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", ttftMs: 5000 }).catch(() => {});
' >/dev/null 2>&1
have "a run with no session id stays ephemeral" 'grep -q -- "--no-session" /tmp/pi-argv.txt'
rm -rf "$STUB" /tmp/pi-argv.txt

out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
console.log(JSON.stringify(a2a.agentCard("https://example.invalid")));')
# The A2A AgentCard schema has NO metadata field, and the a2a-sdk drops what it
# does not know — so anything published there is invisible to every compliant
# client. These assert the SPEC-LEGAL home instead.
have "card extras live in capabilities.extensions" 'echo "$out" | grep -q "\"extensions\""'
have "  ...advertising the model registry"        'echo "$out" | grep -q "resident-registry/v1"'
have "  ...and how to hold a conversation"        'echo "$out" | grep -q "resident-conversation/v1"'
have "the card has NO non-spec metadata field"    '! echo "$out" | grep -q "\"metadata\""'
have "the conversation key is named"              'echo "$out" | grep -q "contextId"'

# One registered model means nothing to choose, so a caller need not name it.
out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
a2a.messageSend({ message: { role: "user", kind: "message", messageId: "m1",
  parts: [{ kind: "text", text: "hi" }] } })
  .then((t) => console.log("MODEL=" + t.metadata.model))
  .catch((e) => console.log("THREW: " + e.message));')
have "a sole registered model is used when none is requested" 'echo "$out" | grep -q "MODEL=openrouter/z-ai/glm-5.3-flash"'

echo "=== 6d. tool policy (D8) ==="
# pi enables read/bash/edit/write by default and said so nowhere. The point of
# these assertions is that the set is EXPLICIT: what an agent can do must be
# readable from the command line and from /health, not inferred from pi's docs.
STUB=$(mktemp -d)
printf '#!/bin/sh\necho "$@" > /tmp/pi-argv.txt\n' > "$STUB/pi"; chmod +x "$STUB/pi"
PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", ttftMs: 5000 }).catch(() => {});' >/dev/null 2>&1
have "the tool set is always passed explicitly" 'grep -q -- "--tools" /tmp/pi-argv.txt'

RESIDENT_TOOLS="read" PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", ttftMs: 5000 }).catch(() => {});' >/dev/null 2>&1
have "RESIDENT_TOOLS narrows the policy"     'grep -q -- "--tools read" /tmp/pi-argv.txt'
have "  ...and bash is then absent"          '! grep -q -- "bash" /tmp/pi-argv.txt'

PATH="$STUB:$PATH" timeout 30 node --input-type=module -e '
import { runPi } from "/usr/local/bin/lib/pi.mjs";
runPi({ model: "m", prompt: "hi", tools: [], ttftMs: 5000 }).catch(() => {});' >/dev/null 2>&1
have "an empty policy disables tools entirely" 'grep -q -- "--no-tools" /tmp/pi-argv.txt'
rm -rf "$STUB" /tmp/pi-argv.txt

echo "=== 6e. concurrency ceiling ==="
# Instances are SINGLETONS and do not autoscale, so the box has a fixed
# ceiling. The only choice is whether it is hit politely or by OOM — M0 saw the
# cgroup kill the child while the container survived, i.e. one caller silently
# killing another's run.
out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
console.log(JSON.stringify(a2a.runStats()));')
have "the agent reports its run ceiling"   'echo "$out" | grep -q "\"max\""'
have "  ...and starts idle"                'echo "$out" | grep -q "\"active\":0"'

out=$(RESIDENT_MAX_CONCURRENT_RUNS=0 A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
a2a.messageSend({ message: { role:"user", kind:"message", messageId:"m1",
  parts:[{kind:"text",text:"hi"}] } })
  .then(() => console.log("ACCEPTED"))
  .catch((e) => console.log("REFUSED: " + e.message));')
have "a full agent refuses rather than OOMs"    'echo "$out" | grep -q "REFUSED"'
have "  ...and the refusal names the ceiling"   'echo "$out" | grep -q "maximum 0 concurrent"'
have "  ...and says instances do not autoscale" 'echo "$out" | grep -q "do not autoscale"'

echo "=== 6f. keyless providers (design P6) ==="
# A registry whose providers authenticate by ambient identity must start with
# NO key in the container. The shared-key path is what makes one hostile prompt
# in any user's thread everyone's problem.
VERTEX_REG='{"providers":{"vertex":{"api":"google-vertex","models":[{"id":"gemini-3.8-flash","maxTokens":65536,"contextWindow":1048576,"reasoning":true}]}}}'
out=$(MODELS_JSON="$VERTEX_REG" GOOGLE_CLOUD_PROJECT=test-project RESIDENT_PORT=8094 \
      timeout 40 /usr/local/bin/boot.sh 2>&1)
have "a keyless registry boots"              'echo "$out" | grep -q "model registry: 1 models"'
have "  ...and says no keys are present"     'echo "$out" | grep -q "provider keys: NONE"'
have "  ...and configures vertex ADC"        'echo "$out" | grep -q "vertex: ADC as"'
have "  ...defaulting to the global endpoint" 'echo "$out" | grep -q "location=global"'

# Fail closed: a vertex provider with no project fails EVERY call at call time,
# which reads as a model fault rather than a boot misconfiguration.
out=$(MODELS_JSON="$VERTEX_REG" RESIDENT_PORT=8095 timeout 40 env -u GOOGLE_CLOUD_PROJECT \
      /usr/local/bin/boot.sh 2>&1)
have "vertex without a project refuses to start" 'echo "$out" | grep -q "GOOGLE_CLOUD_PROJECT is unset"'
have "  ...and names the consequence"            'echo "$out" | grep -q "looking like a model fault"'

echo "=== 6g. idleness for the sweep (M4) ==="
# Instances do not autoscale, do not stop themselves, and a stopped one does
# NOT wake on a request — its URL 404s until something calls :start (probed
# 2026-09-03). So idle time is billed in full until a sweep acts, and the sweep
# can only act on a number the agent reports.
out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
console.log(JSON.stringify(a2a.runStats()));')
have "the agent reports its idle seconds"   'echo "$out" | grep -q "idle_s"'

# The number must count WORK, not traffic. A sweep that counted health probes
# would never stop anything, because the sweep's own probe is traffic.
out=$(A2A 'import * as a2a from "/usr/local/bin/lib/a2a.mjs";
const before = a2a.runStats().idle_s;
a2a.noteActivity();
console.log(JSON.stringify({before, after: a2a.runStats().idle_s}));')
have "activity resets idleness"             'echo "$out" | grep -q "\"after\":0"'

echo "=== 6h. observability (M8) ==="
# The tracer's own suite, run against the artefact that will be deployed rather
# than against a checkout that may have moved on — same reason test-image.sh
# ships inside the image at all.
out=$(RESIDENT_LIB=/usr/local/bin/lib node --test /usr/local/bin/test-otel.mjs /usr/local/bin/test-observatory.mjs /usr/local/bin/test-a2a-persistence.mjs /usr/local/bin/test-pi-staging.mjs 2>&1)
have "telemetry suites pass in the image"   'echo "$out" | grep -q "fail 0"'

# DURABILITY OF THE CONVERSATION, not of the task store — a separate mount
# write that M6's fsync fix did not cover. The node suite above proves the
# copy is faithful and CANNOT prove the fsync: a tmpdir has no upload gap,
# which is exactly why every simulated restart passed before M6. So the
# fsync is asserted here as a property of the shipped source, and as an
# OBJECT in GCS by verify-resident-chaos.sh. A plain cpSync/writeFileSync
# returns once the bytes reach the FUSE buffer, and the object may never
# exist; the symptom is a resident that forgets a conversation after an
# idle stop, which reads as a model problem rather than a storage one.
have "staged sessions are fsynced to the mount" \
     'grep -q "fsyncSync" /usr/local/bin/lib/pi.mjs'
have "staging does not fall back to a plain cpSync" \
     '! grep -q "cpSync" /usr/local/bin/lib/pi.mjs'

# TWO planes, and only the second is what an operator reads. A resident that
# emitted spans and no session would be "traced" and still absent from the view
# that shows agent jobs and Claude Code sessions — which is the complaint M8
# was written to fix.
have "runs are reported as observatory sessions" 'grep -q "observatory.startRun" /usr/local/bin/lib/a2a.mjs'
have "pi events are forwarded, not re-derived"   'grep -q "reported.event(ev)" /usr/local/bin/lib/a2a.mjs'
have "a failed run still closes its session"     'grep -q "reported.finish({ ok: false" /usr/local/bin/lib/a2a.mjs'
have "all THREE observatory planes are posted"   'grep -q "api/observatory/hooks" /usr/local/bin/lib/observatory.mjs && grep -q "api/exec/sessions" /usr/local/bin/lib/observatory.mjs && grep -q "api/exec/events" /usr/local/bin/lib/observatory.mjs'
have "SessionStart always carries a workspace"   'grep -q "event: \"SessionStart\", workspace" /usr/local/bin/lib/observatory.mjs'
have "the observatory URL is read"               'grep -q "AILANG_OBSERVATORY_URL" /usr/local/bin/server.mjs'

# Wiring, asserted on the source: a tracer nothing calls is the failure this
# milestone is fixing, one layer up.
have "the A2A dispatch is traced"           'grep -q "otel.startSpan(\`a2a." /usr/local/bin/server.mjs'
have "the server configures a tracer"       'grep -q "OTEL_EXPORTER_OTLP_ENDPOINT" /usr/local/bin/server.mjs'
have "task state changes emit spans"        'grep -q "a2a.task.\${state}" /usr/local/bin/lib/a2a.mjs'
have "the caller's trace is continued"      'grep -q "req.headers.traceparent" /usr/local/bin/server.mjs'

# The failure that matters more than any of the above: telemetry must never be
# the reason a turn fails. Boot with an endpoint that cannot possibly answer
# and assert the agent still serves.
out=$(A2A 'process.env.OTEL_EXPORTER_OTLP_ENDPOINT = "http://127.0.0.1:1";
import * as otel from "/usr/local/bin/lib/otel.mjs";
otel.configure({ endpoint: "http://127.0.0.1:1", serviceName: "t" });
const s = otel.startSpan("a2a.message/send"); s.setAttribute("a2a.task.id","t"); s.end();
await otel.flush();
console.log("SURVIVED");')
have "an unreachable collector does not throw" 'echo "$out" | grep -q "SURVIVED"'

# And the boot log says which state it is in, so "invisible in the observatory"
# is noticed rather than discovered later.
have "boot states its trace posture"        'grep -q "observability: .*trace" /tmp/boot.log'
have "boot states its session posture"      'grep -q "observability: .*\(sessions\|observatory\)" /tmp/boot.log'

echo "=== 7. restart idempotence ==="
# The 7-day ceiling makes restarts routine, so a second boot must behave like
# the first rather than tripping over its own leftovers.
kill $BOOT 2>/dev/null; pkill -f "herdr server" 2>/dev/null; sleep 3
/usr/local/bin/boot.sh > /tmp/boot2.log 2>&1 &
for i in $(seq 1 90); do [ "$(curl -s localhost:8080/livez 2>/dev/null)" = "ok" ] && break; sleep 1; done
have "second boot serves again"           '[ "$(curl -s localhost:8080/livez)" = "ok" ]'
have "second boot found the home writable" 'grep -q "agent home writable" /tmp/boot2.log'

echo "=== 7b. surviving a STOP, not just a restart (M6) ==="
# A restart keeps the writable layer; the idle sweep's stop/resume does not.
# That distinction is the whole of M6, and it is only testable by destroying
# the local layer the way Cloud Run does and booting onto the mount alone.
# This boot gets its OWN PORT rather than competing for 8080, because the
# contention is not worth fighting and twice cost a false negative: `$BOOT`
# still names the FIRST boot here (section 7 restarts without recapturing it),
# so killing it left section 7's server holding the port, the third boot never
# bound, and the two assertions below reported as PRODUCT failures on code the
# node suite passed. Killing is still attempted, but nothing depends on it.
pkill -f "server.mjs" 2>/dev/null; pkill -f "boot.sh" 2>/dev/null
pkill -f "herdr server" 2>/dev/null
# WAIT for it to be gone before seeding. pkill is asynchronous and the server's
# own SIGTERM handler checkpoints on the way out, so seeding straight after the
# signal races that write — and the loser is the fixture this section depends on.
for i in $(seq 1 20); do pgrep -f "server.mjs" >/dev/null 2>&1 || break; sleep 1; done
M6_PORT=8081
mkdir -p "$AGENT_HOME/.resident"
cat > "$AGENT_HOME/.resident/tasks.json" <<'JSON'
{"v":2,
 "tasks":[{"kind":"task","id":"task-stopped","contextId":"ctx-stop",
           "status":{"state":"working","timestamp":"2026-09-04T12:00:00.000Z"},
           "history":[],"metadata":{"model":"z-ai/glm-5.3-flash"}}],
 "push":[["task-stopped",{"url":"http://127.0.0.1:1/gone","token":"t"}]]}
JSON
rm -rf /home/ailang/.resident/tasks.json          # the stop resets the writable layer
RESIDENT_PORT=$M6_PORT /usr/local/bin/boot.sh > /tmp/boot3.log 2>&1 &
BOOT=$!
for i in $(seq 1 90); do [ "$(curl -s localhost:$M6_PORT/livez 2>/dev/null)" = "ok" ] && break; sleep 1; done
# A boot that never bound proves nothing either way, so say WHICH it is rather
# than letting the two greps below fail for a reason that is not the product's.
have "the M6 boot came up on $M6_PORT" '[ "$(curl -s localhost:'"$M6_PORT"'/livez)" = "ok" ]'
have "a stop/resume restores from the mount"  'grep -q "restored 1 task(s) from checkpoint" /tmp/boot3.log'
# The silent hang M6 closes: without this the caller was told an answer would
# arrive, the poll had already given up, and nothing would ever speak again.
have "an interrupted run is terminalised"     'grep -q "reaped 1 task(s) interrupted by a restart" /tmp/boot3.log'
# The webhook in the seeded task points at a dead port. Reaping must survive
# that: the terminal state is not best-effort, only the notice is.
have "  ...and an unreachable webhook is survived" '[ "$(curl -s localhost:'"$M6_PORT"'/livez)" = "ok" ]'
grep -q "restored 1 task(s) from checkpoint" /tmp/boot3.log || { echo "  --- boot3.log ---"; sed 's/^/    /' /tmp/boot3.log | tail -25; }

cleanup; sleep 1
echo
echo "=== $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
