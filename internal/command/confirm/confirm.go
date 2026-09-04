package confirm

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewConfirmCmd unlocks the full results of an earlier `redpine preview` via
// the `confirm` tool. This is the call that bills.
func NewConfirmCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:     "confirm <queryId> [resultId...]",
		Aliases: []string{"unlock"},
		Short:   "Pay for and receive the full results of a preview",
		Long: `Unlocks results from a 'redpine preview'. With only a queryId, every
previewed result is unlocked; with result ids, only those. Re-sending an id
you already paid for costs nothing, so a failed confirm is safe to retry
with the same queryId. A preview stays unlockable for 7 days.

This command charges your balance. Check the price on the preview first.`,
		Example: `  redpine confirm qry_a1b2c3d4e5f6
  redpine confirm qry_a1b2c3d4e5f6 abc123 def456
  redpine unlock  qry_a1b2c3d4e5f6            # same command, REST naming`,
		Args: cobra.MinimumNArgs(1),
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

			confirmArgs := map[string]interface{}{"queryId": args[0]}
			if len(args) > 1 {
				confirmArgs["result_ids"] = args[1:]
			}

			var result *mcp.ToolCallResult
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				result, callErr = c.CallTool("confirm", confirmArgs)
				return callErr
			}); err != nil {
				return output.ServerError(err)
			}
			return f.IOStreams().WriteMCPResult(result, f.JSONFlag, f.PrettyFlag)
		},
	}
}
