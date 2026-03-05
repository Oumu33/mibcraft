package generator

import (
	"fmt"
	"regexp"
	"strings"

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
