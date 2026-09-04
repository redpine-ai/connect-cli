package output

import (
	"errors"
	"testing"
)

func TestServerError_MapsRejectedTokenToAuth(t *testing.T) {
	e := ServerError(errors.New(`initialize failed: HTTP 401: {"error":{"code":"UNAUTHORIZED"}}`))
	if e.Code != "auth_error" || e.ExitCode != ExitAuth {
		t.Fatalf("401 mapped to %s / exit %d, want auth_error / %d", e.Code, e.ExitCode, ExitAuth)
	}
	e = ServerError(errors.New("server error (HTTP 503)"))
	if e.Code != "server_error" || e.ExitCode != ExitServer {
		t.Fatalf("503 mapped to %s / exit %d, want server_error / %d", e.Code, e.ExitCode, ExitServer)
	}
}

func TestNotAuthenticated(t *testing.T) {
	e := NotAuthenticated()
	if e.ExitCode != ExitAuth || e.Code != "not_authenticated" {
		t.Fatalf("got %s / %d", e.Code, e.ExitCode)
	}
}
