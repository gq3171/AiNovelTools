package tools

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/AiNovelTools/internal/ai"
)

type Manager struct {
	tools map[string]Tool
}

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

type ToolResult struct {
	ToolName string
	Result   string
	Error    error
}

func NewManager() *Manager {
	m := &Manager{
		tools: make(map[string]Tool),
	}
	
	// 注册内置工具
	m.RegisterTool(&ReadFileTool{})
	m.RegisterTool(&WriteFileTool{})
	m.RegisterTool(&EditFileTool{})
	m.RegisterTool(&ListFilesTool{})
	m.RegisterTool(&CreateDirectoryTool{})
	m.RegisterTool(&DeleteFileTool{})
	m.RegisterTool(&RenameFileTool{})
	m.RegisterTool(&CopyFileTool{})
	m.RegisterTool(&MoveFileTool{})
	m.RegisterTool(&FileInfoTool{})
	m.RegisterTool(&ExecuteCommandTool{})
	m.RegisterTool(&SearchTool{})
	m.RegisterTool(&ReplaceTextTool{})
	
	return m
}

func (m *Manager) RegisterTool(tool Tool) {
	m.tools[tool.Name()] = tool
}

func (m *Manager) GetTool(name string) (Tool, bool) {
	tool, exists := m.tools[name]
	return tool, exists
}

func (m *Manager) ExecuteTools(ctx context.Context, toolCalls []ai.ToolCall) ([]ToolResult, error) {
	var results []ToolResult
	
	for _, call := range toolCalls {
		if call.Type != "function" {
			continue
		}
		
		funcName, ok := call.Function["name"].(string)
		if !ok {
			continue
		}
		
		params, ok := call.Function["arguments"].(map[string]interface{})
		if !ok {
			params = make(map[string]interface{})
		}
		
		tool, exists := m.tools[funcName]
		if !exists {
			results = append(results, ToolResult{
				ToolName: funcName,
				Error:    fmt.Errorf("unknown tool: %s", funcName),
			})
			continue
		}
		
		result, err := tool.Execute(ctx, params)
		results = append(results, ToolResult{
			ToolName: funcName,
			Result:   result,
			Error:    err,
		})
	}
	
	return results, nil
}

// ReadFileTool - 读取文件内容
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read the contents of a file" }

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path parameter is required")
	}
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	return string(content), nil
}

// WriteFileTool - 写入文件内容
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string { return "Write content to a file" }

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path parameter is required")
	}
	
	content, ok := params["content"].(string)
	if !ok {
		return "", fmt.Errorf("content parameter is required")
	}
	
	// 创建目录如果不存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return fmt.Sprintf("File written successfully: %s", filePath), nil
}

// ListFilesTool - 列出目录内容
type ListFilesTool struct{}

func (t *ListFilesTool) Name() string { return "list_files" }
func (t *ListFilesTool) Description() string { return "List files and directories in a path" }

func (t *ListFilesTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		path = "."
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	
	var result strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("%s/\n", entry.Name()))
		} else {
			result.WriteString(fmt.Sprintf("%s\n", entry.Name()))
		}
	}
	
	return result.String(), nil
}

// ExecuteCommandTool - 执行系统命令
type ExecuteCommandTool struct{}

func (t *ExecuteCommandTool) Name() string { return "execute_command" }
func (t *ExecuteCommandTool) Description() string { return "Execute a system command" }

func (t *ExecuteCommandTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	command, ok := params["command"].(string)
	if !ok {
		return "", fmt.Errorf("command parameter is required")
	}
	
	args := strings.Fields(command)
	if len(args) == 0 {
		return "", fmt.Errorf("empty command")
	}
	
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	output, err := cmd.CombinedOutput()
	
	result := string(output)
	if err != nil {
		result += fmt.Sprintf("\nError: %v", err)
	}
	
	return result, nil
}

// SearchTool - 搜索文件内容
type SearchTool struct{}

