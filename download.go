package solc

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const SOLC_BINARIES_BASE_URL = "https://binaries.soliditylang.org/wasm"

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

func downloadSolcBinary(filename string) (string, error) {
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

	return string(body), nil
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

	binaryContent, err := downloadSolcBinary(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to download solc binary for version %s: %w", version, err)
	}

	return New(binaryContent)
}
