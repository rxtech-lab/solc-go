#!/bin/bash

# download-binaries.sh - Download Solidity compiler binaries
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <latest_version>"
    exit 1
fi

LATEST_VERSION="$1"

echo "Downloading Solidity compiler binaries for version $LATEST_VERSION..."

# Get the filename for the latest version
LATEST_FILENAME=$(curl -s https://binaries.soliditylang.org/bin/list.json | jq -r ".releases[\"$LATEST_VERSION\"]")

if [ "$LATEST_FILENAME" = "null" ] || [ -z "$LATEST_FILENAME" ]; then
    echo "❌ Error: Could not find binary filename for version $LATEST_VERSION"
    exit 1
fi

echo "Latest binary filename: $LATEST_FILENAME"

# Create directory if it doesn't exist
mkdir -p embedded-binaries

# Download the latest version binary
echo "📥 Downloading $LATEST_FILENAME..."
curl -f -L -o "embedded-binaries/$LATEST_FILENAME" "https://binaries.soliditylang.org/bin/$LATEST_FILENAME"

if [ ! -f "embedded-binaries/$LATEST_FILENAME" ]; then
    echo "❌ Error: Failed to download $LATEST_FILENAME"
    exit 1
fi

# Verify the download
FILESIZE=$(stat -f%z "embedded-binaries/$LATEST_FILENAME" 2>/dev/null || stat -c%s "embedded-binaries/$LATEST_FILENAME" 2>/dev/null || echo "0")
if [ "$FILESIZE" -lt 1000000 ]; then  # Less than 1MB is suspicious
    echo "❌ Error: Downloaded file is too small ($FILESIZE bytes), likely corrupted"
    exit 1
fi

echo "✅ Successfully downloaded $LATEST_FILENAME ($FILESIZE bytes)"

# Keep 0.8.21 as a stable LTS version (re-download to ensure integrity)
V0821_FILENAME=$(curl -s https://binaries.soliditylang.org/bin/list.json | jq -r '.releases["0.8.21"]')
echo "📥 Downloading LTS version: $V0821_FILENAME..."
curl -f -L -o "embedded-binaries/$V0821_FILENAME" "https://binaries.soliditylang.org/bin/$V0821_FILENAME"

if [ ! -f "embedded-binaries/$V0821_FILENAME" ]; then
    echo "❌ Error: Failed to download LTS version $V0821_FILENAME"
    exit 1
fi

# Verify the LTS download
LTS_FILESIZE=$(stat -f%z "embedded-binaries/$V0821_FILENAME" 2>/dev/null || stat -c%s "embedded-binaries/$V0821_FILENAME" 2>/dev/null || echo "0")
if [ "$LTS_FILESIZE" -lt 1000000 ]; then
    echo "❌ Error: LTS file is too small ($LTS_FILESIZE bytes), likely corrupted"
    exit 1
fi

echo "✅ Successfully downloaded LTS version $V0821_FILENAME ($LTS_FILESIZE bytes)"

# Export filenames for use by other scripts
echo "latest_filename=$LATEST_FILENAME" >> $GITHUB_OUTPUT
echo "lts_filename=$V0821_FILENAME" >> $GITHUB_OUTPUT

echo "🎉 Binary downloads completed successfully!"