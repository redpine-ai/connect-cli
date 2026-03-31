package tools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

var multiSpaceRe = regexp.MustCompile(`\s+`)

func NewListCmd(f *factory.Factory) *cobra.Command {
	var query string
	var integration string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available upstream MCP tools",
		Example: `  redpine tools list
  redpine tools list --query aircraft
  redpine tools list --integration aviation
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
				allTools, callErr = c.FindTools(query, integration)
				return callErr
			}); err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}

			// Cache for use by info/call commands
			tc := f.ToolCache()
			tc.Save(allTools)

			ios := f.IOStreams()
			if ios.OutputMode(f.JSONFlag != "", f.PrettyFlag) == output.ModePretty {
				if len(allTools) > 0 {
					headers := []string{"TOOL", "DESCRIPTION"}
					var rows [][]string
					for _, t := range allTools {
						desc := shortDesc(t.Description)
						rows = append(rows, []string{t.Name, desc})
					}
					output.RenderTable(ios.Out, headers, rows)
				} else {
					if query != "" {
						fmt.Fprintf(ios.Out, "No tools matching %q. Try a broader search.\n", query)
					} else if integration != "" {
						fmt.Fprintf(ios.Out, "No tools found for integration %q. Run 'redpine tools list' to see all.\n", integration)
					} else {
						fmt.Fprintln(ios.Out, "No tools available. Use 'redpine search' and 'redpine collections' for built-in features.")
					}
				}
			} else {
				ios.WriteJSON(output.NewSuccessEnvelope(allTools))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search tools by keyword")
	cmd.Flags().StringVar(&integration, "integration", "", "Filter by integration prefix")
	return cmd
}

// shortDesc collapses whitespace, extracts the first sentence, and caps at 80 chars.
func shortDesc(s string) string {
	s = strings.TrimSpace(s)
	s = multiSpaceRe.ReplaceAllString(s, " ")

	// Cut at first sentence boundary
	for _, sep := range []string{". ", ".\n"} {
		if idx := strings.Index(s, sep); idx != -1 {
			s = s[:idx+1]
			break
		}
	}

	if len(s) > 80 {
		s = s[:77] + "..."
	}
	return s
}
