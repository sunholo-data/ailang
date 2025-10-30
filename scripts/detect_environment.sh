#!/bin/bash
# Environment Detection for Claude Code
# Detects whether running in cloud (web UI) or local environment

set -euo pipefail

# Detect environment type
detect_environment() {
    # Primary detection: CLAUDE_CODE_REMOTE environment variable
    if [[ "${CLAUDE_CODE_REMOTE:-}" == "true" ]]; then
        echo "cloud"
        return 0
    fi

    # Secondary detection: CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE
    if [[ "${CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE:-}" == "cloud_default" ]]; then
        echo "cloud"
        return 0
    fi

    # Tertiary detection: Sandbox + Docker (fallback)
    if [[ "${IS_SANDBOX:-}" == "yes" ]] && [[ -f "/.dockerenv" ]]; then
        echo "cloud"
        return 0
    fi

    # Default to local
    echo "local"
    return 0
}

# Get PATH adjustments based on environment
get_path_for_environment() {
    local env=$(detect_environment)

    if [[ "$env" == "cloud" ]]; then
        # Cloud: Go installs to /root/go/bin
        echo "/root/go/bin"
    else
        # Local: Go installs to ~/go/bin
        echo "$HOME/go/bin"
    fi
}

# Setup PATH for ailang commands
setup_ailang_path() {
    local go_bin=$(get_path_for_environment)

    # Only add if not already in PATH
    if [[ ":$PATH:" != *":$go_bin:"* ]]; then
        export PATH="$PATH:$go_bin"
    fi
}

# Verbose diagnostic output
show_environment_info() {
    local env=$(detect_environment)
    local go_bin=$(get_path_for_environment)

    echo "=== Claude Code Environment Detection ==="
    echo ""
    echo "Environment Type: $env"
    echo "Go Bin Directory: $go_bin"
    echo ""
    echo "Primary Indicators:"
    echo "  CLAUDE_CODE_REMOTE: ${CLAUDE_CODE_REMOTE:-unset}"
    echo "  CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE: ${CLAUDE_CODE_REMOTE_ENVIRONMENT_TYPE:-unset}"
    echo "  IS_SANDBOX: ${IS_SANDBOX:-unset}"
    echo ""
    echo "System Info:"
    echo "  HOME: ${HOME}"
    echo "  USER: ${USER:-unset}"
    echo "  HOSTNAME: $(hostname 2>/dev/null || echo 'unknown')"
    echo "  /.dockerenv: $([ -f /.dockerenv ] && echo 'exists' || echo 'absent')"
    echo ""
    echo "AILANG Binary:"
    if command -v ailang &>/dev/null; then
        echo "  ✅ Found: $(which ailang)"
        echo "  Version: $(ailang --version 2>&1 | head -1)"
    else
        echo "  ❌ Not found in PATH"
        echo "  💡 Run: make quick-install"
        echo "  💡 Then: export PATH=\$PATH:$go_bin"
    fi
    echo ""

    if [[ "$env" == "cloud" ]]; then
        echo "🌐 Cloud Environment Detected"
        echo "   Suggestions:"
        echo "   - Use: export PATH=\$PATH:/root/go/bin"
        echo "   - Or: source scripts/detect_environment.sh && setup_ailang_path"
    else
        echo "💻 Local Environment Detected"
        echo "   Suggestions:"
        echo "   - Standard Go PATH should work: ~/go/bin"
        echo "   - Add to .bashrc: export PATH=\$PATH:\$HOME/go/bin"
    fi
    echo ""
    echo "========================================="
}

# If script is executed (not sourced), show info
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Script is being executed directly
    case "${1:-info}" in
        detect)
            detect_environment
            ;;
        path)
            get_path_for_environment
            ;;
        setup)
            setup_ailang_path
            echo "PATH updated: $(get_path_for_environment) added"
            ;;
        info|*)
            show_environment_info
            ;;
    esac
else
    # Script is being sourced - just make functions available
    :
fi
