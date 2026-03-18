package auth

import (
	"os"
	"path/filepath"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewLogoutCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials and terminate session",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token != "" {
				client := f.MCPClient(token)
				cfg, _ := f.Config()
				serverURL := f.ServerFlag
				if serverURL == "" {
					serverURL = cfg.ServerURL
				}
				sc := mcp.DefaultSessionCache(serverURL)
				if sid := sc.Load(); sid != "" {
					client.SetSessionID(sid)
					client.DeleteSession()
					sc.Delete()
				}
			}

			kr := f.Keyring()
			kr.Delete()

			credsPath := filepath.Join(config.ConfigDir(), "credentials.json")
			os.Remove(credsPath)

			ios := f.IOStreams()
			ios.WriteJSON(output.NewSuccessEnvelope(map[string]string{
				"message": "Logged out successfully",
			}))
			return nil
		},
	}
}
