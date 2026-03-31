package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestClient_FindTools(t *testing.T) {
	toolsJSON := `[{"name":"search","description":"Search docs","inputSchema":{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}},{"name":"aviation--aircraft_finder","description":"Find aircraft","inputSchema":{"type":"object","properties":{},"required":[]}}]`
	callResult := `{"content":[{"type":"text","text":` + strconv.Quote(toolsJSON) + `}]}`

	server := newMockMCPServer(t, map[string]json.RawMessage{
		"initialize": json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
		"tools/call": json.RawMessage(callResult),
	})
	defer server.Close()

	client := NewClient(server.URL, "token")
	if err := client.Initialize(); err != nil {
		t.Fatal(err)
	}

	tools, err := client.FindTools("", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 2 {
		t.Errorf("got %d tools, want 2", len(tools))
	}
	if tools[0].Name != "search" {
		t.Errorf("got name %q, want %q", tools[0].Name, "search")
	}
}

func TestClient_FindToolsWithQuery(t *testing.T) {
	toolsJSON := `[{"name":"aviation--aircraft_finder","description":"Find aircraft","inputSchema":{}}]`
	callResult := `{"content":[{"type":"text","text":` + strconv.Quote(toolsJSON) + `}]}`

	server := newMockMCPServer(t, map[string]json.RawMessage{
		"initialize": json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
		"tools/call": json.RawMessage(callResult),
	})
	defer server.Close()

	client := NewClient(server.URL, "token")
	client.Initialize()

	tools, err := client.FindTools("aircraft", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Errorf("got %d tools, want 1", len(tools))
	}
}

func TestClient_InspectTool(t *testing.T) {
	toolJSON := `{"name":"search","description":"Search docs","inputSchema":{"type":"object","properties":{"query":{"type":"string","description":"Search query"}},"required":["query"]},"annotations":{"readOnlyHint":true}}`
	callResult := `{"content":[{"type":"text","text":` + strconv.Quote(toolJSON) + `}]}`

	server := newMockMCPServer(t, map[string]json.RawMessage{
		"initialize": json.RawMessage(`{"protocolVersion":"2025-03-26","serverInfo":{"name":"test"},"capabilities":{}}`),
		"tools/call": json.RawMessage(callResult),
	})
	defer server.Close()

	client := NewClient(server.URL, "token")
	client.Initialize()

	tool, err := client.InspectTool("search")
	if err != nil {
		t.Fatal(err)
	}
	if tool == nil {
		t.Fatal("nil tool")
	}
	if tool.Name != "search" {
		t.Errorf("got name %q, want %q", tool.Name, "search")
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
