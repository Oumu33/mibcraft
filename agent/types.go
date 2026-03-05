package agent

import (
	"time"
)

// Event 表示一个监控事件
type Event struct {
	ID          string            `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	PluginName  string            `json:"plugin_name"`
	Severity    Severity          `json:"severity"`
	Status      Status            `json:"status"`
	Summary     string            `json:"summary"`
	Detail      string            `json:"detail,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Metrics     map[string]any    `json:"metrics,omitempty"`
	Diagnosis   *DiagnosisResult  `json:"diagnosis,omitempty"`
}

// Severity 事件严重级别
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Status 事件状态
type Status string

const (
	StatusFiring   Status = "firing"
	StatusResolved Status = "resolved"
)

// DiagnosisResult AI 诊断结果
type DiagnosisResult struct {
	Timestamp     time.Time        `json:"timestamp"`
	RootCause     string           `json:"root_cause"`
	Analysis      string           `json:"analysis"`
	Recommendations []string       `json:"recommendations"`
	ToolsUsed     []string         `json:"tools_used,omitempty"`
	Duration      time.Duration    `json:"duration"`
}

// PluginConfig 插件配置接口
type PluginConfig interface {
	GetName() string
	GetEnabled() bool
	GetInterval() time.Duration
	GetTimeout() time.Duration
}

// BasePluginConfig 基础插件配置
type BasePluginConfig struct {
	Name     string        `toml:"name"`
	Enabled  bool          `toml:"enabled"`
	Interval time.Duration `toml:"interval"`
	Timeout  time.Duration `toml:"timeout"`
}

func (c *BasePluginConfig) GetName() string        { return c.Name }
func (c *BasePluginConfig) GetEnabled() bool       { return c.Enabled }
func (c *BasePluginConfig) GetInterval() time.Duration { return c.Interval }
func (c *BasePluginConfig) GetTimeout() time.Duration  { return c.Timeout }

// CheckResult 检查结果
type CheckResult struct {
	OK      bool              `json:"ok"`
	Message string            `json:"message"`
	Metrics map[string]any    `json:"metrics,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// DiagnosticTool 诊断工具接口
type DiagnosticTool interface {
	Name() string
	Description() string
	Execute(ctx map[string]any) (string, error)
}

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Plugin    string        `json:"plugin"`
	Healthy   bool          `json:"healthy"`
	Message   string        `json:"message"`
	Timestamp time.Time     `json:"timestamp"`
	Latency   time.Duration `json:"latency"`
}
