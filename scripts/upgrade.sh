#!/bin/bash

# Open Station - Upgrade Script
# Upgrades existing installation to latest version

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

REPO="zhaojiewen/open-station"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/open-station"

echo "=========================================="
echo "   Open Station - Upgrader"
echo "=========================================="
echo ""

# Check if installed
if [ ! -f "$INSTALL_DIR/open-station" ]; then
    echo -e "${RED}Open Station is not installed${NC}"
    echo "Run './scripts/install-binary.sh' first"
    exit 1
fi

# Get current version
CURRENT_VERSION=$("$INSTALL_DIR/open-station" --version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
echo -e "${GREEN}Current version: $CURRENT_VERSION${NC}"

# Detect system
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    armv7l|armv7) ARCH="armv7" ;;
    armv6l) ARCH="armv6" ;;
esac

# Get latest version
echo "Fetching latest version..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')
echo -e "${GREEN}Latest version: v$LATEST_VERSION${NC}"

# Check if already latest
if [ "$CURRENT_VERSION" = "v$LATEST_VERSION" ]; then
    echo -e "${GREEN}Already running latest version!${NC}"
    exit 0
fi

# Download new version
echo ""
echo "Downloading new version..."

DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$LATEST_VERSION/open-station-$LATEST_VERSION-$OS-$ARCH.tar.gz"
TEMP_DIR=$(mktemp -d)

if ! curl -fSL --progress-bar -o "$TEMP_DIR/package.tar.gz" "$DOWNLOAD_URL"; then
    echo -e "${RED}Download failed${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Extract
tar -xzf "$TEMP_DIR/package.tar.gz" -C "$TEMP_DIR"
BINARY_FILE=$(find "$TEMP_DIR" -name "open-station" -type f | head -1)

if [ ! -f "$BINARY_FILE" ]; then
    echo -e "${RED}Binary not found in package${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Stop service if running
if sudo systemctl is-active --quiet open-station; then
    echo "Stopping service..."
    sudo systemctl stop open-station
fi

# Replace binary
echo "Installing new binary..."
sudo cp "$BINARY_FILE" "$INSTALL_DIR/open-station"
sudo chmod +x "$INSTALL_DIR/open-station"

# Verify
NEW_VERSION=$("$INSTALL_DIR/open-station" --version 2>&1 | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+')
echo -e "${GREEN}✅ Upgraded to $NEW_VERSION${NC}"

# Cleanup
rm -rf "$TEMP_DIR"

# Start service
read -p "Start service now? [Y/n]: " START_SERVICE
START_SERVICE=${START_SERVICE:-Y}

if [[ "$START_SERVICE" =~ ^[Yy]$ ]]; then
    sudo systemctl start open-station
    sleep 3

    if sudo systemctl is-active --quiet open-station; then
        echo -e "${GREEN}✅ Service started${NC}"
    else
        echo -e "${RED}Service failed to start. Check logs:${NC}"
        echo "    sudo journalctl -u open-station -n 50"
    fi
fi

echo ""
echo "=========================================="
echo "   Upgrade Complete!"
echo "=========================================="
echo ""
echo "Previous: $CURRENT_VERSION"
echo "Current:  $NEW_VERSION"
echo ""