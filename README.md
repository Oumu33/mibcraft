# Mibcraft - SNMP 配置智能生成器

基于 MIB 文件的 SNMP 监控配置智能生成工具，支持自然语言对话、多种采集器配置生成、基础设施监控全覆盖。

## 功能特性

### 核心功能
- **MIB 文件解析** - 解析标准 MIB 文件，提取 OID 定义
- **自然语言对话** - AI 驱动的智能配置生成
- **多格式配置输出** - Categraf、SNMP Exporter、Telegraf
- **MIB 压缩包解压** - 支持 zip/tar.gz/tar.bz2/tar/gz

### 基础设施监控
- **网络设备**: 华为、华三、锐捷、Cisco、Juniper 等 12+ 厂商
- **物理服务器**: Dell iDRAC、HPE iLO、Lenovo XClarity、Supermicro
- **虚拟化平台**: VMware vSphere、Proxmox VE
- **服务探测**: HTTP/HTTPS、ICMP、TCP 端口

## 快速开始

### 安装

```bash
# 克隆仓库
git clone https://github.com/Oumu33/mibcraft.git
cd mibcraft

# 编译
go build -o mibcraft .

# 或使用构建脚本
./build.sh
```

### 启动交互模式

```bash
# 启动对话模式
./mibcraft

# 指定配置文件
./mibcraft -config conf.d/config.toml
```

### 命令行模式

```bash
# 生成 Categraf 配置
./mibcraft -mode cli -gen categraf -mib mibs/IF-MIB.mib -oids "1.3.6.1.2.1.2" -output ./output

# 生成基础设施监控配置
./mibcraft --infra --infra-config conf.d/infra_devices.toml --output ./output/infra
```

## 对话模式命令

启动后进入交互式对话模式：

```
╔════════════════════════════════════════════════════════════╗
║            MIB-Agent - SNMP 配置生成助手                    ║
╚════════════════════════════════════════════════════════════╝

📁 当前 MIB 目录: ./mibs
📋 已发现 1 个 MIB 文件

>>> 
```

### 命令列表

| 命令 | 说明 | 示例 |
|------|------|------|
| `/help` | 显示帮助信息 | `/help` |
| `/load <file>` | 加载 MIB 文件 | `/load mibs/CISCO-IF-MIB.mib` |
| `/extract <archive>` | 解压 MIB 压缩包 | `/extract vendor-mibs.zip` |
| `/mibdir [path]` | 设置/查看 MIB 目录 | `/mibdir /path/to/mibs` |
| `/scan` | 扫描 MIB 目录 | `/scan` |
| `/list` | 列出已加载的 MIB | `/list` |
| `/search <name>` | 搜索 MIB 对象 | `/search interface` |
| `/show <oid>` | 显示 OID 详情 | `/show 1.3.6.1.2.1.1.1` |
| `/gen [oids]` | 生成配置文件 | `/gen 1.3.6.1.2.1.2` |
| `/infra` | 生成基础设施配置 | `/infra` |
| `/clear` | 清除对话历史 | `/clear` |
| `/exit` | 退出程序 | `/exit` |

### 自然语言对话

配置 AI 模型后，可直接用自然语言描述需求：

```
>>> 我想监控华为交换机的 CPU 使用率

[AI 自动搜索相关 OID 并生成配置]

>>> 帮我生成监控 Dell 服务器温度的配置

[AI 生成 Redfish 配置]
```

### AI 工具调用示例

Chat 模式内置 10 个 AI 工具，支持自然语言调用：

#### 主机监控 (Node Exporter)

```
>>> 帮我监控 3 台 Linux 服务器
>>> 主机: web-01(192.168.1.10), web-02(192.168.1.11), db-01(192.168.1.20)

✅ Node Exporter 配置已生成: ./output/infra/config/vmagent/targets/node-exporters.json
包含 3 个节点
```

#### 服务探测 (Blackbox Exporter)

```
>>> 帮我探测几个网站的可用性
>>> 网站: https://www.example.com, https://api.example.com, https://grafana.example.com

✅ Blackbox Exporter 配置已生成:
HTTP 探测: 3 个 -> ./output/infra/config/vmagent/targets/blackbox-http.json
```

```
>>> 帮我测试几台服务器的网络连通性
>>> IP: 192.168.1.1, 192.168.1.2, 192.168.1.3

✅ Blackbox Exporter 配置已生成:
ICMP 探测: 3 个 -> ./output/infra/config/vmagent/targets/blackbox-icmp.json
```

#### 硬件监控 (Redfish/iDRAC/iLO)

```
>>> 帮我监控 Dell R740 服务器的硬件状态
>>> 设备: dell-r740-01 (192.168.2.100), 用户名 root, 密码 calvin

✅ Redfish 配置已生成:
- Telegraf: ./output/infra/config/telegraf/telegraf-redfish.conf
- Exporter: ./output/infra/config/redfish-exporter/redfish.yml
```

