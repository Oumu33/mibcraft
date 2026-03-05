package generator

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Oumu33/mibcraft/types"
	"gopkg.in/yaml.v3"
)

// Generator 配置生成器
type Generator struct {
	config *GeneratorConfig
}

// GeneratorConfig 生成器配置
type GeneratorConfig struct {
	DefaultCommunity string
	DefaultVersion   int
	DefaultInterval  string
}

// NewGenerator 创建新的配置生成器
func NewGenerator(cfg *GeneratorConfig) *Generator {
	return &Generator{config: cfg}
}

// GenerateCategrafConfig 生成 Categraf TOML 配置
func (g *Generator) GenerateCategrafConfig(objects []*types.MIBObject, req *types.ConfigRequest) (string, error) {
	config := &types.CategrafConfig{
		Interval:  g.config.DefaultInterval,
		Targets:   []string{"127.0.0.1:161"},
		Community: g.config.DefaultCommunity,
		Version:   g.config.DefaultVersion,
		Timeout:   "5s",
		Retries:   3,
		Collect:   make([]types.CategrafCollectConfig, 0),
		Labels:    req.Labels,
	}
	
	// 为每个对象生成采集配置
	for _, obj := range objects {
		// 跳过不可读的对象
		if obj.Access != "read-only" && obj.Access != "read-write" {
			continue
		}
		
		metricName := obj.Name
		if name, ok := req.MetricNames[obj.OID]; ok {
			metricName = name
		}
		
		collect := types.CategrafCollectConfig{
			MetricName: g.sanitizeMetricName(metricName),
			OID:        obj.OID,
			Type:       g.mapTypeToCategraf(obj.Type),
			Labels:     obj.Labels,
		}
		
		config.Collect = append(config.Collect, collect)
	}
	
	// 序列化为 TOML
	return g.marshalCategrafTOML(config)
}

// marshalCategrafTOML 序列化为 TOML 格式
func (g *Generator) marshalCategrafTOML(config *types.CategrafConfig) (string, error) {
	var sb strings.Builder
	
	sb.WriteString("# Categraf SNMP 采集配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")
	
	sb.WriteString(fmt.Sprintf("interval = \"%s\"\n", config.Interval))
	sb.WriteString(fmt.Sprintf("targets = [\"%s\"]\n", strings.Join(config.Targets, "\", \"")))
	sb.WriteString(fmt.Sprintf("community = \"%s\"\n", config.Community))
	sb.WriteString(fmt.Sprintf("version = %d\n", config.Version))
	
	if config.Timeout != "" {
		sb.WriteString(fmt.Sprintf("timeout = \"%s\"\n", config.Timeout))
	}
	sb.WriteString(fmt.Sprintf("retries = %d\n", config.Retries))
	
	// 写入标签
	if len(config.Labels) > 0 {
		sb.WriteString("\n[labels]\n")
		for k, v := range config.Labels {
			sb.WriteString(fmt.Sprintf("  %s = \"%s\"\n", k, v))
		}
	}
	
	// 写入采集配置
	sb.WriteString("\n[[collect]]\n")
	for i, collect := range config.Collect {
		if i > 0 {
			sb.WriteString("\n[[collect]]\n")
		}
		sb.WriteString(fmt.Sprintf("  metric_name = \"%s\"\n", collect.MetricName))
		sb.WriteString(fmt.Sprintf("  oid = \"%s\"\n", collect.OID))
		sb.WriteString(fmt.Sprintf("  type = \"%s\"\n", collect.Type))
		
		if len(collect.Labels) > 0 {
			sb.WriteString("  [collect.labels]\n")
			for k, v := range collect.Labels {
				sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
			}
		}
	}
	
	return sb.String(), nil
}

