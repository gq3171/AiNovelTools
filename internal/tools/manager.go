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
	ToolName   string
	Result     string
	Error      error
	ToolCallID string
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
	m.RegisterTool(&SmartTaskPlannerTool{})
	
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
		
		params := make(map[string]interface{})
		if arguments, exists := call.Function["arguments"]; exists {
			switch args := arguments.(type) {
			case map[string]interface{}:
				params = args
			case string:
				// 如果arguments是JSON字符串，尝试解析
				if err := json.Unmarshal([]byte(args), &params); err != nil {
					results = append(results, ToolResult{
						ToolName:   funcName,
						Error:      fmt.Errorf("failed to parse arguments JSON: %w", err),
						ToolCallID: call.ID,
					})
					continue
				}
			}
		}
		
		tool, exists := m.tools[funcName]
		if !exists {
			results = append(results, ToolResult{
				ToolName:   funcName,
				Error:      fmt.Errorf("unknown tool: %s", funcName),
				ToolCallID: call.ID,
			})
			continue
		}
		
		result, err := tool.Execute(ctx, params)
		results = append(results, ToolResult{
			ToolName:   funcName,
			Result:     result,
			Error:      err,
			ToolCallID: call.ID,
		})
	}
	
	return results, nil
}

// ReadFileTool - 读取文件内容
type ReadFileTool struct{}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string { 
	return "读取文件内容。🚨重要：使用此工具前必须先调用list_files获取准确的文件名和路径，绝对禁止使用假设路径或/path/to/这样的占位符。必须使用list_files结果中的确切文件名。"
}

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
func (t *ListFilesTool) Description() string { 
	return "列出目录中的文件和子目录。🔥必须第一步：任何涉及文件的请求都必须先调用此工具了解环境！这是获取准确文件名和路径的唯一方法，其他工具依赖此工具的结果。"
}

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
	query, ok := params["query"].(string)
	if !ok {
		return "", fmt.Errorf("query parameter is required")
	}
	
	path, ok := params["path"].(string)
	if !ok {
		path = "."
	}
	
	filePattern, _ := params["file_pattern"].(string)
	useRegex, _ := params["use_regex"].(bool)
	caseSensitive, _ := params["case_sensitive"].(bool)
	showLineNumbers, _ := params["show_line_numbers"].(bool)
	maxResults, _ := params["max_results"].(float64)
	
	if maxResults == 0 {
		maxResults = 50 // 默认最多显示50个结果
	}
	
	var result strings.Builder
	var foundCount int
	
	// 编译正则表达式（如果使用正则模式）
	var regexPattern *regexp.Regexp
	if useRegex {
		var err error
		flags := ""
		if !caseSensitive {
			flags = "(?i)"
		}
		regexPattern, err = regexp.Compile(flags + query)
		if err != nil {
			return "", fmt.Errorf("invalid regex pattern: %w", err)
		}
	}
	
	result.WriteString(fmt.Sprintf("🔍 搜索结果 - 查询: \"%s\"\n", query))
	if useRegex {
		result.WriteString("📝 模式: 正则表达式\n")
	} else {
		result.WriteString("📝 模式: 文本匹配\n")
	}
	result.WriteString(fmt.Sprintf("📁 路径: %s\n\n", path))
	
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || foundCount >= int(maxResults) {
			return nil
		}
		
		if info.IsDir() {
			return nil
		}
		
		// 检查文件模式匹配
		if filePattern != "" {
			matched, _ := filepath.Match(filePattern, filepath.Base(filePath))
			if !matched {
				return nil
			}
		}
		
		// 只搜索文本文件
		if !isTextFile(filePath) {
			return nil
		}
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil
		}
		
		lines := strings.Split(string(content), "\n")
		fileHasMatch := false
		var matchLines []string
		
		for lineNum, line := range lines {
			var matched bool
			
			if useRegex {
				if regexPattern.MatchString(line) {
					matched = true
				}
			} else {
				// 文本搜索
				searchLine := line
				searchQuery := query
				if !caseSensitive {
					searchLine = strings.ToLower(line)
					searchQuery = strings.ToLower(query)
				}
				
				if strings.Contains(searchLine, searchQuery) {
					matched = true
				}
			}
			
			if matched {
				if !fileHasMatch {
					fileHasMatch = true
					foundCount++
					result.WriteString(fmt.Sprintf("📄 %s\n", filePath))
				}
				
				if showLineNumbers {
					matchLines = append(matchLines, fmt.Sprintf("  第%d行: %s", lineNum+1, strings.TrimSpace(line)))
				} else {
					matchLines = append(matchLines, fmt.Sprintf("  %s", strings.TrimSpace(line)))
				}
				
				// 限制每个文件显示的匹配行数
				if len(matchLines) >= 5 {
					break
				}
			}
		}
		
		if fileHasMatch {
			for _, line := range matchLines {
				result.WriteString(line + "\n")
			}
			result.WriteString("\n")
		}
		
		return nil
	})
	
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}
	
	if foundCount == 0 {
		result.WriteString("❌ 未找到匹配的内容\n")
	} else {
		result.WriteString(fmt.Sprintf("✅ 共找到 %d 个文件包含匹配内容", foundCount))
		if foundCount >= int(maxResults) {
			result.WriteString(fmt.Sprintf("（已限制显示前%d个结果）", int(maxResults)))
		}
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
	result.WriteString("🧠 === 智能环境分析 === 🧠\n\n")
	
	// 获取上下文摘要
	contextSummary := t.contextManager.GetContextSummary()
	result.WriteString(contextSummary)
	
	// 获取智能建议
	suggestions := t.contextManager.GetWorkingSuggestions()
	if len(suggestions) > 0 {
		result.WriteString("💡 智能建议:\n")
		for _, suggestion := range suggestions {
			result.WriteString(fmt.Sprintf("  %s\n", suggestion))
		}
		result.WriteString("\n")
	}
	
	// 项目分析（结合基础工具）
	projectTool := &GetProjectInfoTool{}
	projectInfo, err := projectTool.Execute(ctx, nil)
	if err == nil {
		result.WriteString("🔍 当前项目分析:\n")
		// 只显示关键信息，避免重复
		lines := strings.Split(projectInfo, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Project Type:") || 
			   strings.Contains(line, "AI Suggestions:") ||
			   strings.Contains(line, "Dependencies & Configuration:") {
				// 翻译关键术语
				if strings.Contains(line, "Project Type:") {
					line = strings.Replace(line, "Project Type:", "🎯 项目类型:", 1)
				}
				if strings.Contains(line, "AI Suggestions:") {
					line = strings.Replace(line, "AI Suggestions:", "💡 AI建议:", 1)
				}
				if strings.Contains(line, "Dependencies & Configuration:") {
					line = strings.Replace(line, "Dependencies & Configuration:", "📦 依赖和配置:", 1)
				}
				result.WriteString(line + "\n")
			}
		}
		result.WriteString("\n")
	}
	
	// 环境状态
	result.WriteString("🌐 环境状态:\n")
	result.WriteString(fmt.Sprintf("操作系统: %s/%s | Go版本: %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version()))
	result.WriteString(fmt.Sprintf("工作目录: %s\n", currentDir))
	result.WriteString(fmt.Sprintf("上下文更新时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// AI助手能力提醒
	result.WriteString("🤖 AI助手已就绪!\n")
	result.WriteString("我具备完整的环境感知能力，可以为您提供:\n")
	result.WriteString("• 基于项目类型的智能文件操作\n")
	result.WriteString("• 上下文感知的代码分析和建议\n")
	result.WriteString("• 项目特定的开发工作流自动化\n")
	result.WriteString("• 智能内容搜索和修改\n")
	result.WriteString("• 基于您工作模式的个性化助手\n")
	
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
		parameters := getToolParameters(name)
		required := getRequiredParameters(name)
		
		toolDef := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type":       "object",
					"properties": parameters,
					"required":   required,
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
				"description": "要读取的文件路径",
			},
		}
	case "write_file":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string", 
				"description": "要写入的文件路径",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "要写入的文件内容",
			},
		}
	case "list_files":
		return map[string]interface{}{
			"directory": map[string]interface{}{
				"type":        "string",
				"description": "要列出的目录路径（可选，默认为当前目录）",
			},
		}
	case "search":
		return map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "要搜索的文本内容或正则表达式",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "搜索路径（可选，默认当前目录）",
			},
			"file_pattern": map[string]interface{}{
				"type":        "string", 
				"description": "文件匹配模式（可选，如*.txt）",
			},
			"use_regex": map[string]interface{}{
				"type":        "boolean",
				"description": "是否使用正则表达式搜索",
			},
			"case_sensitive": map[string]interface{}{
				"type":        "boolean",
				"description": "是否区分大小写",
			},
			"show_line_numbers": map[string]interface{}{
				"type":        "boolean",
				"description": "是否显示行号",
			},
			"max_results": map[string]interface{}{
				"type":        "integer",
				"description": "最大结果数量（默认50）",
			},
		}
	case "execute_command":
		return map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "要执行的系统命令",
			},
		}
	case "file_info":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "要获取信息的文件或目录路径",
			},
		}
	case "edit_file":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "要编辑的文件路径",
			},
			"start_line": map[string]interface{}{
				"type":        "integer",
				"description": "开始编辑的行号（可选）",
			},
			"end_line": map[string]interface{}{
				"type":        "integer", 
				"description": "结束编辑的行号（可选）",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "新的文件内容",
			},
		}
	case "create_directory":
		return map[string]interface{}{
			"directory_path": map[string]interface{}{
				"type":        "string",
				"description": "要创建的目录路径",
			},
		}
	case "delete_file":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "要删除的文件或目录路径",
			},
		}
	case "copy_file":
		return map[string]interface{}{
			"source_path": map[string]interface{}{
				"type":        "string",
				"description": "源文件路径",
			},
			"destination_path": map[string]interface{}{
				"type":        "string",
				"description": "目标文件路径",
			},
		}
	case "move_file", "rename_file":
		return map[string]interface{}{
			"old_path": map[string]interface{}{
				"type":        "string",
				"description": "原文件路径",
			},
			"new_path": map[string]interface{}{
				"type":        "string",
				"description": "新文件路径",
			},
		}
	case "replace_text":
		return map[string]interface{}{
			"file_path": map[string]interface{}{
				"type":        "string",
				"description": "要替换文本的文件路径",
			},
			"old_text": map[string]interface{}{
				"type":        "string",
				"description": "要被替换的文本",
			},
			"new_text": map[string]interface{}{
				"type":        "string",
				"description": "新的替换文本",
			},
		}
	case "get_project_info":
		return map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "项目路径（可选，默认为当前目录）",
			},
		}
	case "smart_task_planner":
		return map[string]interface{}{
			"task_description": map[string]interface{}{
				"type":        "string",
				"description": "需要规划的任务描述",
			},
		}
	default:
		return map[string]interface{}{}
	}
}

