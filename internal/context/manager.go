package context

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ContextManager 智能上下文管理器
type ContextManager struct {
	currentProject *ProjectContext
	workHistory    []WorkingSession
	preferences    *UserPreferences
	mutex          sync.RWMutex
}

// ProjectContext 项目上下文信息
type ProjectContext struct {
	Path           string            `json:"path"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	LastAccessed   time.Time         `json:"last_accessed"`
	RecentFiles    []string          `json:"recent_files"`
	Bookmarks      []string          `json:"bookmarks"`
	CustomSettings map[string]string `json:"custom_settings"`
}

// WorkingSession 工作会话记录
type WorkingSession struct {
	Timestamp   time.Time `json:"timestamp"`
	ProjectPath string    `json:"project_path"`
	Actions     []string  `json:"actions"`
	Duration    int64     `json:"duration"` // 秒
}

// UserPreferences 用户偏好设置
type UserPreferences struct {
	PreferredEditor     string            `json:"preferred_editor"`
	DefaultFileEncoding string            `json:"default_file_encoding"`
	AutoSaveContext    bool              `json:"auto_save_context"`
	MaxHistoryDays     int               `json:"max_history_days"`
	CustomCommands     map[string]string `json:"custom_commands"`
}

// NewContextManager 创建新的上下文管理器
func NewContextManager() *ContextManager {
	return &ContextManager{
		workHistory: make([]WorkingSession, 0),
		preferences: &UserPreferences{
			PreferredEditor:     "default",
			DefaultFileEncoding: "utf-8",
			AutoSaveContext:    true,
			MaxHistoryDays:     30,
			CustomCommands:     make(map[string]string),
		},
	}
}

// LoadContext 加载保存的上下文
func (cm *ContextManager) LoadContext() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	
	contextFile := filepath.Join(configDir, "ai-assistant-context.json")
	
	if _, err := os.Stat(contextFile); os.IsNotExist(err) {
		// 文件不存在，使用默认配置
		return nil
	}
	
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return fmt.Errorf("failed to read context file: %w", err)
	}
	
	var contextData struct {
		CurrentProject *ProjectContext   `json:"current_project"`
		WorkHistory    []WorkingSession  `json:"work_history"`
		Preferences    *UserPreferences  `json:"preferences"`
	}
	
	if err := json.Unmarshal(data, &contextData); err != nil {
		return fmt.Errorf("failed to parse context file: %w", err)
	}
	
	cm.currentProject = contextData.CurrentProject
	cm.workHistory = contextData.WorkHistory
	if contextData.Preferences != nil {
		cm.preferences = contextData.Preferences
	}
	
	return nil
}

// SaveContext 保存当前上下文
func (cm *ContextManager) SaveContext() error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if !cm.preferences.AutoSaveContext {
		return nil
	}
	
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}
	
	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	contextData := struct {
		CurrentProject *ProjectContext   `json:"current_project"`
		WorkHistory    []WorkingSession  `json:"work_history"`
		Preferences    *UserPreferences  `json:"preferences"`
	}{
		CurrentProject: cm.currentProject,
		WorkHistory:    cm.workHistory,
		Preferences:    cm.preferences,
	}
	
	data, err := json.MarshalIndent(contextData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}
	
	contextFile := filepath.Join(configDir, "ai-assistant-context.json")
	if err := os.WriteFile(contextFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}
	
	return nil
}

// UpdateCurrentProject 更新当前项目上下文
func (cm *ContextManager) UpdateCurrentProject(projectPath string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	// 检测项目类型
	projectType := detectProjectType(projectPath)
	projectName := filepath.Base(projectPath)
	
	cm.currentProject = &ProjectContext{
		Path:           projectPath,
		Name:           projectName,
		Type:           projectType,
		LastAccessed:   time.Now(),
		RecentFiles:    make([]string, 0),
		Bookmarks:      make([]string, 0),
		CustomSettings: make(map[string]string),
	}
	
	// 自动保存
	go func() {
		if err := cm.SaveContext(); err != nil {
			fmt.Printf("Warning: Failed to save context: %v\n", err)
		}
	}()
	
	return nil
}

// AddRecentFile 添加最近访问的文件
func (cm *ContextManager) AddRecentFile(filePath string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	if cm.currentProject == nil {
		return
	}
	
	// 移除已存在的相同文件
	for i, file := range cm.currentProject.RecentFiles {
		if file == filePath {
			cm.currentProject.RecentFiles = append(
				cm.currentProject.RecentFiles[:i],
				cm.currentProject.RecentFiles[i+1:]...,
			)
			break
		}
	}
	
	// 添加到开头
	cm.currentProject.RecentFiles = append([]string{filePath}, cm.currentProject.RecentFiles...)
	
	// 限制最大数量
	if len(cm.currentProject.RecentFiles) > 10 {
		cm.currentProject.RecentFiles = cm.currentProject.RecentFiles[:10]
	}
}

// RecordAction 记录用户操作
func (cm *ContextManager) RecordAction(action string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	
	now := time.Now()
	
	// 如果是同一个项目的连续操作，添加到当前会话
	if len(cm.workHistory) > 0 {
		lastSession := &cm.workHistory[len(cm.workHistory)-1]
		if lastSession.ProjectPath == cm.GetCurrentProjectPath() &&
			now.Sub(lastSession.Timestamp) < 30*time.Minute {
			lastSession.Actions = append(lastSession.Actions, action)
			lastSession.Duration = int64(now.Sub(lastSession.Timestamp).Seconds())
			return
		}
	}
	
	// 创建新会话
	session := WorkingSession{
		Timestamp:   now,
		ProjectPath: cm.GetCurrentProjectPath(),
		Actions:     []string{action},
		Duration:    0,
	}
	
	cm.workHistory = append(cm.workHistory, session)
	
	// 清理旧记录
	cm.cleanupOldHistory()
}

// GetCurrentProjectPath 获取当前项目路径
func (cm *ContextManager) GetCurrentProjectPath() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	if cm.currentProject == nil {
		currentDir, _ := os.Getwd()
		return currentDir
	}
	
	return cm.currentProject.Path
}

// GetContextSummary 获取上下文摘要
func (cm *ContextManager) GetContextSummary() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	var summary strings.Builder
	summary.WriteString("🧠 === AI Assistant Context Summary === 🧠\n\n")
	
	// 当前项目信息
	if cm.currentProject != nil {
		summary.WriteString("📁 Current Project:\n")
		summary.WriteString(fmt.Sprintf("  Name: %s\n", cm.currentProject.Name))
		summary.WriteString(fmt.Sprintf("  Path: %s\n", cm.currentProject.Path))
		summary.WriteString(fmt.Sprintf("  Type: %s\n", cm.currentProject.Type))
		summary.WriteString(fmt.Sprintf("  Last Accessed: %s\n", cm.currentProject.LastAccessed.Format("2006-01-02 15:04:05")))
		
		if len(cm.currentProject.RecentFiles) > 0 {
			summary.WriteString("  Recent Files:\n")
			for i, file := range cm.currentProject.RecentFiles {
				if i >= 5 { // 只显示前5个
					break
				}
				summary.WriteString(fmt.Sprintf("    - %s\n", filepath.Base(file)))
			}
		}
		summary.WriteString("\n")
	}
	
	// 工作历史
	if len(cm.workHistory) > 0 {
		summary.WriteString("📊 Recent Activity:\n")
		for i := len(cm.workHistory) - 1; i >= 0 && len(cm.workHistory)-1-i < 3; i-- {
			session := cm.workHistory[i]
			summary.WriteString(fmt.Sprintf("  %s - %s (%d actions)\n",
				session.Timestamp.Format("15:04"),
				filepath.Base(session.ProjectPath),
				len(session.Actions)))
		}
		summary.WriteString("\n")
	}
	
	// 用户偏好
	summary.WriteString("⚙️ Preferences:\n")
	summary.WriteString(fmt.Sprintf("  Editor: %s\n", cm.preferences.PreferredEditor))
	summary.WriteString(fmt.Sprintf("  Encoding: %s\n", cm.preferences.DefaultFileEncoding))
	summary.WriteString(fmt.Sprintf("  Auto-save: %v\n", cm.preferences.AutoSaveContext))
	
	return summary.String()
}

// GetWorkingSuggestions 基于上下文提供工作建议
func (cm *ContextManager) GetWorkingSuggestions() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	
	suggestions := []string{}
	
	if cm.currentProject == nil {
		suggestions = append(suggestions, "🔍 Use 'get_working_context' to analyze current environment")
		suggestions = append(suggestions, "📁 Use 'get_project_info' to understand project structure")
		return suggestions
	}
	
	// 基于项目类型的建议
	if strings.Contains(cm.currentProject.Type, "Go") {
		suggestions = append(suggestions, "🏃 Run 'go run main.go' to start the application")
		suggestions = append(suggestions, "🔧 Use 'go mod tidy' to clean dependencies")
		suggestions = append(suggestions, "🏗️ Use 'go build' to compile the project")
	}
	
	if strings.Contains(cm.currentProject.Type, "Node.js") {
		suggestions = append(suggestions, "📦 Run 'npm install' to install dependencies")
		suggestions = append(suggestions, "🚀 Use 'npm start' to start development server")
	}
	
	// 基于最近文件的建议
	if len(cm.currentProject.RecentFiles) > 0 {
		suggestions = append(suggestions, "📄 Continue editing recent files:")
		for i, file := range cm.currentProject.RecentFiles {
			if i >= 3 {
				break
			}
			suggestions = append(suggestions, fmt.Sprintf("   - %s", filepath.Base(file)))
		}
	}
	
	// 基于工作历史的建议
	if len(cm.workHistory) > 0 {
		lastSession := cm.workHistory[len(cm.workHistory)-1]
		if len(lastSession.Actions) > 0 {
			lastAction := lastSession.Actions[len(lastSession.Actions)-1]
			if strings.Contains(lastAction, "read_file") {
				suggestions = append(suggestions, "✏️ Consider editing the file you just read")
			}
			if strings.Contains(lastAction, "search") {
				suggestions = append(suggestions, "🔄 Use 'replace_text' to modify found content")
			}
		}
	}
	
	return suggestions
}

// 辅助函数
func (cm *ContextManager) cleanupOldHistory() {
	maxAge := time.Duration(cm.preferences.MaxHistoryDays) * 24 * time.Hour
	cutoff := time.Now().Add(-maxAge)
	
	filtered := make([]WorkingSession, 0)
	for _, session := range cm.workHistory {
		if session.Timestamp.After(cutoff) {
			filtered = append(filtered, session)
		}
	}
	
	cm.workHistory = filtered
}

func getConfigDir() (string, error) {
	// 获取用户配置目录
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "ai-assistant"), nil
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	switch {
	case os.Getenv("OS") == "Windows_NT":
		return filepath.Join(homeDir, "AppData", "Roaming", "AI-Assistant"), nil
	case fileExists("/System/Library/CoreServices/SystemVersion.plist"): // macOS
		return filepath.Join(homeDir, "Library", "Application Support", "AI-Assistant"), nil
	default: // Linux
		return filepath.Join(homeDir, ".config", "ai-assistant"), nil
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detectProjectType(projectPath string) string {
	projectIndicators := map[string]string{
		"go.mod":         "Go Module Project",
		"package.json":   "Node.js Project",
		"pom.xml":        "Java Maven Project",
		"Cargo.toml":     "Rust Project",
		"requirements.txt": "Python Project",
	}
	
	entries, err := os.ReadDir(projectPath)
	if err != nil {
		return "Unknown"
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			if projectType, exists := projectIndicators[entry.Name()]; exists {
				return projectType
			}
		}
	}
	
	return "Generic Project"
}