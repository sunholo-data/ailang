#!/usr/bin/env bash
# Acceptance suite for the agent-pi image (M-PI-HARNESS-UPGRADE M4). Runs INSIDE
# the built image, as the runtime user, with no provider credentials — every
# check here is model-free on purpose, so it can gate a Cloud Build step that
# is NOT allowFailure.
#
# What a green result proves:
#   1. the installed pi is exactly the pinned version (the Dockerfile asserts
#      this too; here it is asserted where `ailang` and pi meet)
#   2. Node meets pi's floor
#   3. the embedded extension suite is materialised in pi's GLOBAL dir
#   4. the global dir EXECUTES with no trust decision — a load-time sentinel
#      writes a marker before pi even resolves the model (design doc V27)
#   5. a sentinel under <workspace>/.pi/extensions/ does NOT execute — the
#      workspace is a cloned branch and must not run code beside `bash` (D7)
#
# What it does NOT prove: that a model will call a tool from the suite. That is
# the runbook's `quota_report` probe, which needs a provider and runs on the rig.
set -euo pipefail

PI_VERSION="${PI_VERSION:?PI_VERSION build arg must be passed into the test}"
pass=0; fail=0
have() { # have "<label>" '<shell assertion>'
  if eval "$2"; then echo "  ok   $1"; pass=$((pass+1)); else echo "  FAIL $1"; fail=$((fail+1)); fi
}

# Never `cmd | grep -q` under pipefail: grep exits on first match, the writer
# takes SIGPIPE, and the pipeline reads as FAILED. Capture, then grep the string.
NPM_GLOBAL="$(npm ls -g --depth=0 2>/dev/null || true)"
PI_STATUS="$(ailang pi status 2>/dev/null || true)"

echo "=== 1. pi pin ==="
have "pi --version == ${PI_VERSION}"   '[ "$(pi --version </dev/null)" = "$PI_VERSION" ]'
have "pi is the maintained package"    'grep -q "@earendil-works/pi-coding-agent@${PI_VERSION}" <<<"$NPM_GLOBAL"'
have "abandoned package absent"        '! grep -q "@mariozechner/pi-coding-agent" <<<"$NPM_GLOBAL"'

echo "=== 2. node floor ==="
have "node >= 22.19.0" 'node -e "const [a,b]=process.versions.node.split(\".\").map(Number); process.exit(a>22||(a===22&&b>=19)?0:1)"'

echo "=== 3. embedded suite in the global dir ==="
EXT="$HOME/.pi/agent/extensions"
have "global dir exists"               '[ -d "$EXT" ]'
have "provider-quota.ts installed"     '[ -f "$EXT/provider-quota.ts" ]'
have "ailang pi status: nothing MISSING" '! grep -q "MISSING" <<<"$PI_STATUS"'
have "ailang pi status: suite listed"    'grep -q "provider-quota.ts" <<<"$PI_STATUS"'

echo "=== 4./5. extension execution: global runs, workspace stays inert (D7) ==="
WS="$(mktemp -d)"; OUT="$(mktemp -d)"
mkdir -p "$WS/.pi/extensions"
cat > "$EXT/zz-sentinel-global.ts" <<TS
import { writeFileSync } from "node:fs";
export default function (pi: any) { writeFileSync("$OUT/global.marker", "loaded\n"); }
TS
cat > "$WS/.pi/extensions/sentinel-workspace.ts" <<TS
import { writeFileSync } from "node:fs";
export default function (pi: any) { writeFileSync("$OUT/workspace.marker", "loaded\n"); }
TS
# A bogus model: pi loads extensions, then fails to resolve the model and exits 1.
# No credentials needed, no network. The markers say what executed.
( cd "$WS" && pi --mode json --no-session --no-context-files --model ollama/does-not-exist -p "hi" </dev/null >/dev/null 2>"$OUT/err.txt" ) || true
rm -f "$EXT/zz-sentinel-global.ts"
have "global sentinel executed"        '[ -f "$OUT/global.marker" ]'
have "workspace sentinel did NOT run"  '[ ! -f "$OUT/workspace.marker" ]'
have "no trust.json was needed"        '[ ! -f "$HOME/.pi/agent/trust.json" ]'
have "pi failed on the MODEL, not earlier" 'grep -q "not found" "$OUT/err.txt"'