#### IPMI 监控 (老旧服务器)

```
>>> 帮我通过 IPMI 监控几台老服务器
>>> 服务器: server-old-01(192.168.3.10), server-old-02(192.168.3.11)

✅ IPMI Exporter 配置已生成:
- File SD: ./output/infra/config/vmagent/targets/ipmi-devices.json
- Telegraf: ./output/infra/config/telegraf/telegraf-ipmi.conf
包含 2 台服务器
```

#### 网络设备监控 (SNMP)

```
>>> 帮我监控核心交换机
>>> 设备: core-sw-01 (192.168.1.100), 华为, community public

✅ 网络设备配置已生成: ./output/infra/config/vmagent/targets/snmp-devices.json
```

#### VMware vSphere 监控

```
>>> 帮我监控 vCenter
>>> 地址: https://vcenter.example.com/sdk, 用户名 administrator@vsphere.local

✅ VMware 配置已生成:
- Telegraf: ./output/infra/config/telegraf/telegraf.conf
- Targets: ./output/infra/config/vmagent/targets/vmware-vcenters.json
```

#### Proxmox VE 监控

```
>>> 帮我监控 Proxmox 集群
>>> 节点: pve-node1(192.168.10.100), pve-node2(192.168.10.101)

✅ Proxmox VE 配置已生成:
- 环境变量: ./output/infra/config/proxmox.env
- Scrape 配置: ./output/infra/config/proxmox-scrape.yml
包含 2 个节点
```

## MIB 管理

### 设置 MIB 目录

```bash
>>> /mibdir /opt/mibs
✅ MIB 目录已设置为: /opt/mibs
```

### 解压 MIB 压缩包

```bash
>>> /extract vendor-mibs.tar.gz
正在解压: vendor-mibs.tar.gz
目标目录: ./mibs
✅ 成功解压 15 个 MIB 文件:
  - CISCO-PROCESS-MIB.mib
  - CISCO-MEMORY-POOL-MIB.mib
  - HUAWEI-DEVICE-MIB.mib
  ...
```

### 扫描 MIB 目录

```bash
>>> /scan
扫描目录: ./mibs

📄 MIB 文件 (23 个):
  1. IF-MIB.mib
  2. CISCO-IF-MIB.mib
  3. HUAWEI-DEVICE-MIB.mib
  ...

📦 压缩包 (2 个):
  1. vendor-extra.tar.gz
  2. legacy-mibs.zip

💡 使用 /extract <文件名> 解压 MIB 压缩包
```

## 配置生成

### SNMP 配置生成

```bash
# 搜索相关 OID
>>> /search interface

找到 15 个匹配对象:
  ifNumber (.1.3.6.1.2.1.2.1) - 网络接口数量
  ifTable (.1.3.6.1.2.1.2.2) - 接口表
  ifEntry (.1.3.6.1.2.1.2.2.1) - 接口条目
  ...

# 生成配置
>>> /gen --format both .1.3.6.1.2.1.2

=== Categraf 配置 ===
interval = "30s"
targets = ["127.0.0.1:161"]
...

=== SNMP Exporter 配置 ===
module: default
metrics:
  - name: if_number
    oid: .1.3.6.1.2.1.2.1
    type: gauge
  ...
```

### 基础设施监控配置

```bash
>>> /infra

📊 生成基础设施监控配置...
配置文件: conf.d/infra_devices.toml
输出目录: ./output/infra

✅ 配置生成完成！

🚀 启动命令:
   cd ./output/infra && docker-compose up -d
```

## 基础设施配置文件

编辑 `conf.d/infra_devices.toml` 定义监控目标：

```toml
# 全局标签
[global_labels]
env = "production"
region = "cn-east-1"

# Linux 服务器
[[node_exporters]]
name = "web-server-01"
host = "192.168.1.10"
port = 9100

# SNMP 网络设备
[[snmp_devices]]
name = "core-switch-01"
host = "192.168.1.100"
vendor = "huawei"
device_tier = "core"
community = "public"

# Redfish 服务器
[[redfish_devices]]
name = "dell-r740-01"
host = "192.168.2.100"
username = "root"
password = "calvin"
vendor = "dell_idrac"

# IPMI 老服务器
[[ipmi_devices]]
name = "old-server-01"
host = "192.168.3.10"
username = "ADMIN"
password = "ADMIN"

# VMware vCenter
[[vmware_vcenters]]
name = "vcenter-main"
url = "https://vcenter.example.com/sdk"
username = "monitoring@vsphere.local"
password = "YourPassword"
```

## AI 工具列表

Chat 模式内置 10 个 AI 工具：

