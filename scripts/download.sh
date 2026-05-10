#!/bin/bash

# Open Station - Quick Download Script
# Simple script to download the correct binary for your system

set -e

REPO="zhaojiewen/open-station"

echo "=========================================="
echo "   Open Station - Quick Download"
echo "=========================================="
echo ""

# Detect system
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l|armv7)
        ARCH="armv7"
        ;;
    armv6l)
        ARCH="armv6"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "System: $OS/$ARCH"

# Get version
VERSION=""
if [ -n "$1" ]; then
    VERSION="$1"
else
    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
fi

if [ -z "$VERSION" ]; then
    VERSION="latest"
fi

echo "Version: $VERSION"

# Download
DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${VERSION}/open-station-${VERSION}-${OS}-${ARCH}.tar.gz"
echo ""
echo "Downloading from:"
echo "$DOWNLOAD_URL"
echo ""

curl -fSL --progress-bar -o "open-station.tar.gz" "$DOWNLOAD_URL"

echo ""
echo "✅ Download complete: open-station.tar.gz"
echo ""
echo "Extract with:    tar xzf open-station.tar.gz"
echo "Install with:    ./install.sh"
echo ""