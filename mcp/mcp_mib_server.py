#!/usr/bin/env python3
"""
MCP MIB Server - 用于解析 MIB 文件的 MCP 服务器
通过 stdio 进行 JSON-RPC 通信
"""

import json
import sys
import os
import re
from typing import Any, Dict, List, Optional

class MIBParser:
    """简单的 MIB 文件解析器"""
    
    def __init__(self):
        self.mib_dirs: List[str] = []
        self.modules: Dict[str, Dict] = {}
    
    def add_mib_dir(self, path: str):
        """添加 MIB 目录"""
        if os.path.isdir(path) and path not in self.mib_dirs:
            self.mib_dirs.append(path)
    
    def parse_file(self, filepath: str) -> Dict[str, Any]:
        """解析 MIB 文件"""
        if not os.path.exists(filepath):
            return {"error": f"文件不存在: {filepath}"}
        
        try:
            with open(filepath, 'r') as f:
                content = f.read()
        except Exception as e:
            return {"error": f"读取文件失败: {e}"}
        
        module = {
            "file_path": filepath,
            "name": "",
            "objects": [],
            "imports": []
        }
        
        # 提取模块名
        name_match = re.search(r'^(\w+)\s+DEFINITIONS\s+::=\s+BEGIN', content, re.MULTILINE)
        if name_match:
            module["name"] = name_match.group(1)
        
        # 解析 OBJECT-TYPE 定义
        object_pattern = re.compile(
            r'(\w+)\s+OBJECT-TYPE\s+'
            r'SYNTAX\s+([^\n]+)\s+'
            r'(?:ACCESS|MAX-ACCESS)\s+(\w+)\s+'
            r'STATUS\s+\w+\s+'
            r'DESCRIPTION\s+"([^"]*)"\s+'
            r'.*?::=\s*\{\s*([^\}]+)\s*\}',
            re.DOTALL
        )
        
        for match in object_pattern.finditer(content):
            obj = {
                "name": match.group(1).strip(),
                "syntax": self._parse_syntax(match.group(2)),
                "type": self._get_type(match.group(2)),
                "access": match.group(3).lower(),
                "description": match.group(4).strip(),
                "oid": self._normalize_oid(match.group(5)),
                "enum_values": self._parse_enum(match.group(2))
            }
            module["objects"].append(obj)
        
        self.modules[filepath] = module
        return module
    
    def _parse_syntax(self, syntax: str) -> str:
        """解析语法类型"""
        return syntax.strip()
    
    def _get_type(self, syntax: str) -> str:
        """获取简化的类型"""
        syntax = syntax.upper()
        if 'COUNTER32' in syntax or 'COUNTER64' in syntax:
            return 'counter'
        if 'GAUGE' in syntax:
            return 'gauge'
        if 'TIMETICKS' in syntax:
            return 'timeticks'
        if 'OCTET STRING' in syntax:
            return 'string'
        if 'IPADDRESS' in syntax:
            return 'ipaddress'
        if 'INTEGER' in syntax:
            return 'integer'
        if 'BITS' in syntax:
            return 'bits'
        return 'unknown'
    
    def _normalize_oid(self, oid_str: str) -> str:
        """标准化 OID"""
        return oid_str.strip()
    
    def _parse_enum(self, syntax: str) -> Dict[str, int]:
        """解析枚举值"""
        enums = {}
        enum_pattern = re.compile(r'(\w+)\s*\((\d+)\)')
        for match in enum_pattern.finditer(syntax):
            enums[match.group(1)] = int(match.group(2))
        return enums
    
    def search_objects(self, query: str) -> List[Dict]:
        """搜索对象"""
        results = []
        query_lower = query.lower()
        
        for module in self.modules.values():
            for obj in module.get("objects", []):
                # 按名称或描述搜索
                if query_lower in obj["name"].lower() or query_lower in obj.get("description", "").lower():
                    obj_copy = obj.copy()
                    obj_copy["mib"] = module["name"]
                    results.append(obj_copy)
                # 按 OID 前缀搜索
                elif obj["oid"].startswith(query):
                    obj_copy = obj.copy()
                    obj_copy["mib"] = module["name"]
                    results.append(obj_copy)
        
        return results
    
    def list_files(self) -> List[str]:
        """列出所有 MIB 文件"""
        files = []
        for mib_dir in self.mib_dirs:
            if os.path.isdir(mib_dir):
                for root, _, filenames in os.walk(mib_dir):
                    for filename in filenames:
                        ext = os.path.splitext(filename)[1].lower()
                        if ext in ['.mib', '.my'] or ext == '':
                            files.append(os.path.join(root, filename))
        return files


