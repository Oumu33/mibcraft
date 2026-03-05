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
	CategrafConfig     string    `json:"categraf_config,omitempty"`
	SNMPExporterConfig string    `json:"snmp_exporter_config,omitempty"`
	TelegrafConfig     string    `json:"telegraf_config,omitempty"`
	GeneratedAt        time.Time `json:"generated_at"`
	MIBObjectsUsed     []string  `json:"mib_objects_used"`
	Warnings           []string  `json:"warnings,omitempty"`
}

// TelegrafConfig Telegraf SNMP 输入插件配置
type TelegrafConfig struct {
	Interval     string                  `toml:"interval,omitempty"`
	NamePrefix   string                  `toml:"name_prefix,omitempty"`
	NameSuffix   string                  `toml:"name_suffix,omitempty"`
	Collection   []TelegrafInputConfig   `toml:"-"`
	Labels       map[string]string       `toml:"tags,omitempty"`
}

// TelegrafInputConfig Telegraf inputs.snmp 配置
type TelegrafInputConfig struct {
	Agents          []string                     `toml:"agents"`
	Community       string                       `toml:"community,omitempty"`
	Version         int                          `toml:"version,omitempty"`
	Timeout         string                       `toml:"timeout,omitempty"`
	Retries         int                          `toml:"retries,omitempty"`
	MaxRepetitions  int                          `toml:"max_repetitions,omitempty"`
	Fields          []TelegrafFieldConfig        `toml:"field"`
	Tables          []TelegrafTableConfig        `toml:"table,omitempty"`
	Tags            map[string]string            `toml:"tags,omitempty"`
	NameOverride    string                       `toml:"name_override,omitempty"`
	NamePrefix      string                       `toml:"name_prefix,omitempty"`
}

// TelegrafFieldConfig Telegraf 字段配置
type TelegrafFieldConfig struct {
	Name       string `toml:"name"`
	Oid        string `toml:"oid"`
	Conversion string `toml:"conversion,omitempty"`
}

// TelegrafTableConfig Telegraf 表配置（用于表格型 OID）
type TelegrafTableConfig struct {
	Name       string                  `toml:"name"`
	Oid        string                  `toml:"oid,omitempty"`
	Fields     []TelegrafFieldConfig   `toml:"field"`
	InheritTags []string               `toml:"inherit_tags,omitempty"`
}

// ==================== IPMI/Redfish/VMware 配置类型 ====================

// IPMIConfig IPMI监控配置 (Telegraf inputs.ipmi_sensor)
type IPMIConfig struct {
	Servers       []IPMIServerConfig `toml:"server" json:"servers"`
	Interval      string             `toml:"interval,omitempty" json:"interval,omitempty"`
	MetricVersion int                `toml:"metric_version,omitempty" json:"metric_version,omitempty"`
	Timeout       string             `toml:"timeout,omitempty" json:"timeout,omitempty"`
}

// IPMIServerConfig IPMI服务器配置
type IPMIServerConfig struct {
	Host        string            `toml:"host" json:"host"`
	Username    string            `toml:"username" json:"username"`
	Password    string            `toml:"password" json:"password"`
	Interface   string            `toml:"interface,omitempty" json:"interface,omitempty"` // lan, lanplus, open
	Port        int               `toml:"port,omitempty" json:"port,omitempty"`
	Privilege   string            `toml:"privilege,omitempty" json:"privilege,omitempty"` // user, operator, admin
	Labels      map[string]string `toml:"tags,omitempty" json:"tags,omitempty"`
}

// RedfishConfig Redfish监控配置 (Telegraf inputs.redfish)
type RedfishConfig struct {
	Servers      []RedfishServerConfig `toml:"server" json:"servers"`
	Interval     string                `toml:"interval,omitempty" json:"interval,omitempty"`
	Timeout      string                `toml:"timeout,omitempty" json:"timeout,omitempty"`
}

// RedfishServerConfig Redfish服务器配置 (Dell iDRAC, HPE iLO, etc.)
type RedfishServerConfig struct {
	Name         string            `toml:"name" json:"name"`
	Host         string            `toml:"address" json:"host"`
	Username     string            `toml:"username" json:"username"`
	Password     string            `toml:"password" json:"password"`
	Port         int               `toml:"port,omitempty" json:"port,omitempty"`
	Insecure     bool              `toml:"insecure_skip_verify,omitempty" json:"insecure,omitempty"`
	IncludeMetrics []string        `toml:"include_metrics,omitempty" json:"include_metrics,omitempty"`
	Labels       map[string]string `toml:"tags,omitempty" json:"tags,omitempty"`
}

