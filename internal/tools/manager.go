package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/AiNovelTools/internal/ai"
	contextmgr "github.com/AiNovelTools/internal/context"
	"github.com/AiNovelTools/internal/novel"
)

type Manager struct {
	tools          map[string]Tool
	contextManager *contextmgr.ContextManager
	novelManager   *novel.NovelManager
}

type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, params map[string]interface{}) (string, error)
}

// EnhancedTool 提供更详细的工具信息给AI模型
type EnhancedTool interface {
	Tool
	GetParameterSchema() map[string]interface{}
	GetUsageExamples() []string
	GetCategory() string
}

type ToolResult struct {
	ToolName string
	Result   string
	Error    error
}

func NewManager() *Manager {
	contextManager := contextmgr.NewContextManager()
	// 尝试加载已保存的上下文
	contextManager.LoadContext()
	
	// 初始化小说管理器
	currentDir, _ := os.Getwd()
	novelManager := novel.NewNovelManager(currentDir)
	novelManager.LoadProject() // 尝试加载已有项目
	
	m := &Manager{
		tools:          make(map[string]Tool),
		contextManager: contextManager,
		novelManager:   novelManager,
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
	
	// 环境感知工具
	m.RegisterTool(&GetCurrentDirectoryTool{})
	m.RegisterTool(&GetSystemInfoTool{})
	m.RegisterTool(&GetProjectInfoTool{})
	m.RegisterTool(&GetWorkingContextTool{})
	m.RegisterTool(&GetSmartContextTool{contextManager: m.contextManager})
	
	// 小说写作工具
	m.RegisterTool(&InitNovelProjectTool{novelManager: m.novelManager})
	m.RegisterTool(&GetNovelContextTool{novelManager: m.novelManager})
	m.RegisterTool(&AddCharacterTool{novelManager: m.novelManager})
	m.RegisterTool(&AddPlotLineTool{novelManager: m.novelManager})
	m.RegisterTool(&GetChapterContextTool{novelManager: m.novelManager})
	m.RegisterTool(&SearchNovelHistoryTool{novelManager: m.novelManager})
	
	return m
}

func (m *Manager) RegisterTool(tool Tool) {
	m.tools[tool.Name()] = tool
}

func (m *Manager) GetTool(name string) (Tool, bool) {
	tool, exists := m.tools[name]
	return tool, exists
}

func (m *Manager) GetAllTools() map[string]Tool {
	return m.tools
}

func (m *Manager) GetToolsInfo() string {
	var result strings.Builder
	result.WriteString("🛠️ === AI Assistant Tools Documentation === 🛠️\n\n")
	
	categories := map[string][]string{
		"📖 File Operations": {},
		"📁 Directory Management": {},
		"🔍 Search & Replace": {},
		"⚡ System Commands": {},
		"🤖 Environment Awareness": {},
		"📚 Novel Writing": {},
	}
	
	// 分类工具
	for name := range m.tools {
		category := getToolCategory(name)
		if tools, exists := categories[category]; exists {
			categories[category] = append(tools, name)
		}
	}
	
	// 生成文档
	for category, toolNames := range categories {
		if len(toolNames) > 0 {
			result.WriteString(fmt.Sprintf("%s\n", category))
			for _, toolName := range toolNames {
				if tool, exists := m.tools[toolName]; exists {
					result.WriteString(fmt.Sprintf("  • %s - %s\n", toolName, tool.Description()))
				}
			}
			result.WriteString("\n")
		}
	}
	
	result.WriteString("💡 Usage Tips:\n")
	result.WriteString("• Use get_working_context first to understand current environment\n")
	result.WriteString("• Use get_project_info to analyze project structure\n")
	result.WriteString("• Use search tool to find specific code or content\n")
	result.WriteString("• Always use file_info before modifying important files\n")
	result.WriteString("• Use execute_command for system operations like git, npm, etc.\n")
	
	return result.String()
}

func getToolCategory(toolName string) string {
	fileOps := []string{"read_file", "write_file", "edit_file", "file_info", "copy_file", "move_file", "rename_file", "delete_file"}
	dirOps := []string{"list_files", "create_directory"}
	searchOps := []string{"search", "replace_text"}
	sysOps := []string{"execute_command"}
	envOps := []string{"get_current_directory", "get_system_info", "get_project_info", "get_working_context", "get_smart_context"}
	novelOps := []string{"init_novel_project", "get_novel_context", "add_character", "add_plot_line", "get_chapter_context", "search_novel_history"}
	
	for _, op := range fileOps {
		if op == toolName {
			return "📖 File Operations"
		}
	}
	for _, op := range dirOps {
		if op == toolName {
			return "📁 Directory Management"
		}
	}
	for _, op := range searchOps {
		if op == toolName {
			return "🔍 Search & Replace"
		}
	}
	for _, op := range sysOps {
		if op == toolName {
			return "⚡ System Commands"
		}
	}
	for _, op := range envOps {
		if op == toolName {
			return "🤖 Environment Awareness"
		}
	}
	for _, op := range novelOps {
		if op == toolName {
			return "📚 Novel Writing"
		}
	}
	
	return "🔧 Other Tools"
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

// ======================== 环境感知工具 ========================

// GetCurrentDirectoryTool - 获取当前工作目录
type GetCurrentDirectoryTool struct{}

func (t *GetCurrentDirectoryTool) Name() string { return "get_current_directory" }
func (t *GetCurrentDirectoryTool) Description() string { 
	return "Get the current working directory and basic directory information. This tool helps AI understand the current location in the file system."
}

func (t *GetCurrentDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// 获取目录信息
	info, err := os.Stat(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to get directory info: %w", err)
	}
	
	// 列出当前目录的内容（仅第一级）
	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Current Working Directory: %s\n", currentDir))
	result.WriteString(fmt.Sprintf("Directory Name: %s\n", filepath.Base(currentDir)))
	result.WriteString(fmt.Sprintf("Parent Directory: %s\n", filepath.Dir(currentDir)))
	result.WriteString(fmt.Sprintf("Modified: %s\n", info.ModTime().Format(time.RFC3339)))
	result.WriteString(fmt.Sprintf("Total Items: %d\n\n", len(entries)))
	
	result.WriteString("Directory Contents:\n")
	fileCount, dirCount := 0, 0
	for _, entry := range entries {
		if entry.IsDir() {
			result.WriteString(fmt.Sprintf("📁 %s/\n", entry.Name()))
			dirCount++
		} else {
			result.WriteString(fmt.Sprintf("📄 %s\n", entry.Name()))
			fileCount++
		}
		if fileCount+dirCount >= 20 { // 限制显示数量
			result.WriteString("... (更多项目)\n")
			break
		}
	}
	
	result.WriteString(fmt.Sprintf("\nSummary: %d directories, %d files\n", dirCount, fileCount))
	
	return result.String(), nil
}

// GetSystemInfoTool - 获取系统信息
type GetSystemInfoTool struct{}

func (t *GetSystemInfoTool) Name() string { return "get_system_info" }
func (t *GetSystemInfoTool) Description() string { 
	return "Get comprehensive system information including OS, architecture, Go version, and environment variables. Helps AI understand the runtime environment."
}

func (t *GetSystemInfoTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	var result strings.Builder
	
	// 基本系统信息
	result.WriteString("=== System Information ===\n")
	result.WriteString(fmt.Sprintf("Operating System: %s\n", runtime.GOOS))
	result.WriteString(fmt.Sprintf("Architecture: %s\n", runtime.GOARCH))
	result.WriteString(fmt.Sprintf("Go Version: %s\n", runtime.Version()))
	result.WriteString(fmt.Sprintf("CPU Cores: %d\n", runtime.NumCPU()))
	
	// 获取主机名
	if hostname, err := os.Hostname(); err == nil {
		result.WriteString(fmt.Sprintf("Hostname: %s\n", hostname))
	}
	
	// 当前用户
	if user := os.Getenv("USER"); user == "" {
		user = os.Getenv("USERNAME") // Windows
	} else {
		result.WriteString(fmt.Sprintf("Current User: %s\n", user))
	}
	
	// 环境变量（重要的）
	result.WriteString("\n=== Environment Variables ===\n")
	importantEnvs := []string{"PATH", "HOME", "GOPATH", "GOROOT", "GOPROXY", "PWD"}
	for _, env := range importantEnvs {
		if value := os.Getenv(env); value != "" {
			// 对于PATH，只显示前几个路径
			if env == "PATH" {
				paths := strings.Split(value, string(os.PathListSeparator))
				if len(paths) > 5 {
					result.WriteString(fmt.Sprintf("%s: %s ... (and %d more)\n", env, strings.Join(paths[:5], string(os.PathListSeparator)), len(paths)-5))
				} else {
					result.WriteString(fmt.Sprintf("%s: %s\n", env, value))
				}
			} else {
				result.WriteString(fmt.Sprintf("%s: %s\n", env, value))
			}
		}
	}
	
	// 磁盘空间信息（当前目录）
	currentDir, _ := os.Getwd()
	result.WriteString(fmt.Sprintf("\n=== Current Directory Context ===\n"))
	result.WriteString(fmt.Sprintf("Working Directory: %s\n", currentDir))
	
	return result.String(), nil
}

// GetProjectInfoTool - 获取项目信息
type GetProjectInfoTool struct{}

func (t *GetProjectInfoTool) Name() string { return "get_project_info" }
func (t *GetProjectInfoTool) Description() string { 
	return "Analyze and identify project type, structure, dependencies, and provide intelligent insights about the current project."
}

func (t *GetProjectInfoTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	projectPath, ok := params["path"].(string)
	if !ok {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		projectPath = currentDir
	}
	
	var result strings.Builder
	result.WriteString(fmt.Sprintf("=== Project Analysis: %s ===\n", filepath.Base(projectPath)))
	result.WriteString(fmt.Sprintf("Project Path: %s\n\n", projectPath))
	
	// 检测项目类型
	projectType := detectProjectType(projectPath)
	result.WriteString(fmt.Sprintf("🎯 Project Type: %s\n\n", projectType))
	
	// 分析项目结构
	structure := analyzeProjectStructure(projectPath)
	result.WriteString("📁 Project Structure:\n")
	result.WriteString(structure)
	result.WriteString("\n")
	
	// 检测依赖和配置文件
	dependencies := analyzeDependencies(projectPath)
	if dependencies != "" {
		result.WriteString("📦 Dependencies & Configuration:\n")
		result.WriteString(dependencies)
		result.WriteString("\n")
	}
	
	// 提供智能建议
	suggestions := generateProjectSuggestions(projectPath, projectType)
	if suggestions != "" {
		result.WriteString("💡 AI Suggestions:\n")
		result.WriteString(suggestions)
	}
	
	return result.String(), nil
}

// GetWorkingContextTool - 获取完整工作上下文
type GetWorkingContextTool struct{}

func (t *GetWorkingContextTool) Name() string { return "get_working_context" }
func (t *GetWorkingContextTool) Description() string { 
	return "Get comprehensive working context including current directory, system info, project details, and recent activity. Provides complete environment awareness for AI."
}

func (t *GetWorkingContextTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	var result strings.Builder
	result.WriteString("🤖 === AI Assistant Working Context === 🤖\n\n")
	
	// 获取当前目录信息
	currentDirTool := &GetCurrentDirectoryTool{}
	dirInfo, err := currentDirTool.Execute(ctx, nil)
	if err == nil {
		result.WriteString("📍 " + dirInfo + "\n")
	}
	
	// 获取项目信息
	projectTool := &GetProjectInfoTool{}
	projectInfo, err := projectTool.Execute(ctx, nil)
	if err == nil {
		result.WriteString(projectInfo + "\n")
	}
	
	// 系统信息摘要
	result.WriteString("💻 System Summary:\n")
	result.WriteString(fmt.Sprintf("OS: %s/%s | Go: %s | CPU: %d cores\n", runtime.GOOS, runtime.GOARCH, runtime.Version(), runtime.NumCPU()))
	
	if hostname, err := os.Hostname(); err == nil {
		result.WriteString(fmt.Sprintf("Host: %s | ", hostname))
	}
	
	currentDir, _ := os.Getwd()
	result.WriteString(fmt.Sprintf("PWD: %s\n\n", currentDir))
	
	// AI工作建议
	result.WriteString("🎯 AI Assistant Ready!\n")
	result.WriteString("Available capabilities:\n")
	result.WriteString("• File operations (read, write, edit, search, replace)\n")
	result.WriteString("• Directory management (create, delete, move, copy)\n")
	result.WriteString("• System commands execution\n")
	result.WriteString("• Project analysis and structure understanding\n")
	result.WriteString("• Context-aware assistance based on project type\n")
	
	return result.String(), nil
}

// ======================== 辅助函数 ========================

func detectProjectType(projectPath string) string {
	// 检查常见的项目标识文件
	projectIndicators := map[string]string{
		"go.mod":          "Go Module Project",
		"go.sum":          "Go Project",
		"package.json":    "Node.js/JavaScript Project",
		"pom.xml":         "Java Maven Project",
		"build.gradle":    "Java Gradle Project",
		"Cargo.toml":      "Rust Project",
		"pyproject.toml":  "Python Project",
		"requirements.txt": "Python Project",
		"composer.json":   "PHP Project",
		"Gemfile":         "Ruby Project",
		"CMakeLists.txt":  "C/C++ CMake Project",
		"Makefile":        "C/C++ Make Project",
		"Dockerfile":      "Docker Project",
		"docker-compose.yml": "Docker Compose Project",
	}
	
	detectedTypes := []string{}
	
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return "Unknown Project Type"
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			if projectType, exists := projectIndicators[entry.Name()]; exists {
				detectedTypes = append(detectedTypes, projectType)
			}
		}
	}
	
	if len(detectedTypes) == 0 {
		return "Generic Directory"
	}
	
	return strings.Join(detectedTypes, " + ")
}

