#!/usr/bin/env bash
# Pearls installer — downloads the latest release for your platform.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/justrnr500/pearls/main/scripts/install.sh | bash
#
set -euo pipefail

REPO="justrnr500/pearls"
INSTALL_DIR="/usr/local/bin"
USE_SUDO="true"

# --- Detect platform ---

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  darwin) ;;
  linux)  ;;
  *)
    echo "Error: Unsupported OS: $OS"
    echo "Pearls supports macOS and Linux."
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    echo "Pearls supports amd64 and arm64."
    exit 1
    ;;
esac

echo "Detected platform: ${OS}/${ARCH}"

# --- Check for sudo if needed ---

if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo &>/dev/null; then
    USE_SUDO="true"
  else
    # Fall back to ~/.local/bin
    INSTALL_DIR="${HOME}/.local/bin"
    USE_SUDO="false"
    mkdir -p "$INSTALL_DIR"
    echo "No write access to /usr/local/bin; installing to ${INSTALL_DIR}"
  fi
fi

do_install() {
  if [ "$USE_SUDO" = "true" ] && [ ! -w "$INSTALL_DIR" ]; then
    sudo "$@"
  else
    "$@"
  fi
}

# --- Fetch latest release ---

echo "Fetching latest release..."
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/latest"
TAG=$(curl -fsSL "$RELEASE_URL" | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Error: Could not determine latest release."
  echo "Check https://github.com/${REPO}/releases"
  exit 1
fi

VERSION="${TAG#v}"
echo "Latest version: ${TAG}"

# --- Download and extract ---

ARCHIVE="pearls_${VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${ARCHIVE}..."
if ! curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$DOWNLOAD_URL"; then
  echo "Error: Download failed."
  echo "URL: ${DOWNLOAD_URL}"
  echo "Check that a release exists for your platform at:"
  echo "  https://github.com/${REPO}/releases/tag/${TAG}"
  exit 1
fi

echo "Extracting..."
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

# --- Install binaries ---

echo "Installing to ${INSTALL_DIR}..."
do_install install -m 755 "${TMPDIR}/pearls" "${INSTALL_DIR}/pearls"
do_install install -m 755 "${TMPDIR}/pl" "${INSTALL_DIR}/pl"

# --- Verify ---

if command -v pearls &>/dev/null; then
  INSTALLED_VERSION="$(pearls --version 2>&1 | head -1)"
  echo ""
  echo "Successfully installed: ${INSTALLED_VERSION}"
else
  echo ""
  echo "Installed to ${INSTALL_DIR}/pearls"
  if [[ ":$PATH:" != *":${INSTALL_DIR}:"* ]]; then
    echo ""
    echo "Note: ${INSTALL_DIR} is not in your PATH."
    echo "Add it with:  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi
fi

echo ""
echo "Get started:"
echo "  cd your-project"
echo "  pearls init"
echo "  pearls onboard --hooks"
