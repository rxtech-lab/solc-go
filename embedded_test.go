package solc

import (
	"strings"
	"testing"
)

func TestEmbeddedVersions(t *testing.T) {
	versions := GetEmbeddedVersions()

	// Should have exactly 2 embedded versions
	if len(versions) != 2 {
		t.Fatalf("Expected 2 embedded versions, got %d", len(versions))
	}

	// Should contain 0.8.30 and 0.8.21
	hasLatest := false
	hasLTS := false

	for _, version := range versions {
		if version == "0.8.30" {
			hasLatest = true
		}
		if version == "0.8.21" {
			hasLTS = true
		}
	}

	if !hasLatest {
		t.Error("Missing embedded version 0.8.30")
	}

	if !hasLTS {
		t.Error("Missing embedded version 0.8.21")
	}
}

func TestEmbeddedBinaryAccess(t *testing.T) {
	// Test that we can access embedded binaries
	binary, exists := getEmbeddedBinary("0.8.30")
	if !exists {
		t.Fatal("0.8.30 embedded binary not found")
	}

	if len(binary) == 0 {
		t.Fatal("0.8.30 embedded binary is empty")
	}

	// Should be a JavaScript file
	if !strings.Contains(binary, "Module") {
		t.Error("0.8.30 binary doesn't appear to be a valid solc emscripten binary")
	}

	// Test LTS version
	binary, exists = getEmbeddedBinary("0.8.21")
	if !exists {
		t.Fatal("0.8.21 embedded binary not found")
	}

	if len(binary) == 0 {
		t.Fatal("0.8.21 embedded binary is empty")
	}

	// Should be a JavaScript file
	if !strings.Contains(binary, "Module") {
		t.Error("0.8.21 binary doesn't appear to be a valid solc emscripten binary")
	}
}

func TestNewWithVersionEmbedded(t *testing.T) {
	// Test creating compiler with embedded version
	solc, err := NewWithVersion("0.8.30")
	if err != nil {
		t.Fatalf("Failed to create solc with embedded version 0.8.30: %v", err)
	}
	defer solc.Close()

	// Should have a valid version
	version := solc.Version()
	if !strings.Contains(version, "0.8.30") {
		t.Errorf("Expected version to contain 0.8.30, got: %s", version)
	}

	// Should have a valid license
	license := solc.License()
	if len(license) == 0 {
		t.Error("License should not be empty")
	}
}