func analyzeProjectStructure(projectPath string) string {
	var result strings.Builder
	
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return "Unable to analyze project structure"
	}
	
	directories := []string{}
	files := []string{}
	
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		} else {
			files = append(files, entry.Name())
		}
	}
	
	// 分析目录结构
	for _, dir := range directories {
		dirType := classifyDirectory(dir)
		result.WriteString(fmt.Sprintf("📁 %s/ - %s\n", dir, dirType))
	}
	
	// 分析重要文件
	for _, file := range files {
		fileType := classifyFile(file)
		if fileType != "" {
			result.WriteString(fmt.Sprintf("📄 %s - %s\n", file, fileType))
		}
	}
	
	return result.String()
}

func classifyDirectory(dirName string) string {
	dirTypes := map[string]string{
		"src":        "Source Code Directory",
		"internal":   "Internal Package Directory",
		"pkg":        "Package Directory", 
		"cmd":        "Command Directory",
		"api":        "API Definition Directory",
		"web":        "Web Assets Directory",
		"static":     "Static Files Directory",
		"templates":  "Template Directory",
		"config":     "Configuration Directory",
		"docs":       "Documentation Directory",
		"test":       "Test Directory",
		"tests":      "Test Directory",
		"build":      "Build Output Directory",
		"dist":       "Distribution Directory",
		"vendor":     "Vendor Dependencies Directory",
		"node_modules": "Node.js Dependencies Directory",
		"target":     "Build Target Directory",
		"bin":        "Binary Directory",
		"scripts":    "Scripts Directory",
		"tools":      "Tools Directory",
		"examples":   "Examples Directory",
		"frontend":   "Frontend Code Directory",
		"backend":    "Backend Code Directory",
	}
	
	if description, exists := dirTypes[strings.ToLower(dirName)]; exists {
		return description
	}
	
	return "Project Directory"
}

