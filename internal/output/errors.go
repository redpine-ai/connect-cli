package output

import (
	"fmt"
	"io"
	"strings"
)

const (
	ExitOK     = 0
	ExitError  = 1
	ExitAuth   = 2
	ExitInput  = 3
	ExitServer = 4
)

type CLIError struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
	Hint        string   `json:"hint,omitempty"`
	DocURL      string   `json:"doc_url,omitempty"`
	RequestID   string   `json:"request_id,omitempty"`
	ExitCode    int      `json:"-"`
}

func (e *CLIError) Error() string {
	return e.Message
}

func (e *CLIError) WritePretty(w io.Writer) {
	fmt.Fprintf(w, "Error: %s\n", e.Message)
	if len(e.Suggestions) > 0 {
		fmt.Fprintf(w, "Did you mean: %s?\n", strings.Join(e.Suggestions, ", "))
	}
	if e.Hint != "" {
		fmt.Fprintf(w, "Hint: %s\n", e.Hint)
	}
}
