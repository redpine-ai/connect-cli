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
			allTools, err := client.ListTools()
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			tc := f.ToolCache()
			tc.Save(allTools)
			upstream := filterUpstreamTools(allTools)
			builtin := filterBuiltinTools(allTools)
			ios := f.IOStreams()
			if ios.OutputMode(f.JSONFlag != "", f.PrettyFlag) == output.ModePretty {
				if len(upstream) > 0 {
					fmt.Fprintln(ios.Out, "Upstream tools:")
					headers := []string{"TOOL", "DESCRIPTION"}
					var rows [][]string
					for _, t := range upstream {
						rows = append(rows, []string{t.Name, t.Description})
					}
					output.RenderTable(ios.Out, headers, rows)
				} else {
					fmt.Fprintln(ios.Out, "No upstream MCP tools registered.")
					fmt.Fprintln(ios.Out, "Ask your admin to register upstream servers in Connect.")
				}
				if len(builtin) > 0 {
					fmt.Fprintln(ios.Out)
					fmt.Fprintln(ios.Out, "Built-in (use via dedicated commands):")
					for _, t := range builtin {
						fmt.Fprintf(ios.Out, "  %-25s %s\n", t.Name, t.Description)
					}
					fmt.Fprintln(ios.Out)
					fmt.Fprintln(ios.Out, "Run 'redpine search' and 'redpine collections list' to use built-in tools.")
				}
			} else {
				ios.WriteJSON(output.NewSuccessEnvelope(map[string]interface{}{
					"upstream": upstream,
					"builtin":  builtin,
				}))
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

func filterBuiltinTools(tools []mcp.Tool) []mcp.Tool {
	var builtin []mcp.Tool
	for _, t := range tools {
		if !strings.Contains(t.Name, "--") {
			builtin = append(builtin, t)
		}
	}
	return builtin
}
