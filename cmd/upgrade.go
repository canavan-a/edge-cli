package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade edge-cli to the latest release from GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isDevBuild() {
			return fmt.Errorf("upgrade is not available in dev builds (repo not set)")
		}
		fmt.Printf("Current version: %s\n", version)
		fmt.Println("Checking for latest release...")

		latest, err := fetchLatestTag()
		if err != nil {
			return fmt.Errorf("failed to check latest version: %w", err)
		}

		if latest == version {
			fmt.Printf("Already up to date (%s).\n", version)
			return nil
		}

		fmt.Printf("New version available: %s\n", latest)

		suffix, err := platformSuffix()
		if err != nil {
			return err
		}

		url := fmt.Sprintf("https://github.com/%s/releases/download/%s/edge-cli-%s", repo, latest, suffix)
		fmt.Printf("Downloading %s...\n", url)

		self, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not determine current executable path: %w", err)
		}

		// Download to a temp file in /tmp.
		tmp, err := os.CreateTemp("", "edge-cli-upgrade-*")
		if err != nil {
			return err
		}
		tmpPath := tmp.Name()
		defer os.Remove(tmpPath)

		resp, err := http.Get(url)
		if err != nil {
			tmp.Close()
			return fmt.Errorf("download failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			tmp.Close()
			return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
		}

		if _, err := io.Copy(tmp, resp.Body); err != nil {
			tmp.Close()
			return fmt.Errorf("failed to write download: %w", err)
		}
		tmp.Close()

		if err := os.Chmod(tmpPath, 0755); err != nil {
			return err
		}

		// Try a direct move first. If permission is denied, retry with sudo.
		if err := moveFile(tmpPath, self); err != nil {
			fmt.Println("Root access required to replace binary, trying sudo...")
			if sudoErr := sudoMove(tmpPath, self); sudoErr != nil {
				return fmt.Errorf("failed to replace binary: %w", sudoErr)
			}
		}

		fmt.Printf("Upgraded to %s.\n", latest)
		return nil
	},
}

// moveFile moves src to dst within the same filesystem using rename,
// or falls back to a copy+rename using a temp file in the same directory as dst.
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	// Cross-device: write a temp file beside the destination, then rename.
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, "edge-cli-replace-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	in, err := os.Open(src)
	if err != nil {
		tmp.Close()
		return err
	}
	defer in.Close()

	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	tmp.Close()

	if err := os.Chmod(tmp.Name(), 0755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

func sudoMove(src, dst string) error {
	out, err := exec.Command("sudo", "mv", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(out))
	}
	return exec.Command("sudo", "chmod", "+x", dst).Run()
}

func fetchLatestTag() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.TagName == "" {
		return "", fmt.Errorf("no releases found")
	}
	return result.TagName, nil
}

func platformSuffix() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("upgrade only supported on Linux")
	}
	switch runtime.GOARCH {
	case "amd64":
		return "linux-amd64", nil
	case "arm64":
		return "linux-arm64", nil
	case "arm":
		return "linux-armv7", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
}