echo "=== 6. ailang_run is present and default-deny (M-AGENT-AILANG-ONLY-EXECUTION) ==="
have "ailang-exec.ts installed"           '[ -f "$EXT/ailang-exec.ts" ]'
PROFILE_ARGS="$(ailang pi tool-profile ailang_only 2>/dev/null || true)"
# M-EXECUTOR-POLICY-HARDENING M4: the file tools are the policy's sandboxed
# ailang_read/edit/write, never pi's native read/edit/write.
have "ailang pi tool-profile ailang_only: allowlist, no bash" 'case "$PROFILE_ARGS" in *"--no-builtin-tools --tools ailang_read,ailang_edit,ailang_write,ailang_check,ailang_run,builtins_search,examples_search,ailang_cli") true;; *) false;; esac'
have "  ...and no native read/edit/write"                     'case "$PROFILE_ARGS" in *"--tools read,"*|*",read,"*|*",write,"*|*",edit,"*) false;; *) true;; esac'
have "  ...carries its extensions with discovery off"         'case "$PROFILE_ARGS" in "--no-extensions -e "*ailang-exec.ts*|"--no-extensions -e "*ailang-lsp-lite.ts*) true;; *) false;; esac'
have "  ...and the carried files exist"                       'for f in $PROFILE_ARGS; do case "$f" in *.ts) [ -f "$f" ] || exit 1;; esac; done'
# Model-free: the extension's gate is gateFromEnv (the environment) followed by
# gateWithSummary (the Go endpoint's verdict — `ailang policy-tool` op=summary,
# M-EXECUTOR-POLICY-HARDENING M4). Both halves run for real here.
# (capture-then-grep, never `cmd | grep -q` — see the pipefail note above)
GATE_JS='import("'"$EXT"'/ailang-exec.ts").then(async m => { let g = m.gateFromEnv(process.env); if (!g.refusal && g.policyPath) { g = m.gateWithSummary(g, await m.defaultToolRunner()(g.policyPath, { op: "summary" })).gate; } console.log(JSON.stringify(g)); })'
POL="$(mktemp -d)"
G_UNSET="$(env -u AILANG_AGENT_POLICY node --experimental-strip-types -e "$GATE_JS" 2>/dev/null || true)"
# FS needs a sandbox; a policy whose sandbox contains its own directory is D4.
printf 'allowed_caps = ["IO", "FS"]\nfs_sandbox = "%s"\nentry = "main"\n' "$POL" > "$POL/policy.toml"
G_INSIDE="$(AILANG_AGENT_POLICY="$POL/policy.toml" node --experimental-strip-types -e "$GATE_JS" 2>/dev/null || true)"
printf 'allowed_caps = ["IO", "FS"]\nfs_sandbox = "%s"\nentry = "main"\n' "$WS" > "$POL/policy.toml"
G_OUTSIDE="$(AILANG_AGENT_POLICY="$POL/policy.toml" node --experimental-strip-types -e "$GATE_JS" 2>/dev/null || true)"
RUN_HELP="$(ailang run --help 2>&1 || true)"
have "no policy env -> refusal names AILANG_AGENT_POLICY" 'grep -q "AILANG_AGENT_POLICY is unset" <<<"$G_UNSET"'
have "policy inside its own sandbox -> refused (D4)"      'grep -q "rewrite the policy" <<<"$G_INSIDE"'
have "policy outside the sandbox -> granted"              'grep -q "\"refusal\":null" <<<"$G_OUTSIDE"'
have "ailang run --policy exists"                         'grep -q -- "-policy string" <<<"$RUN_HELP"'

echo "=== 7. Z3 is in the image and ai-check uses it ==="
# ai-check degrades to verify.available=false when z3 is absent — a pass
# that proves nothing. Assert the binary AND that ai-check reports it.
Z3V="$(z3 --version 2>&1 || true)"
have "z3 binary present" 'grep -q "Z3 version" <<<"$Z3V"'
printf 'module probe\nexport func main() -> () ! {} = ()\n' > "$WS/probe.ail"
AICHECK="$(cd "$WS" && ailang ai-check probe.ail 2>/dev/null || true)"
have "ai-check reports verify.available=true" 'grep -Eq "\"available\": *true" <<<"$AICHECK"'

echo "=== 8. ailang_cli: typed requests to the Go endpoint; execution stays gated ==="
# The decision is the endpoint's (`ailang policy-tool`), not the extension's:
# iface builds an argv; run is gate-only; messages has no schema; a path that
# leaves the sandbox is refused by the root handle. ($POL/policy.toml is the
# outside-the-sandbox policy from section 6, sandbox = $WS.)
cli_ok() { local out; out="$(printf '%s' "$1" | AILANG_AGENT_POLICY="$POL/policy.toml" ailang policy-tool 2>/dev/null || true)"; grep -Eq '"ok": *true' <<<"$out"; }
have "iface std/fs -> ok (argv built in Go)"        'cli_ok "{\"op\":\"iface\",\"module\":\"std/fs\"}"'
have "run -> refused (gate-only)"                    '! cli_ok "{\"op\":\"run\",\"path\":\"x.ail\"}"'
have "messages -> refused (no schema)"               '! cli_ok "{\"op\":\"messages\",\"query\":\"send\"}"'
have "fmt ../../etc/x -> refused (leaves the sandbox)" '! cli_ok "{\"op\":\"fmt\",\"path\":\"../../etc/x\"}"'
have "read outside the sandbox -> refused"           '! cli_ok "{\"op\":\"read\",\"path\":\"/etc/passwd\"}"'

echo
echo "passed: $pass  failed: $fail"
[ "$fail" -eq 0 ]
