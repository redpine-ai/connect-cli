package mcp

import (
	"encoding/json"
	"fmt"

	"github.com/redpine-ai/connect-cli/internal/version"
)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ToolCallResult struct {
	Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Client struct {
	transport *Transport
	idCounter int
}

func NewClient(serverURL, token string) *Client {
	return &Client{
		transport: NewTransport(serverURL, token),
	}
}

func (c *Client) SessionID() string {
	return c.transport.SessionID()
}

func (c *Client) SetSessionID(id string) {
	c.transport.SetSessionID(id)
}

func (c *Client) nextID() int {
	c.idCounter++
	return c.idCounter
}

func (c *Client) Initialize() error {
	resp, err := c.transport.Send(&RPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"clientInfo": map[string]string{
				"name":    "connect-cli",
				"version": version.Version,
			},
			"capabilities": map[string]interface{}{},
		},
	})
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send notifications/initialized
	_, err = c.transport.SendBatch([]interface{}{
		&RPCNotification{
			JSONRPC: "2.0",
			Method:  "notifications/initialized",
		},
	})
	_ = err // Notification responses are optional

	return nil
}

func (c *Client) ListTools() ([]Tool, error) {
	resp, err := c.transport.Send(&RPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/list",
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (c *Client) CallTool(name string, args map[string]interface{}) (*ToolCallResult, error) {
	resp, err := c.transport.Send(&RPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID(),
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name":      name,
			"arguments": args,
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/call error: %s", resp.Error.Message)
	}

	var result ToolCallResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteSession() error {
	return c.transport.Delete()
}
