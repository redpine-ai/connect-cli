package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"golang.org/x/term"
)

// OutputMode determines how output is rendered.
type OutputMode int

const (
	ModeJSON OutputMode = iota
	ModePretty
)

// IOStreams provides access to stdout and stderr with TTY awareness.
type IOStreams struct {
	Out    io.Writer
	ErrOut io.Writer
	tty    bool
	Color  bool
}

// New creates an IOStreams wired to real stdout/stderr with TTY detection.
func New() *IOStreams {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	colorEnabled := isTTY
	if os.Getenv("NO_COLOR") != "" {
		colorEnabled = false
	}
	if os.Getenv("CLICOLOR_FORCE") != "" {
		colorEnabled = true
	}
	return &IOStreams{
		Out:    os.Stdout,
		ErrOut: os.Stderr,
		tty:    isTTY,
		Color:  colorEnabled,
	}
}

// NewTestIOStreams creates an IOStreams backed by buffers for testing.
func NewTestIOStreams() *IOStreams {
	return &IOStreams{
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
		tty:    false,
	}
}

// IsTTY reports whether stdout is a terminal.
func (ios *IOStreams) IsTTY() bool {
	return ios.tty
}

// OutputMode resolves the effective output mode from flags and TTY state.
// The jsonFlag and prettyFlag correspond to --json and --pretty CLI flags.
func (ios *IOStreams) OutputMode(jsonFlag, prettyFlag bool) OutputMode {
	if jsonFlag {
		return ModeJSON
	}
	if prettyFlag {
		return ModePretty
	}
	if ios.tty {
		return ModePretty
	}
	return ModeJSON
}

// Envelope is the standard JSON response wrapper.
type Envelope struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  *CLIError   `json:"error,omitempty"`
}

// NewSuccessEnvelope wraps data in a success envelope.
func NewSuccessEnvelope(data interface{}) *Envelope {
	return &Envelope{Status: "ok", Data: data}
}

// NewErrorEnvelope wraps a CLIError in an error envelope.
func NewErrorEnvelope(err *CLIError) *Envelope {
	return &Envelope{Status: "error", Error: err}
}

// WriteJSON serialises the envelope as JSON to the output stream.
func (ios *IOStreams) WriteJSON(env *Envelope) error {
	enc := json.NewEncoder(ios.Out)
	return enc.Encode(env)
}
