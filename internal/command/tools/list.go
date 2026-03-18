package tools

import (
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
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'connect auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}
			client := f.MCPClient(token)
			if err := client.Initialize(); err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			allTools, err := client.ListTools()
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			tc := f.ToolCache()
			tc.Save(allTools)
			upstream := filterUpstreamTools(allTools)
			ios := f.IOStreams()
			ios.WriteJSON(output.NewSuccessEnvelope(upstream))
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
