#!/bin/bash

# update-embedded.sh - Update embedded.go with new Solidity versions
set -e

if [ -z "$1" ] || [ -z "$2" ]; then
    echo "Usage: $0 <latest_version> <latest_filename>"
    exit 1
fi

LATEST_VERSION="$1"
LATEST_FILENAME="$2"

echo "Updating embedded.go with Solidity $LATEST_VERSION..."

# Generate variable name (e.g., 0.8.30 -> solc0830Binary)
VAR_NAME="solc$(echo $LATEST_VERSION | sed 's/\.//g')Binary"

echo "Generated variable name: $VAR_NAME"

# Backup existing embedded.go if it exists
if [ -f embedded.go ]; then
    cp embedded.go embedded.go.backup
    echo "📁 Backed up existing embedded.go"
fi

# Create new embedded.go file
cat > embedded.go << EOF
package solc

import (
	_ "embed"
)

// Embedded Solidity compiler binaries
// These are predownloaded and embedded into the package for better performance

//go:embed embedded-binaries/$LATEST_FILENAME
var ${VAR_NAME} string

//go:embed embedded-binaries/soljson-v0.8.21+commit.d9974bed.js
var solc0821Binary string

// embeddedVersions maps version strings to their embedded binary content
var embeddedVersions = map[string]string{
	"$LATEST_VERSION": ${VAR_NAME},
	"0.8.21": solc0821Binary,
}

// getEmbeddedBinary returns the embedded binary for a given version if available
func getEmbeddedBinary(version string) (string, bool) {
	binary, exists := embeddedVersions[version]
	return binary, exists
}

// GetEmbeddedVersions returns a list of all embedded Solidity versions
func GetEmbeddedVersions() []string {
	versions := make([]string, 0, len(embeddedVersions))
	for version := range embeddedVersions {
		versions = append(versions, version)
	}
	return versions
}
EOF

# Verify the file was created correctly
if [ ! -f embedded.go ]; then
    echo "❌ Error: Failed to create embedded.go"
    if [ -f embedded.go.backup ]; then
        mv embedded.go.backup embedded.go
        echo "🔄 Restored backup"
    fi
    exit 1
fi

# Check that the file contains expected content
if ! grep -q "$LATEST_VERSION" embedded.go || ! grep -q "$VAR_NAME" embedded.go; then
    echo "❌ Error: embedded.go doesn't contain expected version or variable name"
    if [ -f embedded.go.backup ]; then
        mv embedded.go.backup embedded.go
        echo "🔄 Restored backup"
    fi
    exit 1
fi

echo "✅ Successfully updated embedded.go with version $LATEST_VERSION"

# Clean up backup on success
if [ -f embedded.go.backup ]; then
    rm embedded.go.backup
fi

# Show a preview of the changes
echo "📄 Preview of embedded.go:"
head -n 20 embedded.go
echo "..."
echo "🎉 embedded.go update completed!"