class MCPServer:
    """MCP 服务器实现"""
    
    def __init__(self):
        self.parser = MIBParser()
        self.tools = [
            {
                "name": "load_mib",
                "description": "加载并解析 MIB 文件",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "file_path": {
                            "type": "string",
                            "description": "MIB 文件路径"
                        }
                    },
                    "required": ["file_path"]
                }
            },
            {
                "name": "search_mib_objects",
                "description": "搜索 MIB 对象，可以按名称、描述或 OID 前缀搜索",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "query": {
                            "type": "string",
                            "description": "搜索关键词"
                        }
                    },
                    "required": ["query"]
                }
            },
            {
                "name": "list_mib_files",
                "description": "列出所有可用的 MIB 文件",
                "inputSchema": {
                    "type": "object",
                    "properties": {}
                }
            },
            {
                "name": "add_mib_dir",
                "description": "添加 MIB 文件搜索目录",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "path": {
                            "type": "string",
                            "description": "目录路径"
                        }
                    },
                    "required": ["path"]
                }
            },
            {
                "name": "get_object_details",
                "description": "获取特定 OID 的详细信息",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "oid": {
                            "type": "string",
                            "description": "OID"
                        }
                    },
                    "required": ["oid"]
                }
            }
        ]
    
    def handle_request(self, request: Dict) -> Dict:
        """处理 JSON-RPC 请求"""
        method = request.get("method", "")
        params = request.get("params", {})
        request_id = request.get("id")
        
        try:
            if method == "initialize":
                return self._initialize(params, request_id)
            elif method == "tools/list":
                return self._list_tools(request_id)
            elif method == "tools/call":
                return self._call_tool(params, request_id)
            elif method == "resources/list":
                return self._list_resources(request_id)
            else:
                return self._error_response(request_id, -32601, f"方法不存在: {method}")
        except Exception as e:
            return self._error_response(request_id, -32603, str(e))
    
    def _initialize(self, params: Dict, request_id: Any) -> Dict:
        """初始化"""
        return self._success_response(request_id, {
            "protocolVersion": "2024-11-05",
            "serverInfo": {
                "name": "mcp-mib-server",
                "version": "1.0.0"
            },
            "capabilities": {
                "tools": {}
            }
        })
    
    def _list_tools(self, request_id: Any) -> Dict:
        """列出工具"""
        return self._success_response(request_id, {
            "tools": self.tools
        })
    
    def _call_tool(self, params: Dict, request_id: Any) -> Dict:
        """调用工具"""
        tool_name = params.get("name", "")
        arguments = params.get("arguments", {})
        
        if tool_name == "load_mib":
            result = self.parser.parse_file(arguments.get("file_path", ""))
        elif tool_name == "search_mib_objects":
            result = self.parser.search_objects(arguments.get("query", ""))
        elif tool_name == "list_mib_files":
            result = self.parser.list_files()
        elif tool_name == "add_mib_dir":
            self.parser.add_mib_dir(arguments.get("path", ""))
            result = {"status": "ok"}
        elif tool_name == "get_object_details":
            result = self.parser.search_objects(arguments.get("oid", ""))
        else:
            return self._error_response(request_id, -32602, f"未知工具: {tool_name}")
        
        return self._success_response(request_id, {
            "content": [{
                "type": "text",
                "text": json.dumps(result, ensure_ascii=False, indent=2)
            }]
        })
    
    def _list_resources(self, request_id: Any) -> Dict:
        """列出资源"""
        return self._success_response(request_id, {
            "resources": []
        })
    
    def _success_response(self, request_id: Any, result: Any) -> Dict:
        """成功响应"""
        return {
            "jsonrpc": "2.0",
            "id": request_id,
            "result": result
        }
    
    def _error_response(self, request_id: Any, code: int, message: str) -> Dict:
        """错误响应"""
        return {
            "jsonrpc": "2.0",
            "id": request_id,
            "error": {
                "code": code,
                "message": message
            }
        }
    
    def run(self):
        """运行服务器"""
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            
            try:
                request = json.loads(line)
                response = self.handle_request(request)
                print(json.dumps(response), flush=True)
            except json.JSONDecodeError as e:
                error_response = self._error_response(None, -32700, f"解析错误: {e}")
                print(json.dumps(error_response), flush=True)


if __name__ == "__main__":
    server = MCPServer()
    server.run()
