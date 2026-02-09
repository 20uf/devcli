package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
)

const (
	repoOwner   = "20uf"
	repoName    = "devcli"
	releasesURL = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases"
)

type githubRelease struct {
	TagName    string  `json:"tag_name"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Check queries GitHub for the most recent release and returns whether an update is available.
// If preRelease is false, only stable releases are considered.
func Check(currentVersion string, preRelease bool) (latestVersion string, hasUpdate bool, err error) {
	if !preRelease {
		return checkStable(currentVersion)
	}
	return checkAll(currentVersion)
}

func checkStable(currentVersion string) (string, bool, error) {
	resp, err := http.Get(releasesURL + "/latest")
	if err != nil {
		return "", false, fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("no stable release found (status %d)", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	return compareVersions(currentVersion, release.TagName)
}

func checkAll(currentVersion string) (string, bool, error) {
	resp, err := http.Get(releasesURL + "?per_page=1")
	if err != nil {
		return "", false, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", false, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(releases) == 0 {
		return "", false, fmt.Errorf("no releases found")
	}

	return compareVersions(currentVersion, releases[0].TagName)
}

func compareVersions(currentVersion, latestTag string) (string, bool, error) {
	latest := ensureVPrefix(latestTag)
	current := ensureVPrefix(currentVersion)

	if !semver.IsValid(current) || !semver.IsValid(latest) {
		return strings.TrimPrefix(latest, "v"), current != latest, nil
	}

	hasUpdate := semver.Compare(current, latest) < 0
	return strings.TrimPrefix(latest, "v"), hasUpdate, nil
}

// Apply downloads and replaces the current binary with the specified version.
func Apply(version string) error {
	release, err := fetchRelease(version)
	if err != nil {
		return err
	}

	assetName := buildAssetName()
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no asset found for %s/%s (%s)", runtime.GOOS, runtime.GOARCH, assetName)
	}

	return downloadAndReplace(downloadURL)
}

func fetchRelease(version string) (*githubRelease, error) {
	tag := ensureVPrefix(version)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", repoOwner, repoName, tag)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("release %s not found (status %d)", tag, resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func downloadAndReplace(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "devcli-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	tmpFile.Close()

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), execPath); err != nil {
		// Permission denied — retry with sudo
		if os.IsPermission(err) {
			fmt.Println("Permission denied, retrying with sudo...")
			cmd := exec.Command("sudo", "mv", tmpFile.Name(), execPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if sudoErr := cmd.Run(); sudoErr != nil {
				return fmt.Errorf("failed to replace binary with sudo: %w", sudoErr)
			}
			return nil
		}
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

func buildAssetName() string {
	return fmt.Sprintf("devcli_%s_%s", runtime.GOOS, runtime.GOARCH)
}

func ensureVPrefix(v string) string {
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}
