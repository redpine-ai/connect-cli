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

// NotAuthenticated is the error every server-backed command returns when no
// token could be resolved. One definition so the hint names the right env var.
func NotAuthenticated() *CLIError {
	return &CLIError{
		Code:     "not_authenticated",
		Message:  "Not authenticated",
		Hint:     "Run 'redpine auth login', or set REDPINE_API_KEY (CONNECT_API_KEY still works)",
		ExitCode: ExitAuth,
	}
}

// ServerError wraps a transport or tool failure. A rejected token is reported
// as an auth failure (exit 2) rather than a server error, so scripts that key
// off the exit code re-authenticate instead of retrying.
func ServerError(err error) *CLIError {
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "http 401") || strings.Contains(lower, "unauthorized") {
		return &CLIError{
			Code:     "auth_error",
			Message:  msg,
			Hint:     "Run 'redpine auth login' again, or set REDPINE_API_KEY to a valid key",
			ExitCode: ExitAuth,
		}
	}
	return &CLIError{Code: "server_error", Message: msg, ExitCode: ExitServer}
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
