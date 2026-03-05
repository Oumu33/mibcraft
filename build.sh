#!/bin/bash

# MIB-Agent 构建脚本

VERSION=${VERSION:-"dev"}
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.buildDate=${BUILD_DATE}"

echo "构建 mibcraft ${VERSION}..."

# 创建输出目录
mkdir -p bin

# 构建
go build -ldflags "${LDFLAGS}" -o bin/mibcraft .

if [ $? -eq 0 ]; then
    echo "构建成功: bin/mibcraft"
    echo ""
    echo "使用方法:"
    echo "  ./bin/mibcraft                    # 启动交互式对话"
    echo "  ./bin/mibcraft -h                 # 显示帮助"
    echo "  ./bin/mibcraft --version          # 显示版本"
    echo ""
    echo "  # 命令行生成模式:"
    echo "  ./bin/mibcraft -gen categraf -mib /path/to/file.mib -oids 1.3.6.1.2.1.2.2.1"
    echo "  ./bin/mibcraft -gen both -mib /path/to/file.mib -oids 1.3.6.1.2.1.2.2.1 -output ./output"
else
    echo "构建失败"
    exit 1
fi
