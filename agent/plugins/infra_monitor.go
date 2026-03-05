package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Oumu33/mibcraft/agent"
	"github.com/Oumu33/mibcraft/types"
)

// InfraMonitorPlugin 基础设施监控配置生成插件
// 生成完整的监控栈配置，包括所有 Monitoring-deployment 中的组件
type InfraMonitorPlugin struct {
	config  *InfraMonitorPluginConfig
	lastGen time.Time
}

// InfraMonitorPluginConfig 基础设施监控插件配置
type InfraMonitorPluginConfig struct {
	agent.BasePluginConfig
	// 输出目录
	OutputDir string `toml:"output_dir"`
	// 设备配置文件路径
	DevicesFile string `toml:"devices_file"`
	// 是否监听变化
	WatchChanges bool `toml:"watch_changes"`
	// 环境信息
	Environment string `toml:"environment"`
	Datacenter  string `toml:"datacenter"`
	Cluster     string `toml:"cluster"`
	// 全局标签
	GlobalLabels map[string]string `toml:"global_labels"`

	// 监控目标配置（可直接在配置中定义）
	NodeExporters   []types.NodeExporterTarget   `toml:"node_exporters"`
	SNMPDevices     []types.SNMPDeviceTarget     `toml:"snmp_devices"`
	BlackboxTargets []types.BlackboxTarget       `toml:"blackbox_targets"`
	RedfishDevices  []types.RedfishDeviceConfig  `toml:"redfish_devices"`
	IPMIDevices     []types.IPMIDeviceConfig     `toml:"ipmi_devices"`
	VMwareVCenters  []types.VMwareVCenterConfig  `toml:"vmware_vcenters"`
	ProxmoxHosts    []types.ProxmoxHostConfig    `toml:"proxmox_hosts"`
	TopologyDevices []types.TopologyDeviceConfig `toml:"topology_devices"`
}

// InfraDevicesConfig 设备配置文件格式
type InfraDevicesConfig struct {
	Environment     string                        `toml:"environment"`
	Datacenter      string                        `toml:"datacenter"`
	Cluster         string                        `toml:"cluster"`
	NodeExporters   []types.NodeExporterTarget    `toml:"node_exporters"`
	SNMPDevices     []types.SNMPDeviceTarget      `toml:"snmp_devices"`
	BlackboxTargets []types.BlackboxTarget        `toml:"blackbox_targets"`
	RedfishDevices  []types.RedfishDeviceConfig   `toml:"redfish_devices"`
	IPMIDevices     []types.IPMIDeviceConfig      `toml:"ipmi_devices"`
	VMwareVCenters  []types.VMwareVCenterConfig   `toml:"vmware_vcenters"`
	ProxmoxHosts    []types.ProxmoxHostConfig     `toml:"proxmox_hosts"`
	TopologyDevices []types.TopologyDeviceConfig  `toml:"topology_devices"`
	GlobalLabels    map[string]string             `toml:"global_labels"`
}

// NewInfraMonitorPlugin 创建基础设施监控插件
func NewInfraMonitorPlugin() *InfraMonitorPlugin {
	return &InfraMonitorPlugin{}
}

func (p *InfraMonitorPlugin) Name() string {
	return "infra_monitor"
}

func (p *InfraMonitorPlugin) Description() string {
	return "生成完整的基础设施监控配置，支持 Node Exporter、SNMP、Blackbox、Redfish、IPMI、VMware、Proxmox、拓扑发现等"
}

func (p *InfraMonitorPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*InfraMonitorPluginConfig); ok {
			p.config = cfg
		}
	}

	if p.config == nil {
		p.config = &InfraMonitorPluginConfig{
			OutputDir:      "./output/infra",
			WatchChanges:   true,
			Environment:    "production",
			Datacenter:     "dc1",
			Cluster:        "monitoring-cluster",
			GlobalLabels:   make(map[string]string),
		}
	}

	// 确保输出目录存在
	os.MkdirAll(p.config.OutputDir, 0755)
	// 创建子目录
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/vmagent/targets"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/telegraf"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/blackbox-exporter"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/snmp-exporter"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/topology"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/redfish-exporter"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/ipmi-exporter"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/alertmanager"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/vmalert/alerts"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/loki/rules"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/grafana/provisioning/datasources"), 0755)
	os.MkdirAll(filepath.Join(p.config.OutputDir, "config/grafana/provisioning/dashboards"), 0755)

	return nil
}

func (p *InfraMonitorPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Labels:  p.config.GlobalLabels,
	}

	// 加载设备配置
	devices, err := p.LoadDevices()
	if err != nil {
		result.OK = false
		result.Message = fmt.Sprintf("加载设备配置失败: %v", err)
		return result, nil
	}

	// 检查是否有变化
	if p.config.WatchChanges {
		if changed := p.checkChanges(); changed {
			// 重新生成配置
			if err := p.RegenerateConfig(devices); err != nil {
				result.OK = false
				result.Message = fmt.Sprintf("重新生成配置失败: %v", err)
				return result, nil
			}
			result.Message = "检测到设备变化，已重新生成配置"
		} else {
			result.Message = "设备配置无变化"
		}
	}

	// 统计信息
	result.Metrics["node_exporters_count"] = len(devices.NodeExporters)
	result.Metrics["snmp_devices_count"] = len(devices.SNMPDevices)
	result.Metrics["blackbox_targets_count"] = len(devices.BlackboxTargets)
	result.Metrics["redfish_devices_count"] = len(devices.RedfishDevices)
	result.Metrics["ipmi_devices_count"] = len(devices.IPMIDevices)
	result.Metrics["vmware_vcenters_count"] = len(devices.VMwareVCenters)
	result.Metrics["proxmox_hosts_count"] = len(devices.ProxmoxHosts)
	result.Metrics["topology_devices_count"] = len(devices.TopologyDevices)
	result.Metrics["config_last_gen"] = p.lastGen.Unix()

	return result, nil
}

