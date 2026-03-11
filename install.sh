#!/bin/sh
set -e

REPO="alanisme/adb-tui"
BINARY="adb-tui"
INSTALL_DIR="/usr/local/bin"

main() {
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    arch="$(uname -m)"

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        aarch64|arm64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
    esac

    case "$os" in
        linux|darwin) ;;
        *) echo "Unsupported OS: $os (use install.ps1 for Windows)" >&2; exit 1 ;;
    esac

    if [ -n "$1" ]; then
        version="$1"
    else
        version="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
            | grep '"tag_name"' | head -1 | cut -d'"' -f4)"
    fi

    if [ -z "$version" ]; then
        echo "Failed to determine latest version." >&2
        exit 1
    fi

    archive="${BINARY}_${version#v}_${os}_${arch}.tar.gz"
    url="https://github.com/${REPO}/releases/download/${version}/${archive}"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    echo "Downloading ${BINARY} ${version} (${os}/${arch})..."
    curl -fsSL "$url" -o "${tmpdir}/${archive}"

    tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"

    if [ -w "$INSTALL_DIR" ]; then
        mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    else
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    fi

    chmod +x "${INSTALL_DIR}/${BINARY}"
    echo "Installed ${BINARY} ${version} to ${INSTALL_DIR}/${BINARY}"
}

main "$@"
