package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var errPermissionDenied = errors.New("permission denied")

// PlatformTarget returns the release asset name for the current OS and architecture.
func PlatformTarget() string {
	name := fmt.Sprintf("shrugged-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

// DownloadAndReplace downloads the latest release binary, verifies its checksum,
// and atomically replaces the currently running binary.
func DownloadAndReplace(latestTag string) error {
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("could not resolve symlinks: %w", err)
	}

	if err := probeWriteAccess(binaryPath); err != nil {
		if errors.Is(err, errPermissionDenied) {
			return fmt.Errorf("shrugged is installed at %s — re-run with sudo", binaryPath)
		}
		return err
	}

	assetName := PlatformTarget()
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", repo, latestTag)
	binaryURL := baseURL + "/" + assetName
	checksumURL := baseURL + "/checksums.txt"

	tmpPath, err := downloadFile(binaryURL, filepath.Dir(binaryPath))
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := verifyChecksum(assetName, tmpPath, checksumURL); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	if err := atomicReplace(tmpPath, binaryPath); err != nil {
		return err
	}

	return nil
}

func probeWriteAccess(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return errPermissionDenied
		}
		return fmt.Errorf("cannot write to %s: %w", path, err)
	}
	f.Close()
	return nil
}

func downloadFile(url string, dir string) (string, error) {
	client := &http.Client{Timeout: 120 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d for %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp(dir, "shrugged-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}

	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}

	return tmp.Name(), nil
}

func verifyChecksum(assetName, binaryPath, checksumURL string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to fetch checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums.txt returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksums: %w", err)
	}

	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			expectedHash = fields[0]
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s in checksums.txt", assetName)
	}

	f, err := os.Open(binaryPath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	return nil
}

func atomicReplace(src, dst string) error {
	if err := os.Chmod(src, 0o755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}
	return nil
}