// loadDevices 加载设备配置
func (p *InfraMonitorPlugin) LoadDevices() (*InfraDevicesConfig, error) {
	devices := &InfraDevicesConfig{
		Environment:     p.config.Environment,
		Datacenter:      p.config.Datacenter,
		Cluster:         p.config.Cluster,
		NodeExporters:   p.config.NodeExporters,
		SNMPDevices:     p.config.SNMPDevices,
		BlackboxTargets: p.config.BlackboxTargets,
		RedfishDevices:  p.config.RedfishDevices,
		IPMIDevices:     p.config.IPMIDevices,
		VMwareVCenters:  p.config.VMwareVCenters,
		ProxmoxHosts:    p.config.ProxmoxHosts,
		TopologyDevices: p.config.TopologyDevices,
		GlobalLabels:    p.config.GlobalLabels,
	}

	// 如果配置了设备文件，从文件加载
	if p.config.DevicesFile != "" {
		data, err := os.ReadFile(p.config.DevicesFile)
		if err != nil {
			if os.IsNotExist(err) {
				// 文件不存在，使用内联配置
				return devices, nil
			}
			return nil, err
		}

		var fileDevices InfraDevicesConfig
		if _, err := toml.Decode(string(data), &fileDevices); err != nil {
			return nil, err
		}

		// 合并配置
		if fileDevices.Environment != "" {
			devices.Environment = fileDevices.Environment
		}
		if fileDevices.Datacenter != "" {
			devices.Datacenter = fileDevices.Datacenter
		}
		if fileDevices.Cluster != "" {
			devices.Cluster = fileDevices.Cluster
		}
		if len(fileDevices.NodeExporters) > 0 {
			devices.NodeExporters = fileDevices.NodeExporters
		}
		if len(fileDevices.SNMPDevices) > 0 {
			devices.SNMPDevices = fileDevices.SNMPDevices
		}
		if len(fileDevices.BlackboxTargets) > 0 {
			devices.BlackboxTargets = fileDevices.BlackboxTargets
		}
		if len(fileDevices.RedfishDevices) > 0 {
			devices.RedfishDevices = fileDevices.RedfishDevices
		}
		if len(fileDevices.IPMIDevices) > 0 {
			devices.IPMIDevices = fileDevices.IPMIDevices
		}
		if len(fileDevices.VMwareVCenters) > 0 {
			devices.VMwareVCenters = fileDevices.VMwareVCenters
		}
		if len(fileDevices.ProxmoxHosts) > 0 {
			devices.ProxmoxHosts = fileDevices.ProxmoxHosts
		}
		if len(fileDevices.TopologyDevices) > 0 {
			devices.TopologyDevices = fileDevices.TopologyDevices
		}
		if len(fileDevices.GlobalLabels) > 0 {
			if devices.GlobalLabels == nil {
				devices.GlobalLabels = make(map[string]string)
			}
			for k, v := range fileDevices.GlobalLabels {
				devices.GlobalLabels[k] = v
			}
		}
	}

	return devices, nil
}

// checkChanges 检查配置变化
func (p *InfraMonitorPlugin) checkChanges() bool {
	// 检查设备配置文件
	if p.config.DevicesFile != "" {
		info, err := os.Stat(p.config.DevicesFile)
		if err == nil && info.ModTime().After(p.lastGen) {
			return true
		}
	}
	return false
}