func (t *SearchTool) Name() string { return "search" }
func (t *SearchTool) Description() string { return "Search for text in files" }

func (t *SearchTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter is required")
	}
	
	path, ok := params["path"].(string)
	if !ok {
		path = "."
	}
	
	var result strings.Builder
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		if info.IsDir() {
			return nil
		}
		
		// 只搜索文本文件
		if !isTextFile(filePath) {
			return nil
		}
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}
		
		if strings.Contains(string(content), pattern) {
			result.WriteString(fmt.Sprintf("Found in: %s\n", filePath))
		}
		
		return nil
	})
	
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	
	return result.String(), nil
}

func isTextFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	textExts := []string{".go", ".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp", ".html", ".css", ".xml", ".sql", ".sh", ".bat"}
	
	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}
	
	return false
}

// EditFileTool - 编辑文件内容（替换指定行范围或模式）
type EditFileTool struct{}

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string { return "Edit file content by replacing specific lines or patterns" }

func (t *EditFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path parameter is required")
	}
	
	// 读取原文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	lines := strings.Split(string(content), "\n")
	
	// 检查编辑模式
	if oldText, ok := params["old_text"].(string); ok {
		// 模式1: 替换指定文本
		newText, ok := params["new_text"].(string)
		if !ok {
			return "", fmt.Errorf("new_text parameter is required when using old_text")
		}
		
		newContent := strings.ReplaceAll(string(content), oldText, newText)
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		
		return fmt.Sprintf("Replaced text in file: %s", filePath), nil
	}
	
	if startLine, ok := params["start_line"].(float64); ok {
		// 模式2: 替换指定行范围
		endLine, ok := params["end_line"].(float64)
		if !ok {
			endLine = startLine
		}
		
		newContent, ok := params["new_content"].(string)
		if !ok {
			return "", fmt.Errorf("new_content parameter is required when using line numbers")
		}
		
		start := int(startLine) - 1
		end := int(endLine) - 1
		
		if start < 0 || start >= len(lines) || end < 0 || end >= len(lines) || start > end {
			return "", fmt.Errorf("invalid line range: %d-%d", start+1, end+1)
		}
		
		// 替换指定行范围
		newLines := strings.Split(newContent, "\n")
		result := append(lines[:start], newLines...)
		result = append(result, lines[end+1:]...)
		
		if err := os.WriteFile(filePath, []byte(strings.Join(result, "\n")), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}
		
		return fmt.Sprintf("Edited lines %d-%d in file: %s", int(startLine), int(endLine), filePath), nil
	}
	
	return "", fmt.Errorf("either old_text/new_text or start_line/end_line/new_content parameters are required")
}

// CreateDirectoryTool - 创建目录
type CreateDirectoryTool struct{}

func (t *CreateDirectoryTool) Name() string { return "create_directory" }
func (t *CreateDirectoryTool) Description() string { return "Create a directory and its parent directories if needed" }

func (t *CreateDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	dirPath, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	return fmt.Sprintf("Directory created: %s", dirPath), nil
}

// DeleteFileTool - 删除文件或目录
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "delete_file" }
func (t *DeleteFileTool) Description() string { return "Delete a file or directory" }

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %s", path)
	}
	
	if info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			return "", fmt.Errorf("failed to delete directory: %w", err)
		}
		return fmt.Sprintf("Directory deleted: %s", path), nil
	} else {
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("failed to delete file: %w", err)
		}
		return fmt.Sprintf("File deleted: %s", path), nil
	}
}

// RenameFileTool - 重命名文件或目录
type RenameFileTool struct{}

func (t *RenameFileTool) Name() string { return "rename_file" }
func (t *RenameFileTool) Description() string { return "Rename a file or directory" }

