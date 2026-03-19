package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/fuzzy"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

type toolValidationError struct {
	Code        string
	Message     string
	Suggestions []string
	Hint        string
}

func (e *toolValidationError) Error() string { return e.Message }

func NewCallCmd(f *factory.Factory) *cobra.Command {
	var inputJSON string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "call <tool-name> [key=value...]",
		Short: "Call an upstream MCP tool",
		Example: `  redpine tools call analytics--run_query query="SELECT *" limit=10
  echo '{"query": "test"}' | redpine tools call analytics--run_query
  redpine tools call analytics--run_query --input '{"query": "test"}'
  redpine tools call analytics--run_query query="test" --dry-run`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'redpine auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}

			toolName := args[0]

			// Read stdin if piped
			var stdinData []byte
			if inputJSON == "" {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					stdinData, _ = io.ReadAll(os.Stdin)
				}
			}

			toolArgs, err := parseToolArgs(args[1:], inputJSON, stdinData)
			if err != nil {
				return &output.CLIError{Code: "invalid_input", Message: err.Error(), ExitCode: output.ExitInput}
			}

			// Dry run — show what would be sent without executing
			if dryRun {
				payload := map[string]interface{}{
					"jsonrpc": "2.0",
					"method":  "tools/call",
					"params": map[string]interface{}{
						"name":      toolName,
						"arguments": toolArgs,
					},
				}
				ios := f.IOStreams()
				return ios.WriteResult(payload, f.JSONFlag != "", f.PrettyFlag, func(w io.Writer) {
					fmt.Fprintf(w, "Method:    tools/call\n")
					fmt.Fprintf(w, "Tool:      %s\n", toolName)
					fmt.Fprintf(w, "Arguments:\n")
					for k, v := range toolArgs {
						fmt.Fprintf(w, "  %s = %v\n", k, v)
					}
				})
			}

			// Validate against cached tool schemas
			tc := f.ToolCache()
			cachedTools, cacheErr := tc.Load()
			if cacheErr == nil {
				if valErr := validateToolName(toolName, cachedTools); valErr != nil {
					vErr := valErr.(*toolValidationError)
					return &output.CLIError{
						Code: "tool_not_found", Message: vErr.Message,
						Suggestions: vErr.Suggestions, Hint: vErr.Hint, ExitCode: output.ExitInput,
					}
				}
				for _, t := range cachedTools {
					if t.Name == toolName {
						if paramErr := validateParams(toolArgs, t.InputSchema); paramErr != nil {
							return paramErr
						}
						break
					}
				}
			}

			client, sc, err := f.MCPClientWithSession(token)
			if err != nil {
				return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
			}
			defer sc.Save(client.SessionID())

			if cacheErr != nil {
				freshTools, listErr := client.ListTools()
				if listErr == nil {
					tc.Save(freshTools)
				}
			}

			var result *mcp.ToolCallResult
			if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
				var callErr error
				result, callErr = c.CallTool(toolName, toolArgs)
				return callErr
			}); err != nil {
				return &output.CLIError{Code: "tool_error", Message: err.Error(), ExitCode: output.ExitServer}
			}

			ios := f.IOStreams()
			return ios.WriteMCPResult(result, f.JSONFlag != "", f.PrettyFlag)
		},
	}

	cmd.Flags().StringVar(&inputJSON, "input", "", "Tool arguments as JSON string")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be sent without executing")
	return cmd
}

func parseToolArgs(kvArgs []string, jsonInput string, stdinData []byte) (map[string]interface{}, error) {
	// Priority: explicit --input JSON > key=value args > piped stdin auto-wire
	if jsonInput != "" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(jsonInput), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON input: %w", err)
		}
		return result, nil
	}

	result := make(map[string]interface{})

	// Auto-wire: extract fields from piped stdin (previous command's output)
	if len(stdinData) > 0 {
		piped := extractPipedFields(stdinData)
		for k, v := range piped {
			result[k] = v
		}
	}

	// Key=value args override piped values
	for _, arg := range kvArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument %q: expected key=value format", arg)
		}
		result[parts[0]] = coerceValue(parts[1])
	}
	return result, nil
}