func classifyFile(fileName string) string {
	fileTypes := map[string]string{
		"go.mod":          "Go Module Definition",
		"go.sum":          "Go Module Checksums",
		"package.json":    "Node.js Package Definition",
		"package-lock.json": "Node.js Lock File",
		"pom.xml":         "Maven Build Configuration",
		"build.gradle":    "Gradle Build Configuration",
		"Dockerfile":      "Docker Container Definition",
		"docker-compose.yml": "Docker Compose Configuration",
		"README.md":       "Project Documentation",
		"LICENSE":         "License File",
		"Makefile":        "Build Automation File",
		"main.go":         "Go Main Entry Point",
		"main.py":         "Python Main Entry Point",
		"index.js":        "JavaScript Entry Point",
		"index.html":      "HTML Entry Point",
		".gitignore":      "Git Ignore Rules",
		".env":            "Environment Variables",
		"config.yaml":     "YAML Configuration",
		"config.json":     "JSON Configuration",
		"tsconfig.json":   "TypeScript Configuration",
	}
	
	if description, exists := fileTypes[fileName]; exists {
		return description
	}
	
	// 检查文件扩展名
	ext := strings.ToLower(filepath.Ext(fileName))
	extTypes := map[string]string{
		".go":   "Go Source File",
		".py":   "Python Source File",
		".js":   "JavaScript File",
		".ts":   "TypeScript File",
		".java": "Java Source File",
		".cpp":  "C++ Source File",
		".c":    "C Source File",
		".h":    "Header File",
		".md":   "Markdown Documentation",
		".yml":  "YAML Configuration",
		".yaml": "YAML Configuration",
		".json": "JSON Data File",
		".xml":  "XML File",
		".html": "HTML File",
		".css":  "CSS Stylesheet",
		".sql":  "SQL Script",
		".sh":   "Shell Script",
		".bat":  "Batch Script",
	}
	
	if description, exists := extTypes[ext]; exists {
		return description
	}
	
	return "" // 不显示普通文件
}

