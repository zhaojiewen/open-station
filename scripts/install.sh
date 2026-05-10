#!/bin/bash

# Open Station - One-click Installer
# Works on Linux and macOS
# Downloads pre-built binary from GitHub releases

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default settings
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/open-station}"
REPO="zhaojiewen/open-station"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --config-dir)
            CONFIG_DIR="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --help)
            echo "Open Station Installer"
            echo ""
            echo "Usage: curl -fsSL https://raw.githubusercontent.com/zhaojiewen/open-station/main/scripts/install.sh | bash"
            echo ""
            echo "Options:"
            echo "  --dir <path>       Installation directory (default: /usr/local/bin)"
            echo "  --config-dir <path> Configuration directory (default: /etc/open-station)"
            echo "  --version <version> Install specific version (default: latest)"
            echo ""
            echo "Examples:"
            echo "  # Default install"
            echo "  curl -fsSL https://raw.githubusercontent.com/zhaojiewen/open-station/main/scripts/install.sh | bash"
            echo ""
            echo "  # Custom directory"
            echo "  curl -fsSL https://raw.githubusercontent.com/zhaojiewen/open-station/main/scripts/install.sh | bash -s -- --dir /opt/open-station"
            echo ""
            echo "  # Specific version"
            echo "  curl -fsSL https://raw.githubusercontent.com/zhaojiewen/open-station/main/scripts/install.sh | bash -s -- --version 1.0.0"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

echo ""
echo -e "${BLUE}=========================================="
echo "   Open Station - One-click Installer"
echo -e "==========================================${NC}"
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

    echo -e "${GREEN}✅ System detected: $OS/$ARCH${NC}"
}

# Get version
get_version() {
    if [ -z "$VERSION" ]; then
        echo "Fetching latest version..."
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/')

        if [ -z "$VERSION" ]; then
            echo -e "${YELLOW}⚠️  Could not fetch latest version, using 'latest'${NC}"
            VERSION="latest"
        else
            echo -e "${GREEN}✅ Latest version: v$VERSION${NC}"
        fi
    else
        echo -e "${GREEN}✅ Installing version: v$VERSION${NC}"
    fi
}

# Download and install
download_install() {
    echo ""
    echo "Downloading Open Station..."

    TEMP_DIR=$(mktemp -d)

    # Build download URL
    if [ "$VERSION" = "latest" ]; then
        DOWNLOAD_URL="https://github.com/$REPO/releases/latest/download/open-station-latest-$OS-$ARCH.tar.gz"
    else
        DOWNLOAD_URL="https://github.com/$REPO/releases/download/v$VERSION/open-station-$VERSION-$OS-$ARCH.tar.gz"
    fi

    echo -e "${BLUE}URL: $DOWNLOAD_URL${NC}"

    # Download
    if ! curl -fSL --progress-bar -o "$TEMP_DIR/package.tar.gz" "$DOWNLOAD_URL"; then
        echo -e "${RED}❌ Download failed!${NC}"
        echo "Please check if the version exists: https://github.com/$REPO/releases"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    echo -e "${GREEN}✅ Download complete${NC}"

    # Extract
    echo "Extracting..."
    tar -xzf "$TEMP_DIR/package.tar.gz" -C "$TEMP_DIR"

    # Find binary
    BINARY="$TEMP_DIR/bin/open-station"
    if [ ! -f "$BINARY" ]; then
        BINARY=$(find "$TEMP_DIR" -name "open-station" -type f | head -1)
    fi

    if [ ! -f "$BINARY" ]; then
        echo -e "${RED}❌ Binary not found in package${NC}"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    # Install binary
    echo ""
    echo "Installing to $INSTALL_DIR..."

    # Create directory if needed
    if [ ! -d "$INSTALL_DIR" ]; then
        sudo mkdir -p "$INSTALL_DIR"
    fi

    # Copy binary
    sudo cp "$BINARY" "$INSTALL_DIR/open-station"
    sudo chmod +x "$INSTALL_DIR/open-station"

    echo -e "${GREEN}✅ Binary installed: $INSTALL_DIR/open-station${NC}"

    # Install configs
    echo ""
    echo "Installing configuration..."

    if [ ! -d "$CONFIG_DIR" ]; then
        sudo mkdir -p "$CONFIG_DIR"
    fi

    # Copy configs from package if available
    if [ -d "$TEMP_DIR/configs" ]; then
        sudo cp -r "$TEMP_DIR/configs" "$CONFIG_DIR/" 2>/dev/null || true
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

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
  claude:
    api_key: ${ANTHROPIC_API_KEY}

plugins:
  enabled: true
EOF
        echo -e "${GREEN}✅ Default config created: $CONFIG_DIR/config.yaml${NC}"
    else
        echo -e "${GREEN}✅ Config directory: $CONFIG_DIR${NC}"
    fi

    # Cleanup
    rm -rf "$TEMP_DIR"
}

# Verify installation
verify() {
    echo ""
    echo "Verifying installation..."

    if "$INSTALL_DIR/open-station" --version 2>&1 | grep -q "Open Station" || "$INSTALL_DIR/open-station" --help 2>&1 | grep -q "config"; then
        echo -e "${GREEN}✅ Installation verified${NC}"
    else
        echo -e "${YELLOW}⚠️  Binary installed but verification incomplete${NC}"
    fi
}

# Show result
show_result() {
    echo ""
    echo -e "${BLUE}=========================================="
    echo "   Installation Complete!"
    echo -e "==========================================${NC}"
    echo ""
    echo "Binary:      $INSTALL_DIR/open-station"
    echo "Config:      $CONFIG_DIR/config.yaml"
    echo ""
    echo -e "${YELLOW}Next steps:${NC}"
    echo ""
    echo "1. Configure Provider API Keys:"
    echo "   sudo vim $CONFIG_DIR/config.yaml"
    echo ""
    echo "2. Start required services (PostgreSQL + Redis):"
    if [ "$OS" = "darwin" ]; then
        echo "   brew services start postgresql@16"
        echo "   brew services start redis"
    else
        echo "   sudo systemctl start postgresql"
        echo "   sudo systemctl start redis"
    fi
    echo ""
    echo "3. Start Open Station:"
    echo "   $INSTALL_DIR/open-station -config $CONFIG_DIR/config.yaml"
    echo ""
    echo "4. Or use Docker (recommended for quick start):"
    echo "   make start"
    echo ""
    echo "Test API:"
    echo "  curl http://localhost:8080/health"
    echo ""
    echo "Documentation: https://github.com/$REPO#readme"
    echo ""
}

# Main
main() {
    detect_system
    get_version
    download_install
    verify
    show_result
}

main