// regenerateConfig 重新生成所有配置
func (p *InfraMonitorPlugin) RegenerateConfig(devices *InfraDevicesConfig) error {
	outputDir := p.config.OutputDir

	// 1. 生成 docker-compose.yml
	dockerCompose := p.generateDockerCompose(devices)
	if err := os.WriteFile(filepath.Join(outputDir, "docker-compose.yaml"), []byte(dockerCompose), 0644); err != nil {
		return fmt.Errorf("写入 docker-compose.yaml 失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "docker-compose.yaml"))

	// 2. 生成 vmagent prometheus.yml
	vmagentConfig := p.generateVmagentConfig(devices)
	if err := os.WriteFile(filepath.Join(outputDir, "config/vmagent/prometheus.yml"), []byte(vmagentConfig), 0644); err != nil {
		return fmt.Errorf("写入 vmagent 配置失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/vmagent/prometheus.yml"))

	// 3. 生成 File SD 目标文件
	if len(devices.NodeExporters) > 0 {
		targets := p.generateNodeExporterTargets(devices.NodeExporters, devices.GlobalLabels)
		if err := os.WriteFile(filepath.Join(outputDir, "config/vmagent/targets/node-exporters.json"), []byte(targets), 0644); err != nil {
			return fmt.Errorf("写入 node-exporters 目标失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/vmagent/targets/node-exporters.json"))
	}

	if len(devices.SNMPDevices) > 0 {
		targets := p.generateSNMPTargets(devices.SNMPDevices, devices.GlobalLabels)
		if err := os.WriteFile(filepath.Join(outputDir, "config/vmagent/targets/snmp-devices.json"), []byte(targets), 0644); err != nil {
			return fmt.Errorf("写入 SNMP 目标失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/vmagent/targets/snmp-devices.json"))
	}

	if len(devices.BlackboxTargets) > 0 {
		httpTargets, icmpTargets := p.generateBlackboxTargets(devices.BlackboxTargets, devices.GlobalLabels)
		if len(httpTargets) > 0 {
			if err := os.WriteFile(filepath.Join(outputDir, "config/vmagent/targets/blackbox-http.json"), []byte(httpTargets), 0644); err != nil {
				return fmt.Errorf("写入 Blackbox HTTP 目标失败: %w", err)
			}
			fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/vmagent/targets/blackbox-http.json"))
		}
		if len(icmpTargets) > 0 {
			if err := os.WriteFile(filepath.Join(outputDir, "config/vmagent/targets/blackbox-icmp.json"), []byte(icmpTargets), 0644); err != nil {
				return fmt.Errorf("写入 Blackbox ICMP 目标失败: %w", err)
			}
			fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/vmagent/targets/blackbox-icmp.json"))
		}
	}

	// 4. 生成 Telegraf VMware 配置
	if len(devices.VMwareVCenters) > 0 {
		telegrafConfig := p.generateTelegrafVMwareConfig(devices.VMwareVCenters, devices.GlobalLabels)
		if err := os.WriteFile(filepath.Join(outputDir, "config/telegraf/telegraf.conf"), []byte(telegrafConfig), 0644); err != nil {
			return fmt.Errorf("写入 Telegraf 配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/telegraf/telegraf.conf"))
	}

	// 5. 生成 Telegraf Redfish 配置
	if len(devices.RedfishDevices) > 0 {
		redfishConfig := p.generateTelegrafRedfishConfig(devices.RedfishDevices, devices.GlobalLabels)
		if err := os.WriteFile(filepath.Join(outputDir, "config/telegraf/telegraf-redfish.conf"), []byte(redfishConfig), 0644); err != nil {
			return fmt.Errorf("写入 Redfish 配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/telegraf/telegraf-redfish.conf"))
	}

	// 6. 生成 Telegraf IPMI 配置
	if len(devices.IPMIDevices) > 0 {
		ipmiConfig := p.generateTelegrafIPMIConfig(devices.IPMIDevices, devices.GlobalLabels)
		if err := os.WriteFile(filepath.Join(outputDir, "config/telegraf/telegraf-ipmi.conf"), []byte(ipmiConfig), 0644); err != nil {
			return fmt.Errorf("写入 IPMI 配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/telegraf/telegraf-ipmi.conf"))
	}

	// 7. 生成拓扑发现配置
	if len(devices.TopologyDevices) > 0 {
		topologyConfig := p.generateTopologyConfig(devices.TopologyDevices)
		if err := os.WriteFile(filepath.Join(outputDir, "config/topology/devices.yml"), []byte(topologyConfig), 0644); err != nil {
			return fmt.Errorf("写入拓扑配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/topology/devices.yml"))
	}

	// 8. 生成 Blackbox Exporter 配置
	blackboxConfig := p.generateBlackboxExporterConfig()
	if err := os.WriteFile(filepath.Join(outputDir, "config/blackbox-exporter/blackbox.yml"), []byte(blackboxConfig), 0644); err != nil {
		return fmt.Errorf("写入 Blackbox 配置失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/blackbox-exporter/blackbox.yml"))

	// 9. 生成 Redfish Exporter 配置
	if len(devices.RedfishDevices) > 0 {
		redfishExporterConfig := p.generateRedfishExporterConfig(devices.RedfishDevices)
		if err := os.WriteFile(filepath.Join(outputDir, "config/redfish-exporter/redfish.yml"), []byte(redfishExporterConfig), 0644); err != nil {
			return fmt.Errorf("写入 Redfish Exporter 配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/redfish-exporter/redfish.yml"))
	}

	// 10. 生成 Alertmanager 配置
	alertmanagerConfig := p.generateAlertmanagerConfig()
	if err := os.WriteFile(filepath.Join(outputDir, "config/alertmanager/alertmanager.yml"), []byte(alertmanagerConfig), 0644); err != nil {
		return fmt.Errorf("写入 Alertmanager 配置失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", filepath.Join(outputDir, "config/alertmanager/alertmanager.yml"))

	// 11. 生成 .env 文件
	envFile := p.generateEnvFile(devices)
	if err := os.WriteFile(filepath.Join(outputDir, ".env"), []byte(envFile), 0644); err != nil {
		return fmt.Errorf("写入 .env 文件失败: %w", err)
	}
	fmt.Printf("已生成: %s\n", filepath.Join(outputDir, ".env"))

	p.lastGen = time.Now()
	return nil
}

// GenerateConfig 手动触发配置生成
func (p *InfraMonitorPlugin) GenerateConfig() (*types.InfraMonitorResult, error) {
	devices, err := p.LoadDevices()
	if err != nil {
		return nil, err
	}

	result := &types.InfraMonitorResult{
		GeneratedAt:       time.Now(),
		TargetsConfigured: make([]string, 0),
		Warnings:          make([]string, 0),
	}

	// 生成所有配置
	result.DockerCompose = p.generateDockerCompose(devices)
	result.VmagentConfig = p.generateVmagentConfig(devices)
	result.TelegrafConfig = p.generateTelegrafVMwareConfig(devices.VMwareVCenters, devices.GlobalLabels)
	result.BlackboxConfig = p.generateBlackboxExporterConfig()
	result.AlertmanagerConfig = p.generateAlertmanagerConfig()
	result.EnvFile = p.generateEnvFile(devices)

	// 生成目标文件
	if len(devices.NodeExporters) > 0 {
		result.NodeExporterTargets = p.generateNodeExporterTargets(devices.NodeExporters, devices.GlobalLabels)
		for _, t := range devices.NodeExporters {
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("NodeExporter: %s (%s)", t.Name, t.Host))
		}
	}

	if len(devices.SNMPDevices) > 0 {
		result.SNMPTargets = p.generateSNMPTargets(devices.SNMPDevices, devices.GlobalLabels)
		for _, t := range devices.SNMPDevices {
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("SNMP: %s (%s)", t.Name, t.Host))
		}
	}

	if len(devices.RedfishDevices) > 0 {
		result.RedfishConfig = p.generateTelegrafRedfishConfig(devices.RedfishDevices, devices.GlobalLabels)
		result.RedfishTargets = p.generateRedfishExporterConfig(devices.RedfishDevices)
		for _, t := range devices.RedfishDevices {
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("Redfish: %s (%s)", t.Name, t.Host))
		}
	}

	if len(devices.IPMIDevices) > 0 {
		result.IPMIExporterConfig = p.generateTelegrafIPMIConfig(devices.IPMIDevices, devices.GlobalLabels)
		for _, t := range devices.IPMIDevices {
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("IPMI: %s (%s)", t.Name, t.Host))
		}
	}

	if len(devices.VMwareVCenters) > 0 {
		for _, vc := range devices.VMwareVCenters {
			name := vc.Name
			if name == "" {
				name = vc.URL
			}
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("VMware: %s", name))
		}
	}

	if len(devices.TopologyDevices) > 0 {
		result.TopologyConfig = p.generateTopologyConfig(devices.TopologyDevices)
		for _, t := range devices.TopologyDevices {
			result.TargetsConfigured = append(result.TargetsConfigured, fmt.Sprintf("Topology: %s (%s)", t.Name, t.Host))
		}
	}

	p.lastGen = time.Now()
	return result, nil
}