| 工具名称 | 功能 | 适用场景 |
|----------|------|----------|
| `generate_config` | 通用 SNMP 配置生成 | MIB 解析后生成采集配置 |
| `generate_node_config` | Node Exporter 配置 | Linux/Windows 主机监控 |
| `generate_blackbox_config` | Blackbox Exporter 配置 | HTTP/ICMP/TCP 服务探测 |
| `generate_hardware_config` | Redfish 硬件配置 | Dell/HPE/Lenovo 服务器 |
| `generate_ipmi_config` | IPMI 配置 | 老旧服务器硬件监控 |
| `generate_network_config` | 网络设备 SNMP 配置 | 交换机/路由器监控 |
| `generate_vmware_config` | VMware vSphere 配置 | vCenter/ESXi 虚拟化监控 |
| `generate_proxmox_config` | Proxmox VE 配置 | PVE 虚拟化监控 |
| `search_mib` | 搜索 MIB 对象 | 查找 OID 定义 |
| `explain_oid` | 解释 OID 含义 | 了解 OID 用途 |

## 支持的组件

### 采集器 (8种)
- Node Exporter - Linux 服务器监控
- SNMP Exporter - 网络设备监控
- Blackbox Exporter - 服务探测
- Redfish Exporter - 现代服务器硬件
- IPMI Exporter - 老旧服务器硬件
- Telegraf VMware - 虚拟化监控
- Telegraf Redfish - 硬件监控
- Telegraf IPMI - 老旧硬件监控

### 网络设备厂商 (12种)
| 厂商 | 协议支持 |
|------|----------|
| 华为 | NDP + LLDP |
| 华三 | LNP + LLDP |
| 锐捷 | LLDP |
| 迈普 | LLDP |
| 烽火 | LLDP |
| 中兴 | LLDP |
| 迪普 | LLDP |
| Cisco | CDP + LLDP |
| Arista | LLDP |
| Juniper | LLDP |
| HPE | LLDP |

### 物理服务器厂商 (5种)
| 厂商 | 监控方式 |
|------|----------|
| Dell | Redfish/iDRAC |
| HPE | Redfish/iLO |
| Lenovo | Redfish/XClarity |
| Supermicro | Redfish/IPMI |
| Fujitsu | Redfish |

### 虚拟化平台 (2种)
- VMware vSphere (vCenter/ESXi/VM)
- Proxmox VE (PVE/LXC/QEMU)

### 监控栈组件 (12种)
| 组件 | 端口 | 功能 |
|------|------|------|
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

## 配置文件说明

### 主配置文件 `conf.d/config.toml`

```toml
[global]
mib_dirs = ["./mibs", "/usr/share/snmp/mibs"]

[global.labels]
env = "production"
region = "cn-east-1"

[ai]
enabled = true
model_priority = ["deepseek", "gpt4o"]

[ai.models.deepseek]
base_url = "https://api.deepseek.com/v1"
api_key = "${DEEPSEEK_API_KEY}"
model = "deepseek-chat"

[generator]
output_dir = "./output"
default_community = "public"
default_version = 2
default_interval = "30s"

[agent]
enabled = true
check_interval = "30s"
```

## 生成的文件结构

运行 `--infra` 后生成：

```
output/infra/
├── docker-compose.yaml
├── .env
└── config/
    ├── vmagent/
    │   ├── prometheus.yml
    │   └── targets/
    │       ├── node-exporters.json
    │       ├── snmp-devices.json
    │       ├── blackbox-http.json
    │       └── blackbox-icmp.json
    ├── telegraf/
    │   ├── telegraf.conf
    │   ├── telegraf-redfish.conf
    │   └── telegraf-ipmi.conf
    ├── blackbox-exporter/
    │   └── blackbox.yml
    ├── redfish-exporter/
    │   └── redfish.yml
    ├── alertmanager/
    │   └── alertmanager.yml
    └── topology/
        └── devices.yml
```

## 开发

### 项目结构

```
mibcraft/
├── main.go              # 入口文件
├── chat/                # 对话模式
├── agent/               # Agent 模式
│   └── plugins/         # 插件系统
├── config/              # 配置管理
├── generator/           # 配置生成器
├── mibparser/           # MIB 解析器
│   └── extractor.go     # 压缩包解压
├── types/               # 类型定义
├── mcp/                 # MCP 协议支持
└── conf.d/              # 配置文件目录
```

### 添加新插件

```go
// agent/plugins/my_plugin.go
package plugins

type MyPlugin struct {
    config *MyPluginConfig
}

func (p *MyPlugin) Name() string {
    return "my_plugin"
}

func (p *MyPlugin) Check(ctx context.Context) (*agent.CheckResult, error) {
    // 实现检查逻辑
}
```

## License

MIT License
