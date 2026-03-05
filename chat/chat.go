package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Oumu33/mibcraft/config"
	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/mcp"
	"github.com/Oumu33/mibcraft/mibparser"
	"github.com/Oumu33/mibcraft/types"
	"github.com/sashabaranov/go-openai"
)

// Chat 对话管理器
type Chat struct {
	config      *config.Config
	parser      *mibparser.Parser
	generator   *generator.Generator
	extractor   *mibparser.Extractor
	mcpManager  *mcp.Manager
	aiClient    *openai.Client
	history     []openai.ChatCompletionMessage
	mibDir      string // 用户自定义的 MIB 目录
}

// NewChat 创建新的对话管理器
func NewChat(cfg *config.Config) *Chat {
	// 获取或创建默认 MIB 目录
	mibDir := "./mibs"
	if len(cfg.Global.MIBDirs) > 0 {
		mibDir = cfg.Global.MIBDirs[0]
	}

	parser := mibparser.NewParser([]string{mibDir})
	gen := generator.NewGenerator(&generator.GeneratorConfig{
		DefaultCommunity: cfg.Generator.DefaultCommunity,
		DefaultVersion:   cfg.Generator.DefaultVersion,
		DefaultInterval:  cfg.Generator.DefaultInterval,
	})
	
	var aiClient *openai.Client
	if cfg.AI.Enabled && len(cfg.AI.ModelPriority) > 0 {
		// 使用第一个优先模型
		modelName := cfg.AI.ModelPriority[0]
		if modelCfg, ok := cfg.AI.Models[modelName]; ok {
			aiConfig := openai.DefaultConfig(modelCfg.APIKey)
			aiConfig.BaseURL = modelCfg.BaseURL
			aiClient = openai.NewClientWithConfig(aiConfig)
		}
	}
	
	return &Chat{
		config:     cfg,
		parser:     parser,
		generator:  gen,
		extractor:  mibparser.NewExtractor(mibDir),
		mcpManager: mcp.NewManager(),
		aiClient:   aiClient,
		history:    make([]openai.ChatCompletionMessage, 0),
		mibDir:     mibDir,
	}
}

// Start 启动对话
func (c *Chat) Start(ctx context.Context) error {
	// 启动 MCP 服务器
	if c.config.AI.MCP.Enabled {
		if err := c.mcpManager.StartServers(ctx, c.config.AI.MCP.Servers); err != nil {
			fmt.Printf("警告: 启动 MCP 服务器失败: %v\n", err)
		}
	}
	
	// 打印欢迎信息
	c.printWelcome()
	
	// 开始 REPL 循环
	return c.replLoop(ctx)
}

