package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	repo             = "terminally-online/shrugged"
	cacheFileName    = "version_check.json"
	checkInterval    = 1 * time.Hour
	githubAPITimeout = 5 * time.Second
)

type ReleaseInfo struct {
	TagName string `json:"tag_name"`
}

type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	LatestTag string    `json:"latest_tag"`
}

// CheckForUpdate compares the current version against the latest GitHub release.
// Returns the latest tag, whether the current version is outdated, and any error.
// Returns ("", false, nil) when version is "dev", the env opt-out is set, or the cache is fresh and current.
func CheckForUpdate(currentVersion string) (string, bool, error) {
	if currentVersion == "dev" {
		return "", false, nil
	}
	if os.Getenv("SHRUGGED_NO_UPDATE_CHECK") == "1" {
		return "", false, nil
	}

	cached := readCache()
	if !cached.CheckedAt.IsZero() && time.Since(cached.CheckedAt) < checkInterval {
		return cached.LatestTag, isOutdated(currentVersion, cached.LatestTag), nil
	}

	release, err := FetchLatestRelease()
	if err != nil {
		return "", false, fmt.Errorf("failed to check for updates: %w", err)
	}

	writeCache(cacheEntry{
		CheckedAt: time.Now(),
		LatestTag: release.TagName,
	})

	return release.TagName, isOutdated(currentVersion, release.TagName), nil
}

// FetchLatestRelease queries the GitHub API for the latest release of shrugged.
func FetchLatestRelease() (*ReleaseInfo, error) {
	client := &http.Client{Timeout: githubAPITimeout}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var info ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &info, nil
}

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "shrugged")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func readCache() cacheEntry {
	dir, err := cacheDir()
	if err != nil {
		return cacheEntry{}
	}
	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		return cacheEntry{}
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return cacheEntry{}
	}
	return entry
}

func writeCache(e cacheEntry) {
	dir, err := cacheDir()
	if err != nil {
		return
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, cacheFileName), data, 0o644)
}

// isOutdated returns true if latest is a higher version than current.
func isOutdated(current, latest string) bool {
	cur, ok1 := parseVersion(current)
	lat, ok2 := parseVersion(latest)
	if !ok1 || !ok2 {
		return false
	}
	for i := range cur {
		if lat[i] > cur[i] {
			return true
		}
		if lat[i] < cur[i] {
			return false
		}
	}
	return false
}

// parseVersion strips an optional "v" prefix and parses "major.minor.patch" into [3]int.
func parseVersion(v string) ([3]int, bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
