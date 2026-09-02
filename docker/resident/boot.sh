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
printf '%s' "$MODELS_JSON" > "$PI_HOME/agent/models.json" || die "cannot write model registry"
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

# ─── 5. herdr ────────────────────────────────────────────────────────────────
# NEVER `setsid herdr server`: it fails silently — no output, an empty log, and
# a server that never starts (Phase 0 §10). Plain backgrounding works.
# NEVER bare `herdr`: that launches the TUI, which in a container with no
# terminal hangs.
mkdir -p "$(dirname "$HERDR_SOCKET_PATH")"
herdr server >/tmp/herdr-server.log 2>&1 &
HERDR_PID=$!
log "herdr server pid=$HERDR_PID socket=$HERDR_SOCKET_PATH"

# Readiness is probed with `api snapshot`, NOT `herdr status`: status exits 0
# whether or not the server is running and merely prints "status: not running"
# in its body, so a loop keyed on its exit code never waits (Phase 0 §10).
READY=0
for i in $(seq 1 60); do
  if herdr api snapshot >/tmp/herdr-snapshot.json 2>/dev/null; then READY=1; break; fi
  sleep 1
done
[ "$READY" = "1" ] || { sed 's/^/boot |   /' /tmp/herdr-server.log; die "herdr server did not become ready in 60s"; }
log "herdr ready after ${i}s: $(herdr --version 2>&1)"

# ─── 6. Hand over ────────────────────────────────────────────────────────────
# The health endpoint owns the foreground. If it exits the container should
# exit too, so Cloud Run restarts us rather than leaving a box with a live
# supervisor and no way to reach it.
log "resident agent ready"
wait $HEALTH_PID
