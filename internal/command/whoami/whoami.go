package whoami

import (
	"fmt"
	"io"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

// SandboxKeyPrefix marks a sandbox API key: same endpoints, canned results,
// no charge. Mirrors the server's SANDBOX_KEY_PREFIX.
const SandboxKeyPrefix = "sk_test_"

// NewWhoamiCmd reports how the CLI is authenticated. `auth status` is the
// same command under another name.
func NewWhoamiCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show who you're authenticated as",
		RunE:  func(cmd *cobra.Command, args []string) error { return Run(f) },
	}
}

// Run is the shared implementation behind `whoami` and `auth status`.
func Run(f *factory.Factory) error {
	token, source := f.Token(f.APIKeyFlag)
	ios := f.IOStreams()

	if token == "" {
		return output.NotAuthenticated()
	}

	var masked string
	if len(token) < 12 {
		masked = "***"
	} else {
		masked = token[:7] + "..." + token[len(token)-4:]
	}

	tokenType := "OAuth"
	sandbox := false
	switch {
	case strings.HasPrefix(token, SandboxKeyPrefix):
		tokenType = "API key (sandbox)"
		sandbox = true
	case strings.HasPrefix(token, "sk_live_"):
		tokenType = "API key"
	}

	hasRefresh := false
	if tokenType == "OAuth" {
		creds, err := config.LoadCredentialsFrom(config.ConfigDir())
		if err == nil && creds.RefreshToken != "" {
			hasRefresh = true
		}
	}

	cfg, _ := f.Config()
	env := ""
	if cfg != nil {
		env = cfg.Environment
	}
	if env == "" {
		env = "production"
	}

	result := map[string]interface{}{
		"authenticated":     true,
		"source":            source,
		"type":              tokenType,
		"sandbox":           sandbox,
		"token":             masked,
		"refresh_available": hasRefresh,
		"environment":       env,
	}

	return ios.WriteResult(result, f.JSONFlag, f.PrettyFlag, func(w io.Writer) {
		fmt.Fprintf(w, "Authenticated (%s)\n", tokenType)
		fmt.Fprintf(w, "  Token:   %s\n", masked)
		fmt.Fprintf(w, "  Source:  %s\n", source)
		if env != "production" {
			fmt.Fprintf(w, "  Env:     %s\n", env)
		}
		if sandbox {
			fmt.Fprintf(w, "  Mode:    sandbox — results are synthetic fixtures, nothing is charged\n")
		}
		if tokenType == "OAuth" {
			if hasRefresh {
				fmt.Fprintf(w, "  Refresh: available\n")
			} else {
				fmt.Fprintf(w, "  Refresh: not available\n")
			}
		}
	})
}
