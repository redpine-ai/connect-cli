package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Initialize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		if len(body) > 0 && body[0] == '[' {
			w.Header().Set("Mcp-Session-Id", "sess-1")
			json.NewEncoder(w).Encode([]RPCResponse{})
			return
		}

		var req RPCRequest
		json.Unmarshal(body, &req)

		if req.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sess-1")
			json.NewEncoder(w).Encode(RPCResponse{
				JSONRPC: "2.0",
				ID:      jsonRawInt(1),
				Result:  json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
			})
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "token")
	err := client.Initialize()
	if err != nil {
		t.Fatal(err)
	}
	if client.SessionID() == "" {
		t.Error("should have session ID after initialize")
	}
}

func TestClient_ListTools(t *testing.T) {
	server := newMockMCPServer(t, map[string]json.RawMessage{
		"initialize": json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
		"tools/list": json.RawMessage(`{"tools":[{"name":"search","description":"Search docs","inputSchema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}},{"name":"analytics--run_query","description":"Run query","inputSchema":{"type":"object","properties":{"sql":{"type":"string"}},"required":["sql"]}}]}`),
	})
	defer server.Close()

	client := NewClient(server.URL, "token")
	if err := client.Initialize(); err != nil {
		t.Fatal(err)
	}

	tools, err := client.ListTools()
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 {
		t.Errorf("got %d tools, want 2", len(tools))
	}
}

func TestClient_CallTool(t *testing.T) {
	server := newMockMCPServer(t, map[string]json.RawMessage{
		"initialize": json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
		"tools/call": json.RawMessage(`{"content":[{"type":"text","text":"result data"}]}`),
	})
	defer server.Close()

	client := NewClient(server.URL, "token")
	if err := client.Initialize(); err != nil {
		t.Fatal(err)
	}

	result, err := client.CallTool("search", map[string]interface{}{"query": "test"})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("nil result")
	}
}

func newMockMCPServer(t *testing.T, responses map[string]json.RawMessage) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			w.WriteHeader(http.StatusOK)
			return
		}

		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Mcp-Session-Id", "test-session")

		if len(body) > 0 && body[0] == '[' {
			var batch []json.RawMessage
			json.Unmarshal(body, &batch)
			var results []RPCResponse
			for _, item := range batch {
				var req RPCRequest
				if err := json.Unmarshal(item, &req); err != nil {
					continue
				}
				if req.ID == 0 {
					continue
				}
				if result, ok := responses[req.Method]; ok {
					idBytes, _ := json.Marshal(req.ID)
					results = append(results, RPCResponse{
						JSONRPC: "2.0",
						ID:      idBytes,
						Result:  result,
					})
				}
			}
			json.NewEncoder(w).Encode(results)
			return
		}

		var req RPCRequest
		json.Unmarshal(body, &req)
		if result, ok := responses[req.Method]; ok {
			idBytes, _ := json.Marshal(req.ID)
			json.NewEncoder(w).Encode(RPCResponse{
				JSONRPC: "2.0",
				ID:      idBytes,
				Result:  result,
			})
		} else {
			json.NewEncoder(w).Encode(RPCResponse{
				JSONRPC: "2.0",
				ID:      jsonRawInt(req.ID),
				Error:   &RPCError{Code: -32601, Message: "method not found"},
			})
		}
	}))
}
