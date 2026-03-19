package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
		Short: "Update Redpine CLI to the latest version",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			release, err := getLatestRelease()
			if err != nil {
				return &output.CLIError{
					Code: "update_check_failed", Message: fmt.Sprintf("Failed to check for updates: %s", err),
					ExitCode: output.ExitServer,
				}
			}
			latest := strings.TrimPrefix(release.TagName, "v")
			current := version.Version

			if latest == current || current == "dev" {
				fmt.Fprintf(ios.ErrOut, "Already up to date (v%s)\n", current)
				return ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
					"up_to_date": true, "version": current,
				}))
			}

			if checkOnly {
				return &output.CLIError{
					Code: "update_available", Message: fmt.Sprintf("Update available: v%s → v%s", current, latest),
					Hint: "Run 'redpine update' to install", ExitCode: 1,
				}
			}

			// Find the right archive
			assetPrefix := fmt.Sprintf("connect-cli_%s_%s_%s", latest, runtime.GOOS, runtime.GOARCH)
			var downloadURL, assetName string
			for _, asset := range release.Assets {
				if strings.HasPrefix(asset.Name, assetPrefix) {
					downloadURL = asset.BrowserDownloadURL
					assetName = asset.Name
					break
				}
			}
			if downloadURL == "" {
				return &output.CLIError{
					Code: "no_binary", Message: fmt.Sprintf("No binary found for %s/%s", runtime.GOOS, runtime.GOARCH),
					ExitCode: output.ExitError,
				}
			}

			// Download
			fmt.Fprintf(ios.ErrOut, "Downloading v%s...\n", latest)
			resp, err := http.Get(downloadURL)
			if err != nil {
				return &output.CLIError{Code: "download_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return &output.CLIError{
					Code: "download_error", Message: fmt.Sprintf("Download failed (HTTP %d)", resp.StatusCode),
					ExitCode: output.ExitServer,
				}
			}

			archiveData, err := io.ReadAll(resp.Body)
			if err != nil {
				return &output.CLIError{Code: "download_error", Message: err.Error(), ExitCode: output.ExitServer}
			}

			// Extract binary from archive
			var binaryData []byte
			if strings.HasSuffix(assetName, ".tar.gz") {
				binaryData, err = extractFromTarGz(archiveData, "redpine")
			} else if strings.HasSuffix(assetName, ".zip") {
				binaryData, err = extractFromZip(archiveData, "redpine.exe")
			} else {
				err = fmt.Errorf("unknown archive format: %s", assetName)
			}
			if err != nil {
				return &output.CLIError{Code: "extract_error", Message: err.Error(), ExitCode: output.ExitError}
			}

			// Replace current binary
			execPath, err := os.Executable()
			if err != nil {
				return &output.CLIError{Code: "update_error", Message: fmt.Sprintf("Cannot find current binary: %s", err), ExitCode: output.ExitError}
			}
			execPath, err = filepath.EvalSymlinks(execPath)
			if err != nil {
				return &output.CLIError{Code: "update_error", Message: fmt.Sprintf("Cannot resolve binary path: %s", err), ExitCode: output.ExitError}
			}

			// Atomic replace: write temp file next to binary, then rename
			dir := filepath.Dir(execPath)
			tmp, err := os.CreateTemp(dir, ".connect-update-*")
			if err != nil {
				return &output.CLIError{
					Code: "update_error", Message: fmt.Sprintf("Cannot write to %s (try with sudo?): %s", dir, err),
					ExitCode: output.ExitError,
				}
			}
			tmpName := tmp.Name()

			if _, err := tmp.Write(binaryData); err != nil {
				tmp.Close()
				os.Remove(tmpName)
				return &output.CLIError{Code: "update_error", Message: err.Error(), ExitCode: output.ExitError}
			}
			if err := tmp.Chmod(0755); err != nil {
				tmp.Close()
				os.Remove(tmpName)
				return &output.CLIError{Code: "update_error", Message: err.Error(), ExitCode: output.ExitError}
			}
			tmp.Close()

			if err := os.Rename(tmpName, execPath); err != nil {
				os.Remove(tmpName)
				return &output.CLIError{
					Code: "update_error", Message: fmt.Sprintf("Cannot replace binary (try with sudo?): %s", err),
					ExitCode: output.ExitError,
				}
			}

			fmt.Fprintf(ios.ErrOut, "Updated: v%s → v%s\n", current, latest)
			return ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
				"updated": true, "from": current, "to": latest,
			}))
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

func extractFromTarGz(data []byte, binaryName string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip error: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar error: %w", err)
		}
		if filepath.Base(hdr.Name) == binaryName && hdr.Typeflag == tar.TypeReg {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary '%s' not found in archive", binaryName)
}

func extractFromZip(data []byte, binaryName string) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip error: %w", err)
	}
	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary '%s' not found in archive", binaryName)
}
