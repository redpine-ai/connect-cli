package auth

import (
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

			ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
				"authenticated": true,
				"source":        source,
				"token":         masked,
			}))
			return nil
		},
	}
}
