package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Oumu33/mibcraft/config"
)

// Notifier 通知管理器
type Notifier struct {
	config   config.NotifyConfig
	eventChan <-chan *Event

	console  *consoleNotifier
	webapi   *webapiNotifier
	flashduty *flashdutyNotifier
	pagerduty *pagerdutyNotifier

	wg    sync.WaitGroup
	stop  chan struct{}
}

// NewNotifier 创建通知管理器
func NewNotifier(cfg config.NotifyConfig, eventChan <-chan *Event) *Notifier {
	n := &Notifier{
		config:    cfg,
		eventChan: eventChan,
		stop:      make(chan struct{}),
	}

	// 初始化各个通知器
	if cfg.Console != nil && cfg.Console.Enabled {
		n.console = &consoleNotifier{color: cfg.Console.Color}
	}
	if cfg.WebAPI != nil && cfg.WebAPI.Enabled {
		n.webapi = &webapiNotifier{config: cfg.WebAPI}
	}
	if cfg.Flashduty != nil && cfg.Flashduty.Enabled {
		n.flashduty = &flashdutyNotifier{config: cfg.Flashduty}
	}
	if cfg.PagerDuty != nil && cfg.PagerDuty.Enabled {
		n.pagerduty = &pagerdutyNotifier{config: cfg.PagerDuty}
	}

	// 默认启用控制台通知
	if n.console == nil {
		n.console = &consoleNotifier{color: true}
	}

	return n
}

// Start 启动通知器
func (n *Notifier) Start(ctx context.Context) {
	n.wg.Add(1)
	go n.run(ctx)
}

// Stop 停止通知器
func (n *Notifier) Stop() {
	close(n.stop)
	n.wg.Wait()
}

// run 运行通知循环
func (n *Notifier) run(ctx context.Context) {
	defer n.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.stop:
			return
		case event := <-n.eventChan:
			n.notify(event)
		}
	}
}

// notify 发送通知到所有通道
func (n *Notifier) notify(event *Event) {
	// 控制台通知
	if n.console != nil {
		n.console.Notify(event)
	}

	// WebAPI 通知
	if n.webapi != nil {
		go n.webapi.Notify(event)
	}

	// Flashduty 通知
	if n.flashduty != nil {
		go n.flashduty.Notify(event)
	}

	// PagerDuty 通知
	if n.pagerduty != nil {
		go n.pagerduty.Notify(event)
	}
}

// consoleNotifier 控制台通知器
type consoleNotifier struct {
	color bool
}

func (n *consoleNotifier) Notify(event *Event) {
	var icon string
	var colorCode string

	if n.color {
		switch event.Severity {
		case SeverityCritical:
			icon = "🔴"
			colorCode = "\033[31m"
		case SeverityHigh:
			icon = "🟠"
			colorCode = "\033[33m"
		case SeverityMedium:
			icon = "🟡"
			colorCode = "\033[33m"
		case SeverityLow:
			icon = "🔵"
			colorCode = "\033[34m"
		default:
			icon = "ℹ️"
			colorCode = "\033[36m"
		}
		resetCode := "\033[0m"

		fmt.Printf("%s%s [%s] %s - %s%s\n",
			colorCode, icon, event.PluginName, event.Timestamp.Format("15:04:05"), event.Summary, resetCode)

		if event.Detail != "" {
			fmt.Printf("  %s详情: %s%s\n", colorCode, event.Detail, resetCode)
		}

		if event.Diagnosis != nil {
			fmt.Printf("  %s诊断: %s%s\n", colorCode, event.Diagnosis.RootCause, resetCode)
		}
	} else {
		fmt.Printf("[%s] [%s] %s - %s\n",
			event.Severity, event.PluginName, event.Timestamp.Format("15:04:05"), event.Summary)
	}
}

// webapiNotifier WebAPI 通知器
type webapiNotifier struct {
	config *config.WebAPINotifyConfig
	client *http.Client
}

func (n *webapiNotifier) Notify(event *Event) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 10 * time.Second}
	}

	data, err := json.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebAPI 序列化事件失败: %v\n", err)
		return
	}

	method := n.config.Method
	if method == "" {
		method = "POST"
	}

	req, err := http.NewRequest(method, n.config.URL, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebAPI 创建请求失败: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.config.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebAPI 发送失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		fmt.Fprintf(os.Stderr, "WebAPI 返回错误状态: %d\n", resp.StatusCode)
	}
}

// flashdutyNotifier Flashduty 通知器
type flashdutyNotifier struct {
	config *config.FlashdutyNotifyConfig
	client *http.Client
}

func (n *flashdutyNotifier) Notify(event *Event) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 10 * time.Second}
	}

	// Flashduty API 格式
	payload := map[string]interface{}{
		"event_key":    event.ID,
		"event_status": string(event.Status),
		"title":        event.Summary,
		"description":  event.Detail,
		"labels":       event.Labels,
		"timestamp":    event.Timestamp.Unix(),
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST",
		"https://api.flashduty.com/v1/alert/push?integration_key="+n.config.IntegrationKey,
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Flashduty 发送失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

// pagerdutyNotifier PagerDuty 通知器
type pagerdutyNotifier struct {
	config *config.PagerDutyNotifyConfig
	client *http.Client
}

func (n *pagerdutyNotifier) Notify(event *Event) {
	if n.client == nil {
		n.client = &http.Client{Timeout: 10 * time.Second}
	}

	// PagerDuty Events API v2 格式
	action := "trigger"
	if event.Status == StatusResolved {
		action = "resolve"
	}

	payload := map[string]interface{}{
		"routing_key": n.config.RoutingKey,
		"event_action": action,
		"dedup_key":    event.ID,
		"payload": map[string]interface{}{
			"summary":   event.Summary,
			"severity":  string(event.Severity),
			"source":    event.PluginName,
			"timestamp": event.Timestamp.Format(time.RFC3339),
			"custom_details": map[string]interface{}{
				"detail": event.Detail,
				"labels": event.Labels,
			},
		},
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST",
		"https://events.pagerduty.com/v2/enqueue",
		bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "PagerDuty 发送失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
}