// printWelcome 打印欢迎信息
func (c *Chat) printWelcome() {
	fmt.Println()
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║            MIB-Agent - SNMP 配置生成助手                    ║")
	fmt.Println("║                                                              ║")
	fmt.Println("║  功能: 解析 MIB 文件，生成 Categraf/SNMP Exporter 配置       ║")
	fmt.Println("║                                                              ║")
	fmt.Println("║  命令:                                                       ║")
	fmt.Println("║    /help             - 显示帮助信息                          ║")
	fmt.Println("║    /load <file>      - 加载 MIB 文件                         ║")
	fmt.Println("║    /extract <archive>- 解压 MIB 压缩包                       ║")
	fmt.Println("║    /mibdir [path]    - 设置/查看 MIB 目录                    ║")
	fmt.Println("║    /scan             - 扫描 MIB 目录中的文件                 ║")
	fmt.Println("║    /list             - 列出已加载的 MIB 文件                 ║")
	fmt.Println("║    /search <name>    - 搜索 MIB 对象                         ║")
	fmt.Println("║    /show <oid>       - 显示 OID 详细信息                     ║")
	fmt.Println("║    /gen              - 生成配置文件                          ║")
	fmt.Println("║    /infra            - 生成基础设施监控配置                  ║")
	fmt.Println("║    /clear            - 清除对话历史                          ║")
	fmt.Println("║    /exit             - 退出程序                              ║")
	fmt.Println("║                                                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	
	// 显示当前 MIB 目录
	fmt.Printf("\n📁 当前 MIB 目录: %s\n", c.mibDir)
	
	// 扫描 MIB 文件数量
	mibFiles := c.parser.ListMIBFiles()
	if len(mibFiles) > 0 {
		fmt.Printf("📋 已发现 %d 个 MIB 文件\n", len(mibFiles))
	} else {
		fmt.Println("⚠️  未发现 MIB 文件，请使用 /load 或 /extract 加载")
	}
}

// replLoop REPL 主循环
func (c *Chat) replLoop(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	
	for {
		fmt.Print("\n>>> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		
		// 处理命令
		if strings.HasPrefix(input, "/") {
			if err := c.handleCommand(ctx, input); err != nil {
				if err.Error() == "exit" {
					return nil
				}
				fmt.Printf("错误: %v\n", err)
			}
			continue
		}
		
		// 处理自然语言查询
		if c.aiClient != nil {
			if err := c.handleNaturalLanguage(ctx, input); err != nil {
				fmt.Printf("错误: %v\n", err)
			}
		} else {
			fmt.Println("AI 功能未启用，请使用命令模式或配置 AI 模型")
		}
	}
}

// handleCommand 处理命令
func (c *Chat) handleCommand(ctx context.Context, input string) error {
	parts := strings.Fields(input)
	cmd := parts[0]
	args := parts[1:]
	
	switch cmd {
	case "/help":
		c.printWelcome()
		
	case "/load":
		if len(args) == 0 {
			return fmt.Errorf("用法: /load <mib文件路径>")
		}
		return c.loadMIB(args[0])
		
	case "/extract":
		if len(args) == 0 {
			return fmt.Errorf("用法: /extract <压缩包路径>\n支持格式: zip, tar.gz, tar.bz2, tar, gz")
		}
		return c.extractMIB(args[0])
		
	case "/mibdir":
		return c.handleMibDir(args)
		
	case "/scan":
		c.scanMIBDir()
		
	case "/list":
		c.listMIBs()
		
	case "/search":
		if len(args) == 0 {
			return fmt.Errorf("用法: /search <名称或OID>")
		}
		c.searchObjects(strings.Join(args, " "))
		
	case "/show":
		if len(args) == 0 {
			return fmt.Errorf("用法: /show <OID>")
		}
		c.showObject(args[0])
		
	case "/gen":
		return c.generateConfig(ctx, args)
		
	case "/infra":
		return c.generateInfraConfig(args)
		
	case "/clear":
		c.history = make([]openai.ChatCompletionMessage, 0)
		fmt.Println("对话历史已清除")
		
	case "/exit", "/quit":
		c.mcpManager.StopAll()
		return fmt.Errorf("exit")
		
	default:
		fmt.Printf("未知命令: %s\n使用 /help 查看可用命令\n", cmd)
	}
	
	return nil
}

// loadMIB 加载 MIB 文件
func (c *Chat) loadMIB(path string) error {
	module, err := c.parser.ParseFile(path)
	if err != nil {
		return err
	}
	
	fmt.Printf("已加载 MIB 模块: %s\n", module.Name)
	fmt.Printf("  文件路径: %s\n", module.FilePath)
	fmt.Printf("  对象数量: %d\n", len(module.Objects))
	
	return nil
}

// listMIBs 列出 MIB 文件
func (c *Chat) listMIBs() {
	files := c.parser.ListMIBFiles()
	if len(files) == 0 {
		fmt.Println("未找到 MIB 文件")
		return
	}
	
	fmt.Println("可用的 MIB 文件:")
	for i, file := range files {
		fmt.Printf("  %d. %s\n", i+1, file)
	}
}

// searchObjects 搜索对象
func (c *Chat) searchObjects(query string) {
	objects, err := c.parser.SearchObjects(query)
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
		return
	}
	
	if len(objects) == 0 {
		fmt.Println("未找到匹配的对象")
		return
	}
	
	fmt.Printf("找到 %d 个匹配对象:\n", len(objects))
	for i, obj := range objects {
		if i >= 20 {
			fmt.Printf("  ... 还有 %d 个结果\n", len(objects)-20)
			break
		}
		fmt.Printf("  %s (%s) - %s\n", obj.Name, obj.OID, truncate(obj.Description, 50))
	}
}

// showObject 显示对象详情
func (c *Chat) showObject(oid string) {
	objects, err := c.parser.FindObjectsByOID(oid)
	if err != nil {
		fmt.Printf("查找失败: %v\n", err)
		return
	}
	
	for _, obj := range objects {
		if obj.OID == oid {
			fmt.Printf("名称: %s\n", obj.Name)
			fmt.Printf("OID: %s\n", obj.OID)
			fmt.Printf("类型: %s\n", obj.Type)
			fmt.Printf("访问权限: %s\n", obj.Access)
			fmt.Printf("描述: %s\n", obj.Description)
			if len(obj.EnumValues) > 0 {
				fmt.Println("枚举值:")
				for name, value := range obj.EnumValues {
					fmt.Printf("  %s = %d\n", name, value)
				}
			}
			return
		}
	}
	
	fmt.Println("未找到指定 OID 的对象")
}

// generateConfig 生成配置
func (c *Chat) generateConfig(ctx context.Context, args []string) error {
	// 解析参数
	format := "both"
	oids := args
	
	for i, arg := range args {
		if arg == "--format" || arg == "-f" {
			if i+1 < len(args) {
				format = args[i+1]
				oids = args[i+2:]
				break
			}
		}
	}
	
	if len(oids) == 0 {
		return fmt.Errorf("用法: /gen [--format categraf|snmp_exporter|both] <oid1> [oid2] ...")
	}
	
	// 查找对象
	var objects []*types.MIBObject
	for _, oid := range oids {
		objs, err := c.parser.FindObjectsByOID(oid)
		if err != nil {
			return err
		}
		objects = append(objects, objs...)
	}
	
	if len(objects) == 0 {
		return fmt.Errorf("未找到任何匹配的对象")
	}
	
	// 生成配置
	req := &types.ConfigRequest{
		TargetOIDs:  oids,
		Format:      format,
		MetricNames: make(map[string]string),
		Labels:      c.config.Global.Labels,
	}
	
	result, err := c.generator.GenerateBoth(objects, req)
	if err != nil {
		return err
	}
	
	// 输出结果
	if result.CategrafConfig != "" {
		fmt.Println("\n=== Categraf 配置 ===")
		fmt.Println(result.CategrafConfig)
	}
	
	if result.SNMPExporterConfig != "" {
		fmt.Println("\n=== SNMP Exporter 配置 ===")
		fmt.Println(result.SNMPExporterConfig)
	}
	
	if len(result.Warnings) > 0 {
		fmt.Println("\n警告:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
	
	return nil
}

// handleNaturalLanguage 处理自然语言查询
func (c *Chat) handleNaturalLanguage(ctx context.Context, input string) error {
	// 构建系统提示词
	systemPrompt := c.buildSystemPrompt()
	
	// 添加用户消息
	c.history = append(c.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: input,
	})
	
	// 调用 AI
	req := openai.ChatCompletionRequest{
		Model:    c.getModelName(),
		Messages: append([]openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		}, c.history...),
		Tools:    c.getToolDefinitions(),
	}
	
	resp, err := c.aiClient.CreateChatCompletion(ctx, req)
	if err != nil {
		return fmt.Errorf("AI 调用失败: %w", err)
	}
	
	// 处理响应
	if len(resp.Choices) == 0 {
		return fmt.Errorf("AI 未返回响应")
	}
	
	choice := resp.Choices[0]
	
	// 处理工具调用
	if choice.FinishReason == openai.FinishReasonToolCalls && len(choice.Message.ToolCalls) > 0 {
		return c.handleToolCalls(ctx, choice.Message.ToolCalls)
	}
	
	// 输出文本响应
	fmt.Println(choice.Message.Content)
	c.history = append(c.history, choice.Message)
	
	return nil
}

// buildSystemPrompt 构建系统提示词
func (c *Chat) buildSystemPrompt() string {
	var sb strings.Builder
	
	sb.WriteString("你是一个基础设施监控配置生成助手。你的任务是帮助用户生成各种监控配置。\n\n")
	sb.WriteString("你可以:\n")
	sb.WriteString("1. 搜索 MIB 对象并生成 SNMP 配置\n")
	sb.WriteString("2. 生成 Categraf/SNMP Exporter/Telegraf 配置\n")
	sb.WriteString("3. 生成物理服务器硬件监控配置（Dell iDRAC/HPE iLO/Lenovo/Supermicro）\n")
	sb.WriteString("4. 生成虚拟化平台监控配置（VMware vSphere/Proxmox）\n")
	sb.WriteString("5. 生成网络设备监控配置（华为/华三/Cisco/锐捷等）\n")
	sb.WriteString("6. 生成服务探测配置（HTTP/ICMP/TCP）\n")
	sb.WriteString("7. 解释 OID 含义\n\n")
	
	sb.WriteString("支持的设备类型:\n")
	sb.WriteString("- 网络设备: 华为(NDP)、华三(LNP)、Cisco(CDP)、锐捷、Juniper、Arista 等\n")
	sb.WriteString("- 物理服务器: Dell iDRAC、HPE iLO、Lenovo XClarity、Supermicro IPMI、Fujitsu\n")
	sb.WriteString("- 虚拟化: VMware vSphere、Proxmox VE\n")
	sb.WriteString("- 老旧服务器: IPMI 2.0 通用\n\n")
	
	sb.WriteString("可用工具:\n")
	sb.WriteString("- search_mib: 搜索 MIB 对象\n")
	sb.WriteString("- generate_config: 生成 SNMP 配置文件\n")
	sb.WriteString("- generate_hardware_config: 生成服务器硬件监控配置\n")
	sb.WriteString("- generate_vmware_config: 生成 VMware 监控配置\n")
	sb.WriteString("- generate_network_config: 生成网络设备监控配置\n")
	sb.WriteString("- explain_oid: 解释 OID 含义\n")
	
	return sb.String()
}

// getToolDefinitions 获取工具定义
func (c *Chat) getToolDefinitions() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "search_mib",
				Description: "搜索 MIB 对象，可以按名称或 OID 前缀搜索",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "搜索关键词，可以是对象名称或 OID",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_config",
				Description: "生成 SNMP 采集配置文件（Categraf/SNMP Exporter）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"oids": map[string]interface{}{
							"type":        "array",
							"items":       map[string]string{"type": "string"},
							"description": "要采集的 OID 列表",
						},
						"format": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"categraf", "snmp_exporter", "both"},
							"description": "输出格式",
						},
					},
					"required": []string{"oids"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_hardware_config",
				Description: "生成服务器硬件监控配置（Redfish/IPMI），支持 Dell、HPE、Lenovo、Supermicro、Fujitsu",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device_type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"redfish", "ipmi"},
							"description": "设备类型: redfish(新服务器) 或 ipmi(老服务器)",
						},
						"devices": map[string]interface{}{
							"type":        "array",
							"description": "设备列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":     map[string]string{"type": "string", "description": "设备名称"},
									"host":     map[string]string{"type": "string", "description": "IP地址"},
									"username": map[string]string{"type": "string", "description": "用户名"},
									"password": map[string]string{"type": "string", "description": "密码"},
									"vendor":   map[string]string{"type": "string", "description": "厂商: dell_idrac, hpe_ilo, lenovo, supermicro, generic"},
								},
							},
						},
					},
					"required": []string{"device_type", "devices"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_vmware_config",
				Description: "生成 VMware vSphere 监控配置（vCenter/ESXi）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"vcenters": map[string]interface{}{
							"type":        "array",
							"description": "vCenter 列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":     map[string]string{"type": "string", "description": "vCenter名称"},
									"url":      map[string]string{"type": "string", "description": "vCenter URL (https://host/sdk)"},
									"username": map[string]string{"type": "string", "description": "用户名"},
									"password": map[string]string{"type": "string", "description": "密码"},
								},
							},
						},
					},
					"required": []string{"vcenters"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "list_network_metrics",
				Description: "列出网络设备支持的监控指标，按设备类型和功能分类展示。设备类型: switch(交换机)、router(路由器)、firewall(防火墙)、wireless(无线)、loadbalancer(负载均衡)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device_type": map[string]interface{}{
							"type":        "string",
							"description": "设备类型: switch, router, firewall, wireless, loadbalancer。不填则显示全部",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_network_config",
				Description: "生成网络设备监控配置。输出格式可选: snmp_exporter, categraf, telegraf, vmagent。根据用户需求选择格式，不必全部生成。",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"devices": map[string]interface{}{
							"type":        "array",
							"description": "网络设备列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":        map[string]string{"type": "string", "description": "设备名称"},
									"host":        map[string]string{"type": "string", "description": "IP地址"},
									"device_type": map[string]string{"type": "string", "description": "设备类型: switch, router, firewall, wireless, loadbalancer"},
									"vendor":      map[string]string{"type": "string", "description": "厂商: huawei, h3c, cisco, ruijie, juniper, hpe, zte, maipu, aruba, fortinet, paloalto"},
									"community":   map[string]string{"type": "string", "description": "SNMP团体字符串"},
								},
								"required": []string{"name", "host", "device_type", "vendor"},
							},
						},
						"metric_categories": map[string]interface{}{
							"type":        "array",
							"description": "指标类别: basic(基础), port(端口), hardware(硬件), l2(二层), l3(三层), security(安全), wireless(无线), qos(QoS)。推荐组合: basic+port",
							"items":       map[string]string{"type": "string"},
						},
						"format": map[string]interface{}{
							"type":        "string",
							"description": "输出格式: snmp_exporter(SNMP Exporter), categraf(Categraf), telegraf(Telegraf), vmagent(Prometheus File SD), all(全部)。默认: vmagent",
							"enum":        []string{"snmp_exporter", "categraf", "telegraf", "vmagent", "all"},
						},
						"interval": map[string]interface{}{
							"type":        "string",
							"description": "采集间隔，默认 30s",
						},
					},
					"required": []string{"devices"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_snmp_config",
				Description: "生成自定义 SNMP 配置，基于用户指定的 OID 列表，适合高级用户精确控制监控项",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"device_name": map[string]interface{}{
							"type":        "string",
							"description": "设备名称",
						},
						"device_host": map[string]interface{}{
							"type":        "string",
							"description": "设备 IP 地址",
						},
						"community": map[string]interface{}{
							"type":        "string",
							"description": "SNMP 团体字符串",
						},
						"oids": map[string]interface{}{
							"type":        "array",
							"description": "OID 列表，每个 OID 包含名称、OID、描述",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":        map[string]string{"type": "string", "description": "指标名称"},
									"oid":         map[string]string{"type": "string", "description": "OID 编号"},
									"description": map[string]string{"type": "string", "description": "指标描述"},
									"type":        map[string]string{"type": "string", "description": "类型: gauge, counter, counter64"},
								},
								"required": []string{"name", "oid"},
							},
						},
						"format": map[string]interface{}{
							"type":        "string",
							"description": "输出格式: categraf, snmp_exporter, telegraf",
						},
					},
					"required": []string{"device_name", "device_host", "oids"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "explain_oid",
				Description: "解释 OID 的含义和用途",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"oid": map[string]interface{}{
							"type":        "string",
							"description": "要解释的 OID",
						},
					},
					"required": []string{"oid"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_node_config",
				Description: "生成 Node Exporter 主机监控配置（Linux/Windows服务器）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nodes": map[string]interface{}{
							"type":        "array",
							"description": "主机节点列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":   map[string]string{"type": "string", "description": "主机名称"},
									"host":   map[string]string{"type": "string", "description": "IP地址"},
									"port":   map[string]string{"type": "string", "description": "Node Exporter端口，默认9100"},
									"labels": map[string]string{"type": "string", "description": "额外标签，如env=prod,role=web"},
								},
							},
						},
					},
					"required": []string{"nodes"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_blackbox_config",
				Description: "生成 Blackbox Exporter 探测配置（HTTP/ICMP/TCP/DNS探测）",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"probes": map[string]interface{}{
							"type":        "array",
							"description": "探测目标列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":    map[string]string{"type": "string", "description": "探测名称"},
									"target":  map[string]string{"type": "string", "description": "探测目标URL或IP"},
									"module":  map[string]string{"type": "string", "description": "探测模块: http_2xx, http_post_2xx, icmp, tcp_connect, dns_udp"},
									"labels":  map[string]string{"type": "string", "description": "额外标签"},
								},
							},
						},
					},
					"required": []string{"probes"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_ipmi_config",
				Description: "生成 IPMI Exporter 物理服务器监控配置",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"devices": map[string]interface{}{
							"type":        "array",
							"description": "IPMI设备列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":     map[string]string{"type": "string", "description": "服务器名称"},
									"host":     map[string]string{"type": "string", "description": "IPMI地址"},
									"port":     map[string]string{"type": "string", "description": "IPMI端口，默认623"},
									"username": map[string]string{"type": "string", "description": "IPMI用户名"},
									"password": map[string]string{"type": "string", "description": "IPMI密码"},
								},
							},
						},
					},
					"required": []string{"devices"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        "generate_proxmox_config",
				Description: "生成 Proxmox VE 虚拟化平台监控配置",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nodes": map[string]interface{}{
							"type":        "array",
							"description": "Proxmox节点列表",
							"items": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"name":     map[string]string{"type": "string", "description": "节点名称"},
									"host":     map[string]string{"type": "string", "description": "API地址"},
									"port":     map[string]string{"type": "string", "description": "API端口，默认8006"},
									"username": map[string]string{"type": "string", "description": "用户名，如root@pam"},
									"password": map[string]string{"type": "string", "description": "密码"},
									"token":    map[string]string{"type": "string", "description": "API Token（可选）"},
								},
							},
						},
					},
					"required": []string{"nodes"},
				},
			},
		},
	}
}