// GenerateSNMPExporterConfig 生成 SNMP Exporter YAML 配置
func (g *Generator) GenerateSNMPExporterConfig(objects []*types.MIBObject, req *types.ConfigRequest) (string, error) {
	moduleName := "default"
	if len(objects) > 0 && objects[0].MIB != "" {
		moduleName = objects[0].MIB
	}
	
	config := &types.SNMPExporterConfig{
		Module:  moduleName,
		Metrics: make([]types.SNMPExporterMetric, 0),
	}
	
	// 为每个对象生成指标配置
	for _, obj := range objects {
		// 跳过不可读的对象
		if obj.Access != "read-only" && obj.Access != "read-write" {
			continue
		}
		
		metricName := obj.Name
		if name, ok := req.MetricNames[obj.OID]; ok {
			metricName = name
		}
		
		metric := types.SNMPExporterMetric{
			Name:   g.sanitizeMetricName(metricName),
			OID:    obj.OID,
			Type:   g.mapTypeToSNMPExporter(obj.Type),
			Help:   obj.Description,
			Labels: make([]types.SNMPExporterLabel, 0),
		}
		
		// 如果有枚举值，添加到配置
		if len(obj.EnumValues) > 0 {
			metric.EnumValues = obj.EnumValues
		}
		
		config.Metrics = append(config.Metrics, metric)
	}
	
	// 序列化为 YAML
	return g.marshalSNMPExporterYAML(config)
}

// marshalSNMPExporterYAML 序列化为 YAML 格式
func (g *Generator) marshalSNMPExporterYAML(config *types.SNMPExporterConfig) (string, error) {
	var sb strings.Builder
	
	sb.WriteString("# SNMP Exporter 配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")
	
	sb.WriteString(fmt.Sprintf("module: %s\n\n", config.Module))
	sb.WriteString("metrics:\n")
	
	for _, metric := range config.Metrics {
		sb.WriteString(fmt.Sprintf("  - name: %s\n", metric.Name))
		sb.WriteString(fmt.Sprintf("    oid: %s\n", metric.OID))
		sb.WriteString(fmt.Sprintf("    type: %s\n", metric.Type))
		
		// 处理帮助文本（转义引号）
		help := strings.ReplaceAll(metric.Help, "\"", "\\\"")
		help = strings.ReplaceAll(help, "\n", " ")
		if len(help) > 200 {
			help = help[:200] + "..."
		}
		sb.WriteString(fmt.Sprintf("    help: \"%s\"\n", help))
		
		if len(metric.EnumValues) > 0 {
			sb.WriteString("    enum_values:\n")
			for name, value := range metric.EnumValues {
				sb.WriteString(fmt.Sprintf("      %s: %d\n", name, value))
			}
		}
		
		if len(metric.Labels) > 0 {
			sb.WriteString("    labels:\n")
			for _, label := range metric.Labels {
				sb.WriteString(fmt.Sprintf("      - name: %s\n", label.Name))
				sb.WriteString(fmt.Sprintf("        oid: %s\n", label.OID))
			}
		}
	}
	
	return sb.String(), nil
}

// GenerateTelegrafConfig 生成 Telegraf inputs.snmp TOML 配置
func (g *Generator) GenerateTelegrafConfig(objects []*types.MIBObject, req *types.ConfigRequest) (string, error) {
	var sb strings.Builder
	
	sb.WriteString("# Telegraf SNMP 输入插件配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n\n")
	sb.WriteString("[[inputs.snmp]]\n")
	
	// 基础配置
	sb.WriteString(fmt.Sprintf("  agents = [\"udp://127.0.0.1:161\"]\n"))
	sb.WriteString(fmt.Sprintf("  community = \"%s\"\n", g.config.DefaultCommunity))
	sb.WriteString(fmt.Sprintf("  version = %d\n", g.config.DefaultVersion))
	sb.WriteString(fmt.Sprintf("  timeout = \"5s\"\n"))
	sb.WriteString(fmt.Sprintf("  retries = 3\n"))
	sb.WriteString(fmt.Sprintf("  max_repetitions = 50\n"))
	
	// 全局标签
	if len(req.Labels) > 0 {
		sb.WriteString("  [inputs.snmp.tags]\n")
		for k, v := range req.Labels {
			sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
		}
	}
	
	// 为每个对象生成字段配置
	for _, obj := range objects {
		if obj.Access != "read-only" && obj.Access != "read-write" {
			continue
		}
		
		metricName := obj.Name
		if name, ok := req.MetricNames[obj.OID]; ok {
			metricName = name
		}
		
		sb.WriteString("\n  [[inputs.snmp.field]]\n")
		sb.WriteString(fmt.Sprintf("    name = \"%s\"\n", g.sanitizeMetricName(metricName)))
		sb.WriteString(fmt.Sprintf("    oid = \"%s\"\n", obj.OID))
		
		// 转换类型
		conversion := g.mapTypeToTelegrafConversion(obj.Type)
		if conversion != "" {
			sb.WriteString(fmt.Sprintf("    conversion = \"%s\"\n", conversion))
		}
	}
	
	return sb.String(), nil
}

