package auth

import (
	"strings"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewStatusCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, source := f.Token(f.APIKeyFlag)
			ios := f.IOStreams()

			if token == "" {
				err := &output.CLIError{
					Code:     "not_authenticated",
					Message:  "Not authenticated",
					Hint:     "Run 'connect auth login' or set CONNECT_API_KEY",
					ExitCode: output.ExitAuth,
				}
				ios.WriteJSON(output.NewErrorEnvelope(err))
				return err
			}

			var masked string
			if len(token) < 12 {
				masked = "***"
			} else {
				masked = token[:7] + "..." + token[len(token)-4:]
			}

			// Detect token type
			tokenType := "oauth"
			if strings.HasPrefix(token, "sk_live_") || strings.HasPrefix(token, "sk_test_") {
				tokenType = "api_key"
			}

			result := map[string]interface{}{
				"authenticated": true,
				"source":        source,
				"type":          tokenType,
				"token":         masked,
			}

			// Check if refresh token is available (OAuth only)
			if tokenType == "oauth" {
				creds, err := config.LoadCredentialsFrom(config.ConfigDir())
				if err == nil && creds.RefreshToken != "" {
					result["refresh_available"] = true
				}
			}

			return ios.WriteJSON(output.NewSuccessEnvelope(result))
		},
	}
}