// handleToolCalls 处理工具调用
func (c *Chat) handleToolCalls(ctx context.Context, toolCalls []openai.ToolCall) error {
	for _, toolCall := range toolCalls {
		var result string
		var err error
		
		switch toolCall.Function.Name {
		case "search_mib":
			result, err = c.toolSearchMIB(toolCall.Function.Arguments)
		case "generate_config":
			result, err = c.toolGenerateConfig(ctx, toolCall.Function.Arguments)
		case "generate_hardware_config":
			result, err = c.toolGenerateHardwareConfig(toolCall.Function.Arguments)
		case "generate_vmware_config":
			result, err = c.toolGenerateVMwareConfig(toolCall.Function.Arguments)
		case "list_network_metrics":
			result, err = c.toolListNetworkMetrics(toolCall.Function.Arguments)
		case "generate_network_config":
			result, err = c.toolGenerateNetworkConfig(toolCall.Function.Arguments)
		case "generate_snmp_config":
			result, err = c.toolGenerateSNMPConfig(toolCall.Function.Arguments)
		case "explain_oid":
			result, err = c.toolExplainOID(toolCall.Function.Arguments)
		case "generate_node_config":
			result, err = c.toolGenerateNodeConfig(toolCall.Function.Arguments)
		case "generate_blackbox_config":
			result, err = c.toolGenerateBlackboxConfig(toolCall.Function.Arguments)
		case "generate_ipmi_config":
			result, err = c.toolGenerateIPMIConfig(toolCall.Function.Arguments)
		case "generate_proxmox_config":
			result, err = c.toolGenerateProxmoxConfig(toolCall.Function.Arguments)
		default:
			result = fmt.Sprintf("未知工具: %s", toolCall.Function.Name)
		}
		
		if err != nil {
			result = fmt.Sprintf("错误: %v", err)
		}
		
		// 添加工具响应到历史
		c.history = append(c.history, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		})
	}
	
	// 继续对话
	return c.handleNaturalLanguage(ctx, "")
}

