#!/bin/bash

# Open Station - Multi-platform Build Script
# Cross-compiles binaries for Linux, macOS, and Windows

set -e

# Version information
VERSION=${VERSION:-"dev"}
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# Output directory
DIST_DIR="dist"
RELEASE_DIR="release"

# Platforms to build
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm/6"
    "linux/arm/7"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "=========================================="
echo "   Open Station - Multi-platform Build"
echo "=========================================="
echo ""
echo "Version:    $VERSION"
echo "Commit:     $GIT_COMMIT"
echo "Build Time: $BUILD_TIME"
echo "Go Version: $GO_VERSION"
echo ""

# Clean previous builds
clean() {
    echo "Cleaning previous builds..."
    rm -rf $DIST_DIR $RELEASE_DIR
    mkdir -p $DIST_DIR $RELEASE_DIR
}

# Embed version information
embed_version() {
    echo "Embedding version information..."

    # Create version.go if not exists
    VERSION_FILE="internal/version/version.go"
    mkdir -p internal/version

    cat > $VERSION_FILE << EOF
package version

var (
    Version   = "$VERSION"
    Commit    = "$GIT_COMMIT"
    BuildTime = "$BUILD_TIME"
    GoVersion = "$GO_VERSION"
)

func GetVersionInfo() map[string]string {
    return map[string]string{
        "version":    Version,
        "commit":     Commit,
        "build_time": BuildTime,
        "go_version": GoVersion,
    }
}
EOF

    echo "✅ Version information embedded"
}

# Build for specific platform
build_platform() {
    local GOOS=$1
    local GOARCH=$2
    local GOARM=${3:-}

    local OUTPUT_NAME="open-station-${VERSION}-${GOOS}-${GOARCH}"
    if [ -n "$GOARM" ]; then
        OUTPUT_NAME="open-station-${VERSION}-${GOOS}-${GOARCH}v${GOARM}"
    fi

    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo "Building for $GOOS/$GOARCH${GOARM:+v$GOARM}..."

    # Set environment variables
    export GOOS=$GOOS
    export GOARCH=$GOARCH
    if [ -n "$GOARM" ]; then
        export GOARM=$GOARM
    fi

    # Build with optimizations
    local LDFLAGS="-s -w -X internal/version.Version=$VERSION -X internal/version.Commit=$GIT_COMMIT -X internal/version.BuildTime=$BUILD_TIME"

    go build -ldflags "$LDFLAGS" -trimpath -o "$DIST_DIR/$OUTPUT_NAME" ./cmd/server

    # Reset environment
    unset GOOS GOARCH GOARM

    echo "✅ Built: $OUTPUT_NAME"
}

# Build all platforms
build_all() {
    echo ""
    echo "Building for all platforms..."
    echo ""

    for PLATFORM in "${PLATFORMS[@]}"; do
        IFS='/' read -r GOOS GOARCH GOARM <<< "$PLATFORM"
        build_platform "$GOOS" "$GOARCH" "$GOARM"
    done

    echo ""
    echo "All builds completed!"
}