// getRequiredParameters 获取工具的必需参数列表
func getRequiredParameters(toolName string) []string {
	switch toolName {
	case "read_file":
		return []string{"file_path"}
	case "write_file":
		return []string{"file_path", "content"}
	case "search":
		return []string{"query"}
	case "execute_command":
		return []string{"command"}
	case "file_info":
		return []string{"file_path"}
	case "edit_file":
		return []string{"file_path", "content"}
	case "create_directory":
		return []string{"directory_path"}
	case "delete_file":
		return []string{"file_path"}
	case "copy_file":
		return []string{"source_path", "destination_path"}
	case "move_file", "rename_file":
		return []string{"old_path", "new_path"}
	case "replace_text":
		return []string{"file_path", "old_text", "new_text"}
	case "smart_task_planner":
		return []string{"task_description"}
	default:
		return []string{}
	}
}

// SmartTaskPlannerTool - 智能任务规划工具
type SmartTaskPlannerTool struct{}

func (t *SmartTaskPlannerTool) Name() string { return "smart_task_planner" }
func (t *SmartTaskPlannerTool) Description() string {
	return "智能任务规划工具。⚠️重要限制：只能在已经获取了完整项目信息（通过list_files和read_file）后使用。禁止在不了解具体项目内容时使用此工具给出通用建议。"
}