// generateDockerCompose 生成 docker-compose.yaml
func (p *InfraMonitorPlugin) generateDockerCompose(devices *InfraDevicesConfig) string {
	var sb strings.Builder

	sb.WriteString("# ===================================================================\n")
	sb.WriteString("# 基础设施监控平台 - Docker Compose 配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n")
	sb.WriteString("# ===================================================================\n\n")

	sb.WriteString("version: '3.8'\n\n")

	sb.WriteString("services:\n")

	// VictoriaMetrics
	sb.WriteString("  # VictoriaMetrics - 时序数据库\n")
	sb.WriteString("  victoriametrics:\n")
	sb.WriteString("    image: victoriametrics/victoria-metrics:latest\n")
	sb.WriteString("    container_name: victoriametrics\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"8428:8428\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - vmdata:/storage\n")
	sb.WriteString("    command:\n")
	sb.WriteString("      - \"--storageDataPath=/storage\"\n")
	sb.WriteString("      - \"--httpListenAddr=:8428\"\n")
	sb.WriteString("      - \"--retentionPeriod=12\"\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// vmagent
	sb.WriteString("  # vmagent - 指标收集代理\n")
	sb.WriteString("  vmagent:\n")
	sb.WriteString("    image: victoriametrics/vmagent:latest\n")
	sb.WriteString("    container_name: vmagent\n")
	sb.WriteString("    depends_on:\n")
	sb.WriteString("      - victoriametrics\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"8429:8429\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - ./config/vmagent/prometheus.yml:/etc/prometheus/prometheus.yml\n")
	sb.WriteString("      - ./config/vmagent/targets:/etc/prometheus/targets\n")
	sb.WriteString("    command:\n")
	sb.WriteString("      - \"--promscrape.config=/etc/prometheus/prometheus.yml\"\n")
	sb.WriteString("      - \"--remoteWrite.url=http://victoriametrics:8428/api/v1/write\"\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// vmalert
	sb.WriteString("  # vmalert - 告警规则引擎\n")
	sb.WriteString("  vmalert:\n")
	sb.WriteString("    image: victoriametrics/vmalert:latest\n")
	sb.WriteString("    container_name: vmalert\n")
	sb.WriteString("    depends_on:\n")
	sb.WriteString("      - victoriametrics\n")
	sb.WriteString("      - alertmanager\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"8880:8880\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - ./config/vmalert/alerts:/etc/alerts\n")
	sb.WriteString("    command:\n")
	sb.WriteString("      - \"--datasource.url=http://victoriametrics:8428\"\n")
	sb.WriteString("      - \"--remoteRead.url=http://victoriametrics:8428\"\n")
	sb.WriteString("      - \"--remoteWrite.url=http://victoriametrics:8428\"\n")
	sb.WriteString("      - \"--notifier.url=http://alertmanager:9093\"\n")
	sb.WriteString("      - \"--rule=/etc/alerts/*.yml\"\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Alertmanager
	sb.WriteString("  # Alertmanager - 告警管理\n")
	sb.WriteString("  alertmanager:\n")
	sb.WriteString("    image: prom/alertmanager:latest\n")
	sb.WriteString("    container_name: alertmanager\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"9093:9093\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - ./config/alertmanager/alertmanager.yml:/etc/alertmanager/alertmanager.yml\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Grafana
	sb.WriteString("  # Grafana - 可视化\n")
	sb.WriteString("  grafana:\n")
	sb.WriteString("    image: grafana/grafana:latest\n")
	sb.WriteString("    container_name: grafana\n")
	sb.WriteString("    depends_on:\n")
	sb.WriteString("      - victoriametrics\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"3000:3000\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - grafana-data:/var/lib/grafana\n")
	sb.WriteString("      - ./config/grafana/provisioning:/etc/grafana/provisioning\n")
	sb.WriteString("    environment:\n")
	sb.WriteString("      - GF_SECURITY_ADMIN_USER=${GRAFANA_ADMIN_USER:-admin}\n")
	sb.WriteString("      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD:-admin}\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Node Exporter
	sb.WriteString("  # Node Exporter - Linux 节点监控\n")
	sb.WriteString("  node-exporter:\n")
	sb.WriteString("    image: prom/node-exporter:latest\n")
	sb.WriteString("    container_name: node-exporter\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"9100:9100\"\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// SNMP Exporter
	sb.WriteString("  # SNMP Exporter - SNMP 设备监控\n")
	sb.WriteString("  snmp-exporter:\n")
	sb.WriteString("    image: prom/snmp-exporter:latest\n")
	sb.WriteString("    container_name: snmp-exporter\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"9116:9116\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - ./config/snmp-exporter/snmp.yml:/etc/snmp_exporter/snmp.yml\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Blackbox Exporter
	sb.WriteString("  # Blackbox Exporter - 服务可用性探测\n")
	sb.WriteString("  blackbox-exporter:\n")
	sb.WriteString("    image: prom/blackbox-exporter:latest\n")
	sb.WriteString("    container_name: blackbox-exporter\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"9115:9115\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - ./config/blackbox-exporter/blackbox.yml:/etc/blackbox_exporter/config.yml\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Telegraf VMware (如果有配置)
	if len(devices.VMwareVCenters) > 0 {
		sb.WriteString("  # Telegraf - VMware 监控\n")
		sb.WriteString("  telegraf-vmware:\n")
		sb.WriteString("    image: telegraf:latest\n")
		sb.WriteString("    container_name: telegraf-vmware\n")
		sb.WriteString("    depends_on:\n")
		sb.WriteString("      - victoriametrics\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - ./config/telegraf/telegraf.conf:/etc/telegraf/telegraf.conf:ro\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    networks:\n")
		sb.WriteString("      - monitoring\n\n")
	}

	// Telegraf Redfish (如果有配置)
	if len(devices.RedfishDevices) > 0 {
		sb.WriteString("  # Telegraf - Redfish 硬件监控\n")
		sb.WriteString("  telegraf-redfish:\n")
		sb.WriteString("    image: telegraf:latest\n")
		sb.WriteString("    container_name: telegraf-redfish\n")
		sb.WriteString("    depends_on:\n")
		sb.WriteString("      - victoriametrics\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - ./config/telegraf/telegraf-redfish.conf:/etc/telegraf/telegraf.conf:ro\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    networks:\n")
		sb.WriteString("      - monitoring\n\n")
	}

	// Redfish Exporter (如果有配置)
	if len(devices.RedfishDevices) > 0 {
		sb.WriteString("  # Redfish Exporter - 服务器硬件监控\n")
		sb.WriteString("  redfish-exporter:\n")
		sb.WriteString("    image: jenningsloy318/redfish_exporter:latest\n")
		sb.WriteString("    container_name: redfish-exporter\n")
		sb.WriteString("    ports:\n")
		sb.WriteString("      - \"9610:9610\"\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - ./config/redfish-exporter/redfish.yml:/etc/redfish_exporter/redfish.yml:ro\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    networks:\n")
		sb.WriteString("      - monitoring\n\n")
	}

	// IPMI Exporter (如果有配置)
	if len(devices.IPMIDevices) > 0 {
		sb.WriteString("  # IPMI Exporter - 老服务器硬件监控\n")
		sb.WriteString("  ipmi-exporter:\n")
		sb.WriteString("    image: prometheuscommunity/ipmi-exporter:latest\n")
		sb.WriteString("    container_name: ipmi-exporter\n")
	sb.WriteString("    ports:\n")
		sb.WriteString("      - \"9290:9290\"\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    networks:\n")
		sb.WriteString("      - monitoring\n\n")
	}

	// Loki
	sb.WriteString("  # Loki - 日志聚合存储\n")
	sb.WriteString("  loki:\n")
	sb.WriteString("    image: grafana/loki:latest\n")
	sb.WriteString("    container_name: loki\n")
	sb.WriteString("    ports:\n")
	sb.WriteString("      - \"3100:3100\"\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - loki-data:/loki\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Promtail
	sb.WriteString("  # Promtail - 日志采集\n")
	sb.WriteString("  promtail:\n")
	sb.WriteString("    image: grafana/promtail:latest\n")
	sb.WriteString("    container_name: promtail\n")
	sb.WriteString("    depends_on:\n")
	sb.WriteString("      - loki\n")
	sb.WriteString("    volumes:\n")
	sb.WriteString("      - /var/log:/var/log:ro\n")
	sb.WriteString("    restart: unless-stopped\n")
	sb.WriteString("    networks:\n")
	sb.WriteString("      - monitoring\n\n")

	// Topology Discovery (如果有配置)
	if len(devices.TopologyDevices) > 0 {
		sb.WriteString("  # Topology Discovery - LLDP 拓扑自动发现\n")
		sb.WriteString("  topology-discovery:\n")
		sb.WriteString("    build:\n")
		sb.WriteString("      context: .\n")
		sb.WriteString("      dockerfile: Dockerfile.topology\n")
		sb.WriteString("    container_name: topology-discovery\n")
		sb.WriteString("    volumes:\n")
		sb.WriteString("      - ./config/topology/devices.yml:/etc/topology/devices.yml:ro\n")
		sb.WriteString("      - ./data/topology:/data/topology\n")
		sb.WriteString("    environment:\n")
		sb.WriteString("      - DISCOVERY_INTERVAL=300\n")
		sb.WriteString("    restart: unless-stopped\n")
		sb.WriteString("    networks:\n")
		sb.WriteString("      - monitoring\n\n")
	}

	// Volumes
	sb.WriteString("volumes:\n")
	sb.WriteString("  vmdata:\n")
	sb.WriteString("  grafana-data:\n")
	sb.WriteString("  loki-data:\n\n")

	// Networks
	sb.WriteString("networks:\n")
	sb.WriteString("  monitoring:\n")
	sb.WriteString("    driver: bridge\n")

	return sb.String()
}

// generateVmagentConfig 生成 vmagent prometheus.yml
func (p *InfraMonitorPlugin) generateVmagentConfig(devices *InfraDevicesConfig) string {
	var sb strings.Builder

	sb.WriteString("# vmagent 配置文件\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("global:\n")
	sb.WriteString("  scrape_interval: 15s\n")
	sb.WriteString("  scrape_timeout: 10s\n")
	sb.WriteString("  external_labels:\n")
	sb.WriteString(fmt.Sprintf("    cluster: '%s'\n", devices.Cluster))
	sb.WriteString(fmt.Sprintf("    environment: '%s'\n", devices.Environment))
	sb.WriteString(fmt.Sprintf("    datacenter: '%s'\n\n", devices.Datacenter))

	sb.WriteString("scrape_configs:\n")

	// 自监控
	sb.WriteString("  # VictoriaMetrics 自监控\n")
	sb.WriteString("  - job_name: 'victoriametrics'\n")
	sb.WriteString("    static_configs:\n")
	sb.WriteString("      - targets: ['victoriametrics:8428']\n\n")

	sb.WriteString("  # vmagent 自监控\n")
	sb.WriteString("  - job_name: 'vmagent'\n")
	sb.WriteString("    static_configs:\n")
	sb.WriteString("      - targets: ['vmagent:8429']\n\n")

	// Node Exporter
	sb.WriteString("  # Node Exporter\n")
	sb.WriteString("  - job_name: 'node-exporter'\n")
	sb.WriteString("    file_sd_configs:\n")
	sb.WriteString("      - files:\n")
	sb.WriteString("        - /etc/prometheus/targets/node-exporters.json\n")
	sb.WriteString("        refresh_interval: 60s\n\n")

	// SNMP Exporter
	sb.WriteString("  # SNMP 设备监控\n")
	sb.WriteString("  - job_name: 'snmp'\n")
	sb.WriteString("    scrape_interval: 30s\n")
	sb.WriteString("    scrape_timeout: 20s\n")
	sb.WriteString("    metrics_path: /snmp\n")
	sb.WriteString("    params:\n")
	sb.WriteString("      module: [if_mib]\n")
	sb.WriteString("    file_sd_configs:\n")
	sb.WriteString("      - files:\n")
	sb.WriteString("        - /etc/prometheus/targets/snmp-devices.json\n")
	sb.WriteString("        refresh_interval: 60s\n")
	sb.WriteString("    relabel_configs:\n")
	sb.WriteString("      - source_labels: [__address__]\n")
	sb.WriteString("        target_label: __param_target\n")
	sb.WriteString("      - source_labels: [__param_target]\n")
	sb.WriteString("        target_label: instance\n")
	sb.WriteString("      - target_label: __address__\n")
	sb.WriteString("        replacement: snmp-exporter:9116\n\n")

	// Blackbox HTTP
	sb.WriteString("  # Blackbox HTTP 探测\n")
	sb.WriteString("  - job_name: 'blackbox-http'\n")
	sb.WriteString("    scrape_interval: 30s\n")
	sb.WriteString("    metrics_path: /probe\n")
	sb.WriteString("    params:\n")
	sb.WriteString("      module: [http_2xx]\n")
	sb.WriteString("    file_sd_configs:\n")
	sb.WriteString("      - files:\n")
	sb.WriteString("        - /etc/prometheus/targets/blackbox-http.json\n")
	sb.WriteString("        refresh_interval: 60s\n")
	sb.WriteString("    relabel_configs:\n")
	sb.WriteString("      - source_labels: [__address__]\n")
	sb.WriteString("        target_label: __param_target\n")
	sb.WriteString("      - source_labels: [__param_target]\n")
	sb.WriteString("        target_label: instance\n")
	sb.WriteString("      - target_label: __address__\n")
	sb.WriteString("        replacement: blackbox-exporter:9115\n\n")

	// Blackbox ICMP
	sb.WriteString("  # Blackbox ICMP 探测\n")
	sb.WriteString("  - job_name: 'blackbox-icmp'\n")
	sb.WriteString("    scrape_interval: 30s\n")
	sb.WriteString("    metrics_path: /probe\n")
	sb.WriteString("    params:\n")
	sb.WriteString("      module: [icmp]\n")
	sb.WriteString("    file_sd_configs:\n")
	sb.WriteString("      - files:\n")
	sb.WriteString("        - /etc/prometheus/targets/blackbox-icmp.json\n")
	sb.WriteString("        refresh_interval: 60s\n")
	sb.WriteString("    relabel_configs:\n")
	sb.WriteString("      - source_labels: [__address__]\n")
	sb.WriteString("        target_label: __param_target\n")
	sb.WriteString("      - source_labels: [__param_target]\n")
	sb.WriteString("        target_label: instance\n")
	sb.WriteString("      - target_label: __address__\n")
	sb.WriteString("        replacement: blackbox-exporter:9115\n\n")

	// Redfish
	if len(devices.RedfishDevices) > 0 {
		sb.WriteString("  # Redfish 硬件监控\n")
		sb.WriteString("  - job_name: 'redfish'\n")
		sb.WriteString("    scrape_interval: 60s\n")
		sb.WriteString("    metrics_path: /redfish\n")
		sb.WriteString("    static_configs:\n")
		for _, d := range devices.RedfishDevices {
			sb.WriteString(fmt.Sprintf("      - targets: ['%s']\n", d.Name))
			sb.WriteString("        labels:\n")
			sb.WriteString(fmt.Sprintf("          instance: '%s'\n", d.Name))
			if d.Vendor != "" {
				sb.WriteString(fmt.Sprintf("          vendor: '%s'\n", d.Vendor))
			}
		}
		sb.WriteString("    relabel_configs:\n")
		sb.WriteString("      - source_labels: [__address__]\n")
		sb.WriteString("        target_label: __param_target\n")
		sb.WriteString("      - target_label: __address__\n")
		sb.WriteString("        replacement: redfish-exporter:9610\n\n")
	}

	// IPMI
	if len(devices.IPMIDevices) > 0 {
		sb.WriteString("  # IPMI 硬件监控\n")
		sb.WriteString("  - job_name: 'ipmi'\n")
		sb.WriteString("    scrape_interval: 60s\n")
		sb.WriteString("    metrics_path: /ipmi\n")
		sb.WriteString("    static_configs:\n")
		for _, d := range devices.IPMIDevices {
			sb.WriteString(fmt.Sprintf("      - targets: ['%s']\n", d.Host))
			sb.WriteString("        labels:\n")
			sb.WriteString(fmt.Sprintf("          instance: '%s'\n", d.Name))
			if d.Vendor != "" {
				sb.WriteString(fmt.Sprintf("          vendor: '%s'\n", d.Vendor))
			}
		}
		sb.WriteString("    relabel_configs:\n")
		sb.WriteString("      - source_labels: [__address__]\n")
		sb.WriteString("        target_label: __param_target\n")
		sb.WriteString("      - source_labels: [__param_target]\n")
		sb.WriteString("        target_label: instance\n")
		sb.WriteString("      - target_label: __address__\n")
		sb.WriteString("        replacement: ipmi-exporter:9290\n\n")
	}

	return sb.String()
}

// generateNodeExporterTargets 生成 Node Exporter File SD 目标
func (p *InfraMonitorPlugin) generateNodeExporterTargets(targets []types.NodeExporterTarget, globalLabels map[string]string) string {
	fileSD := make([]types.FileSDTarget, 0)

	for _, t := range targets {
		labels := make(map[string]string)
		// 添加全局标签
		for k, v := range globalLabels {
			labels[k] = v
		}
		// 添加设备标签
		labels["instance"] = t.Name
		if t.Role != "" {
			labels["role"] = t.Role
		}
		if t.Priority != "" {
			labels["priority"] = t.Priority
		}
		for k, v := range t.Labels {
			labels[k] = v
		}

		fileSD = append(fileSD, types.FileSDTarget{
			Targets: []string{t.Host},
			Labels:  labels,
		})
	}

	data, _ := json.MarshalIndent(fileSD, "", "  ")
	return string(data)
}

// generateSNMPTargets 生成 SNMP File SD 目标
func (p *InfraMonitorPlugin) generateSNMPTargets(targets []types.SNMPDeviceTarget, globalLabels map[string]string) string {
	fileSD := make([]types.FileSDTarget, 0)

	for _, t := range targets {
		labels := make(map[string]string)
		// 添加全局标签
		for k, v := range globalLabels {
			labels[k] = v
		}
		// 添加设备标签
		labels["instance"] = t.Name
		if t.Type != "" {
			labels["device_type"] = t.Type
		}
		if t.Tier != "" {
			labels["device_tier"] = t.Tier
		}
		if t.Vendor != "" {
			labels["vendor"] = t.Vendor
		}
		for k, v := range t.Labels {
			labels[k] = v
		}

		fileSD = append(fileSD, types.FileSDTarget{
			Targets: []string{t.Host},
			Labels:  labels,
		})
	}

	data, _ := json.MarshalIndent(fileSD, "", "  ")
	return string(data)
}

// generateBlackboxTargets 生成 Blackbox File SD 目标
func (p *InfraMonitorPlugin) generateBlackboxTargets(targets []types.BlackboxTarget, globalLabels map[string]string) (string, string) {
	httpTargets := make([]types.FileSDTarget, 0)
	icmpTargets := make([]types.FileSDTarget, 0)

	for _, t := range targets {
		labels := make(map[string]string)
		// 添加全局标签
		for k, v := range globalLabels {
			labels[k] = v
		}
		// 添加设备标签
		labels["instance"] = t.Name
		if t.Priority != "" {
			labels["priority"] = t.Priority
		}
		for k, v := range t.Labels {
			labels[k] = v
		}

		target := types.FileSDTarget{
			Targets: []string{t.URL},
			Labels:  labels,
		}

		if t.Type == "http" || t.Type == "https" {
			httpTargets = append(httpTargets, target)
		} else if t.Type == "icmp" {
			icmpTargets = append(icmpTargets, target)
		}
	}

	httpData, _ := json.MarshalIndent(httpTargets, "", "  ")
	icmpData, _ := json.MarshalIndent(icmpTargets, "", "  ")
	return string(httpData), string(icmpData)
}

// generateTelegrafVMwareConfig 生成 Telegraf VMware 配置
func (p *InfraMonitorPlugin) generateTelegrafVMwareConfig(vcenters []types.VMwareVCenterConfig, globalLabels map[string]string) string {
	var sb strings.Builder

	sb.WriteString("# Telegraf VMware vSphere 监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("[agent]\n")
	sb.WriteString("  interval = \"60s\"\n")
	sb.WriteString("  round_interval = true\n\n")

	sb.WriteString("[[outputs.http]]\n")
	sb.WriteString("  url = \"http://victoriametrics:8428/api/v1/write\"\n")
	sb.WriteString("  data_format = \"prometheusremotewrite\"\n\n")

	for _, vc := range vcenters {
		sb.WriteString("[[inputs.vsphere]]\n")
		sb.WriteString(fmt.Sprintf("  vcenters = [\"%s\"]\n", vc.URL))
		sb.WriteString(fmt.Sprintf("  username = \"%s\"\n", vc.Username))
		sb.WriteString(fmt.Sprintf("  password = \"%s\"\n", vc.Password))
		sb.WriteString(fmt.Sprintf("  insecure_skip_verify = %v\n", vc.Insecure))
		sb.WriteString("  interval = \"60s\"\n\n")

		sb.WriteString("  vm_metric_include = [\n")
		sb.WriteString("    \"cpu.usage.average\",\n")
		sb.WriteString("    \"mem.usage.average\",\n")
		sb.WriteString("    \"disk.usage.average\",\n")
		sb.WriteString("    \"net.usage.average\",\n")
		sb.WriteString("  ]\n\n")

		sb.WriteString("  host_metric_include = [\n")
		sb.WriteString("    \"cpu.usage.average\",\n")
		sb.WriteString("    \"mem.usage.average\",\n")
		sb.WriteString("  ]\n\n")

		sb.WriteString("  [inputs.vsphere.tags]\n")
		for k, v := range globalLabels {
			sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
		}
		if vc.Name != "" {
			sb.WriteString(fmt.Sprintf("    vcenter = \"%s\"\n", vc.Name))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateTelegrafRedfishConfig 生成 Telegraf Redfish 配置
func (p *InfraMonitorPlugin) generateTelegrafRedfishConfig(devices []types.RedfishDeviceConfig, globalLabels map[string]string) string {
	var sb strings.Builder

	sb.WriteString("# Telegraf Redfish 硬件监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("[agent]\n")
	sb.WriteString("  interval = \"60s\"\n\n")

	sb.WriteString("[[outputs.http]]\n")
	sb.WriteString("  url = \"http://victoriametrics:8428/api/v1/write\"\n")
	sb.WriteString("  data_format = \"prometheusremotewrite\"\n\n")

	for _, d := range devices {
		sb.WriteString("[[inputs.redfish]]\n")
		sb.WriteString(fmt.Sprintf("  name = \"%s\"\n", d.Name))
		sb.WriteString(fmt.Sprintf("  address = \"https://%s\"\n", d.Host))
		sb.WriteString(fmt.Sprintf("  username = \"%s\"\n", d.Username))
		sb.WriteString(fmt.Sprintf("  password = \"%s\"\n", d.Password))
		sb.WriteString(fmt.Sprintf("  insecure_skip_verify = %v\n", d.Insecure))
		sb.WriteString("  include_metrics = [\"thermal\", \"power\", \"system\", \"storage\"]\n\n")

		sb.WriteString("  [inputs.redfish.tags]\n")
		for k, v := range globalLabels {
			sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
		}
		sb.WriteString(fmt.Sprintf("    instance = \"%s\"\n", d.Name))
		if d.Vendor != "" {
			sb.WriteString(fmt.Sprintf("    vendor = \"%s\"\n", d.Vendor))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateTelegrafIPMIConfig 生成 Telegraf IPMI 配置
func (p *InfraMonitorPlugin) generateTelegrafIPMIConfig(devices []types.IPMIDeviceConfig, globalLabels map[string]string) string {
	var sb strings.Builder

	sb.WriteString("# Telegraf IPMI 硬件监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("[agent]\n")
	sb.WriteString("  interval = \"60s\"\n\n")

	sb.WriteString("[[outputs.http]]\n")
	sb.WriteString("  url = \"http://victoriametrics:8428/api/v1/write\"\n")
	sb.WriteString("  data_format = \"prometheusremotewrite\"\n\n")

	sb.WriteString("[[inputs.ipmi_sensor]]\n")
	sb.WriteString("  metric_version = 2\n")
	sb.WriteString("  timeout = \"10s\"\n\n")

	for _, d := range devices {
		sb.WriteString("  [[inputs.ipmi_sensor.server]]\n")
		sb.WriteString(fmt.Sprintf("    host = \"%s\"\n", d.Host))
		if d.Username != "" {
			sb.WriteString(fmt.Sprintf("    username = \"%s\"\n", d.Username))
		}
		if d.Password != "" {
			sb.WriteString(fmt.Sprintf("    password = \"%s\"\n", d.Password))
		}
		iface := d.Interface
		if iface == "" {
			iface = "lanplus"
		}
		sb.WriteString(fmt.Sprintf("    interface = \"%s\"\n", iface))
		sb.WriteString(fmt.Sprintf("    port = %d\n", d.Port))
		if d.Port == 0 {
			sb.WriteString("    port = 623\n")
		}

		sb.WriteString("    [inputs.ipmi_sensor.server.tags]\n")
		for k, v := range globalLabels {
			sb.WriteString(fmt.Sprintf("      %s = \"%s\"\n", k, v))
		}
		sb.WriteString(fmt.Sprintf("      instance = \"%s\"\n", d.Name))
		if d.Vendor != "" {
			sb.WriteString(fmt.Sprintf("      vendor = \"%s\"\n", d.Vendor))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateTopologyConfig 生成拓扑发现配置
func (p *InfraMonitorPlugin) generateTopologyConfig(devices []types.TopologyDeviceConfig) string {
	var sb strings.Builder

	sb.WriteString("# 网络设备配置文件\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("devices:\n")

	for _, d := range devices {
		sb.WriteString(fmt.Sprintf("  - name: %s\n", d.Name))
		sb.WriteString(fmt.Sprintf("    host: %s\n", d.Host))
		if d.Type != "" {
			sb.WriteString(fmt.Sprintf("    type: %s\n", d.Type))
		}
		if d.Tier != "" {
			sb.WriteString(fmt.Sprintf("    tier: %s\n", d.Tier))
		}
		if d.Vendor != "" {
			sb.WriteString(fmt.Sprintf("    vendor: %s\n", d.Vendor))
		}
		if d.Location != "" {
			sb.WriteString(fmt.Sprintf("    location: %s\n", d.Location))
		}
		if d.Community != "" {
			sb.WriteString(fmt.Sprintf("    snmp_community: %s\n", d.Community))
		} else {
			sb.WriteString("    snmp_community: public\n")
		}
		if d.Port > 0 {
			sb.WriteString(fmt.Sprintf("    snmp_port: %d\n", d.Port))
		}
		if d.Protocol != "" {
			sb.WriteString(fmt.Sprintf("    protocol: %s\n", d.Protocol))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateBlackboxExporterConfig 生成 Blackbox Exporter 配置
func (p *InfraMonitorPlugin) generateBlackboxExporterConfig() string {
	return `# Blackbox Exporter 配置
# 由 mibcraft 自动生成

modules:
  http_2xx:
    prober: http
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: [200, 301, 302]
      method: GET
      follow_redirects: true
      fail_if_ssl: false
      fail_if_not_ssl: false
      tls_config:
        insecure_skip_verify: true
      preferred_ip_protocol: "ip4"

  http_2xx_check_ssl:
    prober: http
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: [200, 301, 302]
      method: GET
      fail_if_ssl: false
      fail_if_not_ssl: true
      preferred_ip_protocol: "ip4"

  icmp:
    prober: icmp
    icmp:
      preferred_ip_protocol: "ip4"
      payload_size: 56

  tcp_connect:
    prober: tcp
    tcp:
      preferred_ip_protocol: "ip4"

  dns_udp:
    prober: dns
    dns:
      transport_protocol: "udp"
      preferred_ip_protocol: "ip4"
      query_name: "."
      query_type: "NS"
`
}

// generateRedfishExporterConfig 生成 Redfish Exporter 配置
func (p *InfraMonitorPlugin) generateRedfishExporterConfig(devices []types.RedfishDeviceConfig) string {
	var sb strings.Builder

	sb.WriteString("# Redfish Exporter 配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("hosts:\n")

	for _, d := range devices {
		sb.WriteString(fmt.Sprintf("  %s:\n", d.Name))
		sb.WriteString(fmt.Sprintf("    username: \"%s\"\n", d.Username))
		sb.WriteString(fmt.Sprintf("    password: \"%s\"\n", d.Password))
		sb.WriteString(fmt.Sprintf("    host_address: \"%s\"\n", d.Host))
		sb.WriteString("\n")
	}

	return sb.String()
}

// generateAlertmanagerConfig 生成 Alertmanager 配置
func (p *InfraMonitorPlugin) generateAlertmanagerConfig() string {
	return `# Alertmanager 配置
# 由 mibcraft 自动生成

global:
  smtp_smarthost: '${SMTP_HOST}:${SMTP_PORT}'
  smtp_from: '${SMTP_FROM}'
  smtp_auth_username: '${SMTP_USER}'
  smtp_auth_password: '${SMTP_PASSWORD}'
  smtp_require_tls: true

route:
  receiver: 'email-ops'
  group_by: ['alertname', 'severity', 'device_tier']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h

receivers:
  - name: 'email-ops'
    email_configs:
      - to: '${ALERT_EMAIL}'
        headers:
          Subject: '[{{ .Status }}] {{ .GroupLabels.alertname }}'

inhibit_rules:
  # 主机宕机 -> 抑制该主机的所有其他告警
  - source_match:
      alertname: 'HostDown'
    target_match_re:
      instance: '.*'
    equal: ['instance']

  # 核心交换机故障 -> 抑制下游接入交换机告警
  - source_match:
      device_tier: 'core'
      alertname: 'SwitchDown'
    target_match:
      device_tier: 'access'
    equal: ['datacenter']
`
}

// generateEnvFile 生成 .env 文件
func (p *InfraMonitorPlugin) generateEnvFile(devices *InfraDevicesConfig) string {
	var sb strings.Builder

	sb.WriteString("# 环境变量配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")

	sb.WriteString("# Grafana 配置\n")
	sb.WriteString("GRAFANA_ADMIN_USER=admin\n")
	sb.WriteString("GRAFANA_ADMIN_PASSWORD=admin\n\n")

	sb.WriteString("# SMTP 配置（告警邮件）\n")
	sb.WriteString("SMTP_HOST=smtp.example.com\n")
	sb.WriteString("SMTP_PORT=587\n")
	sb.WriteString("SMTP_USER=monitoring@example.com\n")
	sb.WriteString("SMTP_PASSWORD=your-password\n")
	sb.WriteString("SMTP_FROM=monitoring@example.com\n")
	sb.WriteString("ALERT_EMAIL=ops-team@example.com\n\n")

	sb.WriteString("# 环境信息\n")
	sb.WriteString(fmt.Sprintf("ENVIRONMENT=%s\n", devices.Environment))
	sb.WriteString(fmt.Sprintf("DATACENTER=%s\n", devices.Datacenter))
	sb.WriteString(fmt.Sprintf("CLUSTER=%s\n", devices.Cluster))

	return sb.String()
}

func (p *InfraMonitorPlugin) Close() error {
	return nil
}
