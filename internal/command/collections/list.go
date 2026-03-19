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
					Hint: "Run 'redpine auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}
			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer sc.Save(client.SessionID())
			result, err := client.CallTool("list_collections", map[string]interface{}{})
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			ios := f.IOStreams()
			return ios.WriteMCPResult(result, f.JSONFlag != "", f.PrettyFlag)
		},
	}
}
