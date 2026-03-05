package agent

import (
	"context"
)

// Plugin 插件接口
type Plugin interface {
	// Name 返回插件名称
	Name() string

	// Description 返回插件描述
	Description() string

	// Init 初始化插件
	Init(config PluginConfig) error

	// Check 执行检查，返回结果
	Check(ctx context.Context) (*CheckResult, error)

	// Close 清理资源
	Close() error
}

// PluginRegistry 插件注册表
type PluginRegistry struct {
	plugins map[string]Plugin
}

// NewPluginRegistry 创建插件注册表
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register 注册插件
func (r *PluginRegistry) Register(p Plugin) {
	r.plugins[p.Name()] = p
}

// Get 获取插件
func (r *PluginRegistry) Get(name string) (Plugin, bool) {
	p, ok := r.plugins[name]
	return p, ok
}

// GetAll 获取所有插件
func (r *PluginRegistry) GetAll() map[string]Plugin {
	return r.plugins
}

// List 列出所有插件名称
func (r *PluginRegistry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}
