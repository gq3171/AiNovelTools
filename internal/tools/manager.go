package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	m.RegisterTool(&ListFilesTool{})
	m.RegisterTool(&ExecuteCommandTool{})
	m.RegisterTool(&SearchTool{})
	
	return m
}

func (m *Manager) RegisterTool(tool Tool) {
	m.tools[tool.Name()] = tool
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
	textExts := []string{".go", ".txt", ".md", ".json", ".yaml", ".yml", ".toml", ".js", ".ts", ".py", ".java", ".c", ".cpp", ".h", ".hpp"}
	
	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}
	
	return false
}