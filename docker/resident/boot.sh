#!/bin/bash
# Resident agent boot path (v6.40.0 RESIDENT-P1 M1).
#
# Runs on EVERY start: first boot, the weekly auto-restart at the 7-day
# ceiling, and every stop/resume. It must therefore be idempotent — nothing
# here may assume it is running for the first time.
#
# Failure posture is fail-closed and loud. A resident agent that boots into a
# subtly wrong state (no model registry, a dead supervisor, an unwritable home)
# is worse than one that refuses to start, because nobody is watching it.
set -uo pipefail

log() { echo "boot | $*"; }
# Kill the health server before exiting. An orphan that outlives boot would
# keep answering on $PORT for a container that never came up — and, when boot
# runs under a command substitution, would hold its stdout open forever.
die() { echo "boot | FATAL: $*" >&2; [ -n "${HEALTH_PID:-}" ] && kill "$HEALTH_PID" 2>/dev/null; exit 1; }
HEALTH_PID=""
cleanup() { [ -n "${HEALTH_PID:-}" ] && kill "$HEALTH_PID" 2>/dev/null; }
trap cleanup EXIT INT TERM

PORT="${RESIDENT_PORT:-8080}"
PI_HOME="${PI_HOME:-/home/ailang/.pi}"
AGENT_HOME="${AGENT_HOME:-/agent-home}"

# ─── 1. Validate configuration ───────────────────────────────────────────────
# Ahead of the port bind, deliberately. These checks are instant, so they cost
# nothing against a startup probe, and a container that cannot possibly work
# should die with a reason in the log rather than serve a health endpoint that
# reports its own failure.
#
# ─── Model registry ──────────────────────────────────────────────────────────
# pi has NO max-tokens flag: it reads maxTokens/contextWindow/reasoning from
# ~/.pi/agent/models.json and falls back to 16384 / 128000 / false for any model
# absent from it. Booting without a registry therefore does not fail — it
# quietly runs every model at a fraction of its capability. So: fail closed.
#
# The registry lives on LOCAL disk, never on $AGENT_HOME. It carries the live
# provider key and the home is a GCS bucket.
if [ -z "${MODELS_JSON:-}" ]; then
  die "MODELS_JSON is unset. pi would silently fall back to maxTokens=16384 / contextWindow=128000 / reasoning=false for every model. Refusing to start."
