# Scripts

This directory contains automation scripts for maintaining embedded Solidity compiler binaries.

## Scripts Overview

### 🔍 `fetch-version.sh`
**Purpose**: Fetches the latest Solidity version and compares with current embedded version.

**Usage**: `./fetch-version.sh`

**Outputs**:
- `latest_version`: Latest Solidity version from official API
- `current_version`: Currently embedded version in the codebase
- `update_needed`: Boolean indicating if update is required

### 📥 `download-binaries.sh`
**Purpose**: Downloads Solidity compiler binaries for specified version.

**Usage**: `./download-binaries.sh <version>`

**Example**: `./download-binaries.sh 0.8.30`

**Features**:
- Downloads latest version binary
- Downloads 0.8.21 LTS version (for stability)
- Verifies file integrity (size checks)
- Creates `embedded-binaries/` directory if needed

**Outputs**:
- `latest_filename`: Downloaded binary filename
- `lts_filename`: LTS binary filename

### 🔧 `update-embedded.sh`
**Purpose**: Updates `embedded.go` with new Solidity version mappings.

**Usage**: `./update-embedded.sh <version> <filename>`

**Example**: `./update-embedded.sh 0.8.30 soljson-v0.8.30+commit.73712a01.js`

**Features**:
- Generates Go variable names automatically
- Creates backup of existing file
- Validates generated content
- Rolls back on errors

### 📝 `update-readme.sh`
**Purpose**: Updates README.md with new version references.

**Usage**: `./update-readme.sh <version>`

**Example**: `./update-readme.sh 0.8.30`

**Features**:
- Updates feature descriptions
- Updates example code
- Updates performance notes
- Creates backup before changes

## CI/CD Integration

These scripts are used by the GitHub Actions workflow `.github/workflows/update-solidity-binaries.yml` to automatically:

1. **Check for updates** every Sunday at midnight UTC
2. **Download new binaries** when available
3. **Update source code** with new version mappings
4. **Verify builds** work correctly
5. **Create pull requests** for review

## Manual Usage

You can run these scripts manually for development or testing:

```bash
# Check if updates are available
./scripts/fetch-version.sh

# Download binaries for a specific version
./scripts/download-binaries.sh 0.8.25

# Update embedded.go with new version
./scripts/update-embedded.sh 0.8.25 soljson-v0.8.25+commit.cc1e7c.js

# Update README with new version
./scripts/update-readme.sh 0.8.25

# Test the build
go mod tidy && go build .
```

## Error Handling

All scripts include comprehensive error handling:
- File existence checks
- Network error handling
- Content validation
- Automatic rollbacks on failure
- Detailed logging with emojis for clarity

## Dependencies

The scripts require:
- `curl` - for downloading files and API calls
- `jq` - for JSON parsing
- `grep`, `sed` - for text processing
- Standard Unix tools (`stat`, `head`, etc.)

These are typically pre-installed on GitHub Actions runners and most development environments.