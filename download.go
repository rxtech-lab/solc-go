package solc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const SOLC_BINARIES_BASE_URL = "https://binaries.soliditylang.org/bin"

// getCacheDir returns the cache directory path (~/.solc)
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(homeDir, "solc"), nil
}

// getCachedBinaryPath returns the full path for a cached binary
func getCachedBinaryPath(version string) (string, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, version, "soljson.js"), nil
}

// ensureCacheDir creates the cache directory structure for a version
func ensureCacheDir(version string) error {
	cacheDir, err := getCacheDir()
	if err != nil {
		return err
	}
	versionDir := filepath.Join(cacheDir, version)
	return os.MkdirAll(versionDir, 0755)
}

// loadCachedBinary loads a binary from cache if it exists
func loadCachedBinary(version string) (string, bool) {
	cachePath, err := getCachedBinaryPath(version)
	if err != nil {
		return "", false
	}

	content, err := os.ReadFile(cachePath)
	if err != nil {
		return "", false
	}

	return string(content), true
}

// saveBinaryToCache saves a binary to the cache
func saveBinaryToCache(version string, content string) error {
	if err := ensureCacheDir(version); err != nil {
		return err
	}

	cachePath, err := getCachedBinaryPath(version)
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, []byte(content), 0644)
}

type VersionList struct {
	Builds   []Build           `json:"builds"`
	Releases map[string]string `json:"releases"`
}

type Build struct {
	Path        string `json:"path"`
	Version     string `json:"version"`
	Build       string `json:"build"`
	LongVersion string `json:"longVersion"`
	Keccak256   string `json:"keccak256"`
	SHA256      string `json:"sha256"`
}

func fetchVersionList() (*VersionList, error) {
	resp, err := http.Get(fmt.Sprintf("%s/list.json", SOLC_BINARIES_BASE_URL))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch version list: HTTP %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read version list response: %w", err)
	}

	var versionList VersionList
	if err := json.Unmarshal(body, &versionList); err != nil {
		return nil, fmt.Errorf("failed to parse version list: %w", err)
	}

	return &versionList, nil
}

func resolveVersion(version string) (string, error) {
	versionList, err := fetchVersionList()
	if err != nil {
		return "", err
	}

	filename, exists := versionList.Releases[version]
	if !exists {
		return "", fmt.Errorf("version %s not found", version)
	}

	return filename, nil
}

func downloadSolcBinary(version, filename string) (string, error) {
	// First check if we have it cached
	if content, found := loadCachedBinary(version); found {
		return content, nil
	}

	// Download from remote
	url := fmt.Sprintf("%s/%s", SOLC_BINARIES_BASE_URL, filename)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download solc binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download solc binary: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read solc binary: %w", err)
	}

	content := string(body)

	// Save to cache for future use
	if err := saveBinaryToCache(version, content); err != nil {
		// Log the error but don't fail the download
		fmt.Fprintf(os.Stderr, "Warning: failed to cache binary for version %s: %v\n", version, err)
	}

	return content, nil
}

func NewWithVersion(version string) (Solc, error) {
	// First, check if we have an embedded binary for this version
	if binaryContent, exists := getEmbeddedBinary(version); exists {
		return New(binaryContent)
	}

	// Fall back to downloading from remote if not embedded
	filename, err := resolveVersion(version)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve version %s: %w", version, err)
	}

	binaryContent, err := downloadSolcBinary(version, filename)
	if err != nil {
		return nil, fmt.Errorf("failed to download solc binary for version %s: %w", version, err)
	}

	return New(binaryContent)
}
