#!/bin/sh
set -e

REPO="chaewonkong/nezip"
BINARY="nezip"

# Default: /usr/local/bin if writable (root), otherwise ~/.local/bin (no sudo)
if [ -z "$INSTALL_DIR" ]; then
  if [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux)  OS="linux" ;;
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)          ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Fetch latest version tag
VERSION="${VERSION:-$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')}"

if [ -z "$VERSION" ]; then
  echo "Failed to fetch latest version"
  exit 1
fi

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading nezip ${VERSION} (${OS}/${ARCH})..."
curl -fsSL "$URL" -o "${TMP_DIR}/${ARCHIVE}"
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"

mv "${TMP_DIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

chmod +x "${INSTALL_DIR}/${BINARY}"
echo "Installed: ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" --version 2>/dev/null || true

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "NOTE: ${INSTALL_DIR} is not in PATH. Add the following to your shell profile:" \
     && echo "  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
