#!/usr/bin/env bash
# M-COORD-MULTI-HOST-WORKERS (v0.22.0): install the AILANG coordinator
# daemon as a launchd LaunchAgent on macOS.
# M-COORD-TAG-ROUTING-LASTMILE (v0.23.0): adds --port flag to control the
# daemon's HTTP listener port (default 8765).
#
# Idempotent: re-running on an already-installed host is a no-op that
# reports "already installed" and exits 0.
#
# Usage:
#   tools/launchd/install_coordinator.sh                         # default config
#   tools/launchd/install_coordinator.sh --tags ollama:gemma4-26b-ailang,gpu:m4-max
#   tools/launchd/install_coordinator.sh --host-id studio.eval-rig --tags ollama:gemma4-26b-ailang
#   tools/launchd/install_coordinator.sh --port 9000             # custom HTTP port (default 8765)
#   tools/launchd/install_coordinator.sh --dry-run               # show planned changes
#   tools/launchd/install_coordinator.sh --uninstall             # remove daemon
#   tools/launchd/install_coordinator.sh --help
#
# Environment overrides (rarely needed):
#   AILANG_BIN              path to ailang binary (default: $HOME/go/bin/ailang)
#   AILANG_CLOUD_PROJECT    GCP project for Pub/Sub (default: from gcloud config)
#   AILANG_COORD_HTTP_PORT  override default HTTP port 8765 (or pass --port)

set -euo pipefail

# ───────────────────────────────────────────────────────────────────────────
# Defaults + arg parsing
# ───────────────────────────────────────────────────────────────────────────

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMPLATE="$REPO_ROOT/tools/launchd/dev.ailang.coordinator.plist.template"
LAUNCHAGENT_DIR="$HOME/Library/LaunchAgents"
PLIST_DEST="$LAUNCHAGENT_DIR/dev.ailang.coordinator.plist"
PLIST_BACKUP="${PLIST_DEST}.bak"
LABEL="dev.ailang.coordinator"
CONFIG_FILE="$HOME/.ailang/config.yaml"

AILANG_BIN="${AILANG_BIN:-$HOME/go/bin/ailang}"
PATH_PREFIX="$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
HOST_ID=""
TAGS=""
HTTP_PORT="${AILANG_COORD_HTTP_PORT:-8765}"
DRY_RUN=0
UNINSTALL=0

# Resolve cloud project from gcloud unless explicitly set in environment.
CLOUD_PROJECT="${AILANG_CLOUD_PROJECT:-}"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --tags)
            TAGS="$2"; shift 2 ;;
        --host-id)
            HOST_ID="$2"; shift 2 ;;
        --cloud-project)
            CLOUD_PROJECT="$2"; shift 2 ;;
        --port)
            HTTP_PORT="$2"; shift 2 ;;
        --dry-run)
            DRY_RUN=1; shift ;;
        --uninstall)
            UNINSTALL=1; shift ;;
        --help|-h)
            # Print the comment header up to the first non-comment line.
            awk 'NR>1 && /^[^#]/ {exit} NR>1 {sub(/^# ?/, ""); print}' "$0"
            exit 0 ;;
        *)
            echo "Unknown arg: $1 (use --help)" >&2; exit 2 ;;
    esac
done

# Validate port is a number in the unprivileged range
if ! [[ "$HTTP_PORT" =~ ^[0-9]+$ ]] || (( HTTP_PORT < 1024 || HTTP_PORT > 65535 )); then
    echo "Invalid --port value: $HTTP_PORT (expected unprivileged port 1024-65535)" >&2
    exit 2
fi

# ───────────────────────────────────────────────────────────────────────────
# Helpers
# ───────────────────────────────────────────────────────────────────────────

log() { printf "  %s\n" "$*"; }
ok() { printf "✓ %s\n" "$*"; }
warn() { printf "⚠ %s\n" "$*" >&2; }
err() { printf "✗ %s\n" "$*" >&2; exit 1; }

resolve_cloud_project() {
    if [[ -n "$CLOUD_PROJECT" ]]; then return; fi
    # gcloud is often installed outside the bash PATH (e.g. via path.zsh.inc).
    # Look in common install locations before giving up.
    local gcloud_bin=""
    if command -v gcloud >/dev/null 2>&1; then
        gcloud_bin="gcloud"
    else
        for candidate in \
            "$HOME/google-cloud-sdk/bin/gcloud" \
            "$HOME/dev/google-cloud-sdk/bin/gcloud" \
            "/usr/local/google-cloud-sdk/bin/gcloud" \
            "/opt/google-cloud-sdk/bin/gcloud" \
            "/opt/homebrew/bin/gcloud" \
            "/opt/homebrew/share/google-cloud-sdk/bin/gcloud" \
        ; do
            if [[ -x "$candidate" ]]; then
                gcloud_bin="$candidate"
                break
            fi
        done
    fi
    if [[ -n "$gcloud_bin" ]]; then
        CLOUD_PROJECT="$("$gcloud_bin" config get-value project 2>/dev/null || true)"
    fi
    if [[ -z "$CLOUD_PROJECT" ]]; then
        warn "Could not detect gcloud project. Set --cloud-project or AILANG_CLOUD_PROJECT."
        warn "Defaulting to 'ailang-multivac-dev'."
        CLOUD_PROJECT="ailang-multivac-dev"
    fi
}

render_plist() {
    sed \
        -e "s|@USER_HOME@|$HOME|g" \
        -e "s|@AILANG_BIN@|$AILANG_BIN|g" \
        -e "s|@AILANG_CLOUD_PROJECT@|$CLOUD_PROJECT|g" \
        -e "s|@HTTP_PORT@|$HTTP_PORT|g" \
        -e "s|@PATH_PREFIX@|$PATH_PREFIX|g" \
        "$TEMPLATE"
}

