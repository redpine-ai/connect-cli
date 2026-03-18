package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

	cmd := &cobra.Command{
		Use:   "call <tool-name> [key=value...]",
		Short: "Call an upstream MCP tool",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'connect auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}

			toolName := args[0]

			if inputJSON == "" {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					data, _ := io.ReadAll(os.Stdin)
					if len(data) > 0 {
						inputJSON = string(data)
					}
				}
			}

			toolArgs, err := parseToolArgs(args[1:], inputJSON)
			if err != nil {
				return &output.CLIError{Code: "invalid_input", Message: err.Error(), ExitCode: output.ExitInput}
			}

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
				freshTools, err := client.ListTools()
				if err == nil {
					tc.Save(freshTools)
				}
			}

			result, err := client.CallTool(toolName, toolArgs)
			if err != nil {
				return &output.CLIError{Code: "tool_error", Message: err.Error(), ExitCode: output.ExitServer}
			}

			ios := f.IOStreams()
			ios.WriteJSON(output.NewSuccessEnvelope(result))
			return nil
		},
	}

	cmd.Flags().StringVar(&inputJSON, "input", "", "Tool arguments as JSON string")
	return cmd
}

func parseToolArgs(kvArgs []string, jsonInput string) (map[string]interface{}, error) {
	if jsonInput != "" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(jsonInput), &result); err != nil {
			return nil, fmt.Errorf("invalid JSON input: %w", err)
		}
		return result, nil
	}
	result := make(map[string]interface{})
	for _, arg := range kvArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid argument %q: expected key=value format", arg)
		}
		result[parts[0]] = parts[1]
	}
	return result, nil
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
				Suggestions: suggestions, Hint: "Run 'connect tools list' to see available tools",
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
		Suggestions: suggestions, Hint: "Run 'connect tools list' to see available tools",
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
				Hint:     fmt.Sprintf("Required: %s. Optional: %s", strings.Join(s.Required, ", "), strings.Join(optionals, ", ")),
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
				Hint:        fmt.Sprintf("Known parameters: %s", strings.Join(knownParams, ", ")),
				ExitCode:    output.ExitInput,
			}
		}
	}
	return nil
}
