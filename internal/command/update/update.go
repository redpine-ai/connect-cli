package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/redpine-ai/connect-cli/internal/version"
	"github.com/spf13/cobra"
)

const (
	releasesURL     = "https://api.github.com/repos/redpine-ai/connect-cli/releases/latest"
	checksumsAsset  = "checksums.txt"
	apiTimeout      = 15 * time.Second
	downloadTimeout = 5 * time.Minute
)

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func NewUpdateCmd(f *factory.Factory) *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update the Redpine CLI to the latest release",
		Long: `Downloads the latest GitHub release for this platform, verifies it against
the release's checksums.txt, and replaces the running binary in place.
Homebrew installs should use 'brew upgrade connect-cli' instead.`,
		Args: cobra.NoArgs,
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
					Hint: "Run 'redpine update' to install", ExitCode: output.ExitError,
				}
			}

			assetPrefix := fmt.Sprintf("connect-cli_%s_%s_%s", latest, runtime.GOOS, runtime.GOARCH)
			var archive, sums *githubAsset
			for i := range release.Assets {
				a := &release.Assets[i]
				switch {
				case strings.HasPrefix(a.Name, assetPrefix) && archive == nil:
					archive = a
				case a.Name == checksumsAsset:
					sums = a
				}
			}
			if archive == nil {
				return &output.CLIError{
					Code: "no_binary", Message: fmt.Sprintf("No binary found for %s/%s", runtime.GOOS, runtime.GOARCH),
					ExitCode: output.ExitError,
				}
			}
			if sums == nil {
				return &output.CLIError{
					Code: "no_checksums", Message: "Release has no checksums.txt; refusing to install an unverified binary",
					ExitCode: output.ExitError,
				}
			}

			fmt.Fprintf(ios.ErrOut, "Downloading v%s...\n", latest)
			archiveData, err := download(archive.BrowserDownloadURL)
			if err != nil {
				return &output.CLIError{Code: "download_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			sumsData, err := download(sums.BrowserDownloadURL)
			if err != nil {
				return &output.CLIError{Code: "download_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			if err := verifyChecksum(archive.Name, archiveData, sumsData); err != nil {
				return &output.CLIError{Code: "checksum_mismatch", Message: err.Error(), ExitCode: output.ExitError}
			}

			var binaryData []byte
			switch {
			case strings.HasSuffix(archive.Name, ".tar.gz"):
				binaryData, err = extractFromTarGz(archiveData, "redpine")
			case strings.HasSuffix(archive.Name, ".zip"):
				binaryData, err = extractFromZip(archiveData, "redpine.exe")
			default:
				err = fmt.Errorf("unknown archive format: %s", archive.Name)
			}
			if err != nil {
				return &output.CLIError{Code: "extract_error", Message: err.Error(), ExitCode: output.ExitError}
			}

			if err := replaceExecutable(binaryData); err != nil {
				return err
			}

			fmt.Fprintf(ios.ErrOut, "Updated: v%s → v%s\n", current, latest)
			return ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
				"updated": true, "from": current, "to": latest,
			}))
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check whether an update is available")
	return cmd
}

func httpGet(url string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "redpine-cli/"+version.Version)
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream;q=0.9, */*;q=0.8")
	return (&http.Client{Timeout: timeout}).Do(req)
}

func getLatestRelease() (*githubRelease, error) {
	resp, err := httpGet(releasesURL, apiTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}
	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// download fetches a release asset. The URL comes from the GitHub releases
// API response, never from user input.
func download(url string) ([]byte, error) {
	resp, err := httpGet(url, downloadTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed (HTTP %d)", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum checks data against the `<sha256>  <name>` line for name in
// a GoReleaser checksums.txt.
func verifyChecksum(name string, data, sums []byte) error {
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			want = strings.ToLower(fields[0])
			break
		}
	}
	if want == "" {
		return fmt.Errorf("%s has no entry for %s", checksumsAsset, name)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", name, got, want)
	}
	return nil
}

// replaceExecutable writes the new binary next to the running one and renames
// it into place atomically.
func replaceExecutable(binaryData []byte) error {
	execPath, err := os.Executable()
	if err != nil {
		return &output.CLIError{Code: "update_error", Message: fmt.Sprintf("Cannot find current binary: %s", err), ExitCode: output.ExitError}
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return &output.CLIError{Code: "update_error", Message: fmt.Sprintf("Cannot resolve binary path: %s", err), ExitCode: output.ExitError}
	}

	dir := filepath.Dir(execPath)
	tmp, err := os.CreateTemp(dir, ".redpine-update-*")
	if err != nil {
		return &output.CLIError{
			Code: "update_error", Message: fmt.Sprintf("Cannot write to %s (try with sudo?): %s", dir, err),
			ExitCode: output.ExitError,
		}
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(binaryData); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return &output.CLIError{Code: "update_error", Message: err.Error(), ExitCode: output.ExitError}
	}
	if err := tmp.Chmod(0755); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return &output.CLIError{Code: "update_error", Message: err.Error(), ExitCode: output.ExitError}
	}
	_ = tmp.Close()

	if err := os.Rename(tmpName, execPath); err != nil {
		_ = os.Remove(tmpName)
		return &output.CLIError{
			Code: "update_error", Message: fmt.Sprintf("Cannot replace binary (try with sudo?): %s", err),
			ExitCode: output.ExitError,
		}
	}
	return nil
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
