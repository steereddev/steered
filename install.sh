#!/bin/sh
set -e

REPO="steereddev/steered"
BINARY="steered"
INSTALL_DIR="/usr/local/bin"

# detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# normalize arch
case $ARCH in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)
    echo "unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# normalize OS
case $OS in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "unsupported OS: $OS"
    exit 1
    ;;
esac

# get latest version
VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' \
  | sed -E 's/.*"v([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "failed to fetch latest version"
  exit 1
fi

echo "installing steered v${VERSION} for ${OS}/${ARCH}..."

# download
FILENAME="steered_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/v${VERSION}/${FILENAME}"

TMP=$(mktemp -d)
curl -fsSL "$URL" -o "${TMP}/${FILENAME}"

# extract
tar -xzf "${TMP}/${FILENAME}" -C "$TMP"

# install
chmod +x "${TMP}/${BINARY}"
mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"

# cleanup
rm -rf "$TMP"

echo "✓ steered installed to ${INSTALL_DIR}/${BINARY}"
echo "  run: steered"