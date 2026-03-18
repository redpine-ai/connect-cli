package collections

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/spf13/cobra"
)

func NewCollectionsCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collections",
		Short: "Manage document collections",
	}
	cmd.AddCommand(NewListCmd(f))
	return cmd
}
