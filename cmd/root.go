package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/command/auth"
	"github.com/redpine-ai/connect-cli/internal/command/collections"
	"github.com/redpine-ai/connect-cli/internal/command/docs"
	"github.com/redpine-ai/connect-cli/internal/command/search"
	"github.com/redpine-ai/connect-cli/internal/command/tools"
	updatecmd "github.com/redpine-ai/connect-cli/internal/command/update"
	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/redpine-ai/connect-cli/internal/update"
	"github.com/redpine-ai/connect-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey   string
	flagServer   string
	flagJSON     string
	flagPretty   bool
	flagQuiet    bool
	flagInsecure bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:                        "connect",
		Short:                      "Connect CLI — MCP client for the Connect platform",
		SilenceUsage:               true,
		SilenceErrors:              true,
		SuggestionsMinimumDistance: 2,
		Version:                    version.Full(),
	}

	root.SetVersionTemplate("{{.Version}}\n")

	pf := root.PersistentFlags()
	pf.StringVar(&flagAPIKey, "api-key", "", "API key for authentication (env: CONNECT_API_KEY)")
	pf.StringVar(&flagServer, "server", "", "Server URL (env: CONNECT_SERVER_URL)")
	pf.StringVar(&flagJSON, "json", "", "Force JSON output; optionally select comma-separated fields")
	pf.BoolVar(&flagPretty, "pretty", false, "Force human-readable output")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all output except errors")
	pf.BoolVar(&flagInsecure, "insecure", false, "Allow non-HTTPS server URLs")

	root.PersistentFlags().Lookup("json").NoOptDefVal = "*"

	f := factory.New()
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		f.APIKeyFlag = flagAPIKey
		f.ServerFlag = flagServer
		f.JSONFlag = flagJSON
		f.PrettyFlag = flagPretty
		f.QuietFlag = flagQuiet
		f.InsecureFlag = flagInsecure

		// Version check — skip for update, help, completion
		name := cmd.Name()
		if name == "update" || name == "help" || name == "completion" {
			return nil
		}

		cacheDir := filepath.Join(config.ConfigDir(), "cache")
		result := update.Check(cacheDir)
		if result != nil && result.IsOutdated {
			return &output.CLIError{
				Code:     "update_required",
				Message:  result.FormatWarning(),
				ExitCode: output.ExitError,
			}
		}

		return nil
	}

	root.AddCommand(auth.NewAuthCmd(f))
	root.AddCommand(collections.NewCollectionsCmd(f))
	root.AddCommand(search.NewSearchCmd(f))
	root.AddCommand(tools.NewToolsCmd(f))
	root.AddCommand(docs.NewDocsCmd(f))
	root.AddCommand(updatecmd.NewUpdateCmd(f))

	return root
}

func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		ios := output.New()
		if cliErr, ok := err.(*output.CLIError); ok {
			if ios.OutputMode(false, false) == output.ModeJSON {
				ios.WriteJSON(output.NewErrorEnvelope(cliErr))
			} else {
				cliErr.WritePretty(ios.ErrOut)
			}
			os.Exit(cliErr.ExitCode)
		}
		// For unknown command errors, include Cobra's suggestions
		errMsg := err.Error()
		fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
		if suggestions := findSuggestions(root, errMsg); suggestions != "" {
			fmt.Fprintln(os.Stderr, suggestions)
		}
		os.Exit(1)
	}
}

// findSuggestions extracts a misspelled command name from the error and
// returns suggestion text using Cobra's built-in SuggestionsFor.
func findSuggestions(root *cobra.Command, errMsg string) string {
	// Cobra errors look like: `unknown command "logie" for "connect"`
	// Extract the unknown command name
	const prefix = "unknown command \""
	idx := strings.Index(errMsg, prefix)
	if idx < 0 {
		return ""
	}
	rest := errMsg[idx+len(prefix):]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return ""
	}
	unknown := rest[:end]

	// Also check for subcommand context: `unknown command "logie" for "connect auth"`
	// Find the parent command
	cmd := root
	const forPrefix = "\" for \""
	forIdx := strings.Index(rest, forPrefix)
	if forIdx >= 0 {
		parentPart := rest[forIdx+len(forPrefix):]
		parentEnd := strings.Index(parentPart, "\"")
		if parentEnd >= 0 {
			parentName := parentPart[:parentEnd]
			// Walk to the parent command
			parts := strings.Fields(parentName)
			for _, p := range parts[1:] { // skip root name
				for _, sub := range cmd.Commands() {
					if sub.Name() == p {
						cmd = sub
						break
					}
				}
			}
		}
	}

	suggestions := cmd.SuggestionsFor(unknown)
	if len(suggestions) == 0 {
		return ""
	}
	return fmt.Sprintf("\nDid you mean:\n  %s", strings.Join(suggestions, "\n  "))
}
