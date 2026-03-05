package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// Client MCP 客户端
type Client struct {
	name     string
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   *bufio.Reader
	mu       sync.Mutex
	ready    bool
	tools    []Tool
	resources []Resource
}

// Tool MCP 工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// Resource MCP 资源定义
type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

// JSONRPCRequest JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError RPC 错误
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// initializeParams 初始化参数
type initializeParams struct {
	ProtocolVersion string                 `json:"protocolVersion"`
	ClientInfo      map[string]string      `json:"clientInfo"`
	Capabilities    map[string]interface{} `json:"capabilities"`
}

// NewClient 创建新的 MCP 客户端
func NewClient(name, command string, args []string, env map[string]string) (*Client, error) {
	cmd := exec.Command(command, args...)
	
	// 设置环境变量
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// 创建管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdin 管道失败: %w", err)
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdout 管道失败: %w", err)
	}
	
	// 重定向 stderr 到主进程
	cmd.Stderr = os.Stderr
	
	client := &Client{
		name:   name,
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
	
	// 启动进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 MCP 服务器失败: %w", err)
	}
	
	return client, nil
}

// Initialize 初始化 MCP 连接
func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	params := initializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: map[string]string{
			"name":    "mibcraft",
			"version": "1.0.0",
		},
		Capabilities: map[string]interface{}{
			"tools": map[string]bool{},
		},
	}
	
	var result struct {
		Capabilities map[string]interface{} `json:"capabilities"`
		ServerInfo   map[string]string      `json:"serverInfo"`
	}
	
	if err := c.call(ctx, "initialize", params, &result); err != nil {
		return err
	}
	
	// 发送 initialized 通知
	if err := c.notify(ctx, "notifications/initialized", nil); err != nil {
		return err
	}
	
	c.ready = true
	return nil
}

// ListTools 获取工具列表
func (c *Client) ListTools(ctx context.Context) ([]Tool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var result struct {
		Tools []Tool `json:"tools"`
	}
	
	if err := c.call(ctx, "tools/list", map[string]interface{}{}, &result); err != nil {
		return nil, err
	}
	
	c.tools = result.Tools
	return result.Tools, nil
}

// CallTool 调用工具
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (interface{}, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}
	
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	
	if err := c.call(ctx, "tools/call", params, &result); err != nil {
		return nil, err
	}
	
	if result.IsError {
		if len(result.Content) > 0 {
			return nil, fmt.Errorf("工具调用错误: %s", result.Content[0].Text)
		}
		return nil, fmt.Errorf("工具调用错误")
	}
	
	if len(result.Content) > 0 && result.Content[0].Type == "text" {
		return result.Content[0].Text, nil
	}
	
	return result, nil
}

// ListResources 获取资源列表
func (c *Client) ListResources(ctx context.Context) ([]Resource, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	var result struct {
		Resources []Resource `json:"resources"`
	}
	
	if err := c.call(ctx, "resources/list", map[string]interface{}{}, &result); err != nil {
		return nil, err
	}
	
	c.resources = result.Resources
	return result.Resources, nil
}

// ReadResource 读取资源
func (c *Client) ReadResource(ctx context.Context, uri string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	params := map[string]interface{}{
		"uri": uri,
	}
	
	var result struct {
		Contents []struct {
			URI      string `json:"uri"`
			MimeType string `json:"mimeType"`
			Text     string `json:"text"`
		} `json:"contents"`
	}
	
	if err := c.call(ctx, "resources/read", params, &result); err != nil {
		return "", err
	}
	
	if len(result.Contents) > 0 {
		return result.Contents[0].Text, nil
	}
	
	return "", nil
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}

// call 发送 JSON-RPC 调用
func (c *Client) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := time.Now().UnixNano()
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	
	// 发送请求
	if err := json.NewEncoder(c.stdin).Encode(req); err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	
	// 读取响应
	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}
	
	var resp JSONRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}
	
	if resp.Error != nil {
		return fmt.Errorf("RPC 错误: %s (code: %d)", resp.Error.Message, resp.Error.Code)
	}
	
	if result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("解析结果失败: %w", err)
		}
	}
	
	return nil
}

// notify 发送 JSON-RPC 通知
func (c *Client) notify(ctx context.Context, method string, params interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	
	return json.NewEncoder(c.stdin).Encode(req)
}

// GetName 获取客户端名称
func (c *Client) GetName() string {
	return c.name
}

// GetTools 获取工具列表
func (c *Client) GetTools() []Tool {
	return c.tools
}

// GetResources 获取资源列表
func (c *Client) GetResources() []Resource {
	return c.resources
}

// IsReady 检查客户端是否就绪
func (c *Client) IsReady() bool {
	return c.ready
}