// mapTypeToTelegrafConversion 映射类型到 Telegraf 转换类型
func (g *Generator) mapTypeToTelegrafConversion(mibType string) string {
	switch mibType {
	case "counter":
		return "counter"
	case "timeticks":
		return "duration"
	case "ipaddress":
		return "ipaddr"
	default:
		return ""
	}
}

// GenerateBoth 同时生成多种配置
func (g *Generator) GenerateBoth(objects []*types.MIBObject, req *types.ConfigRequest) (*types.ConfigResult, error) {
	result := &types.ConfigResult{
		MIBObjectsUsed: make([]string, 0),
		Warnings:       make([]string, 0),
	}
	
	// 记录使用的对象
	for _, obj := range objects {
		result.MIBObjectsUsed = append(result.MIBObjectsUsed, fmt.Sprintf("%s (%s)", obj.Name, obj.OID))
	}
	
	// 生成 Categraf 配置
	if req.Format == "categraf" || req.Format == "both" {
		categraf, err := g.GenerateCategrafConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 Categraf 配置失败: %w", err)
		}
		result.CategrafConfig = categraf
	}
	
	// 生成 SNMP Exporter 配置
	if req.Format == "snmp_exporter" || req.Format == "both" {
		snmpExporter, err := g.GenerateSNMPExporterConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 SNMP Exporter 配置失败: %w", err)
		}
		result.SNMPExporterConfig = snmpExporter
	}
	
	// 生成 Telegraf 配置
	if req.Format == "telegraf" || req.Format == "both" {
		telegraf, err := g.GenerateTelegrafConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 Telegraf 配置失败: %w", err)
		}
		result.TelegrafConfig = telegraf
	}
	
	// 如果是 all，生成全部三种
	if req.Format == "all" {
		categraf, err := g.GenerateCategrafConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 Categraf 配置失败: %w", err)
		}
		result.CategrafConfig = categraf
		
		snmpExporter, err := g.GenerateSNMPExporterConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 SNMP Exporter 配置失败: %w", err)
		}
		result.SNMPExporterConfig = snmpExporter
		
		telegraf, err := g.GenerateTelegrafConfig(objects, req)
		if err != nil {
			return nil, fmt.Errorf("生成 Telegraf 配置失败: %w", err)
		}
		result.TelegrafConfig = telegraf
	}
	
	return result, nil
}

// sanitizeMetricName 清理指标名称
func (g *Generator) sanitizeMetricName(name string) string {
	// 转换为小写
	name = strings.ToLower(name)
	// 替换非法字符
	name = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(name, "_")
	// 去除前导和尾随下划线
	name = strings.Trim(name, "_")
	// 合并连续下划线
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
	return name
}

// mapTypeToCategraf 映射类型到 Categraf 类型
func (g *Generator) mapTypeToCategraf(mibType string) string {
	switch mibType {
	case "integer":
		return "gauge"
	case "counter":
		return "counter"
	case "gauge":
		return "gauge"
	case "timeticks":
		return "gauge" // 通常转换为秒
	case "string":
		return "string"
	case "ipaddress":
		return "string"
	case "bits":
		return "gauge"
	case "boolean":
		return "gauge"
	default:
		return "gauge"
	}
}