// extractPipedFields parses piped JSON from a previous command and extracts
// useful fields. Handles both the envelope format {"status":"ok","data":{...}}
// and raw JSON objects. Extracts ID-like fields (workspace_id, etc.) and
// all top-level string/number/bool values.
func extractPipedFields(data []byte) map[string]interface{} {
	result := make(map[string]interface{})

	// Try envelope format first
	var envelope struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data != nil {
		extractFlat(envelope.Data, result)
		return result
	}

	// Try raw JSON object
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err == nil {
		extractFlat(raw, result)
		return result
	}

	return result
}

// extractFlat pulls scalar values and recursively extracts from nested
// content blocks (MCP tool results).
func extractFlat(src map[string]interface{}, dst map[string]interface{}) {
	for k, v := range src {
		switch val := v.(type) {
		case string:
			dst[k] = val
		case float64:
			dst[k] = val
		case bool:
			dst[k] = val
		case map[string]interface{}:
			// Recurse one level for nested objects
			extractFlat(val, dst)
		}
	}
}

func coerceValue(s string) interface{} {
	// Boolean
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	// Integer
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n
	}
	// Float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// String
	return s
}

func validateToolName(name string, tools []mcp.Tool) error {
	for _, t := range tools {
		if t.Name == name {
			return nil
		}
	}
	if strings.Contains(name, "--") {
		prefix, toolPart, _ := strings.Cut(name, "--")
		prefixes := map[string][]string{}
		for _, t := range tools {
			if strings.Contains(t.Name, "--") {
				p, tn, _ := strings.Cut(t.Name, "--")
				prefixes[p] = append(prefixes[p], tn)
			}
		}
		prefixList := make([]string, 0, len(prefixes))
		for p := range prefixes {
			prefixList = append(prefixList, p)
		}
		if _, exists := prefixes[prefix]; !exists {
			suggestions := fuzzy.ClosestMatches(prefix, prefixList, 3)
			return &toolValidationError{
				Code: "prefix_not_found", Message: fmt.Sprintf("Upstream '%s' not found", prefix),
				Suggestions: suggestions, Hint: "Run 'redpine tools list' to see available tools",
			}
		}
		suggestions := fuzzy.ClosestMatches(toolPart, prefixes[prefix], 3)
		return &toolValidationError{
			Code: "tool_not_found", Message: fmt.Sprintf("Tool '%s' not found on '%s'", toolPart, prefix),
			Suggestions: suggestions, Hint: fmt.Sprintf("Available tools on '%s': %s", prefix, strings.Join(prefixes[prefix], ", ")),
		}
	}
	allNames := make([]string, 0, len(tools))
	for _, t := range tools {
		allNames = append(allNames, t.Name)
	}
	suggestions := fuzzy.ClosestMatches(name, allNames, 3)
	return &toolValidationError{
		Code: "tool_not_found", Message: fmt.Sprintf("Tool '%s' not found", name),
		Suggestions: suggestions, Hint: "Run 'redpine tools list' to see available tools",
	}
}

func validateParams(args map[string]interface{}, schema json.RawMessage) error {
	if len(schema) == 0 {
		return nil
	}
	var s struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schema, &s); err != nil {
		return nil
	}
	for _, req := range s.Required {
		if _, ok := args[req]; !ok {
			var optionals []string
			for name := range s.Properties {
				isRequired := false
				for _, r := range s.Required {
					if r == name {
						isRequired = true
						break
					}
				}
				if !isRequired {
					optionals = append(optionals, name)
				}
			}
			return &output.CLIError{
				Code:     "missing_param",
				Message:  fmt.Sprintf("Missing required parameter '%s'", req),
				Hint:     fmt.Sprintf("Required: %s. Optional: %s. Run 'redpine tools info <tool>' for details", strings.Join(s.Required, ", "), strings.Join(optionals, ", ")),
				ExitCode: output.ExitInput,
			}
		}
	}
	knownParams := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		knownParams = append(knownParams, name)
	}
	for param := range args {
		found := false
		for _, known := range knownParams {
			if param == known {
				found = true
				break
			}
		}
		if !found && len(knownParams) > 0 {
			suggestions := fuzzy.ClosestMatches(param, knownParams, 3)
			return &output.CLIError{
				Code:        "unknown_param",
				Message:     fmt.Sprintf("Unknown parameter '%s'", param),
				Suggestions: suggestions,
				Hint:        fmt.Sprintf("Known parameters: %s. Run 'redpine tools info <tool>' for details", strings.Join(knownParams, ", ")),
				ExitCode:    output.ExitInput,
			}
		}
	}
	return nil
}
