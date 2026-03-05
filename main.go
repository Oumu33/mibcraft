package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Oumu33/mibcraft/agent"
	"github.com/Oumu33/mibcraft/agent/plugins"
	"github.com/Oumu33/mibcraft/chat"
	"github.com/Oumu33/mibcraft/config"
	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/mibparser"
	"github.com/Oumu33/mibcraft/types"
)

var (
	version   = "dev"
	buildDate = "unknown"
)

func main() {
	// 解析命令行参数
	var (
		configPath      string
		showVersion     bool
		runMode         string
		genMode         string
		mibFile         string
		oids            string
		outputDir       string
		infraMode       bool
		infraConfigPath string
	)

	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.StringVar(&runMode, "mode", "chat", "运行模式: agent, chat, cli, infra")
	flag.StringVar(&genMode, "gen", "", "生成模式: categraf, snmp_exporter, telegraf, all")
	flag.StringVar(&mibFile, "mib", "", "MIB 文件路径")
	flag.StringVar(&oids, "oids", "", "OID 列表 (逗号分隔)")
	flag.StringVar(&outputDir, "output", "", "输出目录")
	flag.BoolVar(&infraMode, "infra", false, "基础设施配置生成模式")
	flag.StringVar(&infraConfigPath, "infra-config", "", "基础设施配置文件路径")
	flag.Parse()

	if showVersion {
		fmt.Printf("mibcraft version %s (built %s)\n", version, buildDate)
		os.Exit(0)
	}

	// 基础设施配置生成模式
	if infraMode || runMode == "infra" {
		if err := generateInfraConfig(infraConfigPath, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "生成基础设施配置失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 加载配置
	if configPath == "" {
		configPath = config.GetConfigPath()
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// CLI 生成模式
	if runMode == "cli" || (genMode != "" && mibFile != "" && oids != "") {
		if err := generateFromCLI(cfg, genMode, mibFile, oids, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "生成配置失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Agent 模式
	if runMode == "agent" {
		if err := runAgentMode(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Agent 错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 默认交互式对话模式
	if err := runInteractiveMode(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// generateFromCLI 命令行生成模式
func generateFromCLI(cfg *config.Config, genMode, mibFile, oids, outputDir string) error {
	parser := mibparser.NewParser(cfg.Global.MIBDirs)

	// 加载 MIB 文件
	if _, err := parser.ParseFile(mibFile); err != nil {
		return fmt.Errorf("解析 MIB 文件失败: %w", err)
	}

	// 解析 OID 列表
	oidList := strings.Split(oids, ",")
	for i, oid := range oidList {
		oidList[i] = strings.TrimSpace(oid)
	}

	// 查找对象
	var objects []*types.MIBObject
	for _, oid := range oidList {
		objs, err := parser.FindObjectsByOID(oid)
		if err != nil {
			return fmt.Errorf("查找 OID %s 失败: %w", oid, err)
		}
		objects = append(objects, objs...)
	}

	// 生成配置
	gen := generator.NewGenerator(&generator.GeneratorConfig{
		DefaultCommunity: cfg.Generator.DefaultCommunity,
		DefaultVersion:   cfg.Generator.DefaultVersion,
		DefaultInterval:  cfg.Generator.DefaultInterval,
	})

	req := &types.ConfigRequest{
		TargetOIDs:  oidList,
		Format:      genMode,
		MetricNames: make(map[string]string),
		Labels:      cfg.Global.Labels,
	}

	result, err := gen.GenerateBoth(objects, req)
	if err != nil {
		return fmt.Errorf("生成配置失败: %w", err)
	}

	// 输出结果
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("创建输出目录失败: %w", err)
		}

		if result.CategrafConfig != "" {
			if err := os.WriteFile(outputDir+"/snmp.toml", []byte(result.CategrafConfig), 0644); err != nil {
				return fmt.Errorf("写入 Categraf 配置失败: %w", err)
			}
			fmt.Printf("Categraf 配置已写入: %s/snmp.toml\n", outputDir)
		}

		if result.SNMPExporterConfig != "" {
			if err := os.WriteFile(outputDir+"/snmp.yml", []byte(result.SNMPExporterConfig), 0644); err != nil {
				return fmt.Errorf("写入 SNMP Exporter 配置失败: %w", err)
			}
			fmt.Printf("SNMP Exporter 配置已写入: %s/snmp.yml\n", outputDir)
		}
	} else {
		if result.CategrafConfig != "" {
			fmt.Println("\n=== Categraf 配置 ===")
			fmt.Println(result.CategrafConfig)
		}

		if result.SNMPExporterConfig != "" {
			fmt.Println("\n=== SNMP Exporter 配置 ===")
			fmt.Println(result.SNMPExporterConfig)
		}
	}

	return nil
}

// generateInfraConfig 生成基础设施监控配置
func generateInfraConfig(configPath, outputDir string) error {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       基础设施监控配置生成器                                 ║")
	fmt.Println("║   生成完整的监控栈配置，支持多种采集器                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")

	// 默认配置文件路径
	if configPath == "" {
		configPath = "conf.d/infra_devices.toml"
	}

	// 默认输出目录
	if outputDir == "" {
		outputDir = "./output/infra"
	}

	// 创建插件实例
	infraPlugin := plugins.NewInfraMonitorPlugin()

	// 加载配置
	pluginConfig := &plugins.InfraMonitorPluginConfig{
		OutputDir:      outputDir,
		DevicesFile:    configPath,
		WatchChanges:   false,
		Environment:    "production",
		Datacenter:     "dc1",
		Cluster:        "monitoring-cluster",
		GlobalLabels:   make(map[string]string),
	}

	if err := infraPlugin.Init(pluginConfig); err != nil {
		return fmt.Errorf("初始化插件失败: %w", err)
	}

	// 加载设备配置
	devices, err := infraPlugin.LoadDevices()
	if err != nil {
		return fmt.Errorf("加载设备配置失败: %w", err)
	}

	// 生成并写入配置文件
	if err := infraPlugin.RegenerateConfig(devices); err != nil {
		return fmt.Errorf("生成配置失败: %w", err)
	}

	// 统计信息
	var targetCount int
	targetCount += len(devices.NodeExporters)
	targetCount += len(devices.SNMPDevices)
	targetCount += len(devices.BlackboxTargets)
	targetCount += len(devices.RedfishDevices)
	targetCount += len(devices.IPMIDevices)
	targetCount += len(devices.VMwareVCenters)
	targetCount += len(devices.ProxmoxHosts)
	targetCount += len(devices.TopologyDevices)

	fmt.Printf("\n✅ 配置生成成功！\n")
	fmt.Printf("📁 输出目录: %s\n", outputDir)
	fmt.Printf("📊 已配置目标: %d 个\n", targetCount)

	fmt.Printf("\n📋 生成的文件:\n")
	fmt.Printf("  - docker-compose.yaml    Docker Compose 部署配置\n")
	fmt.Printf("  - config/vmagent/prometheus.yml    vmagent 采集配置\n")
	fmt.Printf("  - config/vmagent/targets/*.json    File SD 目标文件\n")
	fmt.Printf("  - config/telegraf/*.conf           Telegraf 配置\n")
	fmt.Printf("  - config/blackbox-exporter/blackbox.yml    Blackbox 配置\n")
	fmt.Printf("  - config/alertmanager/alertmanager.yml     Alertmanager 配置\n")
	fmt.Printf("  - .env    环境变量配置\n")

	fmt.Printf("\n🚀 启动命令:\n")
	fmt.Printf("   cd %s && docker-compose up -d\n", outputDir)

	return nil
}

// runInteractiveMode 运行交互式模式
func runInteractiveMode(cfg *config.Config) error {
	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号...")
		cancel()
	}()

	// 启动对话
	c := chat.NewChat(cfg)
	defer c.Stop()

	return c.Start(ctx)
}

// runAgentMode 运行 Agent 模式
func runAgentMode(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建 Agent
	ag := agent.NewAgent(cfg)

	// 注册内置插件
	ag.RegisterPlugin(plugins.NewSNMPPlugin())
	ag.RegisterPlugin(plugins.NewMIBValidatorPlugin())
	ag.RegisterPlugin(plugins.NewOIDMonitorPlugin())
	ag.RegisterPlugin(plugins.NewConfigGenPlugin())
	ag.RegisterPlugin(plugins.NewHardwareMonitorPlugin())

	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║              MIB-Agent 监控模式                             ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n已加载插件: %v\n", ag.GetPlugins())
	fmt.Println("\n按 Ctrl+C 退出...")

	return ag.Start(ctx)
}