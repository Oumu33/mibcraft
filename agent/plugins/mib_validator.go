package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Oumu33/mibcraft/agent"
)

// MIBValidatorPlugin MIB 文件验证插件
type MIBValidatorPlugin struct {
	config *MIBValidatorPluginConfig
}

// MIBValidatorPluginConfig MIB 验证插件配置
type MIBValidatorPluginConfig struct {
	agent.BasePluginConfig
	MIBDirs   []string `toml:"mib_dirs"`
	CheckSync bool     `toml:"check_sync"` // 检查文件同步状态
}

// NewMIBValidatorPlugin 创建 MIB 验证插件
func NewMIBValidatorPlugin() *MIBValidatorPlugin {
	return &MIBValidatorPlugin{}
}

func (p *MIBValidatorPlugin) Name() string {
	return "mib_validator"
}

func (p *MIBValidatorPlugin) Description() string {
	return "验证 MIB 文件格式和完整性"
}

func (p *MIBValidatorPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*MIBValidatorPluginConfig); ok {
			p.config = cfg
		}
	}

	if p.config == nil {
		p.config = &MIBValidatorPluginConfig{
			MIBDirs:   []string{"./mibs"},
			CheckSync: false,
		}
	}

	return nil
}

func (p *MIBValidatorPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Message: "MIB 验证通过",
	}

	totalFiles := 0
	validFiles := 0
	invalidFiles := []string{}

	for _, dir := range p.config.MIBDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}

			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".mib" && ext != ".my" && ext != "" {
				return nil
			}

			totalFiles++

			if valid, err := p.validateMIBFile(path); err != nil || !valid {
				invalidFiles = append(invalidFiles, path)
			} else {
				validFiles++
			}

			return nil
		})
	}

	result.Metrics["mib_total_files"] = totalFiles
	result.Metrics["mib_valid_files"] = validFiles
	result.Metrics["mib_invalid_files"] = len(invalidFiles)

	if len(invalidFiles) > 0 {
		result.OK = false
		result.Message = fmt.Sprintf("发现 %d 个无效 MIB 文件", len(invalidFiles))
	}

	return result, nil
}

// validateMIBFile 验证 MIB 文件
func (p *MIBValidatorPlugin) validateMIBFile(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	// 检查基本结构
	contentStr := string(content)

	// 必须包含 DEFINITIONS ::= BEGIN
	if !strings.Contains(contentStr, "DEFINITIONS") {
		return false, fmt.Errorf("缺少 DEFINITIONS 声明")
	}

	if !strings.Contains(contentStr, "BEGIN") {
		return false, fmt.Errorf("缺少 BEGIN 声明")
	}

	if !strings.Contains(contentStr, "END") {
		return false, fmt.Errorf("缺少 END 声明")
	}

	// 检查 OBJECT-TYPE 定义
	objectTypeRegex := regexp.MustCompile(`OBJECT-TYPE\s+SYNTAX`)
	if !objectTypeRegex.MatchString(contentStr) {
		return false, fmt.Errorf("未找到有效的 OBJECT-TYPE 定义")
	}

	return true, nil
}

func (p *MIBValidatorPlugin) Close() error {
	return nil
}
