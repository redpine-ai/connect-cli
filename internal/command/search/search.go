package search

import (
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSearchCmd(f *factory.Factory) *cobra.Command {
	var collection string
	var limit int

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documents across collections",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'connect auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}
			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer sc.Save(client.SessionID())
			searchArgs := map[string]interface{}{
				"query": strings.Join(args, " "),
			}
			if collection != "" {
				searchArgs["collection"] = collection
			}
			if limit > 0 {
				searchArgs["limit"] = limit
			}
			result, err := client.CallTool("search", searchArgs)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			ios := f.IOStreams()
			return ios.WriteMCPResult(result, f.JSONFlag != "", f.PrettyFlag)
		},
	}

	cmd.Flags().StringVar(&collection, "collection", "", "Search within a specific collection")
	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	return cmd
}
