package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestCLIError_Error(t *testing.T) {
	err := &CLIError{Code: "test", Message: "something broke"}
	if err.Error() != "something broke" {
		t.Errorf("got %q", err.Error())
	}
}

func TestCLIError_WritePretty(t *testing.T) {
	var buf bytes.Buffer
	err := &CLIError{
		Code:        "tool_not_found",
		Message:     "Upstream 'alleas' not found",
		Suggestions: []string{"allears"},
		Hint:        "Run 'redpine tools list'",
	}
	err.WritePretty(&buf)

	out := buf.String()
	if !strings.Contains(out, "alleas") {
		t.Error("should contain error message")
	}
	if !strings.Contains(out, "allears") {
		t.Error("should contain suggestion")
	}
	if !strings.Contains(out, "redpine tools list") {
		t.Error("should contain hint")
	}
}

func TestCLIError_WritePretty_NoSuggestions(t *testing.T) {
	var buf bytes.Buffer
	err := &CLIError{Code: "auth", Message: "not authenticated"}
	err.WritePretty(&buf)

	if !strings.Contains(buf.String(), "not authenticated") {
		t.Error("should contain message")
	}
	if strings.Contains(buf.String(), "Did you mean") {
		t.Error("should not contain suggestions")
	}
}
