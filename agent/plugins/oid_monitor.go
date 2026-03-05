package plugins

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Oumu33/mibcraft/agent"
)

// OIDMonitorPlugin OID 值监控插件
type OIDMonitorPlugin struct {
	config *OIDMonitorPluginConfig
}

// OIDMonitorPluginConfig OID 监控插件配置
type OIDMonitorPluginConfig struct {
	agent.BasePluginConfig
	Targets   []OIDTarget       `toml:"target"`
	Community string            `toml:"community"`
	Version   int               `toml:"version"`
	Timeout   time.Duration     `toml:"timeout"`
	Labels    map[string]string `toml:"labels"`
}

// OIDTarget OID 目标配置
type OIDTarget struct {
	Address   string    `toml:"address"`
	OIDs      []OIDSpec `toml:"oid"`
	Community string    `toml:"community,omitempty"`
}

// OIDSpec OID 规范
type OIDSpec struct {
	OID         string  `toml:"oid"`
	Name        string  `toml:"name"`
	Type        string  `toml:"type"`
	Threshold   float64 `toml:"threshold,omitempty"`
	AlertOn     string  `toml:"alert_on,omitempty"` // above, below, change
	Description string  `toml:"description,omitempty"`
}

// NewOIDMonitorPlugin 创建 OID 监控插件
func NewOIDMonitorPlugin() *OIDMonitorPlugin {
	return &OIDMonitorPlugin{}
}

func (p *OIDMonitorPlugin) Name() string {
	return "oid_monitor"
}

func (p *OIDMonitorPlugin) Description() string {
	return "监控指定 OID 的值变化和阈值告警"
}

func (p *OIDMonitorPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*OIDMonitorPluginConfig); ok {
			p.config = cfg
		}
	}

	if p.config == nil {
		p.config = &OIDMonitorPluginConfig{
			Community: "public",
			Version:   2,
			Timeout:   5 * time.Second,
		}
	}

	return nil
}

func (p *OIDMonitorPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Labels:  p.config.Labels,
	}

	alerts := []string{}

	for _, target := range p.config.Targets {
		community := target.Community
		if community == "" {
			community = p.config.Community
		}

		for _, oidSpec := range target.OIDs {
			// 模拟获取 OID 值
			value := p.simulateOIDValue(oidSpec.OID)
			metricName := oidSpec.Name
			if metricName == "" {
				metricName = fmt.Sprintf("oid_%s", oidSpec.OID)
			}

			result.Metrics[metricName] = value

			// 检查阈值
			if oidSpec.Threshold > 0 && oidSpec.AlertOn != "" {
				alert := p.checkThreshold(metricName, value, oidSpec.Threshold, oidSpec.AlertOn)
				if alert != "" {
					alerts = append(alerts, alert)
				}
			}
		}
	}

	if len(alerts) > 0 {
		result.OK = false
		result.Message = fmt.Sprintf("OID 阈值告警: %s", strings.Join(alerts, "; "))
	} else {
		result.Message = "OID 监控正常"
	}

	return result, nil
}

// simulateOIDValue 模拟获取 OID 值
func (p *OIDMonitorPlugin) simulateOIDValue(oid string) float64 {
	// 实际实现应使用 gosnmp 库
	return 42.0
}

// checkThreshold 检查阈值
func (p *OIDMonitorPlugin) checkThreshold(name string, value, threshold float64, alertOn string) string {
	switch alertOn {
	case "above":
		if value > threshold {
			return fmt.Sprintf("%s 值 %.2f 超过阈值 %.2f", name, value, threshold)
		}
	case "below":
		if value < threshold {
			return fmt.Sprintf("%s 值 %.2f 低于阈值 %.2f", name, value, threshold)
		}
	}
	return ""
}

func (p *OIDMonitorPlugin) Close() error {
	return nil
}