func analyzeDependencies(projectPath string) string {
	var result strings.Builder
	
	// Go项目依赖分析
	if goModPath := filepath.Join(projectPath, "go.mod"); fileExists(goModPath) {
		content, err := os.ReadFile(goModPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			result.WriteString("🔧 Go Dependencies (go.mod):\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module") {
					result.WriteString(fmt.Sprintf("  Module: %s\n", strings.TrimPrefix(line, "module ")))
				} else if strings.HasPrefix(line, "go ") {
					result.WriteString(fmt.Sprintf("  Go Version: %s\n", strings.TrimPrefix(line, "go ")))
				} else if strings.Contains(line, "require") && !strings.Contains(line, "//") {
					result.WriteString(fmt.Sprintf("  %s\n", line))
				}
			}
			result.WriteString("\n")
		}
	}
	
	// Node.js项目依赖分析
	if packageJsonPath := filepath.Join(projectPath, "package.json"); fileExists(packageJsonPath) {
		content, err := os.ReadFile(packageJsonPath)
		if err == nil {
			result.WriteString("📦 Node.js Dependencies (package.json):\n")
			var packageData map[string]interface{}
			if json.Unmarshal(content, &packageData) == nil {
				if name, ok := packageData["name"].(string); ok {
					result.WriteString(fmt.Sprintf("  Package: %s\n", name))
				}
				if version, ok := packageData["version"].(string); ok {
					result.WriteString(fmt.Sprintf("  Version: %s\n", version))
				}
				if scripts, ok := packageData["scripts"].(map[string]interface{}); ok {
					result.WriteString("  Available Scripts:\n")
					for script := range scripts {
						result.WriteString(fmt.Sprintf("    - npm run %s\n", script))
					}
				}
			}
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

func generateProjectSuggestions(projectPath, projectType string) string {
	var suggestions strings.Builder
	
	// 基于项目类型的建议
	if strings.Contains(projectType, "Go") {
		suggestions.WriteString("• Use 'go run main.go' to run the application\n")
		suggestions.WriteString("• Use 'go build' to compile the project\n")
		suggestions.WriteString("• Use 'go mod tidy' to clean up dependencies\n")
		suggestions.WriteString("• Check 'internal/' directory for internal packages\n")
	}
	
	if strings.Contains(projectType, "Node.js") || strings.Contains(projectType, "JavaScript") {
		suggestions.WriteString("• Use 'npm install' to install dependencies\n")
		suggestions.WriteString("• Use 'npm start' or 'npm run dev' to start development\n")
		suggestions.WriteString("• Check package.json for available scripts\n")
	}
	
	if strings.Contains(projectType, "Java") {
		suggestions.WriteString("• Use 'mvn compile' or 'gradle build' to build\n")
		suggestions.WriteString("• Check src/main/java for source code\n")
		suggestions.WriteString("• Look for application.properties for configuration\n")
	}
	
	// 通用建议
	if fileExists(filepath.Join(projectPath, "README.md")) {
		suggestions.WriteString("• Read README.md for project documentation\n")
	}
	
	if fileExists(filepath.Join(projectPath, "Makefile")) {
		suggestions.WriteString("• Use 'make' commands for build automation\n")
	}
	
	if fileExists(filepath.Join(projectPath, "docker-compose.yml")) {
		suggestions.WriteString("• Use 'docker-compose up' to start services\n")
	}
	
	return suggestions.String()
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// GetSmartContextTool - 智能上下文感知工具
type GetSmartContextTool struct {
	contextManager *contextmgr.ContextManager
}

func (t *GetSmartContextTool) Name() string { return "get_smart_context" }
func (t *GetSmartContextTool) Description() string { 
	return "Get intelligent context including project history, user preferences, recent activities, and personalized suggestions. Provides the most comprehensive environment awareness."
}

func (t *GetSmartContextTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// 更新当前项目上下文
	currentDir, _ := os.Getwd()
	t.contextManager.UpdateCurrentProject(currentDir)
	
	var result strings.Builder
	result.WriteString("🧠 === Smart Context Analysis === 🧠\n\n")
	
	// 获取上下文摘要
	contextSummary := t.contextManager.GetContextSummary()
	result.WriteString(contextSummary)
	
	// 获取智能建议
	suggestions := t.contextManager.GetWorkingSuggestions()
	if len(suggestions) > 0 {
		result.WriteString("💡 Smart Suggestions:\n")
		for _, suggestion := range suggestions {
			result.WriteString(fmt.Sprintf("  %s\n", suggestion))
		}
		result.WriteString("\n")
	}
	
	// 项目分析（结合基础工具）
	projectTool := &GetProjectInfoTool{}
	projectInfo, err := projectTool.Execute(ctx, nil)
	if err == nil {
		result.WriteString("🔍 Current Project Analysis:\n")
		// 只显示关键信息，避免重复
		lines := strings.Split(projectInfo, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Project Type:") || 
			   strings.Contains(line, "AI Suggestions:") ||
			   strings.Contains(line, "Dependencies & Configuration:") {
				result.WriteString(line + "\n")
			}
		}
		result.WriteString("\n")
	}
	
	// 环境状态
	result.WriteString("🌐 Environment Status:\n")
	result.WriteString(fmt.Sprintf("OS: %s/%s | Go: %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	result.WriteString(fmt.Sprintf("Working Directory: %s\n", currentDir))
	result.WriteString(fmt.Sprintf("Context Last Updated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// AI助手能力提醒
	result.WriteString("🤖 AI Assistant Ready!\n")
	result.WriteString("I have full context awareness and can help you with:\n")
	result.WriteString("• Intelligent file operations based on project type\n")
	result.WriteString("• Context-aware code analysis and suggestions\n")
	result.WriteString("• Project-specific development workflow automation\n")
	result.WriteString("• Smart content search and modification\n")
	result.WriteString("• Personalized assistance based on your work patterns\n")
	
	return result.String(), nil
}

// ======================== 小说写作工具 ========================

// InitNovelProjectTool - 初始化小说项目
type InitNovelProjectTool struct {
	novelManager *novel.NovelManager
}

func (t *InitNovelProjectTool) Name() string { return "init_novel_project" }
func (t *InitNovelProjectTool) Description() string { 
	return "Initialize a new novel writing project with title, author, genre, and basic settings. Creates the foundation for consistent novel writing with character and plot tracking."
}

func (t *InitNovelProjectTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	title, ok := params["title"].(string)
	if !ok {
		return "", fmt.Errorf("title parameter is required")
	}
	
	author, ok := params["author"].(string)
	if !ok {
		author = "Unknown Author"
	}
	
	genre, ok := params["genre"].(string)
	if !ok {
		genre = "Fiction"
	}
	
	if err := t.novelManager.InitializeProject(title, author, genre); err != nil {
		return "", fmt.Errorf("failed to initialize novel project: %w", err)
	}
	
	return fmt.Sprintf("✅ 小说项目初始化成功！\n标题: %s\n作者: %s\n类型: %s\n\n现在可以开始添加角色、情节线和章节内容了。", title, author, genre), nil
}

// GetNovelContextTool - 获取小说上下文
type GetNovelContextTool struct {
	novelManager *novel.NovelManager
}

func (t *GetNovelContextTool) Name() string { return "get_novel_context" }
func (t *GetNovelContextTool) Description() string { 
	return "Get comprehensive novel writing context including characters, plot lines, world settings, and writing progress. Essential for maintaining consistency across chapters."
}

func (t *GetNovelContextTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	// 这里需要实现获取小说完整上下文的逻辑
	// 由于novel包中的方法是私有的，我们需要在novel包中添加公共方法
	return "📚 小说项目上下文获取功能正在开发中...\n请先使用 init_novel_project 初始化项目。", nil
}

// AddCharacterTool - 添加角色
type AddCharacterTool struct {
	novelManager *novel.NovelManager
}

func (t *AddCharacterTool) Name() string { return "add_character" }
func (t *AddCharacterTool) Description() string { 
	return "Add a new character to the novel with detailed information including personality, background, relationships. Helps maintain character consistency throughout the story."
}

func (t *AddCharacterTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok {
		return "", fmt.Errorf("character name is required")
	}
	
	background, _ := params["background"].(string)
	personality, _ := params["personality"].(string)
	
	return fmt.Sprintf("🎭 角色添加功能正在开发中...\n将添加角色: %s\n背景: %s\n性格: %s", name, background, personality), nil
}

// AddPlotLineTool - 添加情节线
type AddPlotLineTool struct {
	novelManager *novel.NovelManager
}

func (t *AddPlotLineTool) Name() string { return "add_plot_line" }
func (t *AddPlotLineTool) Description() string { 
	return "Add a new plot line to track story progression, conflicts, and resolutions. Essential for maintaining narrative coherence and managing multiple story threads."
}

func (t *AddPlotLineTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	name, ok := params["name"].(string)
	if !ok {
		return "", fmt.Errorf("plot line name is required")
	}
	
	plotType, _ := params["type"].(string)
	description, _ := params["description"].(string)
	
	return fmt.Sprintf("📖 情节线添加功能正在开发中...\n将添加情节线: %s\n类型: %s\n描述: %s", name, plotType, description), nil
}

// GetChapterContextTool - 获取章节上下文
type GetChapterContextTool struct {
	novelManager *novel.NovelManager
}

func (t *GetChapterContextTool) Name() string { return "get_chapter_context" }
func (t *GetChapterContextTool) Description() string { 
	return "Get specific chapter context including relevant characters, active plot lines, and recent discussions. Critical for maintaining chapter-to-chapter continuity."
}

func (t *GetChapterContextTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	chapterNum, ok := params["chapter"].(float64)
	if !ok {
		return "", fmt.Errorf("chapter number is required")
	}
	
	return fmt.Sprintf("📄 第%d章上下文获取功能正在开发中...", int(chapterNum)), nil
}

// SearchNovelHistoryTool - 搜索小说历史
type SearchNovelHistoryTool struct {
	novelManager *novel.NovelManager
}

func (t *SearchNovelHistoryTool) Name() string { return "search_novel_history" }
func (t *SearchNovelHistoryTool) Description() string { 
	return "Search through novel writing history to find previous discussions about characters, plots, or specific content. Crucial for maintaining story consistency and avoiding contradictions."
}

func (t *SearchNovelHistoryTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok {
		return "", fmt.Errorf("search query is required")
	}
	
	maxResults, _ := params["max_results"].(float64)
	if maxResults == 0 {
		maxResults = 10
	}
	
	return fmt.Sprintf("🔍 搜索小说历史功能正在开发中...\n查询: %s\n最大结果数: %d", query, int(maxResults)), nil
}

// GetToolDefinitions 获取所有工具的定义，供AI模型使用
func (m *Manager) GetToolDefinitions() []map[string]interface{} {
	var tools []map[string]interface{}
	
	for name, tool := range m.tools {
		toolDef := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": getToolParameters(name),
				},
			},
		}
		tools = append(tools, toolDef)
	}
	
	return tools
}

// getToolParameters 获取工具的参数定义
func getToolParameters(toolName string) map[string]interface{} {
	switch toolName {
	case "read_file":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "Path to the file to read",
			},
		}
	case "write_file":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string", 
				"description": "Path to the file to write",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "Content to write to the file",
			},
		}
	case "list_files":
		return map[string]interface{}{
			"directory": map[string]interface{}{
				"type":        "string",
				"description": "Directory path to list (optional, defaults to current directory)",
			},
		}
	case "search":
		return map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Text to search for",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string", 
				"description": "File pattern to search in (optional)",
			},
		}
	case "execute_command":
		return map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Command to execute",
			},
		}
	default:
		return map[string]interface{}{}
	}
}