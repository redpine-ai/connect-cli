package balance

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewBalanceCmd calls the free `get_balance` tool: credit balance, trial
// status, and the dashboard and purchase URLs.
func NewBalanceCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show your credit balance and trial status (free)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return output.NotAuthenticated()
			}
			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return output.ServerError(err)
			}
			defer sc.Save(client.SessionID())

			var result *mcp.ToolCallResult
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				result, callErr = c.CallTool("get_balance", map[string]interface{}{})
				return callErr
			}); err != nil {
				return output.ServerError(err)
			}
			return f.IOStreams().WriteMCPResult(result, f.JSONFlag, f.PrettyFlag)
		},
	}
}
