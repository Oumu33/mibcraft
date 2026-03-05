package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Oumu33/mibcraft/config"
	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/mcp"
	"github.com/Oumu33/mibcraft/mibparser"
	"github.com/Oumu33/mibcraft/types"
	"github.com/sashabaranov/go-openai"
)

// Chat 对话管理器
type Chat struct {
	config     *config.Config
	parser     *mibparser.Parser
	generator  *generator.Generator
	mcpManager *mcp.Manager
	aiClient   *openai.Client
	history    []openai.ChatCompletionMessage
}

// NewChat 创建新的对话管理器
func NewChat(cfg *config.Config) *Chat {
	parser := mibparser.NewParser(cfg.Global.MIBDirs)
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
		mcpManager: mcp.NewManager(),
		aiClient:   aiClient,
		history:    make([]openai.ChatCompletionMessage, 0),
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
	fmt.Println("║    /help          - 显示帮助信息                             ║")
	fmt.Println("║    /load <file>   - 加载 MIB 文件                            ║")
	fmt.Println("║    /list          - 列出已加载的 MIB 文件                    ║")
	fmt.Println("║    /search <name> - 搜索 MIB 对象                            ║")
	fmt.Println("║    /show <oid>    - 显示 OID 详细信息                        ║")
	fmt.Println("║    /gen           - 生成配置文件                             ║")
	fmt.Println("║    /clear         - 清除对话历史                             ║")
	fmt.Println("║    /exit          - 退出程序                                 ║")
	fmt.Println("║                                                              ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()
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
	
	sb.WriteString("你是一个 SNMP 配置生成助手。你的任务是帮助用户解析 MIB 文件并生成配置。\n\n")
	sb.WriteString("你可以:\n")
	sb.WriteString("1. 搜索 MIB 对象\n")
	sb.WriteString("2. 生成 Categraf TOML 配置\n")
	sb.WriteString("3. 生成 SNMP Exporter YAML 配置\n")
	sb.WriteString("4. 解释 OID 含义\n\n")
	
	// 添加可用工具说明
	sb.WriteString("可用工具:\n")
	sb.WriteString("- search_mib: 搜索 MIB 对象\n")
	sb.WriteString("- generate_config: 生成配置文件\n")
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
				Description: "生成 SNMP 采集配置文件",
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
		case "explain_oid":
			result, err = c.toolExplainOID(toolCall.Function.Arguments)
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
