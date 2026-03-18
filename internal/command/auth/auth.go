package auth

import (
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/spf13/cobra"
)

func NewAuthCmd(f *factory.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}
	cmd.AddCommand(NewSetKeyCmd(f, ""))
	cmd.AddCommand(NewStatusCmd(f))
	cmd.AddCommand(NewLogoutCmd(f))
	cmd.AddCommand(NewLoginCmd(f))
	return cmd
}