// mapTypeToSNMPExporter 映射类型到 SNMP Exporter 类型
func (g *Generator) mapTypeToSNMPExporter(mibType string) string {
	switch mibType {
	case "integer":
		return "gauge"
	case "counter":
		return "counter"
	case "gauge":
		return "gauge"
	case "timeticks":
		return "gauge"
	case "string":
		return "DisplayString"
	case "ipaddress":
		return "IpAddress"
	case "bits":
		return "Bits"
	case "boolean":
		return "gauge"
	default:
		return "gauge"
	}
}

// GenerateFromTemplate 从模板生成配置
func (g *Generator) GenerateFromTemplate(template string, objects []*types.MIBObject) (string, error) {
	// 简单的模板替换
	result := template
	
	for i, obj := range objects {
		placeholder := fmt.Sprintf("{{.OID%d}}", i)
		result = strings.ReplaceAll(result, placeholder, obj.OID)
		
		placeholder = fmt.Sprintf("{{.Name%d}}", i)
		result = strings.ReplaceAll(result, placeholder, obj.Name)
		
		placeholder = fmt.Sprintf("{{.Type%d}}", i)
		result = strings.ReplaceAll(result, placeholder, g.mapTypeToCategraf(obj.Type))
	}
	
	return result, nil
}

// ValidateConfig 验证配置
func (g *Generator) ValidateConfig(config string, format string) error {
	switch format {
	case "categraf":
		var cfg types.CategrafConfig
		_, err := toml.Decode(config, &cfg)
		return err
	case "snmp_exporter":
		var cfg types.SNMPExporterConfig
		return yaml.Unmarshal([]byte(config), &cfg)
	default:
		return fmt.Errorf("不支持的配置格式: %s", format)
	}
}

// ==================== 硬件监控配置生成方法 ====================

