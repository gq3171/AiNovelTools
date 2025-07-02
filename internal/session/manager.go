package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AiNovelTools/internal/ai"
	"github.com/AiNovelTools/internal/config"
	"github.com/AiNovelTools/internal/tools"
	"github.com/google/uuid"
)

type Manager struct {
	currentSession *Session
	sessionDir     string
}

type Session struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	Messages  []ai.Message  `json:"messages"`
	Context   SessionContext `json:"context"`
}

type SessionContext struct {
	WorkingDirectory string            `json:"working_directory"`
	Environment     map[string]string `json:"environment"`
	ProjectInfo     ProjectInfo       `json:"project_info"`
}

type ProjectInfo struct {
	Name        string   `json:"name"`
	Language    string   `json:"language"`
	Framework   string   `json:"framework"`
	Files       []string `json:"files"`
	Description string   `json:"description"`
}

func NewManager() *Manager {
	configDir, err := config.GetConfigDir()
	if err != nil {
		// 回退到默认路径
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".ai-assistant")
	}
	
	sessionDir := filepath.Join(configDir, "sessions")
	os.MkdirAll(sessionDir, 0755)

	return &Manager{
		sessionDir: sessionDir,
	}
}

func (m *Manager) GetCurrentSession() *Session {
	if m.currentSession == nil {
		m.currentSession = m.NewSession("default")
	}
	return m.currentSession
}

func (m *Manager) NewSession(name string) *Session {
	wd, _ := os.Getwd()
	
	session := &Session{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []ai.Message{},
		Context: SessionContext{
			WorkingDirectory: wd,
			Environment:     make(map[string]string),
			ProjectInfo:     ProjectInfo{},
		},
	}
	
	// 分析当前项目
	session.analyzeProject()
	
	m.currentSession = session
	return session
}

func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, ai.Message{
		Role:    role,
		Content: content,
	})
	s.UpdatedAt = time.Now()
}

func (s *Session) AddToolResult(result tools.ToolResult) {
	content := result.Result
	if result.Error != nil {
		content = fmt.Sprintf("Error: %v", result.Error)
	}
	
	s.AddMessage("tool", fmt.Sprintf("Tool: %s\nResult: %s", result.ToolName, content))
}

func (s *Session) GetMessages() []ai.Message {
	return s.Messages
}

func (s *Session) analyzeProject() {
	wd := s.Context.WorkingDirectory
	
	// 检查是否是Go项目
	if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
		s.Context.ProjectInfo.Language = "Go"
		s.Context.ProjectInfo.Framework = "Go"
		
		// 读取go.mod获取项目名
		if content, err := os.ReadFile(filepath.Join(wd, "go.mod")); err == nil {
			lines := string(content)
			if len(lines) > 0 {
				// 简单解析module名
				for _, line := range []string{lines} {
					if len(line) > 7 && line[:7] == "module " {
						s.Context.ProjectInfo.Name = line[7:]
						break
					}
				}
			}
		}
	}
	
	// 检查是否是Node.js项目
	if _, err := os.Stat(filepath.Join(wd, "package.json")); err == nil {
		s.Context.ProjectInfo.Language = "JavaScript/TypeScript"
		
		// 读取package.json
		if content, err := os.ReadFile(filepath.Join(wd, "package.json")); err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(content, &pkg) == nil {
				if name, ok := pkg["name"].(string); ok {
					s.Context.ProjectInfo.Name = name
				}
				
				// 检测框架
				if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
					if _, hasReact := deps["react"]; hasReact {
						s.Context.ProjectInfo.Framework = "React"
					} else if _, hasVue := deps["vue"]; hasVue {
						s.Context.ProjectInfo.Framework = "Vue"
					} else if _, hasAngular := deps["@angular/core"]; hasAngular {
						s.Context.ProjectInfo.Framework = "Angular"
					}
				}
			}
		}
	}
	
	// 扫描项目文件
	s.scanProjectFiles()
}

func (s *Session) scanProjectFiles() {
	wd := s.Context.WorkingDirectory
	var files []string
	
	err := filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		
		// 跳过隐藏文件和目录
		if info.Name()[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		
		// 跳过常见的无关目录
		skipDirs := []string{"node_modules", "vendor", "target", "build", "dist"}
		for _, skip := range skipDirs {
			if info.IsDir() && info.Name() == skip {
				return filepath.SkipDir
			}
		}
		
		if !info.IsDir() {
			relPath, _ := filepath.Rel(wd, path)
			files = append(files, relPath)
		}
		
		return nil
	})
	
	if err == nil && len(files) < 100 { // 限制文件数量
		s.Context.ProjectInfo.Files = files
	}
}

func (m *Manager) SaveSession(session *Session) error {
	filePath := filepath.Join(m.sessionDir, session.ID+".json")
	
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}
	
	return os.WriteFile(filePath, data, 0644)
}

func (m *Manager) LoadSession(sessionID string) (*Session, error) {
	filePath := filepath.Join(m.sessionDir, sessionID+".json")
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}
	
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	
	return &session, nil
}

func (m *Manager) ListSessions() ([]Session, error) {
	files, err := os.ReadDir(m.sessionDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}
	
	var sessions []Session
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			sessionID := file.Name()[:len(file.Name())-5] // remove .json
			session, err := m.LoadSession(sessionID)
			if err == nil {
				sessions = append(sessions, *session)
			}
		}
	}
	
	return sessions, nil
}

func (m *Manager) SwitchSession(sessionID string) error {
	session, err := m.LoadSession(sessionID)
	if err != nil {
		return err
	}
	
	m.currentSession = session
	return nil
}

func (m *Manager) DeleteSession(sessionID string) error {
	filePath := filepath.Join(m.sessionDir, sessionID+".json")
	
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("session file not found: %s", sessionID)
	}
	
	// 删除会话文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete session file: %w", err)
	}
	
	return nil
}