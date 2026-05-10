#!/bin/bash

# Open Station - Binary Installer for Linux
# Downloads and installs pre-built binary

set -e

# Configuration
REPO="zhaojiewen/open-station"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/open-station"
SERVICE_USER="open-station"
SERVICE_GROUP="open-station"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "   Open Station - Binary Installer"
echo "=========================================="
echo ""

# Detect system
detect_system() {
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
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    echo -e "${GREEN}System detected: $OS/$ARCH${NC}"
}

# Get latest version
get_latest_version() {
    echo "Fetching latest version..."

    VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')

    if [ -z "$VERSION" ]; then
        echo -e "${YELLOW}Could not fetch latest version, using 'latest'${NC}"
        VERSION="latest"
    else
        echo -e "${GREEN}Latest version: v$VERSION${NC}"
    fi
}

# Download binary
download_binary() {
    echo ""
    echo "Downloading Open Station..."

    if [ "$VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/open-station-latest-$OS-$ARCH.tar.gz"
    else
        DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/open-station-$VERSION-$OS-$ARCH.tar.gz"
    fi

    TEMP_DIR=$(mktemp -d)
    PACKAGE_FILE="$TEMP_DIR/open-station.tar.gz"

    echo "Download URL: $DOWNLOAD_URL"

    if ! curl -fSL --progress-bar -o "$PACKAGE_FILE" "$DOWNLOAD_URL"; then
        echo -e "${RED}Download failed!${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    echo -e "${GREEN}✅ Download complete${NC}"

    # Extract
    echo "Extracting package..."
    tar -xzf "$PACKAGE_FILE" -C "$TEMP_DIR"

    BINARY_FILE="$TEMP_DIR/bin/open-station"
    if [ ! -f "$BINARY_FILE" ]; then
        # Try direct binary in archive
        BINARY_FILE=$(find "$TEMP_DIR" -name "open-station" -type f | head -1)
    fi

    if [ ! -f "$BINARY_FILE" ]; then
        echo -e "${RED}Binary not found in package${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
}

# Install binary
install_binary() {
    echo ""
    echo "Installing binary..."

    # Create directories
    sudo mkdir -p "$INSTALL_DIR"

    # Copy binary
    sudo cp "$BINARY_FILE" "$INSTALL_DIR/open-station"
    sudo chmod +x "$INSTALL_DIR/open-station"

    echo -e "${GREEN}✅ Binary installed to $INSTALL_DIR/open-station${NC}"

    # Verify
    if "$INSTALL_DIR/open-station" --version 2>&1 | grep -q "Open Station"; then
        echo -e "${GREEN}✅ Binary verification successful${NC}"
    fi
}

# Install configuration
install_config() {
    echo ""
    echo "Installing configuration..."

    # Create config directory
    sudo mkdir -p "$CONFIG_DIR"

    # Check if configs exist in package
    if [ -d "$TEMP_DIR/configs" ]; then
        sudo cp -r "$TEMP_DIR/configs/*" "$CONFIG_DIR/" 2>/dev/null || true
    fi

    # Create default config if not exists
    if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
        sudo tee "$CONFIG_DIR/config.yaml" > /dev/null << 'EOF'
server:
  port: 8080
  mode: release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: ai_gateway
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 20
  conn_max_lifetime: 1h
  conn_max_idle_time: 10m

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 100
  min_idle_conns: 10

providers:
  openai:
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_API_KEY}
    timeout: 120s
  claude:
    base_url: https://api.anthropic.com/v1
    api_key: ${ANTHROPIC_API_KEY}
    timeout: 120s

logging:
  level: info
  format: json
  output: stdout

rate_limit:
  default_user_rps: 20
  default_tenant_rps: 200
EOF
        echo -e "${GREEN}✅ Default config created at $CONFIG_DIR/config.yaml${NC}"
    else
        echo -e "${GREEN}✅ Config installed to $CONFIG_DIR/${NC}"
    fi
}

