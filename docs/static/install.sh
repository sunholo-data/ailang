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

    # --- Verification ---
    #
    # Preferred: cosign keyless signature check. Every release from v0.14.1+
    # publishes <archive>.sig and <archive>.pem produced by the release
    # workflow via Sigstore's Fulcio CA. The signature is bound to the
    # workflow identity below — a valid signature proves the binary was
    # built by a specific GitHub Actions run in this repo, not just "some
    # Action somewhere".
    #
    # Fallback: SHA256 checksum (v0.13.1+) when cosign isn't installed or
    # the release predates cosign signing.
    #
    # If neither is available, the install aborts rather than proceeding
    # unverified — a silent-skip check is worse than no check.
    #
    # Escape hatch: NO_VERIFY=1 skips all verification. Intended for local
    # CI smoke tests where the artifact is known-trusted; not for end users.
    COSIGN_IDENTITY_REGEXP="^https://github\\.com/$REPO/\\.github/workflows/release\\.yml@refs/tags/v.+$"
    COSIGN_OIDC_ISSUER="https://token.actions.githubusercontent.com"
    SIG_URL="${URL}.sig"
    CERT_URL="${URL}.pem"
    CHECKSUM_URL="${URL}.sha256"

    verify_sha256() {
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
        ok "SHA256 verified."
        cd - >/dev/null
    }

    if [ "${NO_VERIFY:-}" = "1" ]; then
        warn "Verification skipped (NO_VERIFY=1)."
    elif command -v cosign >/dev/null 2>&1 \
        && curl -fsSL "$SIG_URL"  -o "$TMPDIR/$ARCHIVE.sig"  2>/dev/null \
        && curl -fsSL "$CERT_URL" -o "$TMPDIR/$ARCHIVE.pem" 2>/dev/null; then
        info "Verifying cosign signature..."
        if cosign verify-blob \
            --signature   "$TMPDIR/$ARCHIVE.sig" \
            --certificate "$TMPDIR/$ARCHIVE.pem" \
            --certificate-identity-regexp "$COSIGN_IDENTITY_REGEXP" \
            --certificate-oidc-issuer "$COSIGN_OIDC_ISSUER" \
            "$TMPDIR/$ARCHIVE" >/dev/null 2>&1; then
            ok "Signature verified: provenance GitHub Actions $REPO@$VERSION"
        else
            err "cosign verification FAILED — signature mismatch or wrong signer identity"
        fi
    elif curl -fsSL "$CHECKSUM_URL" -o "$TMPDIR/$ARCHIVE.sha256" 2>/dev/null; then
        if ! command -v cosign >/dev/null 2>&1; then
            warn "cosign not found — falling back to SHA256."
            warn "For cryptographic provenance, install cosign:"
            warn "  https://docs.sigstore.dev/cosign/system_config/installation/"
        else
            warn "No cosign signature published for $VERSION — falling back to SHA256."
        fi
        verify_sha256
    else
        # No cosign signature AND no .sha256 — either a very old release or
        # the release assets are incomplete. Refuse to install unverified.
        warn "No verification method available for $VERSION"
        warn "  (no cosign .sig/.pem, no .sha256)."
        warn "Upgrade to a newer release, or re-run with NO_VERIFY=1 to override."
        err "refusing to install unverified archive"
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
