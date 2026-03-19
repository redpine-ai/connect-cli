package output

import (
	"bytes"
	"testing"
)

func TestNewTestIOStreams(t *testing.T) {
	ios := NewTestIOStreams()
	if ios.IsTTY() {
		t.Error("test IOStreams should not be TTY")
	}
}

func TestWriteJSON_Success(t *testing.T) {
	var buf bytes.Buffer
	ios := &IOStreams{Out: &buf, ErrOut: &bytes.Buffer{}, tty: false}

	data := map[string]string{"name": "test"}
	err := ios.WriteJSON(NewSuccessEnvelope(data))
	if err != nil {
		t.Fatal(err)
	}

	want := `{"status":"ok","data":{"name":"test"}}` + "\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestWriteJSON_Error(t *testing.T) {
	var buf bytes.Buffer
	ios := &IOStreams{Out: &buf, ErrOut: &bytes.Buffer{}, tty: false}

	cliErr := &CLIError{
		Code:        "not_found",
		Message:     "not found",
		Suggestions: []string{"try this"},
		Hint:        "run help",
	}
	err := ios.WriteJSON(NewErrorEnvelope(cliErr))
	if err != nil {
		t.Fatal(err)
	}

	if buf.Len() == 0 {
		t.Error("expected output")
	}
}

func TestOutputMode_TTY(t *testing.T) {
	ios := &IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}, tty: true}
	if ios.OutputMode(false, false) != ModePretty {
		t.Error("TTY default should be pretty")
	}
}

func TestOutputMode_NonTTY(t *testing.T) {
	ios := &IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}, tty: false}
	if ios.OutputMode(false, false) != ModePretty {
		t.Error("non-TTY default should be pretty (human-readable is always default)")
	}
}

func TestOutputMode_FlagOverrides(t *testing.T) {
	ios := &IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}, tty: true}
	if ios.OutputMode(true, false) != ModeJSON {
		t.Error("--json flag should force JSON in TTY")
	}

	ios2 := &IOStreams{Out: &bytes.Buffer{}, ErrOut: &bytes.Buffer{}, tty: false}
	if ios2.OutputMode(false, true) != ModePretty {
		t.Error("--pretty flag should force pretty in non-TTY")
	}
}
