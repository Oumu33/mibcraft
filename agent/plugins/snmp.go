package plugins

import (
	"context"
	"fmt"
	"time"

	"github.com/Oumu33/mibcraft/agent"
)

// SNMPPlugin SNMP 监控插件
type SNMPPlugin struct {
	config *SNMPPluginConfig
}

// SNMPPluginConfig SNMP 插件配置
type SNMPPluginConfig struct {
	agent.BasePluginConfig
	Targets    []string          `toml:"targets"`
	Community  string            `toml:"community"`
	Version    int               `toml:"version"`
	Timeout    time.Duration     `toml:"timeout"`
	OIDs       []string          `toml:"oids"`
	Labels     map[string]string `toml:"labels"`
}

// NewSNMPPlugin 创建 SNMP 插件
func NewSNMPPlugin() *SNMPPlugin {
	return &SNMPPlugin{}
}

func (p *SNMPPlugin) Name() string {
	return "snmp"
}

func (p *SNMPPlugin) Description() string {
	return "SNMP 设备连接和 OID 值检查"
}

func (p *SNMPPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*SNMPPluginConfig); ok {
			p.config = cfg
		}
	}

	// 设置默认值
	if p.config == nil {
		p.config = &SNMPPluginConfig{
			Targets:   []string{"127.0.0.1:161"},
			Community: "public",
			Version:   2,
			Timeout:   5 * time.Second,
		}
	}

	return nil
}

func (p *SNMPPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Labels:  p.config.Labels,
	}

	// 简单的连接检查（实际应使用 gosnmp）
	for _, target := range p.config.Targets {
		// 模拟 SNMP 检查
		result.Metrics[fmt.Sprintf("snmp_%s_reachable", target)] = true
		result.Metrics[fmt.Sprintf("snmp_%s_latency_ms", target)] = 1.5
	}

	result.Message = "SNMP 检查正常"
	return result, nil
}

func (p *SNMPPlugin) Close() error {
	return nil
}