# ───────────────────────────────────────────────────────────────────────────
# Uninstall path
# ───────────────────────────────────────────────────────────────────────────

if [[ "$UNINSTALL" -eq 1 ]]; then
    echo "Uninstalling $LABEL..."
    if launchctl list 2>/dev/null | grep -q "$LABEL"; then
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log "would: launchctl unload $PLIST_DEST"
        else
            launchctl unload "$PLIST_DEST" 2>/dev/null || true
            ok "Unloaded launchd job"
        fi
    else
        log "Job not loaded — skipping unload"
    fi

    if [[ -f "$PLIST_DEST" ]]; then
        if [[ "$DRY_RUN" -eq 1 ]]; then
            log "would: rm $PLIST_DEST"
        else
            rm "$PLIST_DEST"
            ok "Removed plist: $PLIST_DEST"
        fi
    else
        log "No plist at $PLIST_DEST — nothing to remove"
    fi

    if [[ -f "$PLIST_BACKUP" ]]; then
        log "Backup retained at: $PLIST_BACKUP"
    fi
    ok "Uninstall complete"
    exit 0
fi

# ───────────────────────────────────────────────────────────────────────────
# Install path
# ───────────────────────────────────────────────────────────────────────────

if [[ ! -f "$TEMPLATE" ]]; then
    err "Template missing: $TEMPLATE"
fi

if [[ ! -x "$AILANG_BIN" ]]; then
    err "ailang binary not found or not executable: $AILANG_BIN (set AILANG_BIN env var)"
fi

resolve_cloud_project

# Header
echo "AILANG coordinator launchd install"
echo "─────────────────────────────────"
log "ailang binary:   $AILANG_BIN"
log "host_id:         ${HOST_ID:-<hostname auto>}"
log "tags:            ${TAGS:-<none>}"
log "cloud project:   $CLOUD_PROJECT"
log "HTTP port:       $HTTP_PORT (curl http://127.0.0.1:$HTTP_PORT/health)"
log "plist target:    $PLIST_DEST"
log "config file:     $CONFIG_FILE"
[[ "$DRY_RUN" -eq 1 ]] && log "MODE:            DRY RUN (no changes will be made)"
echo

# Render the plist into a temp file so we can diff it against any existing
# install and report idempotency cleanly.
RENDERED="$(mktemp -t ailang-coord-plist.XXXXXX)"
render_plist >"$RENDERED"

# Idempotency check on the plist itself.
PLIST_ACTION="install"
if [[ -f "$PLIST_DEST" ]]; then
    if cmp -s "$RENDERED" "$PLIST_DEST"; then
        PLIST_ACTION="unchanged"
    else
        PLIST_ACTION="replace"
    fi
fi

case "$PLIST_ACTION" in
    install)
        log "Plist:           NEW (will create)"
        ;;
    replace)
        log "Plist:           CHANGED (will back up existing → ${PLIST_BACKUP##*/} and replace)"
        ;;
    unchanged)
        log "Plist:           unchanged"
        ;;
esac

# Worker tags / host_id reminder (we don't auto-edit ~/.ailang/config.yaml
# because that file may have many other agents the user owns).
echo
if [[ -n "$TAGS" || -n "$HOST_ID" ]]; then
    echo "Worker advertisement (manual step — script does NOT edit your config):"
    echo
    echo "  Edit $CONFIG_FILE, find the agent you want to advertise"
    echo "  (e.g. eval-rig), and add:"
    echo
    [[ -n "$HOST_ID" ]] && echo "      worker_host_id: $HOST_ID"
    if [[ -n "$TAGS" ]]; then
        echo "      worker_tags:"
        IFS=',' read -r -a TAG_LIST <<< "$TAGS"
        for tag in "${TAG_LIST[@]}"; do
            tag="$(echo "$tag" | xargs)" # trim
            [[ -n "$tag" ]] && echo "        - $tag"
        done
    fi
    echo
fi

# Apply or stop here for dry-run.
if [[ "$DRY_RUN" -eq 1 ]]; then
    log "Skipping launchctl operations (--dry-run)."
    log "Plist that WOULD be written to $PLIST_DEST:"
    echo
    cat "$RENDERED"
    rm "$RENDERED"
    exit 0
fi

mkdir -p "$LAUNCHAGENT_DIR"

if [[ "$PLIST_ACTION" == "replace" ]]; then
    cp "$PLIST_DEST" "$PLIST_BACKUP"
    log "Backed up existing plist → $PLIST_BACKUP"
    # Unload the old job before replacing the file.
    if launchctl list 2>/dev/null | grep -q "$LABEL"; then
        launchctl unload "$PLIST_DEST" 2>/dev/null || true
        log "Unloaded old job"
    fi
fi

if [[ "$PLIST_ACTION" == "install" || "$PLIST_ACTION" == "replace" ]]; then
    install -m 0644 "$RENDERED" "$PLIST_DEST"
    ok "Wrote $PLIST_DEST"
fi

rm "$RENDERED"

# Load (or reload) the job. `launchctl load` is idempotent in practice when
# the previous job was already unloaded, which we ensured above.
if launchctl list 2>/dev/null | grep -q "$LABEL"; then
    log "Job already loaded — leaving as is"
else
    launchctl load "$PLIST_DEST"
    ok "Loaded launchd job"
fi

echo
echo "Verify with:"
echo "  launchctl list | grep $LABEL"
echo "  ailang coordinator status"
echo "  curl -s http://127.0.0.1:$HTTP_PORT/health    # should return {\"status\":\"ok\"}"
echo "  tail -f /tmp/ailang-coordinator-launchd.log"
echo
ok "Install complete"
