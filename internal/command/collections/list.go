package collections

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewListCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available document collections",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'connect auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}
			client := f.MCPClient(token)
			if err := client.Initialize(); err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			result, err := client.CallTool("list_collections", map[string]interface{}{})
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			ios := f.IOStreams()
			ios.WriteJSON(output.NewSuccessEnvelope(result))
			return nil
		},
	}
}
