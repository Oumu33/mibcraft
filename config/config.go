package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config 全局配置
type Config struct {
	Global    GlobalConfig    `toml:"global"`
	Log       LogConfig       `toml:"log"`
	AI        AIConfig        `toml:"ai"`
	MIB       MIBConfig       `toml:"mib"`
	Generator GeneratorConfig `toml:"generator"`
	Notify    NotifyConfig    `toml:"notify"`
	Agent     AgentConfig     `toml:"agent"`
}

// NotifyConfig 通知配置
type NotifyConfig struct {
	Console   *ConsoleNotifyConfig   `toml:"console,omitempty"`
	WebAPI    *WebAPINotifyConfig    `toml:"webapi,omitempty"`
	Flashduty *FlashdutyNotifyConfig `toml:"flashduty,omitempty"`
	PagerDuty *PagerDutyNotifyConfig `toml:"pagerduty,omitempty"`
}

// ConsoleNotifyConfig 控制台通知配置
type ConsoleNotifyConfig struct {
	Enabled bool `toml:"enabled"`
	Color   bool `toml:"color"`
}

// WebAPINotifyConfig WebAPI 通知配置
type WebAPINotifyConfig struct {
	Enabled bool              `toml:"enabled"`
	URL     string            `toml:"url"`
	Method  string            `toml:"method"`
	Timeout string            `toml:"timeout"`
	Headers map[string]string `toml:"headers"`
}

// FlashdutyNotifyConfig Flashduty 通知配置
type FlashdutyNotifyConfig struct {
	Enabled        bool   `toml:"enabled"`
	IntegrationKey string `toml:"integration_key"`
}

// PagerDutyNotifyConfig PagerDuty 通知配置
type PagerDutyNotifyConfig struct {
	Enabled    bool   `toml:"enabled"`
	RoutingKey string `toml:"routing_key"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	Enabled       bool              `toml:"enabled"`
	CheckInterval string            `toml:"check_interval"`
	Plugins       map[string]any    `toml:"plugins"`
	AutoDiagnose  bool              `toml:"auto_diagnose"`
	Labels        map[string]string `toml:"labels"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	MIBDirs []string          `toml:"mib_dirs"`
	Labels  map[string]string `toml:"labels"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `toml:"level"`
	Output string `toml:"output"`
}

// AIConfig AI 配置
type AIConfig struct {
	Enabled       bool                   `toml:"enabled"`
	ModelPriority []string               `toml:"model_priority"`
	Models        map[string]ModelConfig `toml:"models"`
	MCP           MCPConfig              `toml:"mcp"`
}

// ModelConfig 模型配置
type ModelConfig struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
	Model   string `toml:"model"`
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Enabled bool            `toml:"enabled"`
	Servers []MCPServerConfig `toml:"servers"`
}

// MCPServerConfig MCP 服务器配置
type MCPServerConfig struct {
	Name    string   `toml:"name"`
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	Env     map[string]string `toml:"env"`
}

// MIBConfig MIB 配置
type MIBConfig struct {
	CacheDir     string `toml:"cache_dir"`
	AutoLoad     bool   `toml:"auto_load"`
	ParseTimeout int    `toml:"parse_timeout"`
}

// GeneratorConfig 生成器配置
type GeneratorConfig struct {
	OutputDir    string `toml:"output_dir"`
	DefaultCommunity string `toml:"default_community"`
	DefaultVersion  int    `toml:"default_version"`
	DefaultInterval string `toml:"default_interval"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Global: GlobalConfig{
			MIBDirs: []string{"./mibs", "/usr/share/snmp/mibs"},
			Labels:  make(map[string]string),
		},
		Log: LogConfig{
			Level:  "info",
			Output: "stdout",
		},
		AI: AIConfig{
			Enabled:       true,
			ModelPriority: []string{"gpt4o", "deepseek"},
			Models:        make(map[string]ModelConfig),
		},
		MIB: MIBConfig{
			CacheDir:     "./mib_cache",
			AutoLoad:     true,
			ParseTimeout: 30,
		},
		Generator: GeneratorConfig{
			OutputDir:        "./output",
			DefaultCommunity: "public",
			DefaultVersion:   2,
			DefaultInterval:  "30s",
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	
	_, err := toml.DecodeFile(path, cfg)
	if err != nil {
		return nil, err
	}
	
	// 展开环境变量
	cfg.expandEnvVars()
	
	return cfg, nil
}

// expandEnvVars 展开环境变量
func (c *Config) expandEnvVars() {
	expandMap := func(m map[string]string) {
		for k, v := range m {
			if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
				envKey := v[2 : len(v)-1]
				m[k] = os.Getenv(envKey)
			}
		}
	}
	
	expandMap(c.Global.Labels)
	
	for name, model := range c.AI.Models {
		if strings.HasPrefix(model.APIKey, "${") && strings.HasSuffix(model.APIKey, "}") {
			envKey := model.APIKey[2 : len(model.APIKey)-1]
			model.APIKey = os.Getenv(envKey)
			c.AI.Models[name] = model
		}
	}
}

// GetConfigPath 获取配置文件路径
func GetConfigPath() string {
	// 优先级: 环境变量 > 当前目录 > 用户目录 > 系统目录
	if path := os.Getenv("MIB_AGENT_CONFIG"); path != "" {
		return path
	}
	
	candidates := []string{
		"./conf.d/config.toml",
		"./config.toml",
		filepath.Join(os.Getenv("HOME"), ".mibcraft", "config.toml"),
		"/etc/mibcraft/config.toml",
	}
	
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return "./conf.d/config.toml"
}
