package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/command/auth"
	"github.com/redpine-ai/connect-cli/internal/command/balance"
	"github.com/redpine-ai/connect-cli/internal/command/collections"
	"github.com/redpine-ai/connect-cli/internal/command/confirm"
	"github.com/redpine-ai/connect-cli/internal/command/docs"
	"github.com/redpine-ai/connect-cli/internal/command/preview"
	"github.com/redpine-ai/connect-cli/internal/command/search"
	"github.com/redpine-ai/connect-cli/internal/command/tools"
	updatecmd "github.com/redpine-ai/connect-cli/internal/command/update"
	"github.com/redpine-ai/connect-cli/internal/command/whoami"
	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/redpine-ai/connect-cli/internal/update"
	"github.com/redpine-ai/connect-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey string
	flagServer string
	flagJSON   bool
	flagPretty bool
	flagEnv    string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "redpine",
		Short: "Redpine CLI — search Redpine Connect and call MCP tools from the terminal",
		Long: `Redpine CLI — search documents, preview and unlock results, list collections,
and call upstream MCP tools. Built for agents: JSON when piped, readable in a terminal.

Search:
  redpine search <collection> "query"       Search and pay per result
  redpine preview <collection> "query"      Free teasers plus the price to unlock
  redpine confirm <queryId> [resultId...]   Pay for and receive previewed results

Agent quick start:
  1. redpine tools list --json          Get all tools with full input schemas
  2. redpine tools info <tool> --json   Get one tool's schema for param building
  3. redpine tools call <tool> ...      Call a tool (pipe output to chain calls)

Pipe chaining (auto-wires IDs between commands):
  redpine tools call media--create_workspace --input '{"filter":{"query":"X"}}' \
    | redpine tools call media--daily_briefing

Output is JSON when stdout is not a terminal; --json and --pretty force either.`,
		SilenceUsage:               true,
		SilenceErrors:              true,
		SuggestionsMinimumDistance: 2,
		Version:                    version.Full(),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --env was passed standalone, PersistentPreRunE handles it and exits.
			// Otherwise show help.
			return cmd.Help()
		},
	}

	root.SetVersionTemplate("{{.Version}}\n")

	pf := root.PersistentFlags()
	pf.StringVar(&flagAPIKey, "api-key", "", "API key (env: REDPINE_API_KEY, or CONNECT_API_KEY)")
	pf.StringVar(&flagServer, "server", "", "Server URL (env: REDPINE_BASE_URL, or CONNECT_SERVER_URL)")
	pf.BoolVar(&flagJSON, "json", false, "Force the JSON envelope, even in a terminal")
	pf.BoolVar(&flagPretty, "pretty", false, "Force human-readable output, even when piped")

	// Hidden env switch — not shown in help
	pf.StringVar(&flagEnv, "env", "", "")
	root.PersistentFlags().Lookup("env").Hidden = true

	f := factory.New()

	// Handle --env before anything else (works even without a subcommand)
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flagEnv != "" {
			if _, ok := config.EnvURLs[flagEnv]; !ok {
				return &output.CLIError{
					Code:     "invalid_env",
					Message:  fmt.Sprintf("Unknown environment '%s'", flagEnv),
					Hint:     "Valid environments: production, staging",
					ExitCode: output.ExitInput,
				}
			}

			cfg, err := config.Load()
			if err != nil {
				cfg = &config.Config{ServerURL: config.DefaultServerURL}
			}

			if cfg.Environment != flagEnv {
				cfg.Environment = flagEnv
				cfg.ServerURL = config.EnvURLs[flagEnv]
				_ = cfg.Save()

				// Clear auth — tokens are env-specific
				kr := f.Keyring()
				_ = kr.Delete()
				_ = os.Remove(filepath.Join(config.ConfigDir(), "credentials.json"))

				// Clear session cache
				sc := filepath.Join(os.TempDir(), "redpine-session-*")
				matches, _ := filepath.Glob(sc)
				for _, m := range matches {
					_ = os.Remove(m)
				}

				// Clear tool cache
				_ = os.RemoveAll(filepath.Join(config.ConfigDir(), "cache"))

				fmt.Fprintf(os.Stderr, "Switched to %s (%s)\n", flagEnv, config.EnvURLs[flagEnv])
				fmt.Fprintln(os.Stderr, "Auth cleared — run 'redpine auth login' to authenticate.")
				os.Exit(0)
			}
		}

		f.APIKeyFlag = flagAPIKey
		f.ServerFlag = flagServer
		f.JSONFlag = flagJSON
		f.PrettyFlag = flagPretty

		// Version check — skip for update, help, completion, and shell completion hooks
		if shouldSkipUpdateCheck(cmd) {
			return nil
		}

		// A newer release is a notice on stderr, never a block: an outdated
		// binary still works against the server, and blocking here broke
		// scripts and offline commands like `auth set-key`.
		cacheDir := filepath.Join(config.ConfigDir(), "cache")
		if result := update.Check(cacheDir); result != nil && result.IsOutdated {
			fmt.Fprintln(os.Stderr, result.FormatWarning())
		}

		return nil
	}

	root.AddCommand(auth.NewAuthCmd(f))
	root.AddCommand(search.NewSearchCmd(f))
	root.AddCommand(preview.NewPreviewCmd(f))
	root.AddCommand(confirm.NewConfirmCmd(f))
	root.AddCommand(collections.NewCollectionsCmd(f))
	root.AddCommand(balance.NewBalanceCmd(f))
	root.AddCommand(tools.NewToolsCmd(f))
	root.AddCommand(docs.NewDocsCmd(f))
	root.AddCommand(updatecmd.NewUpdateCmd(f))
	root.AddCommand(whoami.NewWhoamiCmd(f))

	return root
}

// shouldSkipUpdateCheck returns true for commands where the update notice
// is noise (update, help, completion subcommands, and Cobra's internal
// shell completion hooks like __complete).
func shouldSkipUpdateCheck(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		name := c.Name()
		if name == "update" || name == "help" || name == "completion" ||
			strings.HasPrefix(name, "__") {
			return true
		}
	}
	return false
}

func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		ios := output.New()
		if cliErr, ok := err.(*output.CLIError); ok {
			if ios.OutputMode(flagJSON, flagPretty) == output.ModeJSON {
				_ = ios.WriteJSON(output.NewErrorEnvelope(cliErr))
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
	// Cobra errors look like: `unknown command "logie" for "redpine"`
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

	// Also check for subcommand context: `unknown command "logie" for "redpine auth"`
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
