package types

import "time"

// MIBObject 表示一个 MIB 对象的定义
type MIBObject struct {
	Name        string            `json:"name"`
	OID         string            `json:"oid"`
	Type        string            `json:"type"`
	Access      string            `json:"access"` // read-only, read-write, write-only, not-accessible
	Description string            `json:"description"`
	Units       string            `json:"units,omitempty"`
	Index       []string          `json:"index,omitempty"`
	Children    []*MIBObject      `json:"children,omitempty"`
	EnumValues  map[string]int    `json:"enum_values,omitempty"`
	MIB         string            `json:"mib"` // 所属 MIB 文件名
	Labels      map[string]string `json:"labels,omitempty"`
}

// MIBModule 表示一个完整的 MIB 模块
type MIBModule struct {
	Name        string       `json:"name"`
	OID         string       `json:"oid"`
	Objects     []*MIBObject `json:"objects"`
	Description string       `json:"description"`
	Imports     []string     `json:"imports,omitempty"`
	FilePath    string       `json:"file_path"`
}

// CategrafConfig Categraf SNMP 采集配置
type CategrafConfig struct {
	Interval   string                    `toml:"interval,omitempty"`
	Targets    []string                  `toml:"targets"`
	Community  string                    `toml:"community"`
	Version    int                       `toml:"version"`
	Timeout    string                    `toml:"timeout,omitempty"`
	Retries    int                       `toml:"retries,omitempty"`
	Collect    []CategrafCollectConfig   `toml:"collect"`
	Labels     map[string]string         `toml:"labels,omitempty"`
}

// CategrafCollectConfig 单个采集配置
type CategrafCollectConfig struct {
	MetricName string            `toml:"metric_name"`
	OID        string            `toml:"oid"`
	Type       string            `toml:"type"`
	Labels     map[string]string `toml:"labels,omitempty"`
}

// SNMPExporterConfig SNMP Exporter 配置
type SNMPExporterConfig struct {
	Module string                  `yaml:"module"`
	Metrics []SNMPExporterMetric   `yaml:"metrics"`
}

// SNMPExporterMetric SNMP Exporter 指标定义
type SNMPExporterMetric struct {
	Name       string                 `yaml:"name"`
	OID        string                 `yaml:"oid"`
	Type       string                 `yaml:"type"`
	Help       string                 `yaml:"help"`
	Labels     []SNMPExporterLabel    `yaml:"labels,omitempty"`
	Regex      string                 `yaml:"regex,omitempty"`
	Scale      float64                `yaml:"scale,omitempty"`
	Offset     float64                `yaml:"offset,omitempty"`
	Indexes    []SNMPExporterIndex    `yaml:"indexes,omitempty"`
	Lookups    []SNMPExporterLookup   `yaml:"lookups,omitempty"`
	EnumValues map[string]int         `yaml:"enum_values,omitempty"`
}

// SNMPExporterLabel 标签定义
type SNMPExporterLabel struct {
	Name    string `yaml:"name"`
	OID     string `yaml:"oid"`
	Type    string `yaml:"type"`
	Regex   string `yaml:"regex,omitempty"`
	Enum    bool   `yaml:"enum,omitempty"`
}

// SNMPExporterIndex 索引定义
type SNMPExporterIndex struct {
	Labelname string `yaml:"labelname"`
	Type      string `yaml:"type"`
	FixedSize int    `yaml:"fixed_size,omitempty"`
	Implied   bool   `yaml:"implied,omitempty"`
	Enum      bool   `yaml:"enum,omitempty"`
}

// SNMPExporterLookup 查找定义
type SNMPExporterLookup struct {
	Labels    []string `yaml:"labels"`
	Labelname string   `yaml:"labelname"`
	Type      string   `yaml:"type"`
	OID       string   `yaml:"oid"`
}

// ConfigRequest 配置生成请求
type ConfigRequest struct {
	MIBFiles     []string          `json:"mib_files"`     // MIB 文件列表
	TargetOIDs   []string          `json:"target_oids"`   // 目标 OID 列表
	MetricNames  map[string]string `json:"metric_names"`  // OID -> 指标名称映射
	Labels       map[string]string `json:"labels"`        // 全局标签
	Format       string            `json:"format"`        // categraf, snmp_exporter, both
	Description  string            `json:"description"`   // 用户描述需求
}

// ConfigResult 配置生成结果
type ConfigResult struct {
	CategrafConfig    string    `json:"categraf_config,omitempty"`
	SNMPExporterConfig string   `json:"snmp_exporter_config,omitempty"`
	GeneratedAt       time.Time `json:"generated_at"`
	MIBObjectsUsed    []string  `json:"mib_objects_used"`
	Warnings          []string  `json:"warnings,omitempty"`
}
