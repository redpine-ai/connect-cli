package auth

import (
	"github.com/redpine-ai/connect-cli/internal/command/whoami"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/spf13/cobra"
)

// NewStatusCmd is `redpine whoami` under the auth namespace.
func NewStatusCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE:  func(cmd *cobra.Command, args []string) error { return whoami.Run(f) },
	}
}
