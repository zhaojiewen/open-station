#!/bin/bash

# Open Station - Uninstaller Script
# Removes binary, configuration, and service

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/open-station"
SERVICE_USER="open-station"

echo "=========================================="
echo "   Open Station - Uninstaller"
echo "=========================================="
echo ""

# Confirm
read -p "Are you sure you want to uninstall Open Station? [y/N]: " CONFIRM
if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
    echo "Uninstall cancelled."
    exit 0
fi

# Stop service
echo "Stopping service..."
if sudo systemctl is-active --quiet open-station; then
    sudo systemctl stop open-station
    echo -e "${GREEN}✅ Service stopped${NC}"
fi

# Disable service
echo "Disabling service..."
if sudo systemctl is-enabled --quiet open-station 2>/dev/null; then
    sudo systemctl disable open-station
    echo -e "${GREEN}✅ Service disabled${NC}"
fi

# Remove service file
echo "Removing systemd service..."
if [ -f "/etc/systemd/system/open-station.service" ]; then
    sudo rm "/etc/systemd/system/open-station.service"
    sudo systemctl daemon-reload
    echo -e "${GREEN}✅ Service file removed${NC}"
fi

# Remove binary
echo "Removing binary..."
if [ -f "$INSTALL_DIR/open-station" ]; then
    sudo rm "$INSTALL_DIR/open-station"
    echo -e "${GREEN}✅ Binary removed${NC}"
fi

# Remove config (optional)
read -p "Remove configuration files? [y/N]: " REMOVE_CONFIG
if [[ "$REMOVE_CONFIG" =~ ^[Yy]$ ]]; then
    echo "Removing configuration..."
    sudo rm -rf "$CONFIG_DIR"
    echo -e "${GREEN}✅ Configuration removed${NC}"
else
    echo -e "${YELLOW}Configuration preserved at $CONFIG_DIR${NC}"
fi

# Remove log directory
echo "Removing log directory..."
sudo rm -rf /var/log/open-station 2>/dev/null || true

# Remove user (optional)
read -p "Remove service user '$SERVICE_USER'? [y/N]: " REMOVE_USER
if [[ "$REMOVE_USER" =~ ^[Yy]$ ]]; then
    if id "$SERVICE_USER" &>/dev/null; then
        sudo userdel "$SERVICE_USER" 2>/dev/null || true
        echo -e "${GREEN}✅ User removed${NC}"
    fi
fi

echo ""
echo "=========================================="
echo "   Uninstall Complete!"
echo "=========================================="
echo ""