func (t *SmartTaskPlannerTool) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
	taskDesc, ok := params["task_description"].(string)
	if !ok {
		return "", fmt.Errorf("task_description parameter is required")
	}
	
	plan := analyzeAndPlanTask(taskDesc)
	return plan, nil
}

func analyzeAndPlanTask(taskDesc string) string {
	taskLower := strings.ToLower(taskDesc)
	var plan strings.Builder
	
	plan.WriteString("🧠 智能任务规划分析\n")
	plan.WriteString("========================\n\n")
	
	// 任务类型识别
	taskType := identifyTaskType(taskLower)
	plan.WriteString(fmt.Sprintf("📋 任务类型: %s\n\n", taskType))
	
	// 根据任务类型生成具体计划
	switch {
	case strings.Contains(taskLower, "小说") || strings.Contains(taskLower, "写作") || strings.Contains(taskLower, "大纲"):
		plan.WriteString(generateNovelWritingPlan(taskDesc))
	case strings.Contains(taskLower, "分析") || strings.Contains(taskLower, "检查") || strings.Contains(taskLower, "评价"):
		plan.WriteString(generateAnalysisPlan(taskDesc))
	case strings.Contains(taskLower, "修改") || strings.Contains(taskLower, "改进") || strings.Contains(taskLower, "优化"):
		plan.WriteString(generateImprovementPlan(taskDesc))
	case strings.Contains(taskLower, "创建") || strings.Contains(taskLower, "生成") || strings.Contains(taskLower, "制作"):
		plan.WriteString(generateCreationPlan(taskDesc))
	default:
		plan.WriteString(generateGeneralPlan(taskDesc))
	}
	
	plan.WriteString("\n🎯 执行建议:\n")
	plan.WriteString("• 按顺序执行各步骤，确保每步完成后再进行下一步\n")
	plan.WriteString("• 在关键节点进行质量检查和用户确认\n")
	plan.WriteString("• 保持文件备份，便于回滚修改\n")
	plan.WriteString("• 及时记录重要发现和改进想法\n")
	
	return plan.String()
}

