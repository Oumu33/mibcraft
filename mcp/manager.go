package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/Oumu33/mibcraft/config"
)

// Manager MCP 管理器
type Manager struct {
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewManager 创建新的 MCP 管理器
func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
	}
}

// StartServers 启动所有配置的 MCP 服务器
func (m *Manager) StartServers(ctx context.Context, servers []config.MCPServerConfig) error {
	for _, server := range servers {
		if err := m.StartServer(ctx, server); err != nil {
			return fmt.Errorf("启动 MCP 服务器 %s 失败: %w", server.Name, err)
		}
	}
	return nil
}

// StartServer 启动单个 MCP 服务器
func (m *Manager) StartServer(ctx context.Context, cfg config.MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 检查是否已存在
	if _, exists := m.clients[cfg.Name]; exists {
		return fmt.Errorf("MCP 服务器 %s 已存在", cfg.Name)
	}
	
	client, err := NewClient(cfg.Name, cfg.Command, cfg.Args, cfg.Env)
	if err != nil {
		return err
	}
	
	// 初始化连接
	if err := client.Initialize(ctx); err != nil {
		client.Close()
		return err
	}
	
	// 获取工具列表
	if _, err := client.ListTools(ctx); err != nil {
		client.Close()
		return err
	}
	
	m.clients[cfg.Name] = client
	return nil
}

// StopServer 停止单个 MCP 服务器
func (m *Manager) StopServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("MCP 服务器 %s 不存在", name)
	}
	
	delete(m.clients, name)
	return client.Close()
}

// StopAll 停止所有 MCP 服务器
func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, client := range m.clients {
		client.Close()
		delete(m.clients, name)
	}
}

// GetClient 获取 MCP 客户端
func (m *Manager) GetClient(name string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	client, exists := m.clients[name]
	return client, exists
}

// GetAllClients 获取所有 MCP 客户端
func (m *Manager) GetAllClients() map[string]*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string]*Client)
	for name, client := range m.clients {
		result[name] = client
	}
	return result
}

// GetAllTools 获取所有工具
func (m *Manager) GetAllTools() map[string][]Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string][]Tool)
	for name, client := range m.clients {
		result[name] = client.GetTools()
	}
	return result
}

// CallTool 调用指定服务器的工具
func (m *Manager) CallTool(ctx context.Context, serverName, toolName string, args map[string]interface{}) (interface{}, error) {
	m.mu.RLock()
	client, exists := m.clients[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("MCP 服务器 %s 不存在", serverName)
	}
	
	return client.CallTool(ctx, toolName, args)
}

// CallToolAny 在任意服务器上调用工具
func (m *Manager) CallToolAny(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for _, client := range m.clients {
		for _, tool := range client.GetTools() {
			if tool.Name == toolName {
				return client.CallTool(ctx, toolName, args)
			}
		}
	}
	
	return nil, fmt.Errorf("工具 %s 未找到", toolName)
}

// FindTool 查找工具所在的服务器
func (m *Manager) FindTool(toolName string) (string, *Tool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for serverName, client := range m.clients {
		for _, tool := range client.GetTools() {
			if tool.Name == toolName {
				return serverName, &tool, true
			}
		}
	}
	
	return "", nil, false
}

// ListResources 列出所有资源
func (m *Manager) ListResources(ctx context.Context) (map[string][]Resource, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[string][]Resource)
	for name, client := range m.clients {
		resources, err := client.ListResources(ctx)
		if err != nil {
			continue
		}
		result[name] = resources
	}
	return result, nil
}

// ReadResource 读取资源
func (m *Manager) ReadResource(ctx context.Context, serverName, uri string) (string, error) {
	m.mu.RLock()
	client, exists := m.clients[serverName]
	m.mu.RUnlock()
	
	if !exists {
		return "", fmt.Errorf("MCP 服务器 %s 不存在", serverName)
	}
	
	return client.ReadResource(ctx, uri)
}

// GetServerNames 获取所有服务器名称
func (m *Manager) GetServerNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// HasServers 检查是否有服务器
func (m *Manager) HasServers() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients) > 0
}
