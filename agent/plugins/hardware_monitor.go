package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/Oumu33/mibcraft/agent"
	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/types"
)

// HardwareMonitorPlugin 硬件监控配置生成插件
// 支持 IPMI、Redfish、VMware 监控配置自动生成
type HardwareMonitorPlugin struct {
	config   *HardwareMonitorPluginConfig
	gen      *generator.Generator
	lastGen  time.Time
}

// HardwareMonitorPluginConfig 硬件监控插件配置
type HardwareMonitorPluginConfig struct {
	agent.BasePluginConfig
	// 输出目录
	OutputDir string `toml:"output_dir"`
	// 设备配置文件路径
	DevicesFile string `toml:"devices_file"`
	// 是否监听变化
	WatchChanges bool `toml:"watch_changes"`
	// 生成格式: telegraf, ipmi_exporter, all
	Format string `toml:"format"`
	// 全局标签
	GlobalLabels map[string]string `toml:"global_labels"`
	// 设备列表（可直接在配置中定义）
	IPMIDevices    []types.IPMIDeviceConfig   `toml:"ipmi_devices"`
	RedfishDevices []types.RedfishDeviceConfig `toml:"redfish_devices"`
	VMwareVCenters []types.VMwareVCenterConfig `toml:"vmware_vcenters"`
}

// DevicesConfig 设备配置文件格式
type DevicesConfig struct {
	IPMIDevices    []types.IPMIDeviceConfig   `toml:"ipmi_devices"`
	RedfishDevices []types.RedfishDeviceConfig `toml:"redfish_devices"`
	VMwareVCenters []types.VMwareVCenterConfig `toml:"vmware_vcenters"`
	GlobalLabels   map[string]string          `toml:"global_labels"`
}

// NewHardwareMonitorPlugin 创建硬件监控插件
func NewHardwareMonitorPlugin() *HardwareMonitorPlugin {
	return &HardwareMonitorPlugin{}
}

func (p *HardwareMonitorPlugin) Name() string {
	return "hardware_monitor"
}

func (p *HardwareMonitorPlugin) Description() string {
	return "生成 IPMI/Redfish/VMware 监控配置，支持物理服务器和虚拟化环境监控"
}

func (p *HardwareMonitorPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*HardwareMonitorPluginConfig); ok {
			p.config = cfg
		}
	}

	if p.config == nil {
		p.config = &HardwareMonitorPluginConfig{
			OutputDir:    "./output/hardware",
			WatchChanges: true,
			Format:       "telegraf",
			GlobalLabels: make(map[string]string),
		}
	}

	p.gen = generator.NewGenerator(&generator.GeneratorConfig{
		DefaultInterval: "60s",
	})

	// 确保输出目录存在
	os.MkdirAll(p.config.OutputDir, 0755)

	return nil
}

func (p *HardwareMonitorPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Labels:  p.config.GlobalLabels,
	}

	// 加载设备配置
	devices, err := p.loadDevices()
	if err != nil {
		result.OK = false
		result.Message = fmt.Sprintf("加载设备配置失败: %v", err)
		return result, nil
	}

	// 检查是否有变化
	if p.config.WatchChanges {
		if changed := p.checkChanges(); changed {
			// 重新生成配置
			if err := p.regenerateConfig(devices); err != nil {
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
	result.Metrics["ipmi_devices_count"] = len(devices.IPMIDevices)
	result.Metrics["redfish_devices_count"] = len(devices.RedfishDevices)
	result.Metrics["vmware_vcenters_count"] = len(devices.VMwareVCenters)
	result.Metrics["config_last_gen"] = p.lastGen.Unix()

	return result, nil
}

// loadDevices 加载设备配置
func (p *HardwareMonitorPlugin) loadDevices() (*DevicesConfig, error) {
	devices := &DevicesConfig{
		IPMIDevices:    p.config.IPMIDevices,
		RedfishDevices: p.config.RedfishDevices,
		VMwareVCenters: p.config.VMwareVCenters,
		GlobalLabels:   p.config.GlobalLabels,
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

		var fileDevices DevicesConfig
		if _, err := toml.Decode(string(data), &fileDevices); err != nil {
			return nil, err
		}

		// 合并配置
		if len(fileDevices.IPMIDevices) > 0 {
			devices.IPMIDevices = fileDevices.IPMIDevices
		}
		if len(fileDevices.RedfishDevices) > 0 {
			devices.RedfishDevices = fileDevices.RedfishDevices
		}
		if len(fileDevices.VMwareVCenters) > 0 {
			devices.VMwareVCenters = fileDevices.VMwareVCenters
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
func (p *HardwareMonitorPlugin) checkChanges() bool {
	// 检查设备配置文件
	if p.config.DevicesFile != "" {
		info, err := os.Stat(p.config.DevicesFile)
		if err == nil && info.ModTime().After(p.lastGen) {
			return true
		}
	}
	return false
}

// regenerateConfig 重新生成配置
func (p *HardwareMonitorPlugin) regenerateConfig(devices *DevicesConfig) error {
	req := &types.HardwareMonitorRequest{
		IPMIDevices:    devices.IPMIDevices,
		RedfishDevices: devices.RedfishDevices,
		VMwareVCenters: devices.VMwareVCenters,
		GlobalLabels:   devices.GlobalLabels,
		Format:         p.config.Format,
	}

	result, err := p.gen.GenerateHardwareMonitorConfig(req)
	if err != nil {
		return err
	}

	// 写入配置文件
	if result.TelegrafIPMIConfig != "" {
		path := filepath.Join(p.config.OutputDir, "telegraf_ipmi.conf")
		if err := os.WriteFile(path, []byte(result.TelegrafIPMIConfig), 0644); err != nil {
			return fmt.Errorf("写入IPMI配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", path)
	}

	if result.TelegrafRedfishConfig != "" {
		path := filepath.Join(p.config.OutputDir, "telegraf_redfish.conf")
		if err := os.WriteFile(path, []byte(result.TelegrafRedfishConfig), 0644); err != nil {
			return fmt.Errorf("写入Redfish配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", path)
	}

	if result.TelegrafVMwareConfig != "" {
		path := filepath.Join(p.config.OutputDir, "telegraf_vmware.conf")
		if err := os.WriteFile(path, []byte(result.TelegrafVMwareConfig), 0644); err != nil {
			return fmt.Errorf("写入VMware配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", path)
	}

	if result.IPMIExporterConfig != "" {
		path := filepath.Join(p.config.OutputDir, "ipmi_exporter.yml")
		if err := os.WriteFile(path, []byte(result.IPMIExporterConfig), 0644); err != nil {
			return fmt.Errorf("写入IPMI Exporter配置失败: %w", err)
		}
		fmt.Printf("已生成: %s\n", path)
	}

	p.lastGen = time.Now()
	return nil
}

// GenerateConfig 手动触发配置生成
func (p *HardwareMonitorPlugin) GenerateConfig() (*types.HardwareMonitorResult, error) {
	devices, err := p.loadDevices()
	if err != nil {
		return nil, err
	}

	req := &types.HardwareMonitorRequest{
		IPMIDevices:    devices.IPMIDevices,
		RedfishDevices: devices.RedfishDevices,
		VMwareVCenters: devices.VMwareVCenters,
		GlobalLabels:   devices.GlobalLabels,
		Format:         p.config.Format,
	}

	result, err := p.gen.GenerateHardwareMonitorConfig(req)
	if err != nil {
		return nil, err
	}

	p.lastGen = time.Now()
	return result, nil
}

func (p *HardwareMonitorPlugin) Close() error {
	return nil
}
