#!/bin/sh
set -e

REPO="20uf/devcli"
BINARY="devcli"
INSTALL_DIR="/usr/local/bin"
PRE_RELEASE=false

# Parse arguments
for arg in "$@"; do
  case "$arg" in
    --pre-release) PRE_RELEASE=true ;;
    *)
      echo "Usage: install.sh [--pre-release]" >&2
      exit 1
      ;;
  esac
done

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Check if already installed
EXISTING=$(command -v "$BINARY" 2>/dev/null || true)
if [ -n "$EXISTING" ]; then
  CURRENT_VERSION=$("$EXISTING" version 2>/dev/null | awk '{print $2}' || echo "unknown")
  echo "${BINARY} is already installed: ${CURRENT_VERSION} (${EXISTING})"
  printf "Reinstall/upgrade? [y/N] "
  read -r REPLY
  case "$REPLY" in
    [yY]|[yY][eE][sS]) ;;
    *)
      echo "Aborted."
      exit 0
      ;;
  esac
fi

# Get version from GitHub API
echo "Fetching latest release..."
if [ "$PRE_RELEASE" = true ]; then
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases" | grep '"tag_name"' | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
else
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$LATEST" ]; then
  if [ "$PRE_RELEASE" = false ]; then
    echo "No stable release found. Try: install.sh --pre-release" >&2
  else
    echo "Failed to fetch latest version" >&2
  fi
  exit 1
fi

ASSET="${BINARY}_${OS}_${ARCH}"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ASSET}"

echo "Downloading ${BINARY} ${LATEST} (${OS}/${ARCH})..."
TMP=$(mktemp)
if ! curl -fsSL -o "$TMP" "$URL"; then
  echo "Download failed. Check that release ${LATEST} has asset ${ASSET}" >&2
  rm -f "$TMP"
  exit 1
fi

chmod +x "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$TMP" "${INSTALL_DIR}/${BINARY}"
fi

echo "${BINARY} ${LATEST} installed to ${INSTALL_DIR}/${BINARY}"
