package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/redpine-ai/connect-cli/internal/version"
	"github.com/spf13/cobra"
)

const releasesURL = "https://api.github.com/repos/redpine-ai/connect-cli/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func NewUpdateCmd(f *factory.Factory) *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update connect CLI to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			release, err := getLatestRelease()
			if err != nil {
				return &output.CLIError{
					Code:     "update_check_failed",
					Message:  fmt.Sprintf("Failed to check for updates: %s", err),
					ExitCode: output.ExitServer,
				}
			}
			latest := strings.TrimPrefix(release.TagName, "v")
			current := version.Version
			if latest == current || current == "dev" {
				ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
					"up_to_date": true, "version": current,
				}))
				return nil
			}
			if checkOnly {
				ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
					"up_to_date": false, "current": current, "latest": latest,
				}))
				return &output.CLIError{
					Code:     "update_available",
					Message:  fmt.Sprintf("Update available: %s -> %s", current, latest),
					Hint:     "Run 'connect update' to install",
					ExitCode: 1,
				}
			}
			assetName := fmt.Sprintf("connect-cli_%s_%s_%s", latest, runtime.GOOS, runtime.GOARCH)
			var downloadURL string
			for _, asset := range release.Assets {
				if strings.Contains(asset.Name, assetName) {
					downloadURL = asset.BrowserDownloadURL
					break
				}
			}
			if downloadURL == "" {
				return &output.CLIError{
					Code:     "no_binary",
					Message:  fmt.Sprintf("No binary found for %s/%s", runtime.GOOS, runtime.GOARCH),
					ExitCode: output.ExitError,
				}
			}
			ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
				"message":      fmt.Sprintf("Update available: %s -> %s", current, latest),
				"download_url": downloadURL,
				"note":         "Self-update download not yet implemented. Download manually from GitHub Releases.",
			}))
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check if an update is available")
	return cmd
}

func getLatestRelease() (*githubRelease, error) {
	resp, err := http.Get(releasesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}
