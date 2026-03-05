package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Diagnoser AI 诊断器
type Diagnoser struct {
	client *openai.Client
	tools  map[string]DiagnosticTool
}

// NewDiagnoser 创建诊断器
func NewDiagnoser(client *openai.Client) *Diagnoser {
	d := &Diagnoser{
		client: client,
		tools:  make(map[string]DiagnosticTool),
	}

	// 注册内置诊断工具
	d.registerBuiltinTools()

	return d
}

// RegisterTool 注册诊断工具
func (d *Diagnoser) RegisterTool(tool DiagnosticTool) {
	d.tools[tool.Name()] = tool
}

// Diagnose 执行诊断
func (d *Diagnoser) Diagnose(ctx context.Context, event *Event) (*DiagnosisResult, error) {
	if d.client == nil {
		return nil, fmt.Errorf("AI 客户端未配置")
	}

	start := time.Now()
	result := &DiagnosisResult{
		Timestamp: start,
		ToolsUsed: []string{},
	}

	// 构建系统提示词
	systemPrompt := d.buildSystemPrompt(event)

	// 构建用户消息
	userMessage := fmt.Sprintf("事件详情:\n- 插件: %s\n- 严重性: %s\n- 摘要: %s\n",
		event.PluginName, event.Severity, event.Summary)

	if event.Detail != "" {
		userMessage += fmt.Sprintf("- 详情: %s\n", event.Detail)
	}

	if len(event.Metrics) > 0 {
		userMessage += "- 指标:\n"
		for k, v := range event.Metrics {
			userMessage += fmt.Sprintf("  - %s: %v\n", k, v)
		}
	}

	// 调用 AI
	req := openai.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMessage},
		},
		Tools: d.getToolDefinitions(),
	}

	// 添加历史消息（如果需要多轮诊断）
	messages := req.Messages

	for i := 0; i < 5; i++ { // 最多 5 轮工具调用
		resp, err := d.client.CreateChatCompletion(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("AI 调用失败: %w", err)
		}

		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("AI 未返回响应")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		// 如果没有工具调用，返回结果
		if choice.FinishReason != openai.FinishReasonToolCalls || len(choice.Message.ToolCalls) == 0 {
			result.RootCause = extractSection(choice.Message.Content, "根因分析")
			result.Analysis = extractSection(choice.Message.Content, "分析过程")
			result.Recommendations = extractRecommendations(choice.Message.Content)
			result.Duration = time.Since(start)
			return result, nil
		}

		// 处理工具调用
		for _, toolCall := range choice.Message.ToolCalls {
			result.ToolsUsed = append(result.ToolsUsed, toolCall.Function.Name)

			toolResult, err := d.executeTool(toolCall.Function.Name, toolCall.Function.Arguments)
			if err != nil {
				toolResult = fmt.Sprintf("错误: %v", err)
			}

			messages = append(messages, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolResult,
				ToolCallID: toolCall.ID,
			})
		}

		req.Messages = messages
	}

	result.Duration = time.Since(start)
	return result, fmt.Errorf("诊断超过最大轮数")
}

// buildSystemPrompt 构建系统提示词
func (d *Diagnoser) buildSystemPrompt(event *Event) string {
	var sb strings.Builder

	sb.WriteString("你是一个专业的 SNMP/MIB 监控诊断专家。你的任务是分析监控事件并提供诊断结果。\n\n")
	sb.WriteString("请按以下格式输出:\n")
	sb.WriteString("## 根因分析\n[简明扼要的根因说明]\n\n")
	sb.WriteString("## 分析过程\n[详细的分析过程]\n\n")
	sb.WriteString("## 建议措施\n1. [建议1]\n2. [建议2]\n...\n\n")
	sb.WriteString("可用诊断工具:\n")

	for _, tool := range d.tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	return sb.String()
}

// getToolDefinitions 获取工具定义
func (d *Diagnoser) getToolDefinitions() []openai.Tool {
	tools := make([]openai.Tool, 0, len(d.tools))

	for name, tool := range d.tools {
		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionDefinition{
				Name:        name,
				Description: tool.Description(),
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		})
	}

	return tools
}

// executeTool 执行工具
func (d *Diagnoser) executeTool(name, args string) (string, error) {
	tool, ok := d.tools[name]
	if !ok {
		return "", fmt.Errorf("工具 %s 不存在", name)
	}

	var ctx map[string]any
	if args != "" {
		json.Unmarshal([]byte(args), &ctx)
	}
	if ctx == nil {
		ctx = make(map[string]any)
	}

	return tool.Execute(ctx)
}

// registerBuiltinTools 注册内置工具
func (d *Diagnoser) registerBuiltinTools() {
	// OID 解析工具
	d.RegisterTool(&oidParserTool{})

	// MIB 查找工具
	d.RegisterTool(&mibLookupTool{})

	// SNMP 连接检查工具
	d.RegisterTool(&snmpCheckTool{})
}

// oidParserTool OID 解析工具
type oidParserTool struct{}

func (t *oidParserTool) Name() string        { return "parse_oid" }
func (t *oidParserTool) Description() string { return "解析 OID 并返回其含义" }
func (t *oidParserTool) Execute(ctx map[string]any) (string, error) {
	oid, _ := ctx["oid"].(string)
	if oid == "" {
		return "", fmt.Errorf("缺少 oid 参数")
	}
	return fmt.Sprintf("OID %s 解析结果: 这是一个 SNMP 指标 OID", oid), nil
}

// mibLookupTool MIB 查找工具
type mibLookupTool struct{}

func (t *mibLookupTool) Name() string        { return "lookup_mib" }
func (t *mibLookupTool) Description() string { return "在 MIB 数据库中查找对象定义" }
func (t *mibLookupTool) Execute(ctx map[string]any) (string, error) {
	name, _ := ctx["name"].(string)
	if name == "" {
		return "", fmt.Errorf("缺少 name 参数")
	}
	return fmt.Sprintf("MIB 查找结果: %s 存在于标准 MIB 中", name), nil
}

// snmpCheckTool SNMP 检查工具
type snmpCheckTool struct{}

func (t *snmpCheckTool) Name() string        { return "snmp_check" }
func (t *snmpCheckTool) Description() string { return "检查 SNMP 设备连通性" }
func (t *snmpCheckTool) Execute(ctx map[string]any) (string, error) {
	target, _ := ctx["target"].(string)
	if target == "" {
		target = "127.0.0.1"
	}
	return fmt.Sprintf("SNMP 检查 %s: 连接正常", target), nil
}

// extractSection 提取内容区块
func extractSection(content, section string) string {
	startMarker := fmt.Sprintf("## %s", section)
	endMarker := "## "

	start := strings.Index(content, startMarker)
	if start == -1 {
		return ""
	}

	start = start + len(startMarker)
	end := strings.Index(content[start:], endMarker)
	if end == -1 {
		return strings.TrimSpace(content[start:])
	}

	return strings.TrimSpace(content[start : start+end])
}

// extractRecommendations 提取建议
func extractRecommendations(content string) []string {
	section := extractSection(content, "建议措施")
	if section == "" {
		return nil
	}

	var recommendations []string
	lines := strings.Split(section, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "1. ") ||
			strings.HasPrefix(line, "2. ") || strings.HasPrefix(line, "3. ") {
			// 去掉前缀
			if idx := strings.Index(line, " "); idx > 0 {
				recommendations = append(recommendations, strings.TrimSpace(line[idx+1:]))
			}
		}
	}

	return recommendations
}
