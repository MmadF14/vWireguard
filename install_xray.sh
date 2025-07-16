#!/bin/bash

# Xray-core installation script for vWireguard

echo "Installing xray-core for vWireguard..."

# Detect architecture
ARCH=$(uname -m)
case $ARCH in
    x86_64)
        ARCH="64"
        ;;
    aarch64|arm64)
        ARCH="arm64-v8a"
        ;;
    armv7l)
        ARCH="arm32-v7a"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Download latest xray-core
echo "Downloading xray-core for $ARCH..."
LATEST_VERSION=$(curl -s https://api.github.com/repos/XTLS/Xray-core/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
DOWNLOAD_URL="https://github.com/XTLS/Xray-core/releases/download/${LATEST_VERSION}/Xray-linux-${ARCH}.zip"

echo "Latest version: $LATEST_VERSION"
echo "Download URL: $DOWNLOAD_URL"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
cd $TEMP_DIR

# Download and extract
wget -O xray.zip "$DOWNLOAD_URL"
if [ $? -ne 0 ]; then
    echo "Failed to download xray-core"
    exit 1
fi

unzip xray.zip
if [ $? -ne 0 ]; then
    echo "Failed to extract xray-core"
    exit 1
fi

# Install xray
sudo mv xray /usr/local/bin/
sudo chmod +x /usr/local/bin/xray

# Verify installation
if [ -f /usr/local/bin/xray ]; then
    echo "xray-core installed successfully!"
    echo "Version: $(/usr/local/bin/xray version)"
else
    echo "Installation failed!"
    exit 1
fi

# Clean up
cd /
rm -rf $TEMP_DIR

echo "Installation completed. Please restart vWireguard service:"
echo "sudo systemctl restart vwireguard" 