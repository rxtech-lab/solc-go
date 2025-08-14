package solc

import (
	_ "embed"
)

// Embedded Solidity compiler binaries
// These are predownloaded and embedded into the package for better performance

//go:embed embedded-binaries/soljson-v0.8.30+commit.73712a01.js
var solc0830Binary string

//go:embed embedded-binaries/soljson-v0.8.21+commit.d9974bed.js
var solc0821Binary string

// embeddedVersions maps version strings to their embedded binary content
var embeddedVersions = map[string]string{
	"0.8.30": solc0830Binary,
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
