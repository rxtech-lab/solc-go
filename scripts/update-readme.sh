#!/bin/bash

# update-readme.sh - Update README.md with new Solidity version
set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <latest_version>"
    exit 1
fi

LATEST_VERSION="$1"

echo "Updating README.md with Solidity $LATEST_VERSION..."

# Backup existing README.md
if [ -f README.md ]; then
    cp README.md README.md.backup
    echo "📁 Backed up existing README.md"
else
    echo "❌ Error: README.md not found"
    exit 1
fi

# Update README to reflect new embedded versions
# Replace version numbers in various contexts:
# 1. In feature descriptions (e.g., "Latest Solidity (0.8.30)")
# 2. In performance notes (e.g., "**0.8.30** (latest)")
# 3. In example code comments

sed -i.tmp "s/Latest Solidity ([0-9]*\.[0-9]*\.[0-9]*)/Latest Solidity ($LATEST_VERSION)/g" README.md
sed -i.tmp "s/\*\*[0-9]*\.[0-9]*\.[0-9]*\*\* (latest)/**$LATEST_VERSION** (latest)/g" README.md

# Update version in example code if present
sed -i.tmp "s/NewWithVersion(\"[0-9]*\.[0-9]*\.[0-9]*\")/NewWithVersion(\"$LATEST_VERSION\")/g" README.md

# Clean up sed backup file
rm -f README.md.tmp

# Verify changes were made
if ! grep -q "$LATEST_VERSION" README.md; then
    echo "⚠️  Warning: README.md may not have been updated correctly"
    echo "   No references to $LATEST_VERSION found in the file"
fi

# Show what was changed
echo "📄 Changes made to README.md:"
if [ -f README.md.backup ]; then
    echo "--- Before ---"
    grep -n "0\.8\.[0-9]\+" README.md.backup | head -5 || true
    echo "--- After ---"
    grep -n "$LATEST_VERSION" README.md | head -5 || true
fi

# Clean up backup on success
if [ -f README.md.backup ]; then
    rm README.md.backup
fi

echo "✅ Successfully updated README.md with version $LATEST_VERSION"
echo "🎉 README.md update completed!"