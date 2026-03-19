package tools

import (
	"fmt"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewListCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available upstream MCP tools",
		Example: `  redpine tools list
  redpine tools list --json`,
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

			var allTools []mcp.Tool
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				allTools, callErr = c.ListTools()
				return callErr
			}); err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			tc := f.ToolCache()
			tc.Save(allTools)
			upstream := filterUpstreamTools(allTools)
			ios := f.IOStreams()
			if ios.OutputMode(f.JSONFlag != "", f.PrettyFlag) == output.ModePretty {
				if len(upstream) > 0 {
					headers := []string{"TOOL", "DESCRIPTION"}
					var rows [][]string
					for _, t := range upstream {
						rows = append(rows, []string{t.Name, t.Description})
					}
					output.RenderTable(ios.Out, headers, rows)
				} else {
					fmt.Fprintln(ios.Out, "No upstream tools. Use 'redpine search' and 'redpine collections' for built-in features.")
				}
			} else {
				ios.WriteJSON(output.NewSuccessEnvelope(upstream))
			}
			return nil
		},
	}
}

func filterUpstreamTools(tools []mcp.Tool) []mcp.Tool {
	var upstream []mcp.Tool
	for _, t := range tools {
		if strings.Contains(t.Name, "--") {
			upstream = append(upstream, t)
		}
	}
	return upstream
}

