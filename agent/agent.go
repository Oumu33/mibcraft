package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Oumu33/mibcraft/config"
	"github.com/sashabaranov/go-openai"
)

// Agent 监控代理
type Agent struct {
	config     *config.Config
	registry   *PluginRegistry
	notifier   *Notifier
	diagnoser  *Diagnoser
	aiClient   *openai.Client

	plugins    []Plugin
	eventChan  chan *Event
	stopChan   chan struct{}
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// NewAgent 创建新的 Agent
func NewAgent(cfg *config.Config) *Agent {
	agent := &Agent{
		config:    cfg,
		registry:  NewPluginRegistry(),
		eventChan: make(chan *Event, 100),
		stopChan:  make(chan struct{}),
	}

	// 初始化 AI 客户端
	if cfg.AI.Enabled && len(cfg.AI.ModelPriority) > 0 {
		modelName := cfg.AI.ModelPriority[0]
		if modelCfg, ok := cfg.AI.Models[modelName]; ok {
			aiConfig := openai.DefaultConfig(modelCfg.APIKey)
			aiConfig.BaseURL = modelCfg.BaseURL
			agent.aiClient = openai.NewClientWithConfig(aiConfig)
		}
	}

	// 初始化通知器
	agent.notifier = NewNotifier(cfg.Notify, agent.eventChan)

	// 初始化诊断器
	agent.diagnoser = NewDiagnoser(agent.aiClient)

	return agent
}

// RegisterPlugin 注册插件
func (a *Agent) RegisterPlugin(p Plugin) {
	a.registry.Register(p)
}

// Start 启动 Agent
func (a *Agent) Start(ctx context.Context) error {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// 启动通知器
	a.notifier.Start(ctx)

	// 启动所有插件
	for name, plugin := range a.registry.GetAll() {
		if err := plugin.Init(nil); err != nil {
			fmt.Printf("插件 %s 初始化失败: %v\n", name, err)
			continue
		}
		a.plugins = append(a.plugins, plugin)

		a.wg.Add(1)
		go a.runPlugin(ctx, plugin)
	}

	fmt.Printf("Agent 已启动，运行 %d 个插件\n", len(a.plugins))

	// 等待信号
	for {
		select {
		case <-ctx.Done():
			a.Stop()
			return ctx.Err()
		case sig := <-sigChan:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Println("\n收到退出信号...")
				a.Stop()
				return nil
			case syscall.SIGHUP:
				fmt.Println("收到 SIGHUP，重新加载配置...")
				// TODO: 实现配置热重载
			}
		case <-a.stopChan:
			return nil
		}
	}
}

// runPlugin 运行插件
func (a *Agent) runPlugin(ctx context.Context, p Plugin) {
	defer a.wg.Done()

	interval := 30 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopChan:
			return
		case <-ticker.C:
			result, err := p.Check(ctx)
			if err != nil {
				fmt.Printf("插件 %s 检查失败: %v\n", p.Name(), err)
				continue
			}

			// 如果检查失败，生成事件
			if !result.OK {
				event := &Event{
					ID:         generateEventID(),
					Timestamp:  time.Now(),
					PluginName: p.Name(),
					Severity:   SeverityHigh,
					Status:     StatusFiring,
					Summary:    result.Message,
					Metrics:    result.Metrics,
					Labels:     result.Labels,
				}
				a.eventChan <- event
			}
		}
	}
}

// Stop 停止 Agent
func (a *Agent) Stop() {
	close(a.stopChan)
	a.wg.Wait()

	// 关闭所有插件
	for _, p := range a.plugins {
		p.Close()
	}

	// 停止通知器
	a.notifier.Stop()
}

// GetPlugins 获取插件列表
func (a *Agent) GetPlugins() []string {
	return a.registry.List()
}

// GetPluginHealth 获取插件健康状态
func (a *Agent) GetPluginHealth(ctx context.Context) []HealthCheckResult {
	var results []HealthCheckResult

	for name, p := range a.registry.GetAll() {
		start := time.Now()
		_, err := p.Check(ctx)
		latency := time.Since(start)

		result := HealthCheckResult{
			Plugin:    name,
			Healthy:   err == nil,
			Timestamp: time.Now(),
			Latency:   latency,
		}
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Message = "OK"
		}
		results = append(results, result)
	}

	return results
}

// RunDiagnosis 运行诊断
func (a *Agent) RunDiagnosis(ctx context.Context, event *Event) (*DiagnosisResult, error) {
	return a.diagnoser.Diagnose(ctx, event)
}

// generateEventID 生成事件 ID
func generateEventID() string {
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}
