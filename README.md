<div align="center">

# 🔮 Mibcraft

**SNMP 配置智能生成器 | MIB-Driven Infrastructure Monitoring Config Generator**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-Oumu33/mibcraft-181717?style=flat&logo=github)](https://github.com/Oumu33/mibcraft)

*基于 MIB 文件的智能监控配置生成工具，支持自然语言对话、多采集器配置输出、基础设施监控全覆盖*

[🚀 快速开始](#-快速开始) · [📖 详细教程](#-详细教程) · [🤖 AI 配置](#-ai-模型配置) · [💡 示例](#-使用示例)

</div>

---

## ✨ 核心特性

<table>
<tr>
<td width="50%">

### 🧠 AI 驱动
- 自然语言对话生成配置
- 支持 DeepSeek / GPT-4o / Claude 等
- 智能识别设备类型
- 自动推荐最佳 OID

</td>
<td width="50%">

### 🔧 多格式输出
- Categraf 配置
- SNMP Exporter 配置
- Telegraf 配置
- vmagent 配置

</td>
</tr>
<tr>
<td width="50%">

### 📦 MIB 管理
- 自动解析 MIB 文件
- 支持压缩包解压 (zip/tar.gz/tar.bz2)
- OID 搜索与解释
- 自定义 MIB 目录

</td>
<td width="50%">

### 🏗️ 基础设施全覆盖
- 🌐 网络设备 (12+ 厂商)
- 🖥️ 物理服务器 (5 大品牌)
- ☁️ 虚拟化平台 (VMware/PVE)
- 🔍 服务探测 (HTTP/ICMP/TCP)

</td>
</tr>
</table>

---

## 🎯 AI 工具列表

Chat 模式内置 **10 个 AI 工具**，一句话生成配置：

| 工具 | 功能 | 示例 |
|:-----|:-----|:-----|
| `generate_node_config` | Node Exporter 主机监控 | "帮我监控 3 台 Linux 服务器" |
| `generate_blackbox_config` | Blackbox 服务探测 | "帮我探测网站可用性" |
| `generate_hardware_config` | Redfish 硬件监控 | "帮我监控 Dell R740 温度" |
| `generate_ipmi_config` | IPMI 老旧服务器 | "帮我校验服务器风扇状态" |
| `generate_network_config` | 网络设备 SNMP | "帮我监控华为核心交换机" |
| `generate_vmware_config` | VMware vSphere | "帮我监控 vCenter 集群" |
| `generate_proxmox_config` | Proxmox VE | "帮我监控 PVE 节点" |
| `generate_config` | 通用 SNMP 配置 | "生成接口流量监控配置" |
| `search_mib` | 搜索 MIB 对象 | "搜索 CPU 相关的 OID" |
| `explain_oid` | 解释 OID 含义 | "解释 1.3.6.1.2.1.1.1" |

---

## 🚀 快速开始

### 📥 安装

```bash
# 克隆仓库
git clone https://github.com/Oumu33/mibcraft.git
cd mibcraft

# 编译
go build -o mibcraft .

# 或使用构建脚本
./build.sh
```

### 🎮 启动

```bash
# 启动交互模式 (推荐)
./mibcraft

# 指定配置文件
./mibcraft -config conf.d/config.toml
```

### ⚡ 命令行模式

```bash
# 生成基础设施监控配置
./mibcraft --infra --output ./output/infra

# 使用自定义设备配置
./mibcraft --infra --infra-config conf.d/infra_devices.toml
```

---

## 🤖 AI 模型配置

### 📝 配置文件位置

主配置文件：`conf.d/config.toml`

### 🔑 获取 API Key

| 模型提供商 | 获取地址 | 推荐指数 |
|:-----------|:---------|:---------|
| DeepSeek | https://platform.deepseek.com/api_keys | ⭐⭐⭐⭐⭐ 便宜好用 |
| OpenAI | https://platform.openai.com/api-keys | ⭐⭐⭐⭐ |
| Claude | https://console.anthropic.com/ | ⭐⭐⭐⭐ |
| Moonshot | https://platform.moonshot.cn/ | ⭐⭐⭐⭐ |
| Zhipu | https://open.bigmodel.cn/ | ⭐⭐⭐⭐ |

### ⚙️ 配置示例

```toml
# conf.d/config.toml

[global]
mib_dirs = ["./mibs", "/usr/share/snmp/mibs"]

[global.labels]
env = "production"
region = "cn-east-1"

# ============================================
# AI 模型配置 (必填)
# ============================================
[ai]
enabled = true
# 模型优先级，按顺序尝试
model_priority = ["deepseek", "gpt4o", "claude"]

# DeepSeek 配置 (推荐，便宜好用)
[ai.models.deepseek]
base_url = "https://api.deepseek.com/v1"
api_key = "${DEEPSEEK_API_KEY}"  # 从环境变量读取
model = "deepseek-chat"          # 或 "deepseek-coder"

# OpenAI GPT-4 配置
[ai.models.gpt4o]
base_url = "https://api.openai.com/v1"
api_key = "${OPENAI_API_KEY}"
model = "gpt-4o"

# Claude 配置
[ai.models.claude]
base_url = "https://api.anthropic.com/v1"
api_key = "${ANTHROPIC_API_KEY}"
model = "claude-3-5-sonnet-20241022"

# Moonshot (月之暗面)
[ai.models.moonshot]
base_url = "https://api.moonshot.cn/v1"
api_key = "${MOONSHOT_API_KEY}"
model = "moonshot-v1-8k"

# 智谱 GLM
[ai.models.zhipu]
base_url = "https://open.bigmodel.cn/api/paas/v4"
api_key = "${ZHIPU_API_KEY}"
model = "glm-4"

# ============================================
# 配置生成器设置
# ============================================
[generator]
output_dir = "./output"
default_community = "public"
default_version = 2
default_interval = "30s"

# ============================================
# Agent 模式设置
# ============================================
[agent]
enabled = true
check_interval = "30s"
```

### 🔐 设置环境变量

```bash
# 方式1: 临时设置 (当前会话)
export DEEPSEEK_API_KEY="sk-xxxxxxxxxxxxxxxx"

# 方式2: 写入 .bashrc (永久)
echo 'export DEEPSEEK_API_KEY="sk-xxxxxxxxxxxxxxxx"' >> ~/.bashrc
source ~/.bashrc

# 方式3: 写入 .env 文件
cat > .env << EOF
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxx
OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxx
ANTHROPIC_API_KEY=sk-ant-xxxxxxxxxxxxxxxx
EOF

# 然后在配置文件中使用 ${DEEPSEEK_API_KEY} 引用
```

### 🧪 验证配置

```bash
# 启动后检查 AI 是否可用
./mibcraft

# 看到以下信息表示配置成功
# 🤖 AI 模型: deepseek-chat (已连接)
```

---

## 📖 详细教程

### 🗨️ 对话模式命令

启动后进入交互式对话模式：

```
╔════════════════════════════════════════════════════════════╗
║            🔮 MIB-Agent - SNMP 配置生成助手                 ║
║                                                              ║
║  MIB 目录: ./mibs                                           ║
║  已发现: 23 个 MIB 文件                                      ║
║  AI 模型: deepseek-chat ✅                                   ║
╚════════════════════════════════════════════════════════════╝

>>> 
```

| 命令 | 说明 | 示例 |
|:-----|:-----|:-----|
| `/help` | 显示帮助信息 | `/help` |
| `/load <file>` | 加载 MIB 文件 | `/load mibs/CISCO-IF-MIB.mib` |
| `/extract <archive>` | 解压 MIB 压缩包 | `/extract vendor-mibs.zip` |
| `/mibdir [path]` | 设置/查看 MIB 目录 | `/mibdir /opt/mibs` |
| `/scan` | 扫描 MIB 目录 | `/scan` |
| `/list` | 列出已加载的 MIB | `/list` |
| `/search <name>` | 搜索 MIB 对象 | `/search cpu` |
| `/show <oid>` | 显示 OID 详情 | `/show 1.3.6.1.2.1.1.1` |
| `/gen [oids]` | 生成配置文件 | `/gen 1.3.6.1.2.1.2` |
| `/infra` | 生成基础设施配置 | `/infra` |
| `/clear` | 清除对话历史 | `/clear` |
| `/exit` | 退出程序 | `/exit` |

---

## 💡 使用示例

### 🖥️ 主机监控 (Node Exporter)

```
>>> 帮我监控 3 台 Linux 服务器

请提供服务器信息：
>>> web-01 (192.168.1.10), web-02 (192.168.1.11), db-01 (192.168.1.20)

✅ Node Exporter 配置已生成！
📁 文件: ./output/infra/config/vmagent/targets/node-exporters.json
📊 包含: 3 个监控节点

🚀 下一步:
   cd ./output/infra && docker-compose up -d
```

### 🌐 服务探测 (Blackbox Exporter)

```
>>> 帮我探测几个网站的可用性

请提供网站地址：
>>> https://www.example.com, https://api.example.com, https://grafana.example.com

✅ Blackbox Exporter 配置已生成！
📁 HTTP 探测: ./output/infra/config/vmagent/targets/blackbox-http.json
📊 包含: 3 个探测目标
```

```
>>> 帮我测试几台服务器的网络连通性

请提供 IP 地址：
>>> 192.168.1.1, 192.168.1.2, 192.168.1.3

✅ Blackbox Exporter 配置已生成！
📁 ICMP 探测: ./output/infra/config/vmagent/targets/blackbox-icmp.json
```

### 🖧 网络设备监控

```
>>> 帮我监控华为核心交换机

请提供设备信息：
>>> 名称: core-sw-01, IP: 192.168.1.100, Community: public

✅ 网络设备 SNMP 配置已生成！
📁 文件: ./output/infra/config/vmagent/targets/snmp-devices.json
📊 厂商: 华为 (支持 NDP + LLDP 拓扑发现)
```

### 🖥️ 硬件监控 (Redfish/iDRAC/iLO)

```
>>> 帮我监控 Dell R740 服务器的硬件状态

请提供服务器信息：
>>> 名称: dell-r740-01, IP: 192.168.2.100, 用户名: root, 密码: calvin

✅ Redfish 配置已生成！
📁 Telegraf: ./output/infra/config/telegraf/telegraf-redfish.conf
📁 Exporter: ./output/infra/config/redfish-exporter/redfish.yml
📊 监控项: 温度、风扇、电源、内存、存储...
```

### ⚡ IPMI 监控 (老旧服务器)

```
>>> 帮我通过 IPMI 监控几台老服务器

请提供服务器信息：
>>> server-old-01 (192.168.3.10), server-old-02 (192.168.3.11)

✅ IPMI Exporter 配置已生成！
📁 File SD: ./output/infra/config/vmagent/targets/ipmi-devices.json
📁 Telegraf: ./output/infra/config/telegraf/telegraf-ipmi.conf
📊 包含: 2 台服务器
```

### ☁️ VMware vSphere 监控

```
>>> 帮我监控 vCenter

请提供 vCenter 信息：
>>> 地址: https://vcenter.example.com/sdk, 用户名: administrator@vsphere.local

✅ VMware 配置已生成！
📁 Telegraf: ./output/infra/config/telegraf/telegraf.conf
📊 监控: vCenter/ESXi/虚拟机/数据存储
```

### 🐧 Proxmox VE 监控

```
>>> 帮我监控 Proxmox 集群

请提供节点信息：
>>> pve-node1 (192.168.10.100), pve-node2 (192.168.10.101)

✅ Proxmox VE 配置已生成！
📁 环境变量: ./output/infra/config/proxmox.env
📁 Scrape 配置: ./output/infra/config/proxmox-scrape.yml
📊 包含: 2 个节点, 监控 VM/LXC/QEMU
```

---

## 🏗️ 基础设施配置

### 📝 设备配置文件

编辑 `conf.d/infra_devices.toml` 定义监控目标：

```toml
# 全局标签
[global_labels]
env = "production"
region = "cn-east-1"

# ============================================
# Linux 服务器 (Node Exporter)
# ============================================
[[node_exporters]]
name = "web-server-01"
host = "192.168.1.10"
port = 9100
labels = { role = "web" }

[[node_exporters]]
name = "db-server-01"
host = "192.168.1.20"
port = 9100
labels = { role = "database" }

# ============================================
# 网络设备 (SNMP)
# ============================================
[[snmp_devices]]
name = "core-switch-01"
host = "192.168.1.100"
vendor = "huawei"        # huawei, h3c, cisco, ruijie, juniper...
device_tier = "core"     # core, aggregation, access
community = "public"

[[snmp_devices]]
name = "access-switch-01"
host = "192.168.1.101"
vendor = "h3c"
device_tier = "access"
community = "public"

# ============================================
# 物理服务器 (Redfish)
# ============================================
[[redfish_devices]]
name = "dell-r740-01"
host = "192.168.2.100"
username = "root"
password = "calvin"
vendor = "dell_idrac"    # dell_idrac, hpe_ilo, lenovo_xclarity, supermicro

[[redfish_devices]]
name = "hpe-dl380-01"
host = "192.168.2.200"
username = "Administrator"
password = "password"
vendor = "hpe_ilo"

# ============================================
# IPMI 设备 (老旧服务器)
# ============================================
[[ipmi_devices]]
name = "old-server-01"
host = "192.168.3.10"
username = "ADMIN"
password = "ADMIN"

# ============================================
# VMware vCenter
# ============================================
[[vmware_vcenters]]
name = "vcenter-main"
url = "https://vcenter.example.com/sdk"
username = "monitoring@vsphere.local"
password = "YourPassword"
```

### 🚀 生成并部署

```bash
# 生成配置
./mibcraft --infra

# 启动监控栈
cd output/infra && docker-compose up -d

# 查看服务状态
docker-compose ps
```

---

## 📦 支持的组件

### 🌐 网络设备厂商 (12+)

| 厂商 | 拓扑协议 | OID 支持 |
|:-----|:---------|:---------|
| 华为 Huawei | NDP + LLDP | ✅ 完整支持 |
| 华三 H3C | LNP + LLDP | ✅ 完整支持 |
| 锐捷 Ruijie | LLDP | ✅ 完整支持 |
| 迈普 Maipu | LLDP | ✅ 完整支持 |
| 烽火 FiberHome | LLDP | ✅ 完整支持 |
| 中兴 ZTE | LLDP | ✅ 完整支持 |
| 迪普 DPtech | LLDP | ✅ 完整支持 |
| Cisco | CDP + LLDP | ✅ 完整支持 |
| Arista | LLDP | ✅ 完整支持 |
| Juniper | LLDP | ✅ 完整支持 |
| HPE | LLDP | ✅ 完整支持 |

### 🖥️ 物理服务器厂商 (5+)

| 厂商 | 监控方式 | 监控项 |
|:-----|:---------|:-------|
| Dell | Redfish/iDRAC | 温度、风扇、电源、内存、存储、RAID |
| HPE | Redfish/iLO | 温度、风扇、电源、内存、存储 |
| Lenovo | Redfish/XClarity | 温度、风扇、电源、内存 |
| Supermicro | Redfish/IPMI | 温度、风扇、电源 |
| Fujitsu | Redfish | 温度、风扇、电源 |

### ☁️ 虚拟化平台 (2)

| 平台 | 监控范围 |
|:-----|:---------|
| VMware vSphere | vCenter、ESXi 主机、虚拟机、数据存储 |
| Proxmox VE | PVE 节点、LXC 容器、QEMU 虚拟机 |

### 📊 监控栈组件 (12+)

| 组件 | 端口 | 功能 |
|:-----|:-----|:-----|
| VictoriaMetrics | 8428 | 时序数据库 |
| vmagent | 8429 | 指标采集代理 |
| vmalert | 8880 | 告警规则引擎 |
| Grafana | 3000 | 可视化平台 |
| Alertmanager | 9093 | 告警管理 |
| Loki | 3100 | 日志聚合存储 |
| Promtail | 9080 | 日志采集 |
| Node Exporter | 9100 | 主机指标 |
| SNMP Exporter | 9116 | SNMP 指标 |
| Blackbox Exporter | 9115 | 服务探测 |
| Redfish Exporter | 9610 | 硬件指标 |
| IPMI Exporter | 9290 | IPMI 指标 |

---

## 📁 项目结构

```
mibcraft/
├── main.go              # 入口文件
├── chat/                # 对话模式
│   └── chat.go          # AI 工具定义
├── agent/               # Agent 模式
│   └── plugins/         # 插件系统
│       ├── infra_monitor.go    # 基础设施监控
│       ├── hardware_monitor.go # 硬件监控
│       ├── oid_monitor.go      # OID 监控
│       └── snmp.go             # SNMP 工具
├── config/              # 配置管理
├── generator/           # 配置生成器
├── mibparser/           # MIB 解析器
│   ├── parser.go        # MIB 解析
│   └── extractor.go     # 压缩包解压
├── types/               # 类型定义
│   ├── types.go         # 通用类型
│   └── infra_types.go   # 基础设施类型
├── mcp/                 # MCP 协议支持
├── conf.d/              # 配置文件目录
│   ├── config.toml      # 主配置
│   ├── infra_devices.toml # 设备配置示例
│   └── hardware_devices.toml # 硬件配置示例
└── mibs/                # MIB 文件目录
```

---

## 🔧 高级配置

### 🎯 自定义 MIB 目录

```bash
# 方式1: 命令行参数
./mibcraft --mib-dir /opt/mibs

# 方式2: 配置文件
[global]
mib_dirs = ["./mibs", "/usr/share/snmp/mibs", "/opt/vendor-mibs"]

# 方式3: 对话模式
>>> /mibdir /opt/mibs
```

### 📦 解压 MIB 压缩包

```bash
>>> /extract vendor-mibs.tar.gz

正在解压: vendor-mibs.tar.gz
目标目录: ./mibs
✅ 成功解压 15 个 MIB 文件
```

### 🔍 OID 搜索与解释

```bash
>>> /search cpu

找到 12 个匹配对象:
  1. hrProcessorLoad (.1.3.6.1.2.1.25.3.3.1.2) - CPU 负载
  2. cpmCPUTotal5min (.1.3.6.1.4.1.9.9.109.1.1.1.1.8) - Cisco CPU 5分钟平均
  ...

>>> /explain 1.3.6.1.2.1.25.3.3.1.2

OID: .1.3.6.1.2.1.25.3.3.1.2
名称: hrProcessorLoad
描述: CPU 处理器负载百分比
类型: Integer (0-100)
访问: Read-only
```

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

```bash
# Fork 后克隆
git clone https://github.com/YOUR_USERNAME/mibcraft.git

# 创建分支
git checkout -b feature/new-feature

# 提交更改
git commit -am "Add new feature"

# 推送分支
git push origin feature/new-feature

# 创建 Pull Request
```

---

## 📄 License

[MIT License](LICENSE)

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给一个 Star！⭐**

Made with ❤️ by [Oumu33](https://github.com/Oumu33)

</div>