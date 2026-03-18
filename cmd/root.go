package cmd

import (
	"fmt"
	"os"

	"github.com/redpine-ai/connect-cli/internal/version"
	"github.com/spf13/cobra"
)

var (
	flagAPIKey   string
	flagServer   string
	flagJSON     string
	flagJQ       string
	flagPretty   bool
	flagQuiet    bool
	flagInsecure bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "connect",
		Short:         "Connect CLI — MCP client for the Connect platform",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.Full(),
	}

	root.SetVersionTemplate("{{.Version}}\n")

	pf := root.PersistentFlags()
	pf.StringVar(&flagAPIKey, "api-key", "", "API key for authentication (env: CONNECT_API_KEY)")
	pf.StringVar(&flagServer, "server", "", "Server URL (env: CONNECT_SERVER_URL)")
	pf.StringVar(&flagJSON, "json", "", "Force JSON output; optionally select comma-separated fields")
	pf.StringVar(&flagJQ, "jq", "", "Filter/transform JSON output with a jq expression")
	pf.BoolVar(&flagPretty, "pretty", false, "Force human-readable output")
	pf.BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all output except errors")
	pf.BoolVar(&flagInsecure, "insecure", false, "Allow non-HTTPS server URLs")

	// --json can be used with or without a value
	root.PersistentFlags().Lookup("json").NoOptDefVal = "*"

	return root
}

func Execute() {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
