package search

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseFilters turns repeated --filter expressions into the flat filter object
// the search API accepts.
//
// Supported forms (the API ANDs top-level keys together):
//
//	issn=1234-5678                          exact match
//	issn=1234-5678,8765-4321                any-of
//	issn!=1234-5678                         exclude
//	publisher!=Elsevier,Springer            exclude any-of
//	journal_metric.2yr_mean_citedness>=5    range
//	publication_date>=2020-01-01            range (dates auto-detected server-side)
//
// Exclusion deliberately mirrors the API's own model — ne / not_in — rather
// than inventing a second way to express it.
//
// Values stay strings unless the comparison is a range, where a numeric value
// is emitted as a number so journal-metric thresholds compare correctly.
// Anything unparseable is passed through as a string for the server to reject
// with a clear message, rather than being guessed at here.
func ParseFilters(exprs []string) (map[string]interface{}, error) {
	if len(exprs) == 0 {
		return nil, nil
	}
	out := map[string]interface{}{}
	for _, expr := range exprs {
		key, op, raw, err := splitExpr(expr)
		if err != nil {
			return nil, err
		}
		if _, clash := out[key]; clash {
			return nil, fmt.Errorf(
				"filter %q: %q given more than once; combine the values with a comma "+
					"or use --filter-json for nested logic", expr, key)
		}
		switch op {
		case "=":
			out[key] = scalarOrList(raw)
		case "!=":
			out[key] = map[string]interface{}{"not": scalarOrList(raw)}
		case ">=", ">", "<=", "<":
			out[key] = map[string]interface{}{rangeOp(op): numberOrString(raw)}
		default:
			return nil, fmt.Errorf("filter %q: unsupported operator %q", expr, op)
		}
	}
	return out, nil
}

// ParseFilterJSON parses a raw filter object, for the structured DSL (OR,
// nesting) that the compact --filter syntax deliberately does not cover.
func ParseFilterJSON(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("--filter-json is not a JSON object: %w", err)
	}
	return parsed, nil
}

// splitExpr finds the operator, longest form first so "!=" is not read as "="
// and ">=" is not read as ">".
func splitExpr(expr string) (key, op, value string, err error) {
	for _, candidate := range []string{"!=", ">=", "<=", "=", ">", "<"} {
		if i := strings.Index(expr, candidate); i > 0 {
			key = strings.TrimSpace(expr[:i])
			value = strings.TrimSpace(expr[i+len(candidate):])
			if key == "" || value == "" {
				return "", "", "", fmt.Errorf(
					"filter %q: expected key%svalue", expr, candidate)
			}
			return key, candidate, value, nil
		}
	}
	return "", "", "", fmt.Errorf(
		"filter %q: expected key=value, key!=value, or key>=value", expr)
}

func rangeOp(op string) string {
	return map[string]string{">=": "gte", ">": "gt", "<=": "lte", "<": "lt"}[op]
}

// scalarOrList emits a list for a comma-separated value so the server treats it
// as any-of, and a plain string otherwise.
func scalarOrList(raw string) interface{} {
	if !strings.Contains(raw, ",") {
		return raw
	}
	parts := []string{}
	for _, p := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	items := make([]interface{}, len(parts))
	for i, p := range parts {
		items[i] = p
	}
	return items
}

func numberOrString(raw string) interface{} {
	if n, err := strconv.ParseFloat(raw, 64); err == nil {
		return n
	}
	return raw
}
