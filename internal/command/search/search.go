package search

import (
	"encoding/json"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewSearchCmd(f *factory.Factory) *cobra.Command {
	var limit int
	var filters []string
	var filterJSON string

	cmd := &cobra.Command{
		Use:   "search <collection> <query>",
		Short: "Search documents in a collection",
		Example: `  redpine search redpine-test "how does authentication work"
  redpine search api-docs "rate limiting" --limit 5

  # filter by journal identity (ISSN survives title spelling variants)
  redpine search corpus "crispr" --filter issn=1664-302X
  redpine search corpus "crispr" --filter issn=1664-302X,1932-6203

  # exclude
  redpine search corpus "crispr" --filter 'issn!=1932-6203'
  redpine search corpus "crispr" --filter 'publisher!=Elsevier'

  # DOI (case-insensitive; a doi.org prefix is accepted)
  redpine search corpus "crispr" --filter doi=10.1345/aph.1g425

  # journal metric threshold (OpenAlex 2-year mean citedness, CC0)
  redpine search corpus "crispr" --filter 'journal_metric.2yr_mean_citedness>=5'

  # full DSL for OR / nesting
  redpine search corpus "crispr" --filter-json '{"or":[{"field":"issn","eq":"1664-302X"},{"field":"issn","eq":"1932-6203"}]}'`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'redpine auth login', or set REDPINE_API_KEY (CONNECT_API_KEY still works)", ExitCode: output.ExitAuth,
				}
			}
			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer sc.Save(client.SessionID())

			searchArgs, argErr := BuildSearchArgs(args[0], strings.Join(args[1:], " "), limit, filters, filterJSON)
			if argErr != nil {
				return argErr
			}
			var result *mcp.ToolCallResult
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				result, callErr = c.CallTool("search", searchArgs)
				return callErr
			}); err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			ios := f.IOStreams()
			return ios.WriteMCPResult(result, f.JSONFlag, f.PrettyFlag)
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
	cmd.Flags().StringArrayVar(&filters, "filter", nil,
		"Metadata filter, repeatable. key=value (comma-separates into any-of), "+
			"key!=value to exclude, or key>=N / key<=N for ranges. Indexed fields: "+
			"doc_id, journal, publisher, keywords, publication_date, doi, issn, article_type, "+
			"section, isbn, open_access, license, chapter_number, chapter_title, chapter_authors, "+
			"plus journal_metric.2yr_mean_citedness / h_index / i10_index (range only). "+
			"Other fields work but are matched by scanning")
	cmd.Flags().StringVar(&filterJSON, "filter-json", "",
		"Raw filter object for OR / nested logic, e.g. "+
			`'{"or":[{"field":"issn","eq":"1664-302X"}]}'`)
	return cmd
}

// BuildSearchArgs assembles the argument object for the MCP `search` tool from
// the CLI's flags. Shared with `preview`, which wraps the same arguments.
func BuildSearchArgs(collection, query string, limit int, filters []string, filterJSON string) (map[string]interface{}, *output.CLIError) {
	searchArgs := map[string]interface{}{
		"collection": collection,
		"query":      query,
	}
	if limit > 0 {
		searchArgs["limit"] = limit
	}

	// --filter and --filter-json are mutually exclusive: the compact form
	// builds a flat object and the JSON form may be a structured DSL node,
	// and the two shapes cannot be merged coherently.
	if len(filters) > 0 && strings.TrimSpace(filterJSON) != "" {
		return nil, &output.CLIError{
			Code: "invalid_input", Message: "Use either --filter or --filter-json, not both",
			Hint: "--filter builds a flat filter; --filter-json takes the full DSL", ExitCode: output.ExitInput,
		}
	}
	parsed, err := ParseFilters(filters)
	if err != nil {
		return nil, &output.CLIError{Code: "invalid_input", Message: err.Error(), ExitCode: output.ExitInput}
	}
	if parsed == nil {
		if parsed, err = ParseFilterJSON(filterJSON); err != nil {
			return nil, &output.CLIError{Code: "invalid_input", Message: err.Error(), ExitCode: output.ExitInput}
		}
	}
	if parsed != nil {
		searchArgs["filters"] = parsed
	}
	return searchArgs, nil
}

// completeCollections fetches collection names live for shell completion.
func completeCollections(f *factory.Factory, prefix string) []string {
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
