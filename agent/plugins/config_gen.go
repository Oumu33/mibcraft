package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Oumu33/mibcraft/agent"
	"github.com/Oumu33/mibcraft/generator"
	"github.com/Oumu33/mibcraft/mibparser"
	"github.com/Oumu33/mibcraft/types"
)

// ConfigGenPlugin 配置生成插件
type ConfigGenPlugin struct {
	config   *ConfigGenPluginConfig
	parser   *mibparser.Parser
	gen      *generator.Generator
	lastGen  time.Time
}

// ConfigGenPluginConfig 配置生成插件配置
type ConfigGenPluginConfig struct {
	agent.BasePluginConfig
	MIBDirs      []string          `toml:"mib_dirs"`
	OutputDir    string            `toml:"output_dir"`
	Format       string            `toml:"format"` // categraf, snmp_exporter, telegraf, all
	WatchChanges bool              `toml:"watch_changes"`
	OIDs         []string          `toml:"oids"`
	Labels       map[string]string `toml:"labels"`
}

// NewConfigGenPlugin 创建配置生成插件
func NewConfigGenPlugin() *ConfigGenPlugin {
	return &ConfigGenPlugin{}
}

func (p *ConfigGenPlugin) Name() string {
	return "config_gen"
}

func (p *ConfigGenPlugin) Description() string {
	return "监控 MIB 文件变化并自动生成配置"
}

func (p *ConfigGenPlugin) Init(config agent.PluginConfig) error {
	if config != nil {
		if cfg, ok := config.(*ConfigGenPluginConfig); ok {
			p.config = cfg
		}
	}

	if p.config == nil {
		p.config = &ConfigGenPluginConfig{
			MIBDirs:      []string{"./mibs"},
			OutputDir:    "./output",
			Format:       "all",
			WatchChanges: true,
		}
	}

	p.parser = mibparser.NewParser(p.config.MIBDirs)
	p.gen = generator.NewGenerator(&generator.GeneratorConfig{
		DefaultCommunity: "public",
		DefaultVersion:   2,
		DefaultInterval:  "30s",
	})

	// 确保输出目录存在
	os.MkdirAll(p.config.OutputDir, 0755)

	return nil
}

func (p *ConfigGenPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
	result := &agent.CheckResult{
		OK:      true,
		Metrics: make(map[string]any),
		Labels:  p.config.Labels,
	}

	// 检查是否有 MIB 文件变化
	if p.config.WatchChanges {
		if changed, err := p.checkMIBChanges(); err != nil {
			result.OK = false
			result.Message = fmt.Sprintf("检查 MIB 变化失败: %v", err)
			return result, nil
		} else if changed {
			// 自动重新生成配置
			if err := p.regenerateConfig(); err != nil {
				result.OK = false
				result.Message = fmt.Sprintf("重新生成配置失败: %v", err)
				return result, nil
			}
			result.Message = "检测到 MIB 变化，已重新生成配置"
		} else {
			result.Message = "MIB 文件无变化"
		}
	}

	// 统计信息
	mibFiles := p.parser.ListMIBFiles()
	result.Metrics["mib_files_count"] = len(mibFiles)
	result.Metrics["config_last_gen"] = p.lastGen.Unix()

	return result, nil
}

// checkMIBChanges 检查 MIB 文件变化
func (p *ConfigGenPlugin) checkMIBChanges() (bool, error) {
	for _, dir := range p.config.MIBDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		var latestMod time.Time
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if info.ModTime().After(latestMod) {
				latestMod = info.ModTime()
			}
			return nil
		})

		if latestMod.After(p.lastGen) {
			return true, nil
		}
	}

	return false, nil
}

// regenerateConfig 重新生成配置
func (p *ConfigGenPlugin) regenerateConfig() error {
	if len(p.config.OIDs) == 0 {
		p.lastGen = time.Now()
		return nil
	}

	// 查找对象
	var objects []*types.MIBObject
	for _, oid := range p.config.OIDs {
		objs, err := p.parser.FindObjectsByOID(oid)
		if err != nil {
			continue
		}
		objects = append(objects, objs...)
	}

	if len(objects) == 0 {
		return nil
	}

	// 生成配置
	req := &types.ConfigRequest{
		TargetOIDs:  p.config.OIDs,
		Format:      p.config.Format,
		MetricNames: make(map[string]string),
		Labels:      p.config.Labels,
	}

	result, err := p.gen.GenerateBoth(objects, req)
	if err != nil {
		return err
	}

	// 写入文件
	if result.CategrafConfig != "" {
		os.WriteFile(filepath.Join(p.config.OutputDir, "snmp.toml"), []byte(result.CategrafConfig), 0644)
	}
	if result.SNMPExporterConfig != "" {
		os.WriteFile(filepath.Join(p.config.OutputDir, "snmp.yml"), []byte(result.SNMPExporterConfig), 0644)
	}
	if result.TelegrafConfig != "" {
		os.WriteFile(filepath.Join(p.config.OutputDir, "telegraf_snmp.conf"), []byte(result.TelegrafConfig), 0644)
	}

	p.lastGen = time.Now()
	return nil
}

func (p *ConfigGenPlugin) Close() error {
	return nil
}
