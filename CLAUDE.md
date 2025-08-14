# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go library that provides Golang bindings for the Solidity compiler. It uses Emscripten-compiled Solidity binaries executed via V8 JavaScript engine to compile Solidity smart contracts.

## Development Commands

### Build and Test (using Makefile)
```bash
# Build the package
make build

# Run all tests
make test

# Format code
make fmt

# Run all steps (fmt, build, test)
make all

# Clean build artifacts
make clean

# Update dependencies
make tidy

# Run linter (if available)
make lint

# Show help
make help
```

### Direct Go Commands
```bash
# Build the package
go build .

# Run all tests
go test -v

# Run tests for a specific file
go test -v -run TestSolc
go test -v -run TestEmbeddedVersions

# Update dependencies
go mod tidy
```

### Manual Binary Management
```bash
# Check for Solidity version updates
./scripts/fetch-version.sh

# Download binaries for specific version
./scripts/download-binaries.sh 0.8.30

# Update embedded.go with new version
./scripts/update-embedded.sh 0.8.30 soljson-v0.8.30+commit.73712a01.js

# Update README with new version references
./scripts/update-readme.sh 0.8.30
```

## Architecture

### Core Components

**solc.go** - Main compiler interface and implementation:
- `Solc` interface: Core API for license, version, compile, and cleanup with proper error handling
- `baseSolc` struct: V8-based implementation using v8go for JavaScript execution
- Thread-safe compilation with mutex protection and closed state tracking
- Proper resource cleanup with `cleanup()` method and error propagation
- Comprehensive documentation and Go best practices compliance

**embedded.go** - Embedded binary management:
- Uses Go 1.16+ `embed` directive to package Solidity binaries
- Currently embeds 0.8.30 (latest) and 0.8.21 (LTS) versions
- `getEmbeddedBinary()`: Internal function for binary retrieval
- `GetEmbeddedVersions()`: Public API to list embedded versions

**download.go** - Dynamic version support:
- `NewWithVersion()`: Primary public API that checks embedded versions first, falls back to download
- `fetchVersionList()`: Retrieves version mappings from binaries.soliditylang.org API
- `downloadSolcBinary()`: Downloads compiler binaries on-demand
- `resolveVersion()`: Maps semantic versions to specific filenames

**input.go/output.go** - Solidity compiler data structures:
- Type definitions matching Solidity's JSON input/output format
- Supports full compiler feature set (optimization, EVM versions, output selection)

### Binary Management Strategy

1. **Embedded Binaries**: Latest and LTS versions are pre-embedded for instant access
2. **On-Demand Download**: Other versions downloaded from official repository as needed
3. **Automatic Updates**: CI/CD pipeline updates embedded versions weekly

### V8 Integration

The library executes Emscripten-compiled Solidity binaries in V8:
- Creates isolated V8 contexts for each compiler instance using v8go v0.9.0+ API
- Binds Emscripten functions (`version`, `license`, `compile`) to Go with proper function casting
- Handles different function naming conventions across Solidity versions
- JSON marshaling for input/output between Go and JavaScript
- Proper resource disposal using `isolate.Dispose()` instead of deprecated `Close()`

### Automation Scripts

Located in `scripts/` directory, these handle embedded binary maintenance:
- **fetch-version.sh**: Version checking and comparison
- **download-binaries.sh**: Binary downloading with integrity verification  
- **update-embedded.sh**: Smart embedded.go generation with rollback
- **update-readme.sh**: Documentation updates with version references

## Key Dependencies

- **v8go v0.9.0**: V8 JavaScript engine bindings (rogchap.com/v8go) with ARM64 support
- **testify v1.9.0**: Testing framework for assertions and test organization
- **Standard library**: HTTP client, JSON, file I/O

## CI/CD Integration

GitHub Actions workflow (`.github/workflows/update-solidity-binaries.yml`):
- Runs every Sunday at midnight UTC
- Automatically detects new Solidity releases
- Downloads binaries and updates embedded versions
- Creates pull requests with comprehensive validation

## Important Notes

- Go 1.24+ required for embed directive support
- v8go v0.9.0+ provides ARM64 (Apple Silicon) compatibility, resolving previous architecture issues
- Embedded binaries are large (~8-9MB each) but provide zero-latency compilation
- The library maintains full backward compatibility with existing Solidity compiler JSON API
- Thread-safe with proper mutex protection and resource cleanup
- Follows Go best practices with comprehensive error handling and documentation

## Version Strategy

- **0.8.30**: Latest embedded version, automatically updated
- **0.8.21**: Stable LTS embedded version
- **Others**: Downloaded on first use from binaries.soliditylang.org