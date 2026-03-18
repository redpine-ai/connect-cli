package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTransport_SendRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("auth header = %q", auth)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("wrong content type")
		}
		w.Header().Set("Mcp-Session-Id", "session-123")
		json.NewEncoder(w).Encode(RPCResponse{
			JSONRPC: "2.0",
			ID:      jsonRawInt(1),
			Result:  json.RawMessage(`{"ok": true}`),
		})
	}))
	defer server.Close()

	tr := NewTransport(server.URL, "test-token", true)
	resp, err := tr.Send(&RPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if tr.SessionID() != "session-123" {
		t.Errorf("session ID = %q", tr.SessionID())
	}
	if resp == nil {
		t.Fatal("nil response")
	}
}

func TestTransport_SendBatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if body[0] != '[' {
			t.Errorf("batch should be array, got %c", body[0])
		}
		w.Header().Set("Mcp-Session-Id", "session-456")
		json.NewEncoder(w).Encode([]RPCResponse{
			{JSONRPC: "2.0", ID: jsonRawInt(1), Result: json.RawMessage(`{}`)},
			{JSONRPC: "2.0", ID: jsonRawInt(2), Result: json.RawMessage(`{}`)},
		})
	}))
	defer server.Close()

	tr := NewTransport(server.URL, "test-token", true)
	responses, err := tr.SendBatch([]interface{}{
		&RPCRequest{JSONRPC: "2.0", ID: 1, Method: "a"},
		&RPCNotification{JSONRPC: "2.0", Method: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(responses) != 2 {
		t.Errorf("got %d responses", len(responses))
	}
}

func TestTransport_RejectsHTTP(t *testing.T) {
	tr := NewTransport("http://example.com", "token", false)
	_, err := tr.Send(&RPCRequest{JSONRPC: "2.0", ID: 1, Method: "test"})
	if err == nil {
		t.Error("should reject non-HTTPS URL")
	}
}

func TestTransport_AllowsHTTPWithInsecure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(RPCResponse{JSONRPC: "2.0", ID: jsonRawInt(1)})
	}))
	defer server.Close()

	tr := NewTransport(server.URL, "token", true)
	_, err := tr.Send(&RPCRequest{JSONRPC: "2.0", ID: 1, Method: "test"})
	if err != nil {
		t.Errorf("insecure should allow HTTP: %v", err)
	}
}

func jsonRawInt(n int) json.RawMessage {
	b, _ := json.Marshal(n)
	return b
}
