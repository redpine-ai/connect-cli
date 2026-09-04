package collections

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewCollectionsCmd lists the collections the key can search. `collections`
// and `collections list` are the same command; the subcommand exists because
// the README and muscle memory both reach for it.
func NewCollectionsCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "List available document collections",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return run(f) },
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available document collections",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return run(f) },
	})
	return cmd
}

func run(f *factory.Factory) error {
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
		result, callErr = c.CallTool("list_collections", map[string]interface{}{})
		return callErr
	}); err != nil {
		return output.ServerError(err)
	}
	return f.IOStreams().WriteMCPResult(result, f.JSONFlag, f.PrettyFlag)
}