func identifyTaskType(taskDesc string) string {
	switch {
	case strings.Contains(taskDesc, "小说") || strings.Contains(taskDesc, "写作"):
		return "小说创作类"
	case strings.Contains(taskDesc, "分析") || strings.Contains(taskDesc, "检查"):
		return "分析评估类"
	case strings.Contains(taskDesc, "修改") || strings.Contains(taskDesc, "改进"):
		return "改进优化类"
	case strings.Contains(taskDesc, "创建") || strings.Contains(taskDesc, "生成"):
		return "创建生成类"
	default:
		return "综合处理类"
	}
}

func generateNovelWritingPlan(taskDesc string) string {
	plan := `🔥 小说创作专业流程:

📚 第一阶段：基础准备
1. 【环境分析】使用 list_files 了解现有文件结构
2. 【设定收集】读取所有相关设定文件（世界观、角色、门派等）
3. 【一致性检查】交叉验证各设定文件的逻辑一致性
4. 【缺失识别】发现设定中的空白和不足之处

📖 第二阶段：内容规划  
5. 【大纲框架】基于现有设定构建故事主线
6. 【角色弧线】设计主要角色的成长轨迹
7. 【冲突设计】安排主要矛盾和转折点
8. 【章节划分】将故事分解为具体章节

✍️ 第三阶段：创作执行
9. 【详细大纲】为每个章节制定详细内容
10. 【写作风格】确定叙述风格和文字特色  
11. 【质量把控】设置检查点和评估标准
12. 【迭代完善】根据反馈持续优化

`
	return plan
}

