#!/bin/sh
set -e

REPO="terminally-online/shrugged"
BINARY_NAME="shrugged"
INSTALL_DIR="${SHRUGGED_INSTALL_DIR:-/usr/local/bin}"

main() {
    detect_platform
    get_latest_version
    download_binary
    install_binary
    install_completions
    verify_install
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            ;;
        *)
            echo "Error: Unsupported operating system: $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo "Error: Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac

    if [ "$OS" = "windows" ]; then
        BINARY_NAME="shrugged.exe"
        ASSET_NAME="shrugged-windows-amd64.exe"
    else
        ASSET_NAME="shrugged-${OS}-${ARCH}"
    fi

    echo "Detected platform: ${OS}/${ARCH}"
}

get_latest_version() {
    if [ -n "$SHRUGGED_VERSION" ]; then
        VERSION="$SHRUGGED_VERSION"
        echo "Using specified version: $VERSION"
        return
    fi

    echo "Fetching latest version..."
    VERSION=$(curl -sI "https://github.com/${REPO}/releases/latest" | grep -i "^location:" | sed 's/.*tag\///' | tr -d '\r\n')

    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version"
        exit 1
    fi

    echo "Latest version: $VERSION"
}

download_binary() {
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"
    CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

    TEMP_DIR=$(mktemp -d)
    TEMP_BINARY="${TEMP_DIR}/${BINARY_NAME}"
    TEMP_CHECKSUMS="${TEMP_DIR}/checksums.txt"

    echo "Downloading ${ASSET_NAME}..."
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_BINARY"; then
        echo "Error: Failed to download binary from $DOWNLOAD_URL"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    echo "Downloading checksums..."
    if curl -fsSL "$CHECKSUM_URL" -o "$TEMP_CHECKSUMS" 2>/dev/null; then
        echo "Verifying checksum..."
        EXPECTED_CHECKSUM=$(grep "$ASSET_NAME" "$TEMP_CHECKSUMS" | awk '{print $1}')

        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL_CHECKSUM=$(sha256sum "$TEMP_BINARY" | awk '{print $1}')
        elif command -v shasum >/dev/null 2>&1; then
            ACTUAL_CHECKSUM=$(shasum -a 256 "$TEMP_BINARY" | awk '{print $1}')
        else
            echo "Warning: No sha256 command found, skipping checksum verification"
            return
        fi

        if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
            echo "Error: Checksum verification failed"
            echo "Expected: $EXPECTED_CHECKSUM"
            echo "Actual:   $ACTUAL_CHECKSUM"
            rm -rf "$TEMP_DIR"
            exit 1
        fi
        echo "Checksum verified"
    else
        echo "Warning: Could not download checksums, skipping verification"
    fi
}

install_binary() {
    chmod +x "$TEMP_BINARY"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$TEMP_BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$TEMP_BINARY" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    rm -rf "$TEMP_DIR"
    echo "Installed to ${INSTALL_DIR}/${BINARY_NAME}"
}

install_completions() {
    if [ "$SHRUGGED_SKIP_COMPLETIONS" = "1" ]; then
        return
    fi

    SHRUGGED_BIN="${INSTALL_DIR}/${BINARY_NAME}"

    if [ ! -x "$SHRUGGED_BIN" ]; then
        echo "Warning: Could not find shrugged binary, skipping completions"
        return
    fi

    echo "Installing shell completions..."

    # Zsh completions
    ZSH_COMP_DIR=""
    if [ -d "/usr/local/share/zsh/site-functions" ]; then
        ZSH_COMP_DIR="/usr/local/share/zsh/site-functions"
    elif [ -d "/usr/share/zsh/site-functions" ]; then
        ZSH_COMP_DIR="/usr/share/zsh/site-functions"
    elif [ -d "${HOME}/.zsh/completions" ]; then
        ZSH_COMP_DIR="${HOME}/.zsh/completions"
    fi

    if [ -n "$ZSH_COMP_DIR" ]; then
        if [ -w "$ZSH_COMP_DIR" ]; then
            "$SHRUGGED_BIN" completion zsh > "${ZSH_COMP_DIR}/_shrugged" 2>/dev/null && \
                echo "  Installed zsh completions to ${ZSH_COMP_DIR}/_shrugged"
        else
            "$SHRUGGED_BIN" completion zsh 2>/dev/null | sudo tee "${ZSH_COMP_DIR}/_shrugged" >/dev/null && \
                echo "  Installed zsh completions to ${ZSH_COMP_DIR}/_shrugged"
        fi
    fi

    # Bash completions
    BASH_COMP_DIR=""
    if [ -d "/etc/bash_completion.d" ]; then
        BASH_COMP_DIR="/etc/bash_completion.d"
    elif [ -d "/usr/local/etc/bash_completion.d" ]; then
        BASH_COMP_DIR="/usr/local/etc/bash_completion.d"
    elif [ -d "/usr/share/bash-completion/completions" ]; then
        BASH_COMP_DIR="/usr/share/bash-completion/completions"
    fi

    if [ -n "$BASH_COMP_DIR" ]; then
        if [ -w "$BASH_COMP_DIR" ]; then
            "$SHRUGGED_BIN" completion bash > "${BASH_COMP_DIR}/shrugged" 2>/dev/null && \
                echo "  Installed bash completions to ${BASH_COMP_DIR}/shrugged"
        else
            "$SHRUGGED_BIN" completion bash 2>/dev/null | sudo tee "${BASH_COMP_DIR}/shrugged" >/dev/null && \
                echo "  Installed bash completions to ${BASH_COMP_DIR}/shrugged"
        fi
    fi

    # Fish completions (user directory, no sudo needed)
    if [ -d "${HOME}/.config/fish/completions" ]; then
        "$SHRUGGED_BIN" completion fish > "${HOME}/.config/fish/completions/shrugged.fish" 2>/dev/null && \
            echo "  Installed fish completions to ~/.config/fish/completions/shrugged.fish"
    elif [ -d "${HOME}/.config/fish" ]; then
        mkdir -p "${HOME}/.config/fish/completions"
        "$SHRUGGED_BIN" completion fish > "${HOME}/.config/fish/completions/shrugged.fish" 2>/dev/null && \
            echo "  Installed fish completions to ~/.config/fish/completions/shrugged.fish"
    fi
}

verify_install() {
    if command -v shrugged >/dev/null 2>&1; then
        echo ""
        echo "Successfully installed shrugged!"
        shrugged --version 2>/dev/null || true
    else
        echo ""
        echo "Installed shrugged to ${INSTALL_DIR}/${BINARY_NAME}"
        echo "Make sure ${INSTALL_DIR} is in your PATH"
    fi
}

main