func (t *RenameFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	oldPath, ok := params["old_path"].(string)
	if !ok {
		return "", fmt.Errorf("old_path parameter is required")
	}
	
	newPath, ok := params["new_path"].(string)
	if !ok {
		return "", fmt.Errorf("new_path parameter is required")
	}
	
	if err := os.Rename(oldPath, newPath); err != nil {
		return "", fmt.Errorf("failed to rename: %w", err)
	}
	
	return fmt.Sprintf("Renamed %s to %s", oldPath, newPath), nil
}

// CopyFileTool - 复制文件
type CopyFileTool struct{}

func (t *CopyFileTool) Name() string { return "copy_file" }
func (t *CopyFileTool) Description() string { return "Copy a file to another location" }

func (t *CopyFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	srcPath, ok := params["src_path"].(string)
	if !ok {
		return "", fmt.Errorf("src_path parameter is required")
	}
	
	dstPath, ok := params["dst_path"].(string)
	if !ok {
		return "", fmt.Errorf("dst_path parameter is required")
	}
	
	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// 复制文件
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()
	
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()
	
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}
	
	return fmt.Sprintf("Copied %s to %s", srcPath, dstPath), nil
}

// MoveFileTool - 移动文件
type MoveFileTool struct{}

func (t *MoveFileTool) Name() string { return "move_file" }
func (t *MoveFileTool) Description() string { return "Move a file to another location" }

func (t *MoveFileTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	srcPath, ok := params["src_path"].(string)
	if !ok {
		return "", fmt.Errorf("src_path parameter is required")
	}
	
	dstPath, ok := params["dst_path"].(string)
	if !ok {
		return "", fmt.Errorf("dst_path parameter is required")
	}
	
	// 创建目标目录
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	if err := os.Rename(srcPath, dstPath); err != nil {
		return "", fmt.Errorf("failed to move file: %w", err)
	}
	
	return fmt.Sprintf("Moved %s to %s", srcPath, dstPath), nil
}

// FileInfoTool - 获取文件信息
type FileInfoTool struct{}

func (t *FileInfoTool) Name() string { return "file_info" }
func (t *FileInfoTool) Description() string { return "Get detailed information about a file or directory" }

func (t *FileInfoTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	path, ok := params["path"].(string)
	if !ok {
		return "", fmt.Errorf("path parameter is required")
	}
	
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Path: %s\n", path))
	result.WriteString(fmt.Sprintf("Name: %s\n", info.Name()))
	result.WriteString(fmt.Sprintf("Size: %d bytes\n", info.Size()))
	result.WriteString(fmt.Sprintf("Mode: %s\n", info.Mode()))
	result.WriteString(fmt.Sprintf("Modified: %s\n", info.ModTime().Format(time.RFC3339)))
	
	if info.IsDir() {
		result.WriteString("Type: Directory\n")
	} else {
		result.WriteString("Type: File\n")
		result.WriteString(fmt.Sprintf("Extension: %s\n", filepath.Ext(path)))
	}
	
	return result.String(), nil
}

// ReplaceTextTool - 批量文本替换（支持正则表达式）
type ReplaceTextTool struct{}

func (t *ReplaceTextTool) Name() string { return "replace_text" }
func (t *ReplaceTextTool) Description() string { return "Replace text in files using patterns or regular expressions" }

func (t *ReplaceTextTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path parameter is required")
	}
	
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter is required")
	}
	
	replacement, ok := params["replacement"].(string)
	if !ok {
		return "", fmt.Errorf("replacement parameter is required")
	}
	
	useRegex, _ := params["use_regex"].(bool)
	
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	
	var newContent string
	var count int
	
	if useRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
		
		newContent = re.ReplaceAllString(string(content), replacement)
		count = len(re.FindAllString(string(content), -1))
	} else {
		oldContent := string(content)
		newContent = strings.ReplaceAll(oldContent, pattern, replacement)
		count = strings.Count(oldContent, pattern)
	}
	
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	
	return fmt.Sprintf("Replaced %d occurrences in %s", count, filePath), nil
}