// GenerateIPMIConfig 生成Telegraf IPMI监控配置
func (g *Generator) GenerateIPMIConfig(devices []types.IPMIDeviceConfig, globalLabels map[string]string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Telegraf IPMI 监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n")
	sb.WriteString("# 用于监控物理服务器：Dell iDRAC, HPE iLO, Supermicro IPMI 等\n\n")

	sb.WriteString("[[inputs.ipmi_sensor]]\n")
	sb.WriteString("  ## 采集间隔\n")
	sb.WriteString(fmt.Sprintf("  interval = \"%s\"\n", g.config.DefaultInterval))
	sb.WriteString("  ## 超时设置\n")
	sb.WriteString("  timeout = \"5s\"\n")
	sb.WriteString("  ## 使用新版指标格式\n")
	sb.WriteString("  metric_version = 2\n\n")

	// 添加服务器配置
	for _, device := range devices {
		sb.WriteString("  [[inputs.ipmi_sensor.server]]\n")
		sb.WriteString(fmt.Sprintf("    ## 服务器名称: %s\n", device.Name))
		if device.Vendor != "" {
			sb.WriteString(fmt.Sprintf("    ## 厂商: %s\n", device.Vendor))
		}
		sb.WriteString(fmt.Sprintf("    host = \"%s\"\n", device.Host))
		
		if device.Username != "" {
			sb.WriteString(fmt.Sprintf("    username = \"%s\"\n", device.Username))
		}
		if device.Password != "" {
			sb.WriteString(fmt.Sprintf("    password = \"%s\"\n", device.Password))
		}
		
		// 接口类型
		iface := device.Interface
		if iface == "" {
			iface = "lanplus" // 默认使用lanplus
		}
		sb.WriteString(fmt.Sprintf("    interface = \"%s\"\n", iface))
		
		// 端口
		port := device.Port
		if port == 0 {
			port = 623
		}
		sb.WriteString(fmt.Sprintf("    port = %d\n", port))
		
		// 标签
		labels := make(map[string]string)
		for k, v := range globalLabels {
			labels[k] = v
		}
		for k, v := range device.Labels {
			labels[k] = v
		}
		labels["device_name"] = device.Name
		if device.Vendor != "" {
			labels["vendor"] = device.Vendor
		}
		labels["monitor_type"] = "ipmi"
		
		if len(labels) > 0 {
			sb.WriteString("    [inputs.ipmi_sensor.server.tags]\n")
			for k, v := range labels {
				sb.WriteString(fmt.Sprintf("      %s = \"%s\"\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GenerateRedfishConfig 生成Telegraf Redfish监控配置
func (g *Generator) GenerateRedfishConfig(devices []types.RedfishDeviceConfig, globalLabels map[string]string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Telegraf Redfish 监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n")
	sb.WriteString("# 用于监控现代服务器：Dell iDRAC9+, HPE iLO5+, Lenovo XClarity 等\n\n")

	sb.WriteString("[[inputs.redfish]]\n")
	sb.WriteString("  ## 采集间隔\n")
	sb.WriteString(fmt.Sprintf("  interval = \"%s\"\n", g.config.DefaultInterval))
	sb.WriteString("  ## 超时设置\n")
	sb.WriteString("  timeout = \"10s\"\n\n")

	// 添加服务器配置
	for _, device := range devices {
		sb.WriteString("  [[inputs.redfish.server]]\n")
		sb.WriteString(fmt.Sprintf("    ## 服务器名称: %s\n", device.Name))
		if device.Vendor != "" {
			sb.WriteString(fmt.Sprintf("    ## 厂商: %s\n", device.Vendor))
		}
		sb.WriteString(fmt.Sprintf("    name = \"%s\"\n", device.Name))
		
		// 构建URL
		port := device.Port
		if port == 0 {
			port = 443
		}
		scheme := "https"
		url := fmt.Sprintf("%s://%s:%d", scheme, device.Host, port)
		sb.WriteString(fmt.Sprintf("    address = \"%s\"\n", url))
		
		sb.WriteString(fmt.Sprintf("    username = \"%s\"\n", device.Username))
		sb.WriteString(fmt.Sprintf("    password = \"%s\"\n", device.Password))
		
		// 是否跳过SSL验证
		sb.WriteString(fmt.Sprintf("    insecure_skip_verify = %v\n", device.Insecure))
		
		// 包含的指标类型
		sb.WriteString("    include_metrics = [\"thermal\", \"power\", \"system\", \"storage\", \"memory\", \"network\"]\n")
		
		// 标签
		labels := make(map[string]string)
		for k, v := range globalLabels {
			labels[k] = v
		}
		for k, v := range device.Labels {
			labels[k] = v
		}
		labels["device_name"] = device.Name
		if device.Vendor != "" {
			labels["vendor"] = device.Vendor
		}
		labels["monitor_type"] = "redfish"
		
		if len(labels) > 0 {
			sb.WriteString("    [inputs.redfish.server.tags]\n")
			for k, v := range labels {
				sb.WriteString(fmt.Sprintf("      %s = \"%s\"\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GenerateVMwareConfig 生成Telegraf VMware vSphere监控配置
func (g *Generator) GenerateVMwareConfig(vcenters []types.VMwareVCenterConfig, globalLabels map[string]string) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Telegraf VMware vSphere 监控配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n")
	sb.WriteString("# 用于监控 ESXi 主机和虚拟机\n\n")

	sb.WriteString("[[inputs.vsphere]]\n")
	sb.WriteString("  ## 采集间隔\n")
	sb.WriteString(fmt.Sprintf("  interval = \"%s\"\n", g.config.DefaultInterval))
	sb.WriteString("  ## 超时设置\n")
	sb.WriteString("  timeout = \"60s\"\n\n")

	// 添加vCenter配置
	for _, vc := range vcenters {
		sb.WriteString("  ## vCenter 服务器\n")
		if vc.Name != "" {
			sb.WriteString(fmt.Sprintf("  ## 名称: %s\n", vc.Name))
		}
		
		sb.WriteString(fmt.Sprintf("  vcenters = [\"%s\"]\n", vc.URL))
		sb.WriteString(fmt.Sprintf("  username = \"%s\"\n", vc.Username))
		sb.WriteString(fmt.Sprintf("  password = \"%s\"\n", vc.Password))
		sb.WriteString(fmt.Sprintf("  insecure_skip_verify = %v\n\n", vc.Insecure))
		
		// VM指标配置
		sb.WriteString("  ## 虚拟机指标\n")
		if len(vc.VMInclude) > 0 {
			sb.WriteString("  vm_metric_include = [\n")
			for _, m := range vc.VMInclude {
				sb.WriteString(fmt.Sprintf("    \"%s\",\n", m))
			}
			sb.WriteString("  ]\n")
		} else {
			sb.WriteString("  vm_metric_include = [\n")
			sb.WriteString("    \"cpu.usage.average\",\n")
			sb.WriteString("    \"cpu.ready.average\",\n")
			sb.WriteString("    \"mem.usage.average\",\n")
			sb.WriteString("    \"mem.swapinRate.average\",\n")
			sb.WriteString("    \"mem.swapoutRate.average\",\n")
			sb.WriteString("    \"disk.usage.average\",\n")
			sb.WriteString("    \"disk.read.average\",\n")
			sb.WriteString("    \"disk.write.average\",\n")
			sb.WriteString("    \"net.usage.average\",\n")
			sb.WriteString("    \"net.bytesRx.average\",\n")
			sb.WriteString("    \"net.bytesTx.average\",\n")
			sb.WriteString("  ]\n")
		}
		
		// 主机指标配置
		sb.WriteString("\n  ## ESXi主机指标\n")
		if len(vc.HostInclude) > 0 {
			sb.WriteString("  host_metric_include = [\n")
			for _, m := range vc.HostInclude {
				sb.WriteString(fmt.Sprintf("    \"%s\",\n", m))
			}
			sb.WriteString("  ]\n")
		} else {
			sb.WriteString("  host_metric_include = [\n")
			sb.WriteString("    \"cpu.usage.average\",\n")
			sb.WriteString("    \"cpu.utilization.average\",\n")
			sb.WriteString("    \"mem.usage.average\",\n")
			sb.WriteString("    \"mem.utilization.average\",\n")
			sb.WriteString("    \"disk.usage.average\",\n")
			sb.WriteString("    \"disk.read.average\",\n")
			sb.WriteString("    \"disk.write.average\",\n")
			sb.WriteString("    \"net.usage.average\",\n")
			sb.WriteString("    \"net.bytesRx.average\",\n")
			sb.WriteString("    \"net.bytesTx.average\",\n")
			sb.WriteString("  ]\n")
		}
		
		// 集群指标配置
		sb.WriteString("\n  ## 集群指标\n")
		sb.WriteString("  cluster_metric_include = [\n")
		sb.WriteString("    \"cpu.usage.average\",\n")
		sb.WriteString("    \"mem.usage.average\",\n")
		sb.WriteString("  ]\n")
		
		// 数据存储指标配置
		sb.WriteString("\n  ## 数据存储指标\n")
		sb.WriteString("  datastore_metric_include = [\n")
		sb.WriteString("    \"disk.used.latest\",\n")
		sb.WriteString("    \"disk.capacity.latest\",\n")
		sb.WriteString("  ]\n")
		
		// 标签
		labels := make(map[string]string)
		for k, v := range globalLabels {
			labels[k] = v
		}
		for k, v := range vc.Labels {
			labels[k] = v
		}
		if vc.Name != "" {
			labels["vcenter_name"] = vc.Name
		}
		labels["monitor_type"] = "vmware"
		
		if len(labels) > 0 {
			sb.WriteString("\n  [inputs.vsphere.tags]\n")
			for k, v := range labels {
				sb.WriteString(fmt.Sprintf("    %s = \"%s\"\n", k, v))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// GenerateIPMIExporterConfig 生成Prometheus IPMI Exporter配置
func (g *Generator) GenerateIPMIExporterConfig(devices []types.IPMIDeviceConfig) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Prometheus IPMI Exporter 配置\n")
	sb.WriteString("# 由 mibcraft 自动生成\n")
	sb.WriteString("# 需要配合 ipmi_exporter 使用\n\n")

	sb.WriteString("modules:\n")

	for _, device := range devices {
		moduleName := g.sanitizeMetricName(device.Name)
		sb.WriteString(fmt.Sprintf("  %s:\n", moduleName))
		sb.WriteString(fmt.Sprintf("    # 服务器: %s\n", device.Name))
		
		if device.Username != "" {
			sb.WriteString(fmt.Sprintf("    user: \"%s\"\n", device.Username))
		}
		if device.Password != "" {
			sb.WriteString(fmt.Sprintf("    pass: \"%s\"\n", device.Password))
		}
		
		// 权限级别
		priv := "admin"
		if device.Vendor == "supermicro" {
			priv = "operator"
		}
		sb.WriteString(fmt.Sprintf("    priv: \"%s\"\n", priv))
		
		// 认证类型
		authType := "md5"
		sb.WriteString(fmt.Sprintf("    auth_type: \"%s\"\n", authType))
		
		// 超时
		sb.WriteString("    timeout: 30\n\n")
	}

	return sb.String(), nil
}

// GenerateHardwareMonitorConfig 统一生成硬件监控配置
func (g *Generator) GenerateHardwareMonitorConfig(req *types.HardwareMonitorRequest) (*types.HardwareMonitorResult, error) {
	result := &types.HardwareMonitorResult{
		GeneratedAt:       time.Now(),
		DevicesConfigured: make([]string, 0),
		Warnings:          make([]string, 0),
	}

	// 生成IPMI配置
	if len(req.IPMIDevices) > 0 {
		config, err := g.GenerateIPMIConfig(req.IPMIDevices, req.GlobalLabels)
		if err != nil {
			return nil, fmt.Errorf("生成IPMI配置失败: %w", err)
		}
		result.TelegrafIPMIConfig = config
		for _, d := range req.IPMIDevices {
			result.DevicesConfigured = append(result.DevicesConfigured, 
				fmt.Sprintf("IPMI: %s (%s)", d.Name, d.Host))
		}
	}

	// 生成Redfish配置
	if len(req.RedfishDevices) > 0 {
		config, err := g.GenerateRedfishConfig(req.RedfishDevices, req.GlobalLabels)
		if err != nil {
			return nil, fmt.Errorf("生成Redfish配置失败: %w", err)
		}
		result.TelegrafRedfishConfig = config
		for _, d := range req.RedfishDevices {
			result.DevicesConfigured = append(result.DevicesConfigured, 
				fmt.Sprintf("Redfish: %s (%s)", d.Name, d.Host))
		}
	}

	// 生成VMware配置
	if len(req.VMwareVCenters) > 0 {
		config, err := g.GenerateVMwareConfig(req.VMwareVCenters, req.GlobalLabels)
		if err != nil {
			return nil, fmt.Errorf("生成VMware配置失败: %w", err)
		}
		result.TelegrafVMwareConfig = config
		for _, vc := range req.VMwareVCenters {
			name := vc.Name
			if name == "" {
				name = vc.URL
			}
			result.DevicesConfigured = append(result.DevicesConfigured, 
				fmt.Sprintf("VMware: %s", name))
		}
	}

	// 如果指定了ipmi_exporter格式
	if req.Format == "ipmi_exporter" && len(req.IPMIDevices) > 0 {
		config, err := g.GenerateIPMIExporterConfig(req.IPMIDevices)
		if err != nil {
			return nil, fmt.Errorf("生成IPMI Exporter配置失败: %w", err)
		}
		result.IPMIExporterConfig = config
	}

	return result, nil
}
