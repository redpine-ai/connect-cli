package tools

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/spf13/cobra"
)

func NewToolsCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Discover and call upstream MCP tools",
	}
	cmd.AddCommand(NewListCmd(f))
	cmd.AddCommand(NewInfoCmd(f))
	cmd.AddCommand(NewSchemaCmd(f))
	cmd.AddCommand(NewCallCmd(f))
	return cmd
}
