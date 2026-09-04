package output

import (
	"bytes"
	"strings"
	"testing"
)

type fakeBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type fakeResult struct {
	Content []fakeBlock `json:"content"`
	IsError bool        `json:"isError,omitempty"`
}

func TestWriteMCPResult_ToolErrorBecomesCLIError(t *testing.T) {
	out := &bytes.Buffer{}
	ios := &IOStreams{Out: out, ErrOut: &bytes.Buffer{}}
	res := fakeResult{Content: []fakeBlock{{Type: "text", Text: "Preview expired"}}, IsError: true}

	err := ios.WriteMCPResult(res, true, false)
	cliErr, ok := err.(*CLIError)
	if !ok {
		t.Fatalf("want *CLIError, got %T (%v)", err, err)
	}
	if cliErr.Code != "tool_error" || cliErr.ExitCode != ExitError || !strings.Contains(cliErr.Message, "Preview expired") {
		t.Fatalf("got %+v", cliErr)
	}
	if out.Len() != 0 {
		t.Fatalf("nothing should be written on a tool error, got %q", out.String())
	}
}

func TestWriteMCPResult_SuccessJSONWhenPiped(t *testing.T) {
	out := &bytes.Buffer{}
	ios := &IOStreams{Out: out, ErrOut: &bytes.Buffer{}, tty: false}
	res := fakeResult{Content: []fakeBlock{{Type: "text", Text: "hello"}}}

	if err := ios.WriteMCPResult(res, false, false); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), `{"status":"ok"`) {
		t.Fatalf("piped output should be the JSON envelope, got %q", out.String())
	}
}

func TestWriteMCPResult_PrettyInTTY(t *testing.T) {
	out := &bytes.Buffer{}
	ios := &IOStreams{Out: out, ErrOut: &bytes.Buffer{}, tty: true}
	res := fakeResult{Content: []fakeBlock{{Type: "text", Text: "hello **there**"}}}

	if err := ios.WriteMCPResult(res, false, false); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "hello there" {
		t.Fatalf("got %q", out.String())
	}
}
