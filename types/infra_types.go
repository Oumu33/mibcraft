package types

import "time"

// ==================== 基础设施监控配置类型 ====================

// InfraMonitorRequest 基础设施监控配置请求
type InfraMonitorRequest struct {
	// 环境信息
	Environment   string `json:"environment"`   // production, staging, test
	Datacenter    string `json:"datacenter"`    // dc1, dc2
	Cluster       string `json:"cluster"`       // monitoring-cluster

	// 监控目标配置
	NodeExporters   []NodeExporterTarget   `json:"node_exporters,omitempty"`
	SNMPDevices     []SNMPDeviceTarget     `json:"snmp_devices,omitempty"`
	BlackboxTargets []BlackboxTarget       `json:"blackbox_targets,omitempty"`
	RedfishDevices  []RedfishDeviceConfig  `json:"redfish_devices,omitempty"`
	IPMIDevices     []IPMIDeviceConfig     `json:"ipmi_devices,omitempty"`
	VMwareVCenters  []VMwareVCenterConfig  `json:"vmware_vcenters,omitempty"`
	ProxmoxHosts    []ProxmoxHostConfig    `json:"proxmox_hosts,omitempty"`
	TopologyDevices []TopologyDeviceConfig `json:"topology_devices,omitempty"`

	// 输出配置
	OutputDir string `json:"output_dir"`
	Format    string `json:"format"` // docker-compose, all

	// 全局标签
	GlobalLabels map[string]string `json:"global_labels,omitempty"`
}

// NodeExporterTarget Node Exporter 目标
type NodeExporterTarget struct {
	Host     string            `json:"host"`     // IP:Port
	Name     string            `json:"name"`     // 实例名称
	Role     string            `json:"role"`     // web, database, cache, monitoring
	Priority string            `json:"priority"` // P0, P1, P2, P3
	Labels   map[string]string `json:"labels,omitempty"`
}

// SNMPDeviceTarget SNMP 设备目标
type SNMPDeviceTarget struct {
	Host      string            `json:"host"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`      // switch, router, firewall
	Tier      string            `json:"tier"`      // core, aggregation, access
	Vendor    string            `json:"vendor"`    // cisco, huawei, h3c, ruijie
	Community string            `json:"community"` // SNMP community
	Version   int               `json:"version"`   // 1, 2, 3
	Module    string            `json:"module"`    // if_mib, cisco_mib, huawei_mib
	Labels    map[string]string `json:"labels,omitempty"`
}

// BlackboxTarget Blackbox 探测目标
type BlackboxTarget struct {
	URL       string            `json:"url"`        // 探测 URL/IP
	Name      string            `json:"name"`       // 名称
	Module    string            `json:"module"`     // http_2xx, icmp, tcp_connect
	Type      string            `json:"type"`       // http, https, icmp, tcp
	Priority  string            `json:"priority"`   // P0, P1, P2, P3
	Labels    map[string]string `json:"labels,omitempty"`
}

// ProxmoxHostConfig Proxmox VE 主机配置
type ProxmoxHostConfig struct {
	Name      string            `json:"name"`
	Host      string            `json:"host"`      // IP 或域名
	Port      int               `json:"port"`      // API 端口，默认 8006
	Username  string            `json:"username"`  // 用户名，如 root@pam
	Password  string            `json:"password"`  // 密码或 API Token
	Insecure  bool              `json:"insecure"`  // 跳过 SSL 验证
	Labels    map[string]string `json:"labels,omitempty"`
}

// TopologyDeviceConfig 拓扑发现设备配置
type TopologyDeviceConfig struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Type      string `json:"type"`      // switch, router, firewall
	Tier      string `json:"tier"`      // core, aggregation, access
	Vendor    string `json:"vendor"`    // huawei, h3c, cisco, ruijie
	Location  string `json:"location"`  // dc1-rack-A01
	Community string `json:"community"` // SNMP community
	Port      int    `json:"port"`      // SNMP 端口
	Protocol  string `json:"protocol"`  // lldp, cdp, ndp, lnp, auto
}

// InfraMonitorResult 基础设施监控配置生成结果
type InfraMonitorResult struct {
	// 配置文件内容
	DockerCompose      string `json:"docker_compose,omitempty"`
	VmagentConfig      string `json:"vmagent_config,omitempty"`
	VMalertConfig      string `json:"vmalert_config,omitempty"`
	AlertmanagerConfig string `json:"alertmanager_config,omitempty"`
	TelegrafConfig     string `json:"telegraf_config,omitempty"`
	BlackboxConfig     string `json:"blackbox_config,omitempty"`
	SNMPExporterConfig string `json:"snmp_exporter_config,omitempty"`
	TopologyConfig     string `json:"topology_config,omitempty"`
	RedfishConfig      string `json:"redfish_config,omitempty"`
	IPMIExporterConfig string `json:"ipmi_exporter_config,omitempty"`
	EnvFile            string `json:"env_file,omitempty"`

	// File SD 目标文件
	NodeExporterTargets   string `json:"node_exporter_targets,omitempty"`
	SNMPTargets           string `json:"snmp_targets,omitempty"`
	BlackboxHTTPTargets   string `json:"blackbox_http_targets,omitempty"`
	BlackboxICMPTargets   string `json:"blackbox_icmp_targets,omitempty"`
	RedfishTargets        string `json:"redfish_targets,omitempty"`
	IPMITargets           string `json:"ipmi_targets,omitempty"`

	GeneratedAt        time.Time `json:"generated_at"`
	TargetsConfigured  []string  `json:"targets_configured"`
	Warnings           []string  `json:"warnings,omitempty"`
}

// FileSDTarget Prometheus File SD 目标格式
type FileSDTarget struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// DockerComposeConfig Docker Compose 配置
type DockerComposeConfig struct {
	Version  string                 `yaml:"version"`
	Services map[string]ServiceConfig `yaml:"services"`
	Volumes  map[string]VolumeConfig `yaml:"volumes,omitempty"`
	Networks map[string]NetworkConfig `yaml:"networks,omitempty"`
}

// ServiceConfig Docker 服务配置
type ServiceConfig struct {
	Image       string            `yaml:"image"`
	ContainerName string          `yaml:"container_name,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Volumes     []string          `yaml:"volumes,omitempty"`
	Command     []string          `yaml:"command,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	EnvFile     []string          `yaml:"env_file,omitempty"`
	DependsOn   []string          `yaml:"depends_on,omitempty"`
	Networks    []string          `yaml:"networks,omitempty"`
	Restart     string            `yaml:"restart,omitempty"`
	Build       *BuildConfig      `yaml:"build,omitempty"`
}

// BuildConfig Docker 构建配置
type BuildConfig struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

// VolumeConfig Docker 卷配置
type VolumeConfig struct {
	Driver string `yaml:"driver,omitempty"`
}

// NetworkConfig Docker 网络配置
type NetworkConfig struct {
	Driver string `yaml:"driver,omitempty"`
}
