package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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
		configPath  string
		showVersion bool
		genMode     string
		mibFile     string
		oids        string
		outputDir   string
	)

	flag.StringVar(&configPath, "config", "", "配置文件路径")
	flag.BoolVar(&showVersion, "version", false, "显示版本信息")
	flag.StringVar(&genMode, "gen", "", "生成模式: categraf, snmp_exporter, both")
	flag.StringVar(&mibFile, "mib", "", "MIB 文件路径")
	flag.StringVar(&oids, "oids", "", "OID 列表 (逗号分隔)")
	flag.StringVar(&outputDir, "output", "", "输出目录")
	flag.Parse()

	if showVersion {
		fmt.Printf("mibcraft version %s (built %s)\n", version, buildDate)
		os.Exit(0)
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

	// 命令行生成模式
	if genMode != "" && mibFile != "" && oids != "" {
		if err := generateFromCLI(cfg, genMode, mibFile, oids, outputDir); err != nil {
			fmt.Fprintf(os.Stderr, "生成配置失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 交互式对话模式
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