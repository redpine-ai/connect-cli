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
		Example: `  redpine tools info search
  redpine tools info media--daily_briefing
  redpine tools info media--create_workspace --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			token, _ := f.Token(f.APIKeyFlag)
			if token == "" {
				return &output.CLIError{
					Code: "not_authenticated", Message: "Not authenticated",
					Hint: "Run 'redpine auth login' or set CONNECT_API_KEY", ExitCode: output.ExitAuth,
				}
			}

			toolName := args[0]

			tc := f.ToolCache()
			allTools, cacheErr := tc.Load()
			if cacheErr != nil {
				client, sc, err := f.MCPClientWithSession(token)
				if err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				defer sc.Save(client.SessionID())
				if err := f.RunWithRefresh(client, sc, func(c *mcp.Client) error {
					var callErr error
					allTools, callErr = c.ListTools()
					return callErr
				}); err != nil {
					return &output.CLIError{Code: "server_error", Message: err.Error(), ExitCode: output.ExitServer}
				}
				tc.Save(allTools)
			}

			var tool *mcp.Tool
			for i := range allTools {
				if allTools[i].Name == toolName {
					tool = &allTools[i]
					break
				}
			}

			if tool == nil {
				names := make([]string, 0, len(allTools))
				for _, t := range allTools {
					names = append(names, t.Name)
				}
				suggestions := fuzzy.ClosestMatches(toolName, names, 3)
				return &output.CLIError{
					Code: "tool_not_found", Message: fmt.Sprintf("Tool '%s' not found", toolName),
					Suggestions: suggestions, Hint: "Run 'redpine tools list' to see available tools",
					ExitCode: output.ExitInput,
				}
			}

			ios := f.IOStreams()
			if ios.OutputMode(f.JSONFlag != "", f.PrettyFlag) == output.ModeJSON {
				// JSON mode: same as schema — full raw schema
				schema := map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
					"inputSchema": json.RawMessage(tool.InputSchema),
				}
				enc := json.NewEncoder(ios.Out)
				enc.SetIndent("", "  ")
				return enc.Encode(schema)
			}

			renderToolInfo(ios.Out, tool, ios.Color)
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
	Children    []paramInfo // nested object properties
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

	// Parse schema (recursive)
	params := parseParams(tool.InputSchema)
	if len(params) == 0 {
		fmt.Fprintln(w, dim("No parameters"))
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s\n", bold("Usage:"))
		fmt.Fprintf(w, "  redpine tools call %s\n", tool.Name)
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

	if len(required) > 0 {
		fmt.Fprintf(w, "%s\n", bold("Required:"))
		for _, p := range required {
			printParam(w, p, green, dim, "  ")
		}
		fmt.Fprintln(w)
	}

	if len(optional) > 0 {
		fmt.Fprintf(w, "%s\n", bold("Optional:"))
		for _, p := range optional {
			printParam(w, p, yellow, dim, "  ")
		}
		fmt.Fprintln(w)
	}

	// Usage example
	fmt.Fprintf(w, "%s\n", bold("Usage:"))
	example := "redpine tools call " + tool.Name
	for _, p := range required {
		if p.Type == "object" {
			example += ` --input '{"` + p.Name + `": {...}}'`
		} else {
			example += fmt.Sprintf(" %s=<%s>", p.Name, p.Type)
		}
	}
	fmt.Fprintf(w, "  %s\n", example)
}

func printParam(w io.Writer, p paramInfo, nameColor, dimColor func(string) string, indent string) {
	typeStr := p.Type

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

	fmt.Fprintf(w, "%s%s %s%s\n", indent, nameColor(p.Name), dimColor(typeStr), constraintStr)
	if p.Description != "" {
		fmt.Fprintf(w, "%s  %s\n", indent, p.Description)
	}

	// Print nested children
	if len(p.Children) > 0 {
		for _, child := range p.Children {
			marker := dimColor("·")
			if child.Required {
				marker = nameColor("·")
			}
			fmt.Fprintf(w, "%s  %s ", indent, marker)

			childConstraints := []string{}
			if len(child.Enum) > 0 {
				childConstraints = append(childConstraints, strings.Join(child.Enum, ", "))
			}
			if child.Minimum != nil || child.Maximum != nil {
				if child.Minimum != nil && child.Maximum != nil {
					childConstraints = append(childConstraints, fmt.Sprintf("%g–%g", *child.Minimum, *child.Maximum))
				} else if child.Minimum != nil {
					childConstraints = append(childConstraints, fmt.Sprintf("min %g", *child.Minimum))
				} else {
					childConstraints = append(childConstraints, fmt.Sprintf("max %g", *child.Maximum))
				}
			}
			if child.Default != nil {
				childConstraints = append(childConstraints, fmt.Sprintf("default: %v", child.Default))
			}

			extra := ""
			if len(childConstraints) > 0 {
				extra = " " + dimColor("("+strings.Join(childConstraints, ", ")+")")
			}

			desc := ""
			if child.Description != "" {
				desc = " — " + child.Description
			}

			fmt.Fprintf(w, "%s %s%s%s\n", child.Name, dimColor(child.Type), extra, desc)

			// Recurse one more level if needed
			if len(child.Children) > 0 {
				for _, grandchild := range child.Children {
					gcDesc := ""
					if grandchild.Description != "" {
						gcDesc = " — " + grandchild.Description
					}
					fmt.Fprintf(w, "%s      %s %s%s\n", indent, grandchild.Name, dimColor(grandchild.Type), gcDesc)
				}
			}
		}
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
		p := parseProperty(name, propRaw, requiredSet[name])
		params = append(params, p)
	}

	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}
		return params[i].Name < params[j].Name
	})

	return params
}

func parseProperty(name string, raw json.RawMessage, required bool) paramInfo {
	var prop struct {
		Type        interface{}                `json:"type"`
		Description string                     `json:"description"`
		Default     interface{}                `json:"default"`
		Enum        []string                   `json:"enum"`
		Minimum     *float64                   `json:"minimum"`
		Maximum     *float64                   `json:"maximum"`
		Properties  map[string]json.RawMessage `json:"properties"`
		Required    []string                   `json:"required"`
		Items       json.RawMessage            `json:"items"`
	}
	json.Unmarshal(raw, &prop)

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

	// For arrays, append item type
	if typeStr == "array" && len(prop.Items) > 0 {
		var itemType struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(prop.Items, &itemType) == nil && itemType.Type != "" {
			typeStr = itemType.Type + "[]"
		}
	}

	p := paramInfo{
		Name:        name,
		Type:        typeStr,
		Description: prop.Description,
		Required:    required,
		Default:     prop.Default,
		Enum:        prop.Enum,
		Minimum:     prop.Minimum,
		Maximum:     prop.Maximum,
	}

	// Recurse into nested object properties
	if len(prop.Properties) > 0 {
		childRequired := make(map[string]bool)
		for _, r := range prop.Required {
			childRequired[r] = true
		}
		for childName, childRaw := range prop.Properties {
			child := parseProperty(childName, childRaw, childRequired[childName])
			p.Children = append(p.Children, child)
		}
		sort.Slice(p.Children, func(i, j int) bool {
			if p.Children[i].Required != p.Children[j].Required {
				return p.Children[i].Required
			}
			return p.Children[i].Name < p.Children[j].Name
		})
	}

	return p
}
