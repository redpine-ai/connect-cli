package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"NAME", "DESCRIPTION"}
	rows := [][]string{
		{"analytics--query", "Run SQL queries"},
		{"billing--invoices", "List invoices"},
	}

	RenderTable(&buf, headers, rows)
	out := buf.String()

	if !strings.Contains(out, "analytics--query") {
		t.Error("should contain tool name")
	}
	if !strings.Contains(out, "Run SQL queries") {
		t.Error("should contain description")
	}
}
