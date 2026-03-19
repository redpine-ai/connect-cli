package tools

import (
	"encoding/json"
	"fmt"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/fuzzy"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSchemaCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "schema <tool-name>",
		Short: "Print raw JSON input schema for a tool",
		Example: `  redpine tools schema search
  redpine tools schema media--daily_briefing`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'redpine auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}

			toolName := args[0]

			// Try cache first, fall back to live
			tc := f.ToolCache()
			allTools, cacheErr := tc.Load()
			if cacheErr != nil {
				client, sc, err := f.MCPClientWithSession(token)
				if err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				defer sc.Save(client.SessionID())
				if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
					var callErr error
					allTools, callErr = c.ListTools()
					return callErr
				}); err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				tc.Save(allTools)
			}

			// Find tool
			var tool *mcp.Tool
			for i := range allTools {
				if allTools[i].Name == toolName {
					tool = &allTools[i]
					break
				}
			}

			if tool == nil {
				names := make([]string, 0, len(allTools))
				for _, t := range allTools {
					names = append(names, t.Name)
				}
				suggestions := fuzzy.ClosestMatches(toolName, names, 3)
				return &output.CLIError{
					Code: "tool_not_found", Message: fmt.Sprintf("Tool '%s' not found", toolName),
					Suggestions: suggestions, Hint: "Run 'redpine tools list' to see available tools",
					ExitCode: output.ExitInput,
				}
			}

			// Output raw schema as JSON
			schema := map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"inputSchema": json.RawMessage(tool.InputSchema),
			}

			enc := json.NewEncoder(f.IOStreams().Out)
			enc.SetIndent("", "  ")
			return enc.Encode(schema)
		},
	}
}
