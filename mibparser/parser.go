package mibparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Oumu33/mibcraft/types"
)

// Parser MIB 文件解析器
type Parser struct {
	mibDirs []string
	cache   map[string]*types.MIBModule
}

// NewParser 创建新的 MIB 解析器
func NewParser(mibDirs []string) *Parser {
	return &Parser{
		mibDirs: mibDirs,
		cache:   make(map[string]*types.MIBModule),
	}
}

// ParseFile 解析单个 MIB 文件
func (p *Parser) ParseFile(path string) (*types.MIBModule, error) {
	// 检查缓存
	if module, ok := p.cache[path]; ok {
		return module, nil
	}
	
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 MIB 文件失败: %w", err)
	}
	
	module, err := p.parseContent(string(content), path)
	if err != nil {
		return nil, err
	}
	
	p.cache[path] = module
	return module, nil
}

// ParseAll 解析所有 MIB 文件
func (p *Parser) ParseAll() ([]*types.MIBModule, error) {
	var modules []*types.MIBModule
	
	for _, dir := range p.mibDirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.mib"))
		if err != nil {
			continue
		}
		
		// 同时支持其他常见扩展名
		mibFiles, _ := filepath.Glob(filepath.Join(dir, "*.MIB"))
		files = append(files, mibFiles...)
		
		myFiles, _ := filepath.Glob(filepath.Join(dir, "*.my"))
		files = append(files, myFiles...)
		
		for _, file := range files {
			module, err := p.ParseFile(file)
			if err != nil {
				continue
			}
			modules = append(modules, module)
		}
	}
	
	return modules, nil
}

// parseContent 解析 MIB 内容
func (p *Parser) parseContent(content, filePath string) (*types.MIBModule, error) {
	module := &types.MIBModule{
		FilePath: filePath,
		Objects:  make([]*types.MIBObject, 0),
	}
	
	// 提取模块名称
	moduleNameRegex := regexp.MustCompile(`(?i)^(\w+)\s+DEFINITIONS\s+::=\s+BEGIN`)
	if match := moduleNameRegex.FindStringSubmatch(content); len(match) > 1 {
		module.Name = match[1]
	}
	
	// 解析 OBJECT-TYPE 定义
	objects := p.parseObjectTypes(content)
	module.Objects = objects
	
	return module, nil
}

// parseObjectTypes 解析所有 OBJECT-TYPE 定义
func (p *Parser) parseObjectTypes(content string) []*types.MIBObject {
	var objects []*types.MIBObject
	
	// 匹配 OBJECT-TYPE 定义块
	// 格式: name OBJECT-TYPE
	//         SYNTAX type
	//         ACCESS access
	//         STATUS status
	//         DESCRIPTION "desc"
	//         ::= { parent oid }
	
	objectRegex := regexp.MustCompile(`(?s)(\w+)\s+OBJECT-TYPE\s+
\s+SYNTAX\s+([^\n]+)
\s+(?:ACCESS|MAX-ACCESS)\s+(\w+)
\s+STATUS\s+\w+
\s+DESCRIPTION\s+"([^"]*)"
(?:[^}]*?)
\s*::=\s*\{\s*([^}]+)\s*\}`)
	
	matches := objectRegex.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 6 {
			obj := &types.MIBObject{
				Name:        strings.TrimSpace(match[1]),
				Type:        p.parseSyntax(match[2]),
				Access:      strings.ToLower(strings.TrimSpace(match[3])),
				Description: strings.TrimSpace(match[4]),
				OID:         p.normalizeOID(match[5]),
				EnumValues:  make(map[string]int),
				Labels:      make(map[string]string),
			}
			
			// 解析枚举值
			if strings.Contains(obj.Type, "INTEGER") {
				obj.EnumValues = p.parseEnumValues(match[2])
			}
			
			objects = append(objects, obj)
		}
	}
	
	return objects
}

