package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type RPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Transport struct {
	url       string
	token     string
	insecure  bool
	sessionID string
	client    *http.Client
}

func NewTransport(url, token string, insecure bool) *Transport {
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, "/mcp")
	return &Transport{
		url:      url,
		token:    token,
		insecure: insecure,
		client:   &http.Client{},
	}
}

func (t *Transport) SessionID() string {
	return t.sessionID
}

func (t *Transport) SetSessionID(id string) {
	t.sessionID = id
}

func (t *Transport) Send(req *RPCRequest) (*RPCResponse, error) {
	if err := t.validateURL(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	respBody, err := t.doPost(body)
	if err != nil {
		return nil, err
	}

	var resp RPCResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("invalid JSON-RPC response: %w", err)
	}
	return &resp, nil
}

func (t *Transport) SendBatch(messages []interface{}) ([]RPCResponse, error) {
	if err := t.validateURL(); err != nil {
		return nil, err
	}

	body, err := json.Marshal(messages)
	if err != nil {
		return nil, err
	}

	respBody, err := t.doPost(body)
	if err != nil {
		return nil, err
	}

	var responses []RPCResponse
	if err := json.Unmarshal(respBody, &responses); err != nil {
		var single RPCResponse
		if err2 := json.Unmarshal(respBody, &single); err2 == nil {
			return []RPCResponse{single}, nil
		}
		return nil, fmt.Errorf("invalid batch response: %w", err)
	}
	return responses, nil
}

func (t *Transport) Delete() error {
	if err := t.validateURL(); err != nil {
		return err
	}

	req, err := http.NewRequest("DELETE", t.url+"/mcp", nil)
	if err != nil {
		return err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (t *Transport) doPost(body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", t.url+"/mcp", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("server unreachable: %w", err)
	}
	defer resp.Body.Close()

	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionID = sid
	}

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody))
	}

	return io.ReadAll(resp.Body)
}

func (t *Transport) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	if t.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", t.sessionID)
	}
}

func (t *Transport) validateURL() error {
	if !t.insecure && !strings.HasPrefix(t.url, "https://") {
		return fmt.Errorf("HTTPS required. Use --insecure for non-HTTPS URLs (local dev only)")
	}
	return nil
}
