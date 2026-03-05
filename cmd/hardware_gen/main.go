package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/types"
)

func separator(title string) {
	line := strings.Repeat("=", 60)
	fmt.Printf("\n%s\n%s\n%s\n", line, title, line)
}

func main() {
	gen := generator.NewGenerator(&generator.GeneratorConfig{
		DefaultInterval: "60s",
	})

	// 测试 IPMI 配置生成
	ipmiDevices := []types.IPMIDeviceConfig{
		{
			Name:      "dell-r740-01",
			Host:      "192.168.1.101",
			Username:  "admin",
			Password:  "calvin",
			Interface: "lanplus",
			Vendor:    "dell",
			Labels:    map[string]string{"rack": "A01"},
		},
		{
			Name:      "hpe-dl380-01",
			Host:      "192.168.1.102",
			Username:  "admin",
			Password:  "password",
			Interface: "lanplus",
			Vendor:    "hpe",
			Labels:    map[string]string{"rack": "A02"},
		},
	}

	globalLabels := map[string]string{
		"env":        "production",
		"datacenter": "dc1",
	}

	separator("IPMI 配置 (Telegraf inputs.ipmi_sensor)")
	ipmiConfig, err := gen.GenerateIPMIConfig(ipmiDevices, globalLabels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成IPMI配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(ipmiConfig)

	// 测试 Redfish 配置生成
	separator("Redfish 配置 (Telegraf inputs.redfish)")

	redfishDevices := []types.RedfishDeviceConfig{
		{
			Name:     "dell-r750-01",
			Host:     "192.168.1.111",
			Username: "root",
			Password: "calvin",
			Vendor:   "dell_idrac",
			Insecure: true,
			Labels:   map[string]string{"rack": "A03"},
		},
	}

	redfishConfig, err := gen.GenerateRedfishConfig(redfishDevices, globalLabels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成Redfish配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(redfishConfig)

	// 测试 VMware 配置生成
	separator("VMware 配置 (Telegraf inputs.vsphere)")

	vcenters := []types.VMwareVCenterConfig{
		{
			Name:     "vcenter-prod",
			URL:      "https://vcenter.example.com/sdk",
			Username: "monitoring@vsphere.local",
			Password: "YourPassword123!",
			Insecure: true,
			Labels:   map[string]string{"environment": "production"},
		},
	}

	vmwareConfig, err := gen.GenerateVMwareConfig(vcenters, globalLabels)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成VMware配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(vmwareConfig)

	// 测试 IPMI Exporter 配置生成
	separator("IPMI Exporter 配置")

	exporterConfig, err := gen.GenerateIPMIExporterConfig(ipmiDevices)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成IPMI Exporter配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(exporterConfig)

	// 测试统一生成
	separator("统一生成测试")

	req := &types.HardwareMonitorRequest{
		IPMIDevices:    ipmiDevices,
		RedfishDevices: redfishDevices,
		VMwareVCenters: vcenters,
		GlobalLabels:   globalLabels,
		Format:         "telegraf",
	}

	result, err := gen.GenerateHardwareMonitorConfig(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "统一生成配置失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("生成时间: %s\n", result.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("配置的设备: %v\n", result.DevicesConfigured)
	fmt.Println("配置生成成功!")
}