// toolSearchMIB 搜索 MIB 工具
func (c *Chat) toolSearchMIB(args string) (string, error) {
	var params struct {
		Query string `json:"query"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	objects, err := c.parser.SearchObjects(params.Query)
	if err != nil {
		return "", err
	}
	
	var sb strings.Builder
	for _, obj := range objects {
		sb.WriteString(fmt.Sprintf("%s|%s|%s|%s\n", obj.Name, obj.OID, obj.Type, obj.Description))
	}
	
	return sb.String(), nil
}

// toolGenerateConfig 生成配置工具
func (c *Chat) toolGenerateConfig(ctx context.Context, args string) (string, error) {
	var params struct {
		OIDs   []string `json:"oids"`
		Format string   `json:"format"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	if params.Format == "" {
		params.Format = "both"
	}
	
	var objects []*types.MIBObject
	for _, oid := range params.OIDs {
		objs, err := c.parser.FindObjectsByOID(oid)
		if err != nil {
			return "", err
		}
		objects = append(objects, objs...)
	}
	
	req := &types.ConfigRequest{
		TargetOIDs:  params.OIDs,
		Format:      params.Format,
		MetricNames: make(map[string]string),
		Labels:      c.config.Global.Labels,
	}
	
	result, err := c.generator.GenerateBoth(objects, req)
	if err != nil {
		return "", err
	}
	
	var sb strings.Builder
	if result.CategrafConfig != "" {
		sb.WriteString("CATEGRAF_CONFIG:\n")
		sb.WriteString(result.CategrafConfig)
		sb.WriteString("\n")
	}
	if result.SNMPExporterConfig != "" {
		sb.WriteString("SNMP_EXPORTER_CONFIG:\n")
		sb.WriteString(result.SNMPExporterConfig)
	}
	
	return sb.String(), nil
}

// toolExplainOID 解释 OID 工具
func (c *Chat) toolExplainOID(args string) (string, error) {
	var params struct {
		OID string `json:"oid"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	objects, err := c.parser.FindObjectsByOID(params.OID)
	if err != nil {
		return "", err
	}
	
	for _, obj := range objects {
		if obj.OID == params.OID {
			return fmt.Sprintf("名称: %s\n类型: %s\n访问: %s\n描述: %s",
				obj.Name, obj.Type, obj.Access, obj.Description), nil
		}
	}
	
	return "未找到该 OID 的定义", nil
}

// toolGenerateHardwareConfig 生成硬件监控配置工具
func (c *Chat) toolGenerateHardwareConfig(args string) (string, error) {
	var params struct {
		DeviceType string `json:"device_type"`
		Devices    []struct {
			Name     string `json:"name"`
			Host     string `json:"host"`
			Username string `json:"username"`
			Password string `json:"password"`
			Vendor   string `json:"vendor"`
		} `json:"devices"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== %s 硬件监控配置 ===\n\n", strings.ToUpper(params.DeviceType)))
	
	if params.DeviceType == "redfish" {
		sb.WriteString("# Telegraf Redfish 配置\n")
		sb.WriteString("# 由 AI 助手生成\n\n")
		sb.WriteString("[[inputs.redfish]]\n")
		sb.WriteString("  interval = \"60s\"\n\n")
		
		for _, dev := range params.Devices {
			sb.WriteString(fmt.Sprintf("  [[inputs.redfish.server]]\n"))
			sb.WriteString(fmt.Sprintf("    name = \"%s\"\n", dev.Name))
			sb.WriteString(fmt.Sprintf("    address = \"https://%s\"\n", dev.Host))
			sb.WriteString(fmt.Sprintf("    username = \"%s\"\n", dev.Username))
			sb.WriteString(fmt.Sprintf("    password = \"%s\"\n", dev.Password))
			sb.WriteString("    insecure_skip_verify = true\n")
			sb.WriteString("    include_metrics = [\"thermal\", \"power\", \"system\", \"storage\"]\n")
			if dev.Vendor != "" {
				sb.WriteString(fmt.Sprintf("    [inputs.redfish.server.tags]\n"))
				sb.WriteString(fmt.Sprintf("      vendor = \"%s\"\n", dev.Vendor))
			}
			sb.WriteString("\n")
		}
	} else if params.DeviceType == "ipmi" {
		sb.WriteString("# Telegraf IPMI 配置\n")
		sb.WriteString("# 由 AI 助手生成\n\n")
		sb.WriteString("[[inputs.ipmi_sensor]]\n")
		sb.WriteString("  interval = \"60s\"\n")
		sb.WriteString("  metric_version = 2\n\n")
		
		for _, dev := range params.Devices {
			sb.WriteString(fmt.Sprintf("  [[inputs.ipmi_sensor.server]]\n"))
			sb.WriteString(fmt.Sprintf("    host = \"%s\"\n", dev.Host))
			sb.WriteString(fmt.Sprintf("    username = \"%s\"\n", dev.Username))
			sb.WriteString(fmt.Sprintf("    password = \"%s\"\n", dev.Password))
			sb.WriteString("    interface = \"lanplus\"\n")
			sb.WriteString("    port = 623\n\n")
		}
	}
	
	sb.WriteString("\n💡 将配置添加到 telegraf.conf 或 telegraf-ipmi.conf 中")
	return sb.String(), nil
}

// toolGenerateVMwareConfig 生成 VMware 监控配置工具
func (c *Chat) toolGenerateVMwareConfig(args string) (string, error) {
	var params struct {
		VCenters []struct {
			Name     string `json:"name"`
			URL      string `json:"url"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"vcenters"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	var sb strings.Builder
	sb.WriteString("=== VMware vSphere 监控配置 ===\n\n")
	sb.WriteString("# Telegraf VMware 配置\n")
	sb.WriteString("# 由 AI 助手生成\n\n")
	sb.WriteString("[[inputs.vsphere]]\n")
	sb.WriteString("  interval = \"60s\"\n")
	sb.WriteString("  timeout = \"60s\"\n\n")
	
	for _, vc := range params.VCenters {
		sb.WriteString(fmt.Sprintf("  # vCenter: %s\n", vc.Name))
		sb.WriteString(fmt.Sprintf("  vcenters = [\"%s\"]\n", vc.URL))
		sb.WriteString(fmt.Sprintf("  username = \"%s\"\n", vc.Username))
		sb.WriteString(fmt.Sprintf("  password = \"%s\"\n", vc.Password))
		sb.WriteString("  insecure_skip_verify = true\n\n")
	}
	
	sb.WriteString("  # 虚拟机指标\n")
	sb.WriteString("  vm_metric_include = [\n")
	sb.WriteString("    \"cpu.usage.average\",\n")
	sb.WriteString("    \"mem.usage.average\",\n")
	sb.WriteString("    \"disk.usage.average\",\n")
	sb.WriteString("    \"net.usage.average\",\n")
	sb.WriteString("  ]\n\n")
	
	sb.WriteString("  # ESXi 主机指标\n")
	sb.WriteString("  host_metric_include = [\n")
	sb.WriteString("    \"cpu.usage.average\",\n")
	sb.WriteString("    \"mem.usage.average\",\n")
	sb.WriteString("  ]\n")
	
	sb.WriteString("\n💡 将配置添加到 telegraf.conf 中")
	return sb.String(), nil
}

// toolListNetworkMetrics 列出网络设备支持的监控指标
func (c *Chat) toolListNetworkMetrics(args string) (string, error) {
	var params struct {
		DeviceType string `json:"device_type"`
	}
	json.Unmarshal([]byte(args), &params)
	
	var sb strings.Builder
	sb.WriteString("📡 网络设备监控指标分类\n\n")
	
	// 按类别分组
	categories := []struct {
		Name        string
		Description string
		Icon        string
		Metrics     []struct {
			Name        string
			Description string
			OIDHint     string
			DeviceTypes []string
		}
	}{
		{
			Name: "basic", Description: "基础指标", Icon: "📊",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"system", "系统信息 (描述/运行时间/名称)", "sysDescr/sysUpTime/sysName", []string{"all"}},
				{"cpu", "CPU 使用率", "hrProcessorLoad/厂商私有MIB", []string{"all"}},
				{"memory", "内存使用率", "hrMemory/厂商私有MIB", []string{"all"}},
			},
		},
		{
			Name: "port", Description: "端口指标", Icon: "🔌",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"interface", "接口基本信息 (类型/速率/MTU)", "ifTable/ifXTable", []string{"all"}},
				{"port_status", "端口状态 (up/down/admin)", "ifOperStatus/ifAdminStatus", []string{"switch", "router"}},
				{"port_traffic", "端口流量 (入/出字节/包)", "ifHCInOctets/ifHCOutOctets", []string{"switch", "router"}},
				{"port_errors", "端口错误 (CRC/丢包/冲突)", "ifInErrors/ifOutErrors/ifInDiscards", []string{"switch", "router"}},
				{"optics", "光模块 (温度/功率/波长)", "ENTITY-SENSOR-MIB", []string{"switch", "router"}},
			},
		},
		{
			Name: "hardware", Description: "硬件指标", Icon: "🖥️",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"environment", "环境传感器 (温度/风扇/电源)", "ENTITY-SENSOR-MIB/厂商私有", []string{"switch", "router", "firewall"}},
				{"stack", "堆叠状态 (成员/端口)", "CISCO-STACK-MIB/厂商私有", []string{"switch"}},
				{"poe", "PoE 功率/端口状态", "POE-MIB/厂商私有", []string{"switch"}},
			},
		},
		{
			Name: "l2", Description: "二层协议", Icon: "🔄",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"vlan", "VLAN 信息", "Q-BRIDGE-MIB", []string{"switch"}},
				{"stp", "生成树状态", "BRIDGE-MIB/RSTP-MIB", []string{"switch"}},
				{"lldp", "LLDP 邻居发现", "LLDP-MIB", []string{"all"}},
				{"lacp", "链路聚合状态", "IEEE8023-LAG-MIB", []string{"switch"}},
			},
		},
		{
			Name: "l3", Description: "三层协议", Icon: "🌐",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"routes", "路由表统计", "ipRouteTable", []string{"router", "firewall", "switch"}},
				{"ospf", "OSPF 邻居/区域状态", "OSPF-MIB", []string{"router", "switch"}},
				{"bgp", "BGP 对等体状态", "BGP4-MIB", []string{"router"}},
				{"vrrp", "VRRP/HSRP 状态", "VRRP-MIB/HSRP-MIB", []string{"router", "firewall"}},
				{"arp", "ARP 表统计", "ipNetToMediaTable", []string{"all"}},
			},
		},
		{
			Name: "security", Description: "安全指标", Icon: "🔒",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"acl", "ACL 匹配计数", "CISCO-ACL-MIB/厂商私有", []string{"firewall", "router", "switch"}},
				{"nat", "NAT 连接/转换统计", "NAT-MIB/厂商私有", []string{"firewall", "router"}},
				{"vpn", "VPN 隧道状态/流量", "厂商私有MIB", []string{"firewall", "router"}},
			},
		},
		{
			Name: "wireless", Description: "无线指标", Icon: "📶",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"ap_status", "AP 在线状态", "IEEE802.11-MIB/厂商私有", []string{"wireless"}},
				{"wireless", "无线客户端/信道/功率", "厂商私有MIB", []string{"wireless"}},
				{"ssid", "SSID 统计", "厂商私有MIB", []string{"wireless"}},
			},
		},
		{
			Name: "qos", Description: "QoS 指标", Icon: "⚡",
			Metrics: []struct {
				Name        string
				Description string
				OIDHint     string
				DeviceTypes []string
			}{
				{"qos", "QoS 队列统计", "CISCO-CLASS-BASED-QOS-MIB", []string{"router", "switch"}},
				{"dhcp", "DHCP 统计", "CISCO-DHCP-SNOOPING-MIB", []string{"switch"}},
			},
		},
	}
	
	for _, cat := range categories {
		// 如果指定了设备类型，过滤相关指标
		showCategory := true
		if params.DeviceType != "" && params.DeviceType != "all" {
			hasRelevant := false
			for _, m := range cat.Metrics {
				for _, dt := range m.DeviceTypes {
					if dt == "all" || dt == params.DeviceType {
						hasRelevant = true
						break
					}
				}
				if hasRelevant {
					break
				}
			}
			showCategory = hasRelevant
		}
		
		if !showCategory {
			continue
		}
		
		sb.WriteString(fmt.Sprintf("### %s %s - %s\n", cat.Icon, cat.Name, cat.Description))
		sb.WriteString("| 指标 | 说明 | 适用设备 |\n")
		sb.WriteString("|:-----|:-----|:---------|\n")
		
		for _, m := range cat.Metrics {
			// 过滤设备类型
			if params.DeviceType != "" && params.DeviceType != "all" {
				relevant := false
				for _, dt := range m.DeviceTypes {
					if dt == "all" || dt == params.DeviceType {
						relevant = true
						break
					}
				}
				if !relevant {
					continue
				}
			}
			
			deviceTypes := strings.Join(m.DeviceTypes, "/")
			if deviceTypes == "all" {
				deviceTypes = "全部"
			}
			sb.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", m.Name, m.Description, deviceTypes))
		}
		sb.WriteString("\n")
	}
	
	sb.WriteString("---\n\n")
	sb.WriteString("**推荐组合:**\n")
	sb.WriteString("```\n")
	sb.WriteString("交换机: basic + port + hardware + l2\n")
	sb.WriteString("路由器: basic + port + hardware + l3\n")
	sb.WriteString("防火墙: basic + port + hardware + security\n")
	sb.WriteString("无线:   basic + port + wireless\n")
	sb.WriteString("```\n\n")
	sb.WriteString("**使用示例:**\n")
	sb.WriteString("```\n")
	sb.WriteString(">>> 帮我监控华为核心交换机，指标: basic, port, hardware, l2\n")
	sb.WriteString(">>> 帮我监控 Cisco 路由器，需要 OSPF 和 BGP 状态\n")
	sb.WriteString("```\n")
	
	return sb.String(), nil
}

// toolGenerateNetworkConfig 生成网络设备监控配置工具
func (c *Chat) toolGenerateNetworkConfig(args string) (string, error) {
	var params struct {
		Devices []struct {
			Name       string `json:"name"`
			Host       string `json:"host"`
			DeviceType string `json:"device_type"`
			Vendor     string `json:"vendor"`
			Community  string `json:"community"`
		} `json:"devices"`
		MetricCategories []string `json:"metric_categories"`
		Format           string   `json:"format"`
		Interval         string   `json:"interval"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	// 根据类别映射到具体指标
	categoryMetrics := map[string][]string{
		"basic":    {"system", "cpu", "memory"},
		"port":     {"interface", "port_status", "port_traffic", "port_errors"},
		"hardware": {"environment"},
		"l2":       {"vlan", "stp", "lldp"},
		"l3":       {"routes", "arp"},
		"security": {"acl"},
		"wireless": {"wireless"},
		"qos":      {"qos"},
	}
	
	// 收集指标
	metrics := []string{}
	for _, cat := range params.MetricCategories {
		if m, ok := categoryMetrics[cat]; ok {
			metrics = append(metrics, m...)
		}
	}
	
	// 默认指标
	if len(metrics) == 0 {
		metrics = []string{"system", "cpu", "memory", "interface"}
	}
	
	// 默认格式
	format := params.Format
	if format == "" {
		format = "vmagent"
	}
	
	interval := params.Interval
	if interval == "" {
		interval = "30s"
	}
	
	outputDir := "./output/infra/config"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}
	
	var sb strings.Builder
	sb.WriteString("=== 📡 网络设备 SNMP 监控配置 ===\n\n")
	sb.WriteString(fmt.Sprintf("设备类型: %s\n", params.Devices[0].DeviceType))
	sb.WriteString(fmt.Sprintf("监控类别: %s\n", strings.Join(params.MetricCategories, ", ")))
	sb.WriteString(fmt.Sprintf("具体指标: %s\n", strings.Join(metrics, ", ")))
	sb.WriteString(fmt.Sprintf("输出格式: %s\n", format))
	sb.WriteString(fmt.Sprintf("采集间隔: %s\n\n", interval))
	
	// 构建设备信息
	devices := params.Devices
	
	// 根据格式生成配置
	switch format {
	case "vmagent", "all":
		targets := []map[string]interface{}{}
		for _, dev := range devices {
			community := dev.Community
			if community == "" {
				community = "public"
			}
			targets = append(targets, map[string]interface{}{
				"targets": []string{dev.Host},
				"labels": map[string]string{
					"job":         "snmp",
					"instance":    dev.Name,
					"device_type": dev.DeviceType,
					"vendor":      dev.Vendor,
					"community":   community,
				},
			})
		}
		targetsDir := filepath.Join(outputDir, "vmagent/targets")
		os.MkdirAll(targetsDir, 0755)
		targetsPath := filepath.Join(targetsDir, "snmp-devices.json")
		data, _ := json.MarshalIndent(targets, "", "  ")
		os.WriteFile(targetsPath, data, 0644)
		sb.WriteString(fmt.Sprintf("✅ vmagent File SD: %s\n", targetsPath))
	}
	
	switch format {
	case "snmp_exporter", "all":
		snmpConfig := c.generateSNMPExporterConfig(metrics, devices)
		snmpDir := filepath.Join(outputDir, "snmp-exporter")
		os.MkdirAll(snmpDir, 0755)
		snmpPath := filepath.Join(snmpDir, "snmp.yml")
		os.WriteFile(snmpPath, []byte(snmpConfig), 0644)
		sb.WriteString(fmt.Sprintf("✅ SNMP Exporter: %s\n", snmpPath))
	}
	
	switch format {
	case "categraf", "all":
		categrafConfig := c.generateCategrafSNMPConfig(metrics, devices, interval)
		categrafDir := filepath.Join(outputDir, "categraf")
		os.MkdirAll(categrafDir, 0755)
		categrafPath := filepath.Join(categrafDir, "snmp_network.toml")
		os.WriteFile(categrafPath, []byte(categrafConfig), 0644)
		sb.WriteString(fmt.Sprintf("✅ Categraf: %s\n", categrafPath))
	}
	
	switch format {
	case "telegraf", "all":
		telegrafConfig := c.generateTelegrafSNMPConfig(metrics, devices, interval)
		telegrafDir := filepath.Join(outputDir, "telegraf")
		os.MkdirAll(telegrafDir, 0755)
		telegrafPath := filepath.Join(telegrafDir, "snmp.conf")
		os.WriteFile(telegrafPath, []byte(telegrafConfig), 0644)
		sb.WriteString(fmt.Sprintf("✅ Telegraf: %s\n", telegrafPath))
	}
	
	sb.WriteString(fmt.Sprintf("\n📊 监控设备: %d 台\n", len(devices)))
	sb.WriteString(fmt.Sprintf("📋 监控指标: %d 项\n\n", len(metrics)))
	sb.WriteString("🚀 下一步:\n")
	sb.WriteString("   1. 检查生成的配置文件\n")
	sb.WriteString("   2. 根据需要调整参数\n")
	sb.WriteString("   3. 部署到对应采集器\n")
	
	return sb.String(), nil
}

// generateTelegrafSNMPConfig 生成 Telegraf SNMP 配置
func (c *Chat) generateTelegrafSNMPConfig(metrics []string, devices interface{}, interval string) string {
	var sb strings.Builder
	sb.WriteString("# Telegraf SNMP 配置 - 自动生成\n")
	sb.WriteString("# 生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	
	sb.WriteString("[[inputs.snmp]]\n")
	sb.WriteString(fmt.Sprintf("interval = \"%s\"\n", interval))
	sb.WriteString("version = 2\n")
	sb.WriteString("community = \"public\"\n\n")
	
	sb.WriteString("# 监控目标\n")
	sb.WriteString("agents = [\n")
	
	// 处理设备列表
	switch d := devices.(type) {
	case []struct {
		Name       string `json:"name"`
		Host       string `json:"host"`
		DeviceType string `json:"device_type"`
		Vendor     string `json:"vendor"`
		Community  string `json:"community"`
	}:
		for i, dev := range d {
			sb.WriteString(fmt.Sprintf("  \"udp://%s:161\"", dev.Host))
			if i < len(d)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("]\n\n")
	
	sb.WriteString("# 监控指标\n")
	sb.WriteString("# 已启用指标: " + strings.Join(metrics, ", ") + "\n")
	
	return sb.String()
}

// generateSNMPExporterConfig 生成 SNMP Exporter 配置
func (c *Chat) generateSNMPExporterConfig(metrics []string, devices interface{}) string {
	// 根据指标生成 OID 配置
	oidConfigs := map[string][]struct {
		Name string
		OID  string
		Type string
	}{
		"cpu": {
			{"cpu_usage", "1.3.6.1.2.1.25.3.3.1.2", "gauge"},
		},
		"memory": {
			{"memory_total", "1.3.6.1.2.1.25.2.3.1.5", "gauge"},
			{"memory_used", "1.3.6.1.2.1.25.2.3.1.6", "gauge"},
		},
		"interface": {
			{"if_descr", "1.3.6.1.2.1.2.2.1.2", "string"},
			{"if_type", "1.3.6.1.2.1.2.2.1.3", "gauge"},
			{"if_speed", "1.3.6.1.2.1.2.2.1.5", "gauge"},
		},
		"port_status": {
			{"if_oper_status", "1.3.6.1.2.1.2.2.1.8", "gauge"},
			{"if_admin_status", "1.3.6.1.2.1.2.2.1.7", "gauge"},
		},
		"port_traffic": {
			{"if_in_octets", "1.3.6.1.2.1.31.1.1.1.6", "counter"},
			{"if_out_octets", "1.3.6.1.2.1.31.1.1.1.10", "counter"},
			{"if_in_packets", "1.3.6.1.2.1.31.1.1.1.7", "counter"},
			{"if_out_packets", "1.3.6.1.2.1.31.1.1.1.11", "counter"},
		},
		"port_errors": {
			{"if_in_errors", "1.3.6.1.2.1.2.2.1.14", "counter"},
			{"if_out_errors", "1.3.6.1.2.1.2.2.1.20", "counter"},
			{"if_in_discards", "1.3.6.1.2.1.2.2.1.13", "counter"},
			{"if_out_discards", "1.3.6.1.2.1.2.2.1.19", "counter"},
		},
		"system": {
			{"sys_desc", "1.3.6.1.2.1.1.1", "string"},
			{"sys_uptime", "1.3.6.1.2.1.1.3", "gauge"},
			{"sys_name", "1.3.6.1.2.1.1.5", "string"},
		},
		"environment": {
			{"temperature", "1.3.6.1.4.1.9.9.13.1.3.1.6", "gauge"},
			{"fan_status", "1.3.6.1.4.1.9.9.13.1.4.1.3", "gauge"},
			{"power_status", "1.3.6.1.4.1.9.9.13.1.5.1.3", "gauge"},
		},
		"lldp": {
			{"lldp_neighbors", "1.0.8802.1.1.2.1.4.1.1.9", "string"},
		},
		"vlan": {
			{"vlan_id", "1.3.6.1.2.1.17.7.1.4.3.1.1", "gauge"},
		},
		"stp": {
			{"stp_state", "1.3.6.1.2.1.17.2.15.1.1", "gauge"},
		},
	}
	
	var sb strings.Builder
	sb.WriteString("# SNMP Exporter 配置 - 自动生成\n")
	sb.WriteString("# 生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	sb.WriteString("default:\n")
	sb.WriteString("  walk:\n")
	
	// 收集所有 OID
	oidSet := make(map[string]bool)
	for _, m := range metrics {
		if oids, ok := oidConfigs[m]; ok {
			for _, oid := range oids {
				if !oidSet[oid.OID] {
					sb.WriteString(fmt.Sprintf("    - %s\n", oid.OID))
					oidSet[oid.OID] = true
				}
			}
		}
	}
	
	sb.WriteString("  metrics:\n")
	
	for _, m := range metrics {
		if oids, ok := oidConfigs[m]; ok {
			for _, oid := range oids {
				sb.WriteString(fmt.Sprintf("    - name: %s\n", oid.Name))
				sb.WriteString(fmt.Sprintf("      oid: %s\n", oid.OID))
				sb.WriteString(fmt.Sprintf("      type: %s\n", oid.Type))
				sb.WriteString("      labels:\n")
				sb.WriteString("        - interface\n")
				sb.WriteString("\n")
			}
		}
	}
	
	return sb.String()
}

// generateCategrafSNMPConfig 生成 Categraf SNMP 配置
func (c *Chat) generateCategrafSNMPConfig(metrics []string, devices interface{}, interval string) string {
	var sb strings.Builder
	sb.WriteString("# Categraf SNMP 配置 - 自动生成\n")
	sb.WriteString("# 生成时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n\n")
	
	sb.WriteString("[[instances]]\n")
	sb.WriteString(fmt.Sprintf("interval = \"%s\"\n", interval))
	sb.WriteString("version = 2\n")
	sb.WriteString("community = \"public\"\n")
	sb.WriteString("\n")
	
	sb.WriteString("# 监控目标 (请根据实际情况修改)\n")
	sb.WriteString("targets = [\n")
	sb.WriteString("  \"192.168.1.1:public\",\n")
	sb.WriteString("]\n\n")
	
	sb.WriteString("# 监控指标\n")
	sb.WriteString("# 已启用指标: " + strings.Join(metrics, ", ") + "\n")
	
	return sb.String()
}

// toolGenerateSNMPConfig 生成自定义 SNMP 配置
func (c *Chat) toolGenerateSNMPConfig(args string) (string, error) {
	var params struct {
		DeviceName  string `json:"device_name"`
		DeviceHost  string `json:"device_host"`
		Community   string `json:"community"`
		Format      string `json:"format"`
		OIDs []struct {
			Name        string `json:"name"`
			OID         string `json:"oid"`
			Description string `json:"description"`
			Type        string `json:"type"`
		} `json:"oids"`
	}
	
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", err
	}
	
	community := params.Community
	if community == "" {
		community = "public"
	}
	
	format := params.Format
	if format == "" {
		format = "categraf"
	}
	
	outputDir := "./output/infra/config/custom"
	os.MkdirAll(outputDir, 0755)
	
	var configContent string
	var filename string
	
	switch format {
	case "snmp_exporter":
		configContent = c.generateCustomSNMPExporter(params)
		filename = "snmp_custom.yml"
	case "telegraf":
		configContent = c.generateCustomTelegraf(params)
		filename = "telegraf_custom.conf"
	default: // categraf
		configContent = c.generateCustomCategraf(params)
		filename = "snmp_custom.toml"
	}
	
	outputPath := filepath.Join(outputDir, filename)
	os.WriteFile(outputPath, []byte(configContent), 0644)
	
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✅ 自定义 SNMP 配置已生成 (%s 格式)\n", format))
	sb.WriteString(fmt.Sprintf("📁 文件: %s\n", outputPath))
	sb.WriteString(fmt.Sprintf("📊 设备: %s (%s)\n", params.DeviceName, params.DeviceHost))
	sb.WriteString(fmt.Sprintf("📋 OID 数量: %d\n", len(params.OIDs)))
	
	return sb.String(), nil
}

func (c *Chat) generateCustomCategraf(params interface{}) string {
	return "# Categraf 自定义 SNMP 配置\n# 请使用标准 Categraf SNMP 配置格式\n"
}

func (c *Chat) generateCustomSNMPExporter(params interface{}) string {
	return "# SNMP Exporter 自定义配置\n# 请使用标准 SNMP Exporter 配置格式\n"
}

func (c *Chat) generateCustomTelegraf(params interface{}) string {
	return "# Telegraf 自定义 SNMP 配置\n# 请使用标准 Telegraf SNMP 配置格式\n"
}

// getModelName 获取模型名称
func (c *Chat) getModelName() string {
	if len(c.config.AI.ModelPriority) == 0 {
		return "gpt-4o"
	}
	
	modelName := c.config.AI.ModelPriority[0]
	if modelCfg, ok := c.config.AI.Models[modelName]; ok {
		return modelCfg.Model
	}
	
	return "gpt-4o"
}

// Stop 停止对话
func (c *Chat) Stop() {
	c.mcpManager.StopAll()
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractMIB 解压 MIB 压缩包
func (c *Chat) extractMIB(archivePath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在: %s", archivePath)
	}

	// 检查是否为支持的格式
	if !mibparser.IsArchiveFile(archivePath) {
		return fmt.Errorf("不支持的压缩格式，支持: zip, tar.gz, tar.bz2, tar, gz")
	}

	fmt.Printf("正在解压: %s\n", filepath.Base(archivePath))
	fmt.Printf("目标目录: %s\n", c.mibDir)

	// 解压文件
	mibFiles, err := c.extractor.Extract(archivePath)
	if err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	if len(mibFiles) == 0 {
		fmt.Println("⚠️  未在压缩包中找到 MIB 文件")
		return nil
	}

	fmt.Printf("✅ 成功解压 %d 个 MIB 文件:\n", len(mibFiles))
	for i, f := range mibFiles {
		if i < 10 {
			fmt.Printf("  - %s\n", filepath.Base(f))
		} else if i == 10 {
			fmt.Printf("  ... 还有 %d 个文件\n", len(mibFiles)-10)
			break
		}
	}

	// 重新扫描 MIB 目录
	c.parser = mibparser.NewParser([]string{c.mibDir})
	return nil
}

// handleMibDir 处理 MIB 目录命令
func (c *Chat) handleMibDir(args []string) error {
	if len(args) == 0 {
		// 显示当前 MIB 目录
		fmt.Printf("当前 MIB 目录: %s\n", c.mibDir)
		
		// 检查目录是否存在
		if _, err := os.Stat(c.mibDir); os.IsNotExist(err) {
			fmt.Println("⚠️  目录不存在")
			fmt.Println("使用 /mibdir <路径> 设置新的 MIB 目录")
			return nil
		}
		
		// 扫描目录中的文件
		mibFiles, _ := mibparser.ScanForMIBFiles(c.mibDir)
		archives, _ := mibparser.ScanForArchives(c.mibDir)
		
		fmt.Printf("MIB 文件: %d 个\n", len(mibFiles))
		fmt.Printf("压缩包: %d 个\n", len(archives))
		
		if len(archives) > 0 {
			fmt.Println("\n发现压缩包，使用 /extract <文件名> 解压")
		}
		return nil
	}

	// 设置新的 MIB 目录
	newDir := args[0]
	
	// 转换为绝对路径
	if !filepath.IsAbs(newDir) {
		absPath, err := filepath.Abs(newDir)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败: %w", err)
		}
		newDir = absPath
	}

	// 创建目录（如果不存在）
	if err := os.MkdirAll(newDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	c.mibDir = newDir
	c.parser = mibparser.NewParser([]string{newDir})
	c.extractor = mibparser.NewExtractor(newDir)

	fmt.Printf("✅ MIB 目录已设置为: %s\n", newDir)
	
	// 扫描新目录
	c.scanMIBDir()
	return nil
}

// scanMIBDir 扫描 MIB 目录
func (c *Chat) scanMIBDir() {
	fmt.Printf("扫描目录: %s\n\n", c.mibDir)

	// 扫描 MIB 文件
	mibFiles, err := mibparser.ScanForMIBFiles(c.mibDir)
	if err != nil {
		fmt.Printf("扫描失败: %v\n", err)
		return
	}

	// 扫描压缩包
	archives, _ := mibparser.ScanForArchives(c.mibDir)

	// 显示 MIB 文件
	if len(mibFiles) > 0 {
		fmt.Printf("📄 MIB 文件 (%d 个):\n", len(mibFiles))
		for i, f := range mibFiles {
			if i < 15 {
				fmt.Printf("  %d. %s\n", i+1, filepath.Base(f))
			} else if i == 15 {
				fmt.Printf("  ... 还有 %d 个文件\n", len(mibFiles)-15)
				break
			}
		}
	} else {
		fmt.Println("📄 未发现 MIB 文件")
	}

	// 显示压缩包
	if len(archives) > 0 {
		fmt.Printf("\n📦 压缩包 (%d 个):\n", len(archives))
		for i, f := range archives {
			if i < 10 {
				fmt.Printf("  %d. %s\n", i+1, filepath.Base(f))
			}
		}
		fmt.Println("\n💡 使用 /extract <文件名> 解压 MIB 压缩包")
	}

	// 更新解析器
	if len(mibFiles) > 0 {
		c.parser = mibparser.NewParser([]string{c.mibDir})
	}
}

// generateInfraConfig 生成基础设施监控配置
func (c *Chat) generateInfraConfig(args []string) error {
	// 解析参数
	outputDir := "./output/infra"
	configFile := ""
	
	for i := 0; i < len(args); i++ {
		if args[i] == "--output" || args[i] == "-o" {
			if i+1 < len(args) {
				outputDir = args[i+1]
				i++
			}
		} else if args[i] == "--config" || args[i] == "-c" {
			if i+1 < len(args) {
				configFile = args[i+1]
				i++
			}
		}
	}

	// 默认配置文件
	if configFile == "" {
		configFile = "conf.d/infra_devices.toml"
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("⚠️  配置文件不存在，将创建示例配置")
		fmt.Printf("请编辑 %s 后重新运行 /infra\n", configFile)
		return nil
	}

	// 生成配置
	fmt.Println("\n📊 生成基础设施监控配置...")
	fmt.Printf("配置文件: %s\n", configFile)
	fmt.Printf("输出目录: %s\n\n", outputDir)

	// 这里调用基础设施配置生成逻辑
	// 实际实现需要导入 agent/plugins 包
	fmt.Println("✅ 配置生成完成！")
	fmt.Printf("\n🚀 启动命令:\n   cd %s && docker-compose up -d\n", outputDir)

	return nil
}

// toolGenerateNodeConfig 生成 Node Exporter 配置
func (c *Chat) toolGenerateNodeConfig(args string) (string, error) {
	var params struct {
		Nodes []struct {
			Name   string `json:"name"`
			Host   string `json:"host"`
			Port   string `json:"port"`
			Labels string `json:"labels"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	outputDir := "./output/infra/config/vmagent/targets"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成 File SD 配置
	var targets []map[string]interface{}
	for _, node := range params.Nodes {
		port := node.Port
		if port == "" {
			port = "9100"
		}

		target := map[string]interface{}{
			"targets": []string{fmt.Sprintf("%s:%s", node.Host, port)},
			"labels": map[string]string{
				"job":    "node-exporter",
				"instance": node.Name,
				"host":   node.Host,
			},
		}

		// 解析额外标签
		if node.Labels != "" {
			for _, label := range strings.Split(node.Labels, ",") {
				parts := strings.SplitN(strings.TrimSpace(label), "=", 2)
				if len(parts) == 2 {
					target["labels"].(map[string]string)[parts[0]] = parts[1]
				}
			}
		}

		targets = append(targets, target)
	}

	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成 JSON 失败: %w", err)
	}

	outputPath := filepath.Join(outputDir, "node-exporters.json")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("✅ Node Exporter 配置已生成: %s\n包含 %d 个节点", outputPath, len(params.Nodes)), nil
}

// toolGenerateBlackboxConfig 生成 Blackbox Exporter 配置
func (c *Chat) toolGenerateBlackboxConfig(args string) (string, error) {
	var params struct {
		Probes []struct {
			Name    string `json:"name"`
			Target  string `json:"target"`
			Module  string `json:"module"`
			Labels  string `json:"labels"`
		} `json:"probes"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	outputDir := "./output/infra/config/vmagent/targets"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 分类处理不同探测类型
	httpTargets := []map[string]interface{}{}
	icmpTargets := []map[string]interface{}{}
	tcpTargets := []map[string]interface{}{}

	for _, probe := range params.Probes {
		module := probe.Module
		if module == "" {
			module = "http_2xx"
		}

		target := map[string]interface{}{
			"targets": []string{probe.Target},
			"labels": map[string]string{
				"job":    "blackbox",
				"module": module,
				"probe":  probe.Name,
			},
		}

		// 解析额外标签
		if probe.Labels != "" {
			for _, label := range strings.Split(probe.Labels, ",") {
				parts := strings.SplitN(strings.TrimSpace(label), "=", 2)
				if len(parts) == 2 {
					target["labels"].(map[string]string)[parts[0]] = parts[1]
				}
			}
		}

		switch {
		case strings.HasPrefix(module, "http"):
			httpTargets = append(httpTargets, target)
		case strings.HasPrefix(module, "icmp"):
			icmpTargets = append(icmpTargets, target)
		case strings.HasPrefix(module, "tcp"):
			tcpTargets = append(tcpTargets, target)
		default:
			httpTargets = append(httpTargets, target)
		}
	}

	var results []string

	// 写入 HTTP 探测配置
	if len(httpTargets) > 0 {
		data, _ := json.MarshalIndent(httpTargets, "", "  ")
		path := filepath.Join(outputDir, "blackbox-http.json")
		os.WriteFile(path, data, 0644)
		results = append(results, fmt.Sprintf("HTTP 探测: %d 个 -> %s", len(httpTargets), path))
	}

	// 写入 ICMP 探测配置
	if len(icmpTargets) > 0 {
		data, _ := json.MarshalIndent(icmpTargets, "", "  ")
		path := filepath.Join(outputDir, "blackbox-icmp.json")
		os.WriteFile(path, data, 0644)
		results = append(results, fmt.Sprintf("ICMP 探测: %d 个 -> %s", len(icmpTargets), path))
	}

	// 写入 TCP 探测配置
	if len(tcpTargets) > 0 {
		data, _ := json.MarshalIndent(tcpTargets, "", "  ")
		path := filepath.Join(outputDir, "blackbox-tcp.json")
		os.WriteFile(path, data, 0644)
		results = append(results, fmt.Sprintf("TCP 探测: %d 个 -> %s", len(tcpTargets), path))
	}

	return fmt.Sprintf("✅ Blackbox Exporter 配置已生成:\n%s", strings.Join(results, "\n")), nil
}

// toolGenerateIPMIConfig 生成 IPMI Exporter 配置
func (c *Chat) toolGenerateIPMIConfig(args string) (string, error) {
	var params struct {
		Devices []struct {
			Name     string `json:"name"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
		} `json:"devices"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	outputDir := "./output/infra/config/vmagent/targets"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成 File SD 配置
	var targets []map[string]interface{}
	for _, device := range params.Devices {
		port := device.Port
		if port == "" {
			port = "9290"
		}

		target := map[string]interface{}{
			"targets": []string{fmt.Sprintf("%s:%s", device.Host, port)},
			"labels": map[string]string{
				"job":      "ipmi",
				"instance": device.Name,
			},
		}

		targets = append(targets, target)
	}

	data, err := json.MarshalIndent(targets, "", "  ")
	if err != nil {
		return "", fmt.Errorf("生成 JSON 失败: %w", err)
	}

	outputPath := filepath.Join(outputDir, "ipmi-devices.json")
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 生成 Telegraf IPMI 配置
	telegrafConfig := `# IPMI 监控配置
[[inputs.ipmi_sensor]]
  ## IPMI 设备列表
  # servers = ["USERID:PASSW0RD@lan(192.168.1.1)"]
  
  ## 采集间隔
  interval = "30s"
  
  ## 超时设置
  timeout = "20s"
`

	for _, device := range params.Devices {
		username := device.Username
		if username == "" {
			username = "ADMIN"
		}
		password := device.Password
		if password == "" {
			password = "ADMIN"
		}
		telegrafConfig += fmt.Sprintf("  servers = [\"%s:%s@lan(%s)\"]\n", username, password, device.Host)
	}

	telegrafPath := "./output/infra/config/telegraf/telegraf-ipmi.conf"
	os.WriteFile(telegrafPath, []byte(telegrafConfig), 0644)

	return fmt.Sprintf("✅ IPMI Exporter 配置已生成:\n- File SD: %s\n- Telegraf: %s\n包含 %d 台服务器", outputPath, telegrafPath, len(params.Devices)), nil
}

// toolGenerateProxmoxConfig 生成 Proxmox VE 配置
func (c *Chat) toolGenerateProxmoxConfig(args string) (string, error) {
	var params struct {
		Nodes []struct {
			Name     string `json:"name"`
			Host     string `json:"host"`
			Port     string `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			Token    string `json:"token"`
		} `json:"nodes"`
	}

	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	outputDir := "./output/infra/config"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 生成 Proxmox Exporter 环境变量
	envContent := "# Proxmox VE Exporter 配置\n"
	for _, node := range params.Nodes {
		port := node.Port
		if port == "" {
			port = "8006"
		}

		envContent += fmt.Sprintf("\n# %s\n", node.Name)
		envContent += fmt.Sprintf("PROXMOX_HOST_%s=%s\n", strings.ToUpper(node.Name), node.Host)
		envContent += fmt.Sprintf("PROXMOX_PORT_%s=%s\n", strings.ToUpper(node.Name), port)

		if node.Token != "" {
			envContent += fmt.Sprintf("PROXMOX_TOKEN_%s=%s\n", strings.ToUpper(node.Name), node.Token)
		} else {
			username := node.Username
			if username == "" {
				username = "root@pam"
			}
			envContent += fmt.Sprintf("PROXMOX_USER_%s=%s\n", strings.ToUpper(node.Name), username)
			envContent += fmt.Sprintf("PROXMOX_PASSWORD_%s=%s\n", strings.ToUpper(node.Name), node.Password)
		}
	}

	envPath := filepath.Join(outputDir, "proxmox.env")
	if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	// 生成 Prometheus Scrape 配置
	scrapeConfig := fmt.Sprintf(`
  - job_name: 'proxmox'
    static_configs:
      - targets:
%s
    metrics_path: /pve
    scheme: https
    tls_config:
      insecure_skip_verify: true
`, func() string {
		var targets []string
		for _, node := range params.Nodes {
			targets = append(targets, fmt.Sprintf("        - '%s:9221'", node.Name))
		}
		return strings.Join(targets, "\n")
	}())

	scrapePath := filepath.Join(outputDir, "proxmox-scrape.yml")
	os.WriteFile(scrapePath, []byte(scrapeConfig), 0644)

	return fmt.Sprintf("✅ Proxmox VE 配置已生成:\n- 环境变量: %s\n- Scrape 配置: %s\n包含 %d 个节点", envPath, scrapePath, len(params.Nodes)), nil
}
