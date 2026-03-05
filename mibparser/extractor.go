package mibparser

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extractor MIB 压缩包解压器
type Extractor struct {
	mibDir string
}

// NewExtractor 创建解压器
func NewExtractor(mibDir string) *Extractor {
	return &Extractor{mibDir: mibDir}
}

// SupportedFormats 支持的压缩格式
var SupportedFormats = []string{".zip", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar", ".gz"}

// IsArchiveFile 判断是否为支持的压缩文件
func IsArchiveFile(filename string) bool {
	ext := strings.ToLower(filename)
	for _, format := range SupportedFormats {
		if strings.HasSuffix(ext, format) {
			return true
		}
	}
	return false
}

// Extract 解压 MIB 压缩包
func (e *Extractor) Extract(archivePath string) ([]string, error) {
	// 确保目标目录存在
	if err := os.MkdirAll(e.mibDir, 0755); err != nil {
		return nil, fmt.Errorf("创建 MIB 目录失败: %w", err)
	}

	ext := strings.ToLower(archivePath)
	var extractedFiles []string
	var err error

	switch {
	case strings.HasSuffix(ext, ".zip"):
		extractedFiles, err = e.extractZip(archivePath)
	case strings.HasSuffix(ext, ".tar.gz") || strings.HasSuffix(ext, ".tgz"):
		extractedFiles, err = e.extractTarGz(archivePath)
	case strings.HasSuffix(ext, ".tar.bz2") || strings.HasSuffix(ext, ".tbz2"):
		extractedFiles, err = e.extractTarBz2(archivePath)
	case strings.HasSuffix(ext, ".tar"):
		extractedFiles, err = e.extractTar(archivePath)
	case strings.HasSuffix(ext, ".gz"):
		extractedFiles, err = e.extractGz(archivePath)
	default:
		return nil, fmt.Errorf("不支持的压缩格式: %s", filepath.Base(archivePath))
	}

	if err != nil {
		return nil, err
	}

	// 过滤只返回 MIB 文件
	var mibFiles []string
	for _, f := range extractedFiles {
		if e.isMIBFile(f) {
			mibFiles = append(mibFiles, f)
		}
	}

	return mibFiles, nil
}

// extractZip 解压 ZIP 文件
func (e *Extractor) extractZip(archivePath string) ([]string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开 ZIP 文件失败: %w", err)
	}
	defer reader.Close()

	var extractedFiles []string
	baseName := strings.TrimSuffix(filepath.Base(archivePath), ".zip")

	for _, file := range reader.File {
		// 跳过目录和非 MIB 文件
		if file.FileInfo().IsDir() {
			continue
		}

		// 只解压 MIB 文件
		fileName := filepath.Base(file.Name)
		if !e.isMIBFile(fileName) && !e.isMIBFileInContent(file) {
			continue
		}

		// 构建目标路径
		destPath := filepath.Join(e.mibDir, baseName, filepath.Base(file.Name))
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// 解压文件
		if err := e.extractZipFile(file, destPath); err != nil {
			continue
		}

		extractedFiles = append(extractedFiles, destPath)
	}

	return extractedFiles, nil
}

// extractZipFile 解压单个 ZIP 文件
func (e *Extractor) extractZipFile(file *zip.File, destPath string) error {
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

// extractTarGz 解压 tar.gz 文件
func (e *Extractor) extractTarGz(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开 tar.gz 文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建 gzip 读取器失败: %w", err)
	}
	defer gzReader.Close()

	return e.extractTarReader(gzReader, archivePath)
}

// extractTarBz2 解压 tar.bz2 文件
func (e *Extractor) extractTarBz2(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开 tar.bz2 文件失败: %w", err)
	}
	defer file.Close()

	bz2Reader := bzip2.NewReader(file)
	return e.extractTarReader(bz2Reader, archivePath)
}

// extractTar 解压 tar 文件
func (e *Extractor) extractTar(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开 tar 文件失败: %w", err)
	}
	defer file.Close()

	return e.extractTarReader(file, archivePath)
}

// extractTarReader 通用的 tar 解压逻辑
func (e *Extractor) extractTarReader(reader io.Reader, archivePath string) ([]string, error) {
	tarReader := tar.NewReader(reader)
	var extractedFiles []string
	baseName := strings.TrimSuffix(filepath.Base(archivePath), filepath.Ext(archivePath))
	baseName = strings.TrimSuffix(baseName, ".tar")

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		// 跳过目录
		if header.Typeflag == tar.TypeDir {
			continue
		}

		// 只解压 MIB 文件
		fileName := filepath.Base(header.Name)
		if !e.isMIBFile(fileName) {
			continue
		}

		// 构建目标路径
		destPath := filepath.Join(e.mibDir, baseName, fileName)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			continue
		}

		// 解压文件
		if err := e.extractTarFile(tarReader, destPath); err != nil {
			continue
		}

		extractedFiles = append(extractedFiles, destPath)
	}

	return extractedFiles, nil
}

// extractTarFile 解压单个 tar 文件
func (e *Extractor) extractTarFile(reader *tar.Reader, destPath string) error {
	writer, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

// extractGz 解压单个 gz 文件
func (e *Extractor) extractGz(archivePath string) ([]string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, fmt.Errorf("打开 gz 文件失败: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("创建 gzip 读取器失败: %w", err)
	}
	defer gzReader.Close()

	// 构建目标文件名
	baseName := strings.TrimSuffix(filepath.Base(archivePath), ".gz")
	destPath := filepath.Join(e.mibDir, baseName)

	writer, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	if _, err := io.Copy(writer, gzReader); err != nil {
		return nil, err
	}

	return []string{destPath}, nil
}

// isMIBFile 判断是否为 MIB 文件
func (e *Extractor) isMIBFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	name := strings.ToLower(filename)

	// 检查扩展名
	if ext == ".mib" || ext == ".my" {
		return true
	}

	// 检查文件名包含 MIB 关键字
	if strings.Contains(name, "mib") {
		return true
	}

	// 检查无扩展名但可能是 MIB 文件
	if ext == "" && len(name) > 3 {
		// 常见的 MIB 文件命名模式
		patterns := []string{"-MIB", "-MIB2", "MIB-", "RFC", "CISCO-", "HUAWEI-", "H3C-"}
		for _, p := range patterns {
			if strings.Contains(strings.ToUpper(name), p) {
				return true
			}
		}
	}

	return false
}

// isMIBFileInContent 检查 ZIP 内文件内容是否为 MIB
func (e *Extractor) isMIBFileInContent(file *zip.File) bool {
	// 通过文件名判断
	return e.isMIBFile(file.Name)
}

// ExtractToDir 解压到指定目录
func ExtractToDir(archivePath, targetDir string) ([]string, error) {
	extractor := NewExtractor(targetDir)
	return extractor.Extract(archivePath)
}

// ScanForArchives 扫描目录中的压缩文件
func ScanForArchives(dir string) ([]string, error) {
	var archives []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if IsArchiveFile(info.Name()) {
			archives = append(archives, path)
		}
		return nil
	})

	return archives, err
}

// ScanForMIBFiles 扫描目录中的 MIB 文件
func ScanForMIBFiles(dir string) ([]string, error) {
	var mibFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mib" || ext == ".my" {
			mibFiles = append(mibFiles, path)
		}
		return nil
	})

	return mibFiles, err
}
