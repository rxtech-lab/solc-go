#!/bin/bash

# fetch-version.sh - Fetch latest Solidity version and compare with current
set -e

echo "Fetching latest Solidity version..."

# Get the latest version from the official API
LATEST_VERSION=$(curl -s https://binaries.soliditylang.org/wasm/list.json | jq -r '.latestRelease')
echo "latest_version=$LATEST_VERSION" >> $GITHUB_OUTPUT
echo "Latest Solidity version: $LATEST_VERSION"

# Get current embedded version from embedded.go
if [ -f embedded.go ]; then
    # Extract version from the embedded versions map (e.g., "0.8.30": solc0830Binary)
    CURRENT_VERSION=$(grep -o '"[0-9]\+\.[0-9]\+\.[0-9]\+"' embedded.go | head -1 | tr -d '"')
    if [ -z "$CURRENT_VERSION" ]; then
        CURRENT_VERSION="none"
    fi
else
    CURRENT_VERSION="none"
fi

echo "current_version=$CURRENT_VERSION" >> $GITHUB_OUTPUT
echo "Current embedded version: $CURRENT_VERSION"

# Check if update is needed
if [ "$LATEST_VERSION" != "$CURRENT_VERSION" ]; then
    echo "update_needed=true" >> $GITHUB_OUTPUT
    echo "✨ Update needed: $CURRENT_VERSION -> $LATEST_VERSION"
else
    echo "update_needed=false" >> $GITHUB_OUTPUT
    echo "✅ No update needed. Already at latest version: $LATEST_VERSION"
fi