// parseSyntax 解析 SYNTAX 类型
func (p *Parser) parseSyntax(syntax string) string {
	syntax = strings.TrimSpace(syntax)
	
	// 处理复杂类型
	if strings.Contains(syntax, "INTEGER") {
		return "integer"
	}
	if strings.Contains(syntax, "Counter32") || strings.Contains(syntax, "Counter64") {
		return "counter"
	}
	if strings.Contains(syntax, "Gauge32") || strings.Contains(syntax, "Gauge64") {
		return "gauge"
	}
	if strings.Contains(syntax, "TimeTicks") {
		return "timeticks"
	}
	if strings.Contains(syntax, "OCTET STRING") {
		return "string"
	}
	if strings.Contains(syntax, "IpAddress") {
		return "ipaddress"
	}
	if strings.Contains(syntax, "BITS") {
		return "bits"
	}
	if strings.Contains(syntax, "TruthValue") {
		return "boolean"
	}
	
	return "unknown"
}

// parseEnumValues 解析枚举值
func (p *Parser) parseEnumValues(syntax string) map[string]int {
	enumValues := make(map[string]int)
	
	// 匹配 INTEGER { name(value), ... }
	enumRegex := regexp.MustCompile(`(\w+)\s*\((\d+)\)`)
	matches := enumRegex.FindAllStringSubmatch(syntax, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			var value int
			fmt.Sscanf(match[2], "%d", &value)
			enumValues[match[1]] = value
		}
	}
	
	return enumValues
}

// normalizeOID 标准化 OID
func (p *Parser) normalizeOID(oidStr string) string {
	oidStr = strings.TrimSpace(oidStr)
	
	// 如果已经是数字形式，直接返回
	if regexp.MustCompile(`^[\d.]+$`).MatchString(oidStr) {
		return oidStr
	}
	
	// 否则返回原始形式（需要通过其他 MIB 解析完整 OID）
	return oidStr
}

// FindObjectsByOID 根据 OID 前缀查找对象
func (p *Parser) FindObjectsByOID(oidPrefix string) ([]*types.MIBObject, error) {
	var results []*types.MIBObject
	
	modules, err := p.ParseAll()
	if err != nil {
		return nil, err
	}
	
	for _, module := range modules {
		for _, obj := range module.Objects {
			if strings.HasPrefix(obj.OID, oidPrefix) {
				obj.MIB = module.Name
				results = append(results, obj)
			}
		}
	}
	
	return results, nil
}

// FindObjectsByName 根据名称查找对象
func (p *Parser) FindObjectsByName(name string) ([]*types.MIBObject, error) {
	var results []*types.MIBObject
	
	modules, err := p.ParseAll()
	if err != nil {
		return nil, err
	}
	
	name = strings.ToLower(name)
	
	for _, module := range modules {
		for _, obj := range module.Objects {
			if strings.Contains(strings.ToLower(obj.Name), name) ||
				strings.Contains(strings.ToLower(obj.Description), name) {
				obj.MIB = module.Name
				results = append(results, obj)
			}
		}
	}
	
	return results, nil
}

// SearchObjects 搜索对象（支持 OID、名称、描述）
func (p *Parser) SearchObjects(query string) ([]*types.MIBObject, error) {
	// 判断是否为 OID 查询
	if regexp.MustCompile(`^[\d.]+$`).MatchString(query) {
		return p.FindObjectsByOID(query)
	}
	
	return p.FindObjectsByName(query)
}

// GetObjectTree 获取对象树结构
func (p *Parser) GetObjectTree(oidPrefix string) (map[string]*types.MIBObject, error) {
	tree := make(map[string]*types.MIBObject)
	
	objects, err := p.FindObjectsByOID(oidPrefix)
	if err != nil {
		return nil, err
	}
	
	for _, obj := range objects {
		tree[obj.OID] = obj
	}
	
	return tree, nil
}

// LoadMIBFile 从指定路径加载 MIB 文件
func (p *Parser) LoadMIBFile(path string) (*types.MIBModule, error) {
	return p.ParseFile(path)
}

// ListMIBFiles 列出所有可用的 MIB 文件
func (p *Parser) ListMIBFiles() []string {
	var files []string
	
	for _, dir := range p.mibDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".mib" || ext == ".my" || ext == "" {
					files = append(files, path)
				}
			}
			return nil
		})
	}
	
	return files
}

// ParseFromFile 从文件句柄解析 MIB
func (p *Parser) ParseFromFile(file *os.File) (*types.MIBModule, error) {
	var content strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return p.parseContent(content.String(), file.Name())
}
