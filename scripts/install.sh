#!/bin/bash
# Install script for dim
# Usage: curl -sSL https://raw.githubusercontent.com/naren-chakraview/dim/master/scripts/install.sh | sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${GREEN}==> $1${NC}"
}

print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_warning() {
    echo -e "${YELLOW}Warning: $1${NC}"
}

# Security note
echo -e "${YELLOW}SECURITY NOTE:${NC}"
echo "This script downloads and executes a pre-built binary from GitHub Releases."
echo "Before running, consider:"
echo "  1. Reviewing this script: https://raw.githubusercontent.com/naren-chakraview/dim/master/scripts/install.sh"
echo "  2. Verifying checksums manually (see GitHub Release page)"
echo "  3. Using 'go install' instead (requires Go): go install github.com/naren-chakraview/dim/cmd/dimd@latest"
echo ""

# Detect OS
OS=$(uname -s)
case "$OS" in
    Linux)
        OS="linux"
        ;;
    Darwin)
        OS="darwin"
        ;;
    MINGW*|MSYS*|CYGWIN*)
        OS="windows"
        ;;
    *)
        print_error "Unsupported OS: $OS"
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

print_header "Detected platform: $OS/$ARCH"

# Determine the latest release
print_header "Fetching latest release information..."
LATEST_RELEASE=$(curl -sSL https://api.github.com/repos/naren-chakraview/dim/releases/latest)
VERSION=$(echo "$LATEST_RELEASE" | grep -oP '"tag_name": "\K[^"]+' | head -1)

if [ -z "$VERSION" ]; then
    print_error "Could not determine latest release version"
    exit 1
fi

print_header "Latest version: $VERSION"

# Build download URL
case "$OS" in
    windows)
        EXT="zip"
        ;;
    *)
        EXT="tar.gz"
        ;;
esac

BINARY_NAME="dim_${VERSION#v}_${OS}_${ARCH}"
DOWNLOAD_URL="https://github.com/naren-chakraview/dim/releases/download/${VERSION}/${BINARY_NAME}.${EXT}"

print_header "Downloading $BINARY_NAME.$EXT..."
TMPDIR=$(mktemp -d)
cd "$TMPDIR"

if ! curl -sSLf -o "$BINARY_NAME.$EXT" "$DOWNLOAD_URL"; then
    print_error "Failed to download $DOWNLOAD_URL"
    rm -rf "$TMPDIR"
    exit 1
fi

# Extract binaries
print_header "Extracting binaries..."
if [ "$EXT" = "zip" ]; then
    unzip -q "$BINARY_NAME.$EXT"
else
    tar xzf "$BINARY_NAME.$EXT"
fi

# Determine install directory
INSTALL_DIR="${INSTALL_DIR:=$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"

# Install binaries
print_header "Installing to $INSTALL_DIR..."
if [ -f "dimd" ]; then
    chmod +x dimd
    cp dimd "$INSTALL_DIR/dimd"
    echo "✓ Installed dimd"
fi

if [ -f "dimctl" ]; then
    chmod +x dimctl
    cp dimctl "$INSTALL_DIR/dimctl"
    echo "✓ Installed dimctl"
fi

# Clean up
cd /
rm -rf "$TMPDIR"

# Check if install dir is in PATH
if [[ ":$PATH:" == *":$INSTALL_DIR:"* ]]; then
    print_header "Installation complete! Run 'dimd --help' to get started."
else
    print_warning "Installation complete, but $INSTALL_DIR is not in your PATH"
    print_warning "Add it with: export PATH=\"$INSTALL_DIR:\$PATH\""
    print_warning "Or move the binaries to /usr/local/bin (requires sudo):"
    print_warning "  sudo mv $INSTALL_DIR/dimd /usr/local/bin/"
    print_warning "  sudo mv $INSTALL_DIR/dimctl /usr/local/bin/"
fi

echo ""
print_header "Next steps:"
echo "  1. Review the docs: https://github.com/naren-chakraview/dim/tree/master/docs"
echo "  2. Build your first route: https://github.com/naren-chakraview/dim/tree/master/docs/GETTING_STARTED.md"
echo "  3. Check out examples: https://github.com/naren-chakraview/dim/tree/master/examples"
