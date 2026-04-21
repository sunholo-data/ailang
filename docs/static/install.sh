#!/bin/sh
# AILANG installer — https://ailang.sunholo.com
# Usage: curl -fsSL https://ailang.sunholo.com/install.sh | bash
#   or:  VERSION=v0.9.0 curl -fsSL https://ailang.sunholo.com/install.sh | bash
set -eu

REPO="sunholo-data/ailang"
BINARY="ailang"

# --- helpers ---

info()  { printf '\033[1;34m%s\033[0m\n' "$*"; }
ok()    { printf '\033[1;32m%s\033[0m\n' "$*"; }
warn()  { printf '\033[1;33m%s\033[0m\n' "$*" >&2; }
err()   { printf '\033[1;31merror: %s\033[0m\n' "$*" >&2; exit 1; }

need_cmd() {
    command -v "$1" >/dev/null 2>&1 || err "need '$1' (command not found)"
}

# --- detect platform ---

detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux)   PLATFORM="linux" ;;
        Darwin)  PLATFORM="darwin" ;;
        MINGW*|MSYS*|CYGWIN*)
            err "Windows is not supported via this installer. Download from:\nhttps://github.com/$REPO/releases/latest" ;;
        *) err "unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH_TAG="x64" ;;
        aarch64|arm64)  ARCH_TAG="arm64" ;;
        *) err "unsupported architecture: $ARCH" ;;
    esac

    ARCHIVE="${PLATFORM}.${ARCH_TAG}.${BINARY}.tar.gz"
}

# --- resolve version ---

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        # Ensure leading 'v'
        case "$VERSION" in
            v*) ;;
            *)  VERSION="v$VERSION" ;;
        esac
        return
    fi

    need_cmd curl
    info "Fetching latest release..."
    VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
        | grep '"tag_name"' | head -1 | cut -d '"' -f 4) \
        || err "failed to fetch latest release from GitHub API"

    [ -n "$VERSION" ] || err "could not determine latest version"
}

# --- choose install directory ---

choose_install_dir() {
    if [ -w /usr/local/bin ]; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO=""
    elif command -v sudo >/dev/null 2>&1; then
        INSTALL_DIR="/usr/local/bin"
        USE_SUDO="sudo"
    else
        INSTALL_DIR="$HOME/.local/bin"
        USE_SUDO=""
        mkdir -p "$INSTALL_DIR"
    fi
}

# --- check existing install ---

check_existing() {
    if command -v "$BINARY" >/dev/null 2>&1; then
        CURRENT=$("$BINARY" --version 2>/dev/null || echo "unknown")
        info "Existing installation: $CURRENT"
        info "Updating to $VERSION..."
    fi
}

# --- download, verify, install ---

do_install() {
    need_cmd curl
    need_cmd tar
    need_cmd mktemp

    TMPDIR=$(mktemp -d)
    trap 'rm -rf "$TMPDIR"' EXIT

    URL="https://github.com/$REPO/releases/download/$VERSION/$ARCHIVE"
    info "Downloading $ARCHIVE ($VERSION)..."
    curl -fSL --progress-bar "$URL" -o "$TMPDIR/$ARCHIVE" \
        || err "download failed — check that $VERSION exists at\nhttps://github.com/$REPO/releases"

    # --- Checksum verification ---
    #
    # Every release from v0.13.1 onward publishes <archive>.sha256 alongside
    # the archive (and a combined SHA256SUMS). The installer requires these
    # for current releases. For older releases that don't carry them, we
    # warn loudly and continue — set NO_VERIFY=1 to suppress the prompt if
    # you're deliberately installing one of those.
    #
    # Set NO_VERIFY=1 to skip verification entirely (not recommended; only
    # useful in CI smoke tests where you control the artifact).
    CHECKSUM_URL="${URL}.sha256"
    if [ "${NO_VERIFY:-}" = "1" ]; then
        warn "Checksum verification skipped (NO_VERIFY=1)."
    elif curl -fsSL "$CHECKSUM_URL" -o "$TMPDIR/$ARCHIVE.sha256" 2>/dev/null; then
        info "Verifying SHA256 checksum..."
        cd "$TMPDIR"
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum -c "$ARCHIVE.sha256" >/dev/null \
                || err "checksum verification FAILED — archive corrupt or tampered"
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 -c "$ARCHIVE.sha256" >/dev/null \
                || err "checksum verification FAILED — archive corrupt or tampered"
        else
            err "neither sha256sum nor shasum available — cannot verify.\nInstall one, or re-run with NO_VERIFY=1 (not recommended)."
        fi
        ok "Checksum verified."
        cd - >/dev/null
    else
        # Older releases (pre-v0.13.1) don't publish per-artifact checksums.
        # Warn loudly rather than silently skipping — a false sense of
        # verification is worse than no verification at all.
        warn "No .sha256 published for $VERSION — cannot verify archive integrity."
        warn "Upgrade to a newer release for cryptographic verification,"
        warn "or re-run with NO_VERIFY=1 to acknowledge and continue."
        err "refusing to install unverified archive (set NO_VERIFY=1 to override)"
    fi

    info "Extracting..."
    tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
    chmod +x "$TMPDIR/$BINARY"

    info "Installing to $INSTALL_DIR/$BINARY..."
    $USE_SUDO install -m 755 "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
}

# --- verify & print result ---

verify() {
    INSTALLED_VERSION=$("$INSTALL_DIR/$BINARY" --version 2>/dev/null) \
        || err "installation failed — binary not working"

    echo ""
    ok "AILANG $INSTALLED_VERSION installed successfully!"
    ok "Location: $INSTALL_DIR/$BINARY"

    # PATH warning for ~/.local/bin
    if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
        case ":$PATH:" in
            *":$INSTALL_DIR:"*) ;;
            *)
                echo ""
                warn "Add $INSTALL_DIR to your PATH:"
                warn "  export PATH=\"$INSTALL_DIR:\$PATH\""
                warn "  # Add to ~/.bashrc or ~/.zshrc to make permanent"
                ;;
        esac
    fi

    echo ""
    info "Get started: ailang --help"
    info "Docs: https://ailang.sunholo.com"
}

# --- main ---

detect_platform
resolve_version
choose_install_dir
check_existing
do_install
verify
