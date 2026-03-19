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

func NewWhoamiCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show who you're authenticated as",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, source := f.Token(f.APIKeyFlag)
			ios := f.IOStreams()

			if token == "" {
				return &output.CLIError{
					Code:     "not_authenticated",
					Message:  "Not authenticated",
					Hint:     "Run 'redpine auth login' or set CONNECT_API_KEY",
					ExitCode: output.ExitAuth,
				}
			}

			var masked string
			if len(token) < 12 {
				masked = "***"
			} else {
				masked = token[:7] + "..." + token[len(token)-4:]
			}

			tokenType := "OAuth"
			if strings.HasPrefix(token, "sk_live_") || strings.HasPrefix(token, "sk_test_") {
				tokenType = "API Key"
			}

			hasRefresh := false
			if tokenType == "OAuth" {
				creds, err := config.LoadCredentialsFrom(config.ConfigDir())
				if err == nil && creds.RefreshToken != "" {
					hasRefresh = true
				}
			}

			result := map[string]interface{}{
				"authenticated":    true,
				"source":           source,
				"type":             tokenType,
				"token":            masked,
				"refresh_available": hasRefresh,
			}

			return ios.WriteResult(result, f.JSONFlag != "", f.PrettyFlag, func(w io.Writer) {
				fmt.Fprintf(w, "Authenticated (%s)\n", tokenType)
				fmt.Fprintf(w, "  Token:   %s\n", masked)
				fmt.Fprintf(w, "  Source:  %s\n", source)
				if tokenType == "OAuth" {
					if hasRefresh {
						fmt.Fprintf(w, "  Refresh: available\n")
					} else {
						fmt.Fprintf(w, "  Refresh: not available\n")
					}
				}
			})
		},
	}
}