func generateAnalysisPlan(taskDesc string) string {
	plan := `🔍 深度分析专业流程:

🎯 第一阶段：信息收集
1. 【目标文件识别】确定需要分析的具体文件
2. 【相关文件发现】查找所有相关联的支持文件
3. 【完整内容读取】获取所有相关文件的完整内容
4. 【背景信息整理】建立完整的上下文环境

🧠 第二阶段：深度分析
5. 【结构分析】评估内容的组织结构和逻辑
6. 【质量评估】从多个维度进行质量判断
7. 【一致性检查】发现内部矛盾和逻辑问题
8. 【完整性验证】识别遗漏和不足之处

💡 第三阶段：洞察提炼
9. 【优势识别】发现内容的亮点和特色
10. 【问题诊断】准确定位具体问题所在
11. 【改进建议】提供针对性的改进方案
12. 【最佳实践】结合行业标准给出专业建议

`
	return plan
}

func generateImprovementPlan(taskDesc string) string {
	plan := `🚀 改进优化专业流程:

📊 第一阶段：现状评估
1. 【现有内容审视】全面了解当前状况
2. 【问题识别定位】准确诊断具体问题  
3. 【影响范围分析】评估修改的连锁效应
4. 【优先级排序】确定改进的先后顺序

🔧 第二阶段：方案设计
5. 【改进策略制定】设计具体的优化方案
6. 【风险评估】预判可能的问题和风险
7. 【备份准备】确保可以安全回滚
8. 【测试计划】设计验证改进效果的方法

⚡ 第三阶段：执行验证
9. 【分步实施】按计划逐步执行改进
10. 【实时监控】跟踪改进过程和效果
11. 【质量验证】确保改进达到预期目标
12. 【文档更新】记录改进过程和结果

`
	return plan
}

func generateCreationPlan(taskDesc string) string {
	plan := `🎨 创建生成专业流程:

🎯 第一阶段：需求分析
1. 【目标明确】准确理解创建要求和目标
2. 【规格定义】确定具体的格式和标准
3. 【素材收集】收集所需的参考资料
4. 【约束识别】了解限制条件和边界

🏗️ 第二阶段：设计规划
5. 【结构设计】规划内容的整体架构
6. 【模板选择】确定最适合的创建模式
7. 【质量标准】设定评估和验收标准
8. 【时间规划】合理安排创建进度

🚀 第三阶段：创建执行
9. 【内容生成】按设计创建具体内容
10. 【质量检查】验证内容的准确性和完整性
11. 【格式调整】确保符合规范要求
12. 【最终验收】完成创建并交付成果

`
	return plan
}

func generateGeneralPlan(taskDesc string) string {
	plan := `⚙️ 综合处理标准流程:

🔍 第一阶段：需求理解
1. 【任务分解】将复杂任务拆分为子任务
2. 【信息收集】获取完成任务所需的所有信息
3. 【依赖分析】识别任务间的依赖关系
4. 【资源评估】确定所需的工具和资源

📋 第二阶段：执行规划
5. 【优先级排序】确定最优的执行顺序
6. 【方法选择】为每个子任务选择最佳方法
7. 【质量控制】设置检查点和验证标准
8. 【风险管控】识别并准备应对潜在问题

✅ 第三阶段：执行验证
9. 【分步执行】按计划逐步完成各子任务
10. 【进度跟踪】监控执行进度和质量
11. 【结果验证】确保每步都达到预期效果
12. 【总结反馈】记录经验和改进建议

`
	return plan
}