# Create service user
create_user() {
    echo ""
    echo "Creating service user..."

    if ! id "$SERVICE_USER" &>/dev/null; then
        sudo useradd -r -s /bin/false -d "$CONFIG_DIR" "$SERVICE_USER"
        echo -e "${GREEN}✅ User '$SERVICE_USER' created${NC}"
    else
        echo -e "${YELLOW}User '$SERVICE_USER' already exists${NC}"
    fi

    # Set ownership
    sudo chown -R "$SERVICE_USER:$SERVICE_GROUP" "$CONFIG_DIR"
}

# Install systemd service
install_service() {
    echo ""
    echo "Installing systemd service..."

    SERVICE_FILE="/etc/systemd/system/open-station.service"

    sudo tee "$SERVICE_FILE" > /dev/null << 'EOF'
[Unit]
Description=Open Station - Enterprise AI Gateway
Documentation=https://github.com/zhaojiewen/open-station
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=open-station
Group=open-station
ExecStart=/usr/local/bin/open-station -config /etc/open-station/config.yaml
Restart=always
RestartSec=5
TimeoutStartSec=30
TimeoutStopSec=30

# Limits
LimitNOFILE=65535
LimitNPROC=4096

# Security
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/open-station /var/log/open-station

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=open-station

[Install]
WantedBy=multi-user.target
EOF

    echo -e "${GREEN}✅ Systemd service installed${NC}"

    # Reload systemd
    sudo systemctl daemon-reload

    # Enable service
    sudo systemctl enable open-station

    echo -e "${GREEN}✅ Service enabled${NC}"
}

# Create log directory
create_log_dir() {
    sudo mkdir -p /var/log/open-station
    sudo chown "$SERVICE_USER:$SERVICE_GROUP" /var/log/open-station
}

# Start service
start_service() {
    echo ""
    echo "Starting service..."

    # Check if PostgreSQL and Redis are running
    if ! sudo systemctl is-active --quiet postgresql; then
        echo -e "${YELLOW}⚠️  PostgreSQL is not running. Please start it first:${NC}"
        echo "    sudo systemctl start postgresql"
    fi

    if ! sudo systemctl is-active --quiet redis; then
        echo -e "${YELLOW}⚠️  Redis is not running. Please start it first:${NC}"
        echo "    sudo systemctl start redis"
    fi

    # Start open-station
    sudo systemctl start open-station

    sleep 3

    # Check status
    if sudo systemctl is-active --quiet open-station; then
        echo -e "${GREEN}✅ Service started successfully${NC}"
    else
        echo -e "${RED}Service failed to start. Check logs:${NC}"
        echo "    sudo journalctl -u open-station -n 50"
    fi
}

# Cleanup
cleanup() {
    rm -rf "$TEMP_DIR"
}

# Show result
show_result() {
    echo ""
    echo "=========================================="
    echo "   Installation Complete!"
    echo "=========================================="
    echo ""
    echo "Binary:     $INSTALL_DIR/open-station"
    echo "Config:     $CONFIG_DIR/config.yaml"
    echo "Service:    open-station.service"
    echo ""
    echo "Commands:"
    echo "  Start:    sudo systemctl start open-station"
    echo "  Stop:     sudo systemctl stop open-station"
    echo "  Status:   sudo systemctl status open-station"
    echo "  Logs:     sudo journalctl -u open-station -f"
    echo "  Restart:  sudo systemctl restart open-station"
    echo ""
    echo "Config edit:"
    echo "  sudo vim $CONFIG_DIR/config.yaml"
    echo "  sudo systemctl restart open-station"
    echo ""
    echo "Test API:"
    echo "  curl http://localhost:8080/health"
    echo ""
}

# Main
main() {
    detect_system
    get_latest_version
    download_binary
    install_binary
    install_config
    create_user
    create_log_dir
    install_service
    cleanup

    read -p "Start service now? [y/N]: " START_NOW
    if [[ "$START_NOW" =~ ^[Yy]$ ]]; then
        start_service
    fi

    show_result
}

# Run
main