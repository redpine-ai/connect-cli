package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/fuzzy"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewInfoCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "info <tool-name>",
		Short: "Show tool details — parameters, types, and usage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'connect auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}

			toolName := args[0]

			// Try cache first, fall back to live fetch
			tc := f.ToolCache()
			allTools, cacheErr := tc.Load()
			if cacheErr != nil {
				client, sc, err := f.MCPClientWithSession(token)
				if err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				defer sc.Save(client.SessionID())
				allTools, err = client.ListTools()
				if err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				tc.Save(allTools)
			}

			// Find the tool
			var tool *mcp.Tool
			for i := range allTools {
				if allTools[i].Name == toolName {
					tool = &allTools[i]
					break
				}
			}

			if tool == nil {
				// Fuzzy suggest
				names := make([]string, 0, len(allTools))
				for _, t := range allTools {
					names = append(names, t.Name)
				}
				suggestions := fuzzy.ClosestMatches(toolName, names, 3)
				return &output.CLIError{
					Code: "tool_not_found", Message: fmt.Sprintf("Tool '%s' not found", toolName),
					Suggestions: suggestions, Hint: "Run 'connect tools list' to see available tools",
					ExitCode: output.ExitInput,
				}
			}

			ios := f.IOStreams()
			if ios.OutputMode(f.JSONFlag != "", f.PrettyFlag) == output.ModePretty {
				renderToolInfo(ios.Out, tool, ios.Color)
			} else {
				ios.WriteJSON(output.NewSuccessEnvelope(tool))
			}
			return nil
		},
	}
}

type paramInfo struct {
	Name        string
	Type        string
	Description string
	Required    bool
	Default     interface{}
	Enum        []string
	Minimum     *float64
	Maximum     *float64
}

func renderToolInfo(w io.Writer, tool *mcp.Tool, color bool) {
	bold := func(s string) string {
		if color {
			return "\033[1m" + s + "\033[0m"
		}
		return s
	}
	dim := func(s string) string {
		if color {
			return "\033[2m" + s + "\033[0m"
		}
		return s
	}
	green := func(s string) string {
		if color {
			return "\033[32m" + s + "\033[0m"
		}
		return s
	}
	yellow := func(s string) string {
		if color {
			return "\033[33m" + s + "\033[0m"
		}
		return s
	}

	// Header
	fmt.Fprintf(w, "%s\n", bold(tool.Name))
	if tool.Description != "" {
		fmt.Fprintf(w, "%s\n", tool.Description)
	}
	fmt.Fprintln(w)

	// Parse schema
	params := parseParams(tool.InputSchema)
	if len(params) == 0 {
		fmt.Fprintln(w, dim("No parameters"))
		return
	}

	// Separate required and optional
	var required, optional []paramInfo
	for _, p := range params {
		if p.Required {
			required = append(required, p)
		} else {
			optional = append(optional, p)
		}
	}

	// Print required params
	if len(required) > 0 {
		fmt.Fprintf(w, "%s\n", bold("Required:"))
		for _, p := range required {
			printParam(w, p, green, dim)
		}
		fmt.Fprintln(w)
	}

	// Print optional params
	if len(optional) > 0 {
		fmt.Fprintf(w, "%s\n", bold("Optional:"))
		for _, p := range optional {
			printParam(w, p, yellow, dim)
		}
		fmt.Fprintln(w)
	}

	// Usage example
	fmt.Fprintf(w, "%s\n", bold("Usage:"))
	example := "connect tools call " + tool.Name
	for _, p := range required {
		example += fmt.Sprintf(" %s=<%s>", p.Name, p.Type)
	}
	fmt.Fprintf(w, "  %s\n", example)
}

func printParam(w io.Writer, p paramInfo, nameColor, dimColor func(string) string) {
	typeStr := p.Type

	// Add constraints
	var constraints []string
	if p.Minimum != nil && p.Maximum != nil {
		constraints = append(constraints, fmt.Sprintf("range: %g–%g", *p.Minimum, *p.Maximum))
	} else if p.Minimum != nil {
		constraints = append(constraints, fmt.Sprintf("min: %g", *p.Minimum))
	} else if p.Maximum != nil {
		constraints = append(constraints, fmt.Sprintf("max: %g", *p.Maximum))
	}
	if len(p.Enum) > 0 {
		constraints = append(constraints, "values: "+strings.Join(p.Enum, ", "))
	}
	if p.Default != nil {
		constraints = append(constraints, fmt.Sprintf("default: %v", p.Default))
	}

	constraintStr := ""
	if len(constraints) > 0 {
		constraintStr = " " + dimColor("("+strings.Join(constraints, ", ")+")")
	}

	fmt.Fprintf(w, "  %s %s%s\n", nameColor(p.Name), dimColor(typeStr), constraintStr)
	if p.Description != "" {
		fmt.Fprintf(w, "    %s\n", p.Description)
	}
}

func parseParams(schema json.RawMessage) []paramInfo {
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

	requiredSet := make(map[string]bool)
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	var params []paramInfo
	for name, propRaw := range s.Properties {
		var prop struct {
			Type        interface{} `json:"type"`
			Description string      `json:"description"`
			Default     interface{} `json:"default"`
			Enum        []string    `json:"enum"`
			Minimum     *float64    `json:"minimum"`
			Maximum     *float64    `json:"maximum"`
		}
		json.Unmarshal(propRaw, &prop)

		typeStr := "any"
		switch v := prop.Type.(type) {
		case string:
			typeStr = v
		case []interface{}:
			var types []string
			for _, t := range v {
				if s, ok := t.(string); ok {
					types = append(types, s)
				}
			}
			typeStr = strings.Join(types, "|")
		}

		params = append(params, paramInfo{
			Name:        name,
			Type:        typeStr,
			Description: prop.Description,
			Required:    requiredSet[name],
			Default:     prop.Default,
			Enum:        prop.Enum,
			Minimum:     prop.Minimum,
			Maximum:     prop.Maximum,
		})
	}

	// Sort: required first, then alphabetical
	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}
		return params[i].Name < params[j].Name
	})

	return params
}
