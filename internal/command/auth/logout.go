package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials and terminate session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ios := f.IOStreams()
			token, _ := f.Token(f.APIKeyFlag)
			if token != "" {
				client := f.MCPClient(token)
				cfg, _ := f.Config()
				serverURL := f.ServerFlag
				if serverURL == "" {
					serverURL = cfg.ServerURLForEnv()
				}
				sc := mcp.DefaultSessionCache(serverURL)
				if sid := sc.Load(); sid != "" {
					client.SetSessionID(sid)
					if err := client.DeleteSession(); err != nil {
						fmt.Fprintf(ios.ErrOut, "Warning: could not terminate server session: %s\n", err)
					}
					sc.Delete()
				}
			}

			kr := f.Keyring()
			kr.Delete()

			credsPath := filepath.Join(config.ConfigDir(), "credentials.json")
			os.Remove(credsPath)

			fmt.Fprintln(ios.ErrOut, "Logged out successfully")
			return nil
		},
	}
}