fi
case "$PI_HOME" in
  "$AGENT_HOME"|"$AGENT_HOME"/*)
    die "PI_HOME ($PI_HOME) is inside AGENT_HOME ($AGENT_HOME). The model registry holds the live provider key and the home is a GCS bucket. Refusing to start." ;;
esac
mkdir -p "$PI_HOME/agent" || die "cannot create $PI_HOME/agent"

# pi reads the provider key FROM the registry file, not the environment — its
# own docs call that "headless-safe". Rather than storing a second copy of the
# key inside MODELS_JSON, the registry carries the literal placeholder
# ${OPENROUTER_API_KEY} and it is substituted here from the env var, which
# Cloud Run injects from Secret Manager. One key, one secret, rotatable without
# touching the registry.
REGISTRY_BODY="$MODELS_JSON"
case "$REGISTRY_BODY" in
  *'${OPENROUTER_API_KEY}'*)
    [ -n "${OPENROUTER_API_KEY:-}" ] || die "the model registry contains the \${OPENROUTER_API_KEY} placeholder but OPENROUTER_API_KEY is unset. pi would authenticate with the literal placeholder and every model call would fail. Refusing to start."
    # Literal split/join, NOT sed/awk substitution. Both treat characters in
    # the REPLACEMENT specially — awk's gsub reads `&` as "the matched text", so
    # a key containing `&` silently becomes the placeholder again and every
    # model call fails looking like a bad key. Caught by the test below, which
    # is why its fixture key contains `&`. split/join interprets nothing.
    REGISTRY_BODY=$(printf '%s' "$REGISTRY_BODY" | node -e '
      const src = require("fs").readFileSync(0, "utf8");
      const key = process.env.OPENROUTER_API_KEY;
      process.stdout.write(src.split("${OPENROUTER_API_KEY}").join(key));
    ') || die "provider key substitution failed"
    log "provider key substituted into the model registry from OPENROUTER_API_KEY"
    ;;
esac
printf '%s' "$REGISTRY_BODY" > "$PI_HOME/agent/models.json" || die "cannot write model registry"
chmod 0600 "$PI_HOME/agent/models.json"
node -e 'JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"))' "$PI_HOME/agent/models.json" \
  || die "MODELS_JSON is not valid JSON"
REG_COUNT=$(node -e '
const c=JSON.parse(require("fs").readFileSync(process.argv[1],"utf8"));
console.log(Object.values(c.providers||{}).reduce((n,p)=>n+((p.models||[]).length),0));' "$PI_HOME/agent/models.json")
[ "${REG_COUNT:-0}" -gt 0 ] || die "model registry declares no models"
log "model registry: $REG_COUNT models -> $PI_HOME/agent/models.json (0600, local disk)"

# ─── 1b. AILANG effect sandbox ───────────────────────────────────────────────
# The containment story for anything this agent PROGRAMS is AILANG's effect
# system: capabilities are deny-by-default (`--caps`), and FS operations are
# confined to AILANG_FS_SANDBOX with escapes rejected and traced.
#
# ⚠️ The default is the dangerous direction: `Sandbox string // (empty = no
# sandbox)`. An unset variable does not fail — it removes the confinement
# silently, which is the same shape as the MODELS_JSON trap above. So: refuse.
#
# ⚠️ SCOPE, stated so nobody over-trusts this: the sandbox bounds AILANG
# PROGRAMS. It does not bound the agent CLI's own file-write or shell tools,
# which reach the filesystem directly. The containment argument therefore holds
# only if AILANG is the agent's sole write path — see the design doc.
AILANG_FS_SANDBOX="${AILANG_FS_SANDBOX:-}"
if [ -z "$AILANG_FS_SANDBOX" ]; then
  die "AILANG_FS_SANDBOX is unset. AILANG treats an empty sandbox as NO sandbox, so effects would run unconfined. Refusing to start."
fi
export AILANG_FS_SANDBOX
mkdir -p "$AILANG_FS_SANDBOX" || die "cannot create sandbox root $AILANG_FS_SANDBOX"

# Prove the sandbox is LIVE rather than merely configured, using ailang's own
# checker: a path outside must be rejected, a path inside must be allowed.
# A misconfigured root that silently allows everything is the failure this
# catches.
if ailang sandbox-check /etc/passwd >/dev/null 2>&1; then
  die "AILANG_FS_SANDBOX=$AILANG_FS_SANDBOX does not reject /etc/passwd — the sandbox is not confining. Refusing to start."
fi
if ! ailang sandbox-check "$AILANG_FS_SANDBOX/probe.txt" >/dev/null 2>&1; then
  die "AILANG_FS_SANDBOX=$AILANG_FS_SANDBOX rejects its own root — misconfigured. Refusing to start."
fi
log "ailang effect sandbox live: $AILANG_FS_SANDBOX (escape rejected, root allowed)"

# ─── 2. Bind the port ────────────────────────────────────────────────────────
# Now that the config is known-good, bind before the slow work. Cloning a repo
# and starting herdr take tens of seconds, and a startup probe that finds
# nothing listening would kill the container mid-boot, turning every
# diagnosable failure into a silent restart loop.
node /usr/local/bin/server.mjs &
HEALTH_PID=$!
log "health endpoint pid=$HEALTH_PID port=$PORT"

# ─── 3. Agent home ───────────────────────────────────────────────────────────
# The mount is scoped per-agent with gcsfuse only-dir, so this is already this
# agent's private prefix. Prove it is writable now rather than discovering it
# on the first tool call.
if [ -d "$AGENT_HOME" ]; then
  if touch "$AGENT_HOME/.boot-probe" 2>/dev/null; then
    rm -f "$AGENT_HOME/.boot-probe"
    log "agent home writable: $AGENT_HOME"
  else
    die "$AGENT_HOME is not writable by uid $(id -u). Check the gcsfuse uid/gid mount options against the container user."
  fi
else
  log "WARNING: $AGENT_HOME is not mounted — session state will not survive a restart"
fi

# ─── 4. Workspace ────────────────────────────────────────────────────────────
# Code lives on LOCAL disk and durability is the git remote, not the mount:
# gcsfuse has no POSIX locking and a .git directory on it corrupts. Re-cloning
# on every boot is the idempotent path; an existing clone is fetched instead.
if [ -n "${WORKSPACE_REPO:-}" ]; then
  WORKSPACE_DIR="${WORKSPACE_DIR:-/workspace}"
  mkdir -p "$WORKSPACE_DIR"
  if [ -d "$WORKSPACE_DIR/.git" ]; then
    log "workspace present, fetching"
    git -C "$WORKSPACE_DIR" fetch --all --prune 2>&1 | sed 's/^/boot |   /'
  else
    log "cloning $WORKSPACE_REPO"
    git clone "$WORKSPACE_REPO" "$WORKSPACE_DIR" 2>&1 | sed 's/^/boot |   /' \
      || die "clone failed"
  fi
fi

# ─── 5. pi extensions ────────────────────────────────────────────────────────
# agent-pi bakes AILANG's dev extension suite in via `ailang pi install`:
# sprint-steward, prepush-gate, microrag-context, session-protocol-gate and the
# rest. They exist for a developer working in the ailang repo and are wrong for
# a resident agent:
#   * they cost startup time and memory on every container start;
#   * session-protocol-gate blocks edit/write and fail-closes bash until
#     `session_protocol_ack`, which a headless resident cannot satisfy the
#     interactive way — it branches on ctx.hasUI and asks for a human keypress.
# Off by default, opt back in with RESIDENT_PI_EXTENSIONS=1 when a resident is
# genuinely doing ailang repo work and can satisfy the protocol.
EXT_DIR="$PI_HOME/agent/extensions"
if [ "${RESIDENT_PI_EXTENSIONS:-0}" = "1" ]; then
  log "pi extensions: ENABLED (RESIDENT_PI_EXTENSIONS=1)"
elif [ -d "$EXT_DIR" ]; then
  mv "$EXT_DIR" "$EXT_DIR.disabled" 2>/dev/null || true
  log "pi extensions: disabled (moved aside; set RESIDENT_PI_EXTENSIONS=1 to keep)"
fi

# ─── 5b. prove pi runs headless ──────────────────────────────────────────────
# A boot that reports "ready" while the agent binary cannot actually run is the
# failure this design keeps repeating: on 2026-09-03 pi spawned, emitted
# nothing, and never exited, and the instance looked perfectly healthy for the
# fifteen minutes the hard timeout allowed. The cause was an inherited stdin
# pipe rather than /dev/null, which `pi --version </dev/null` would have caught
# in one second.
#
# So the boot proves it, with the SAME stdin discipline the executor uses. This
# is a warning, not a die(): a resident that cannot run pi should still serve
# /health so an operator can see why, rather than crash-looping silently.
PI_VERSION=$(timeout 20 pi --version </dev/null 2>&1 | head -1 || true)
if [ -z "$PI_VERSION" ]; then
  log "WARN pi did not answer --version within 20s — the agent path will not work"
else
  log "pi headless check: $PI_VERSION"
fi

# ─── 5c. session persistence (M10) ───────────────────────────────────────────
# A resident whose conversation dies with every call is a host that persists
# and an agent that does not. `--session-id <id>` is documented as "use exact
# project session ID, CREATING IT IF MISSING", which makes resume idempotent
# and lets the CALLER own the identifier — so nothing has to be captured from
# the event stream and stored.
#
# But that flag is verified, never assumed. The image pins its own pi and this
# boot has already been wrong once about what pi does headless. If the flag is
# absent the agent still serves; it serves STATELESS AND SAYS SO, rather than
# passing an unknown flag and having pi fail on every call.
PI_SESSION_DIR="${PI_SESSION_DIR:-$PI_HOME/sessions}"
mkdir -p "$PI_SESSION_DIR"
# The result is written to a FILE, not exported: the server was started back in
# section 2 so it could answer the startup probe before the slow work, which
# means it long predates this env. It re-reads the file per run instead.
CAP_FILE="${TASK_STATE_DIR:-/home/ailang/.resident}/capabilities.json"
mkdir -p "$(dirname "$CAP_FILE")"
if timeout 20 pi --help </dev/null 2>&1 | grep -q -- "--session-id"; then
  SESSION_FLAG="--session-id"
  log "session persistence: ENABLED via --session-id ($PI_SESSION_DIR)"
else
  SESSION_FLAG=""
  log "WARN session persistence DISABLED: pi $PI_VERSION has no --session-id, so every call is stateless"
fi
printf '{"piVersion":"%s","sessionFlag":"%s","sessionDir":"%s","agentHome":"%s"}\n' \
  "$PI_VERSION" "$SESSION_FLAG" "$PI_SESSION_DIR" "$AGENT_HOME" > "$CAP_FILE"

# Sessions live on LOCAL disk and are staged to the mount, never written
# straight to it: gcsfuse has no POSIX locking and pi rewrites a session file
# throughout a run. Same rule as the workspace and for the same reason.
# One restore point, rotated per boot. Extensions have a blast radius they did
# not have before M10: with --no-session a misbehaving extension could ruin one
# call, whereas a compaction extension now rewrites the user's CONVERSATION and
# that rewrite is staged to GCS. One previous generation costs a single copy per
# boot and is the difference between "the assistant forgot last week" and a
# recoverable mistake.
if [ -d "$AGENT_HOME/sessions" ]; then
  rm -rf "$AGENT_HOME/sessions.prev" 2>/dev/null || true
  cp -a "$AGENT_HOME/sessions" "$AGENT_HOME/sessions.prev" 2>/dev/null \
    && log "previous sessions kept at $AGENT_HOME/sessions.prev" \
    || log "WARN could not snapshot sessions to sessions.prev"
fi

if [ -d "$AGENT_HOME/sessions" ]; then
  if cp -a "$AGENT_HOME/sessions/." "$PI_SESSION_DIR/" 2>/dev/null; then
    log "sessions restored from $AGENT_HOME/sessions ($(find "$PI_SESSION_DIR" -type f 2>/dev/null | wc -l | tr -d ' ') files)"
  else
    log "WARN could not restore sessions from $AGENT_HOME/sessions — continuing with a fresh store"
  fi
fi

# ─── 6. herdr (optional) ─────────────────────────────────────────────────────
# herdr is NOT on the task path any more: stream mode runs `pi --mode json`
# directly, because driving pi as a TUI headless does not submit prompts and
# cannot report completion. herdr remains installed for human attach
# (`herdr --remote`), but starting it by default costs a process and memory for
# nothing.
#
# NEVER `setsid herdr server` — it fails silently. NEVER bare `herdr` — that
# launches the TUI and hangs with no terminal.
if [ "${RESIDENT_ENABLE_HERDR:-0}" = "1" ]; then
  mkdir -p "$(dirname "$HERDR_SOCKET_PATH")"
  herdr server >/tmp/herdr-server.log 2>&1 &
  HERDR_PID=$!
  log "herdr server pid=$HERDR_PID socket=$HERDR_SOCKET_PATH"
  # Readiness via `api snapshot`, NOT `herdr status`: status exits 0 whether or
  # not the server runs.
  READY=0
  for i in $(seq 1 60); do
    if herdr api snapshot >/tmp/herdr-snapshot.json 2>/dev/null; then READY=1; break; fi
    sleep 1
  done
  [ "$READY" = "1" ] || { sed 's/^/boot |   /' /tmp/herdr-server.log; die "herdr server did not become ready in 60s"; }
  log "herdr ready after ${i}s: $(herdr --version 2>&1)"
else
  log "herdr: not started (RESIDENT_ENABLE_HERDR=1 to enable for human attach)"
fi

# ─── 6. Hand over ────────────────────────────────────────────────────────────
# The health endpoint owns the foreground. If it exits the container should
# exit too, so Cloud Run restarts us rather than leaving a box with a live
# supervisor and no way to reach it.
log "resident agent ready"
wait $HEALTH_PID