// VMwareConfig VMware vSphere监控配置 (Telegraf inputs.vsphere)
type VMwareConfig struct {
	VCenters     []VMwareVCenterConfig `toml:"vcenter" json:"vcenters"`
	Interval     string                `toml:"interval,omitempty" json:"interval,omitempty"`
	Timeout      string                `toml:"timeout,omitempty" json:"timeout,omitempty"`
}

// VMwareVCenterConfig vCenter服务器配置
type VMwareVCenterConfig struct {
	Name           string            `toml:"name,omitempty" json:"name,omitempty"`
	URL            string            `toml:"vcenters" json:"url"`
	Username       string            `toml:"username" json:"username"`
	Password       string            `toml:"password" json:"password"`
	Insecure       bool              `toml:"insecure_skip_verify,omitempty" json:"insecure,omitempty"`
	Datacenter     string            `toml:"datacenter,omitempty" json:"datacenter,omitempty"`
	Cluster        string            `toml:"cluster,omitempty" json:"cluster,omitempty"`
	VMInclude      []string          `toml:"vm_metric_include,omitempty" json:"vm_metric_include,omitempty"`
	VMExclude      []string          `toml:"vm_metric_exclude,omitempty" json:"vm_metric_exclude,omitempty"`
	HostInclude    []string          `toml:"host_metric_include,omitempty" json:"host_metric_include,omitempty"`
	HostExclude    []string          `toml:"host_metric_exclude,omitempty" json:"host_metric_exclude,omitempty"`
	ClusterInclude []string          `toml:"cluster_metric_include,omitempty" json:"cluster_metric_include,omitempty"`
	DatastoreInclude []string        `toml:"datastore_metric_include,omitempty" json:"datastore_metric_include,omitempty"`
	Labels         map[string]string `toml:"tags,omitempty" json:"tags,omitempty"`
}

// IPMIExporterConfig Prometheus IPMI Exporter配置
type IPMIExporterConfig struct {
	Modules map[string]IPMIExporterModule `yaml:"modules" json:"modules"`
}

// IPMIExporterModule IPMI Exporter模块配置
type IPMIExporterModule struct {
	User     string `yaml:"user,omitempty" json:"user,omitempty"`
	Password string `yaml:"pass,omitempty" json:"password,omitempty"`
	Priv     string `yaml:"priv,omitempty" json:"priv,omitempty"` // admin, operator, user
	AuthType string `yaml:"auth_type,omitempty" json:"auth_type,omitempty"` // md5, sha1, sha256
	Timeout  int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
}

// HardwareMonitorRequest 硬件监控配置请求
type HardwareMonitorRequest struct {
	// 设备列表
	IPMIDevices   []IPMIDeviceConfig   `json:"ipmi_devices,omitempty"`
	RedfishDevices []RedfishDeviceConfig `json:"redfish_devices,omitempty"`
	VMwareVCenters []VMwareVCenterConfig `json:"vmware_vcenters,omitempty"`
	
	// 输出格式
	Format string `json:"format"` // telegraf, ipmi_exporter, redfish_exporter, all
	
	// 全局配置
	GlobalLabels map[string]string `json:"global_labels,omitempty"`
	
	// 用户描述
	Description string `json:"description,omitempty"`
}

// IPMIDeviceConfig IPMI设备配置
type IPMIDeviceConfig struct {
	Name      string            `json:"name"`
	Host      string            `json:"host"`
	Username  string            `json:"username,omitempty"`
	Password  string            `json:"password,omitempty"`
	Interface string            `json:"interface,omitempty"` // lan, lanplus
	Port      int               `json:"port,omitempty"`
	Vendor    string            `json:"vendor,omitempty"` // dell, hpe, supermicro, generic
	Labels    map[string]string `json:"labels,omitempty"`
}

// RedfishDeviceConfig Redfish设备配置
type RedfishDeviceConfig struct {
	Name     string            `json:"name"`
	Host     string            `json:"host"`
	Username string            `json:"username"`
	Password string            `json:"password"`
	Port     int               `json:"port,omitempty"`
	Vendor   string            `json:"vendor,omitempty"` // dell_idrac, hpe_ilo, lenovo_xclarity, generic
	Insecure bool              `json:"insecure,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

// HardwareMonitorResult 硬件监控配置生成结果
type HardwareMonitorResult struct {
	TelegrafIPMIConfig   string `json:"telegraf_ipmi_config,omitempty"`
	TelegrafRedfishConfig string `json:"telegraf_redfish_config,omitempty"`
	TelegrafVMwareConfig string `json:"telegraf_vmware_config,omitempty"`
	IPMIExporterConfig   string `json:"ipmi_exporter_config,omitempty"`
	GeneratedAt          time.Time `json:"generated_at"`
	DevicesConfigured    []string  `json:"devices_configured"`
	Warnings             []string  `json:"warnings,omitempty"`
}
