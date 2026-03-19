package search

import (
	"encoding/json"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSearchCmd(f *factory.Factory) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "search <collection> <query>",
		Short: "Search documents in a collection",
		Example: `  redpine search redpine-test "how does authentication work"
  redpine search api-docs "rate limiting" --limit 5`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'redpine auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}
			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer sc.Save(client.SessionID())

			collection := args[0]
			query := strings.Join(args[1:], " ")

			searchArgs := map[string]interface{}{
				"collection": collection,
				"query":      query,
			}
			if limit > 0 {
				searchArgs["limit"] = limit
			}
			result, err := client.CallTool("search", searchArgs)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			ios := f.IOStreams()
			return ios.WriteMCPResult(result, f.JSONFlag != "", f.PrettyFlag)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				// First arg = collection name — complete from cache or fetch
				return completeCollections(f, toComplete), cobra.ShellCompDirectiveNoFileComp
			}
			// Second arg onwards = query — no completion
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of results")
	return cmd
}

func completeCollections(f *factory.Factory, prefix string) []string {
	// Try to get collection names from tool cache (list_collections result)
	// or from a dedicated collection cache
	tc := f.ToolCache()
	tools, err := tc.Load()
	if err != nil {
		// Try fetching live
		token, _ := f.Token(f.APIKeyFlag)
		if token == "" {
			return nil
		}
		client, sc, clientErr := f.MCPClientWithSession(token)
		if clientErr != nil {
			return nil
		}
		defer sc.Save(client.SessionID())

		// Fetch collections
		result, callErr := client.CallTool("list_collections", map[string]interface{}{})
		if callErr != nil {
			return nil
		}
		return extractCollectionNames(result, prefix)
	}

	// If we have cached tools, try calling list_collections live for names
	// For now, check if there's a collections cache file
	_ = tools
	token, _ := f.Token(f.APIKeyFlag)
	if token == "" {
		return nil
	}
	client, sc, err := f.MCPClientWithSession(token)
	if err != nil {
		return nil
	}
	defer sc.Save(client.SessionID())
	result, callErr := client.CallTool("list_collections", map[string]interface{}{})
	if callErr != nil {
		return nil
	}
	return extractCollectionNames(result, prefix)
}

func extractCollectionNames(result interface{}, prefix string) []string {
	// MCP returns content blocks with text containing collection info
	// Try to parse collection names from the text
	data, err := json.Marshal(result)
	if err != nil {
		return nil
	}

	type contentBlock struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type toolResult struct {
		Content []contentBlock `json:"content"`
	}
	var tr toolResult
	if err := json.Unmarshal(data, &tr); err != nil || len(tr.Content) == 0 {
		return nil
	}

	// Parse lines like "- **collection-name** — description"
	var names []string
	for _, block := range tr.Content {
		for _, line := range strings.Split(block.Text, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "- **") {
				continue
			}
			// Extract name between ** **
			start := strings.Index(line, "**")
			if start < 0 {
				continue
			}
			rest := line[start+2:]
			end := strings.Index(rest, "**")
			if end < 0 {
				continue
			}
			name := rest[:end]
			if prefix == "" || strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
				names = append(names, name)
			}
		}
	}
	return names
}
