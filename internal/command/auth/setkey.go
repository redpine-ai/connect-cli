package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSetKeyCmd(f *factory.Factory, configDir string) *cobra.Command {
	return &cobra.Command{
		Use:   "set-key [key]",
		Short: "Store an API key for authentication",
		Long:  "Store an API key in the system keyring (or config file fallback). If no key argument is provided, reads from stdin.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var key string
			if len(args) > 0 {
				key = args[0]
			} else {
				scanner := bufio.NewScanner(os.Stdin)
				if scanner.Scan() {
					key = strings.TrimSpace(scanner.Text())
				}
			}

			if key == "" {
				return &output.CLIError{
					Code:     "invalid_input",
					Message:  "No API key provided",
					Hint:     "Usage: connect auth set-key <key> or echo <key> | connect auth set-key",
					ExitCode: output.ExitInput,
				}
			}

			kr := f.Keyring()
			if err := kr.Set(key); err != nil {
				dir := configDir
				if dir == "" {
					dir = config.ConfigDir()
				}
				creds := &config.Credentials{Token: key, Type: "api_key"}
				if err := creds.SaveTo(dir); err != nil {
					return fmt.Errorf("failed to store API key: %w", err)
				}
			}

			fmt.Fprintln(f.IOStreams().ErrOut, "API key stored successfully")
			return nil
		},
	}
}
