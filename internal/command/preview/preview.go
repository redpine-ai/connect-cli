package preview

import (
	"strings"

	"github.com/redpine-ai/connect-cli/internal/command/search"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

// NewPreviewCmd runs a search through the `preview` tool: teaser rows plus
// the cost to unlock them, free of charge. The returned queryId is what
// `redpine confirm` takes.
func NewPreviewCmd(f *factory.Factory) *cobra.Command {
	var limit int
	var filters []string
	var filterJSON string
	var tool string

	cmd := &cobra.Command{
		Use:   "preview <collection> <query>",
		Short: "Preview a search for free: teaser rows and the cost to unlock them",
		Long: `Runs the same search as 'redpine search' but returns teasers and a price
instead of billing you. Nothing is charged and no quota slot is spent.

The response carries a queryId, valid for 7 days. Pass it to
'redpine confirm' to pay for and receive the full results; pass result ids
to confirm to unlock only some of them.`,
		Example: `  redpine preview corpus "crispr delivery"
  redpine preview corpus "crispr delivery" --limit 5 --filter issn=1664-302X
  redpine preview corpus "crispr delivery" --json | jq '.data'`,
		Args: cobra.MinimumNArgs(2),
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

			searchArgs, argErr := search.BuildSearchArgs(args[0], strings.Join(args[1:], " "), limit, filters, filterJSON)
			if argErr != nil {
				return argErr
			}
			previewArgs := map[string]interface{}{
				"tool_name": tool,
				"arguments": searchArgs,
			}

			var result *mcp.ToolCallResult
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				result, callErr = c.CallTool("preview", previewArgs)
				return callErr
			}); err != nil {
				return output.ServerError(err)
			}
			return f.IOStreams().WriteMCPResult(result, f.JSONFlag, f.PrettyFlag)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Metadata filter, repeatable; same syntax as 'redpine search --filter'")
	cmd.Flags().StringVar(&filterJSON, "filter-json", "", "Raw filter object for OR / nested logic")
	cmd.Flags().StringVar(&tool, "tool", "search", "Search tool to preview: 'search' or a 'search-<collection>' tool")
	return cmd
}