# Create release packages
create_packages() {
    echo ""
    echo "Creating release packages..."
    echo ""

    for BINARY in $DIST_DIR/open-station-*; do
        if [ -f "$BINARY" ]; then
            local BASENAME=$(basename "$BINARY")
            local PACKAGE_NAME="${BASENAME%.exe}"

            # Create tar.gz for Unix, zip for Windows
            if [[ "$BASENAME" == *"windows"* ]]; then
                # Windows package
                local ZIP_NAME="${RELEASE_DIR}/${PACKAGE_NAME}.zip"

                # Create temp directory
                local TEMP_DIR="temp_package_${PACKAGE_NAME}"
                mkdir -p "$TEMP_DIR"

                # Copy binary and docs
                cp "$BINARY" "$TEMP_DIR/"
                cp README.md "$TEMP_DIR/" 2>/dev/null || echo "# Open Station" > "$TEMP_DIR/README.md"
                cp LICENSE "$TEMP_DIR/" 2>/dev/null || true
                cp -r configs "$TEMP_DIR/" 2>/dev/null || true
                cp -r docs "$TEMP_DIR/" 2>/dev/null || true

                # Create zip
                cd "$TEMP_DIR"
                zip -r "../$ZIP_NAME" . 2>/dev/null || true
                cd ..
                rm -rf "$TEMP_DIR"

                echo "✅ Package: $ZIP_NAME"
            else
                # Unix package
                local TAR_NAME="${RELEASE_DIR}/${PACKAGE_NAME}.tar.gz"

                # Create temp directory
                local TEMP_DIR="temp_package_${PACKAGE_NAME}"
                mkdir -p "$TEMP_DIR/bin"

                # Copy binary
                cp "$BINARY" "$TEMP_DIR/bin/open-station"
                chmod +x "$TEMP_DIR/bin/open-station"

                # Copy docs and configs
                cp README.md "$TEMP_DIR/" 2>/dev/null || echo "# Open Station" > "$TEMP_DIR/README.md"
                cp LICENSE "$TEMP_DIR/" 2>/dev/null || true
                cp -r configs "$TEMP_DIR/" 2>/dev/null || true

                # Create install script
                cat > "$TEMP_DIR/install.sh" << 'INSTALLSCRIPT'
#!/bin/bash
# Open Station Binary Installer

set -e

echo "Installing Open Station..."

# Detect system
OS=$(uname -s)
ARCH=$(uname -m)

# Install binary
sudo mkdir -p /usr/local/bin
sudo cp bin/open-station /usr/local/bin/open-station
sudo chmod +x /usr/local/bin/open-station

# Install config (optional)
if [ -d "configs" ]; then
    sudo mkdir -p /etc/open-station
    sudo cp -r configs/* /etc/open-station/
fi

echo "✅ Open Station installed to /usr/local/bin/open-station"
echo ""
echo "Configuration directory: /etc/open-station"
echo ""
echo "Quick start:"
echo "  open-station -config /etc/open-station/config.yaml"
echo ""
INSTALLSCRIPT
                chmod +x "$TEMP_DIR/install.sh"

                # Create systemd service file
                cat > "$TEMP_DIR/open-station.service" << 'SERVICEFILE'
[Unit]
Description=Open Station - Enterprise AI Gateway
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=open-station
Group=open-station
ExecStart=/usr/local/bin/open-station -config /etc/open-station/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
SERVICEFILE

                # Create tar.gz
                tar -czf "$TAR_NAME" -C "$TEMP_DIR" .
                rm -rf "$TEMP_DIR"

                echo "✅ Package: $TAR_NAME"
            fi
        fi
    done

    # Calculate checksums
    echo ""
    echo "Generating checksums..."
    cd $RELEASE_DIR
    sha256sum * > checksums.txt 2>/dev/null || shasum -a 256 * > checksums.txt
    cd ..

    echo "✅ Checksums generated: $RELEASE_DIR/checksums.txt"
}

# Generate release notes
generate_release_notes() {
    echo ""
    echo "Generating release notes..."

    cat > "${RELEASE_DIR}/RELEASE_NOTES.md" << EOF
# Open Station Release v${VERSION}

## Build Information
- Version: ${VERSION}
- Commit: ${GIT_COMMIT}
- Build Time: ${BUILD_TIME}
- Go Version: ${GO_VERSION}

## Downloads

### Linux
- **AMD64**: open-station-${VERSION}-linux-amd64.tar.gz
- **ARM64**: open-station-${VERSION}-linux-arm64.tar.gz
- **ARM v6**: open-station-${VERSION}-linux-armv6.tar.gz (Raspberry Pi Zero)
- **ARM v7**: open-station-${VERSION}-linux-armv7.tar.gz (Raspberry Pi 3/4)

### macOS
- **Intel**: open-station-${VERSION}-darwin-amd64.tar.gz
- **Apple Silicon**: open-station-${VERSION}-darwin-arm64.tar.gz

### Windows
- **AMD64**: open-station-${VERSION}-windows-amd64.zip
- **ARM64**: open-station-${VERSION}-windows-arm64.zip

## Installation

### Quick Install (Linux/macOS)
\`\`\`bash
# Download and extract
curl -LO https://github.com/zhaojiewen/open-station/releases/download/v${VERSION}/open-station-${VERSION}-$(uname -s)-$(uname -m).tar.gz
tar xzf open-station-${VERSION}-*.tar.gz

# Install
./install.sh
\`\`\`

### Docker
\`\`\`bash
docker pull zhaojiewen/open-station:${VERSION}
docker run -d -p 8080:8080 zhaojiewen/open-station:${VERSION}
\`\`\`

### Homebrew (macOS)
\`\`\`bash
brew tap zhaojiewen/open-station
brew install open-station
\`\`\`\`

## Verification
Verify downloads with checksums.txt:
\`\`\`bash
sha256sum -c checksums.txt
\`\`\`

## Features
- Multi-provider AI Gateway (OpenAI, Claude, Gemini, DeepSeek, GLM)
- MCP Protocol Support for Claude Code integration
- Enterprise billing and usage tracking
- Advanced load balancing with 8 strategies
- Circuit breaker for provider failover
- Real-time metrics and monitoring

## Changelog
See CHANGELOG.md for detailed changes.
EOF

    echo "✅ Release notes generated"
}

# Main build process
main() {
    clean
    embed_version
    build_all
    create_packages
    generate_release_notes

    echo ""
    echo "=========================================="
    echo "   Build Complete!"
    echo "=========================================="
    echo ""
    echo "Distribution packages in: $RELEASE_DIR/"
    echo ""
    ls -lh $RELEASE_DIR/
    echo ""
    echo "Total size: $(du -sh $RELEASE_DIR | cut -f1)"
}

# Run with optional version argument
if [ -n "$1" ]; then
    VERSION="$1"
fi

main