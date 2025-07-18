package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/AiNovelTools/internal/ai"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AI       ai.Config     `yaml:"ai"`
	UI       UIConfig      `yaml:"ui"`
	Features Features      `yaml:"features"`
	Writing  WritingConfig `yaml:"writing"`
}

type UIConfig struct {
	Theme           string `yaml:"theme"`
	ShowTokens      bool   `yaml:"show_tokens"`
	AutoSave        bool   `yaml:"auto_save"`
	MaxHistory      int    `yaml:"max_history"`
	Language        string `yaml:"language"`          // 界面语言
	DateFormat      string `yaml:"date_format"`       // 日期格式
	ShowWelcome     bool   `yaml:"show_welcome"`      // 显示欢迎信息
	CompactMode     bool   `yaml:"compact_mode"`      // 紧凑模式
	ShowProgress    bool   `yaml:"show_progress"`     // 显示进度条
	AnimationSpeed  string `yaml:"animation_speed"`   // 动画速度
}

type Features struct {
	EnableFileWatch bool     `yaml:"enable_file_watch"`
	AllowedCommands []string `yaml:"allowed_commands"`
	SafeMode        bool     `yaml:"safe_mode"`
	AutoCompletion  bool     `yaml:"auto_completion"`   // 自动补全
	SmartSuggestions bool    `yaml:"smart_suggestions"` // 智能建议
	FileIndexing    bool     `yaml:"file_indexing"`     // 文件索引
	BackupFiles     bool     `yaml:"backup_files"`      // 文件备份
	MaxFileSize     int      `yaml:"max_file_size"`     // 最大文件大小(MB)
}

type WritingConfig struct {
	AutoSaveInterval    int      `yaml:"auto_save_interval"`     // 自动保存间隔(秒)
	DefaultGenre        string   `yaml:"default_genre"`          // 默认类型
	PreferredAIModel    string   `yaml:"preferred_ai_model"`     // 首选AI模型
	WritingStyle        string   `yaml:"writing_style"`          // 写作风格
	TargetWordsPerDay   int      `yaml:"target_words_per_day"`   // 每日目标字数
	EnableSpellCheck    bool     `yaml:"enable_spell_check"`     // 拼写检查
	ShowWordCount       bool     `yaml:"show_word_count"`        // 显示字数统计
	RememberContext     bool     `yaml:"remember_context"`       // 记住上下文
	MaxContextLength    int      `yaml:"max_context_length"`     // 最大上下文长度
}

func Load() (*Config, error) {
	configDir, configFile, err := getConfigPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get config paths: %w", err)
	}
	
	// 创建默认配置如果不存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := createDefaultConfig(configDir, configFile); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}
	
	// 读取配置文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// 确保Models map已初始化
	if config.AI.Models == nil {
		config.AI.Models = make(map[ai.Provider]ai.ModelConfig)
	}
	
	// 确保所有提供商都有默认配置
	if _, exists := config.AI.Models[ai.ProviderZhipu]; !exists {
		config.AI.Models[ai.ProviderZhipu] = ai.ModelConfig{
			APIKey:  "",
			BaseURL: "https://open.bigmodel.cn/api/paas/v4",
			Model:   "glm-4",
		}
	}
	
	if _, exists := config.AI.Models[ai.ProviderDeepseek]; !exists {
		config.AI.Models[ai.ProviderDeepseek] = ai.ModelConfig{
			APIKey:  "",
			BaseURL: "https://api.deepseek.com",
			Model:   "deepseek-chat",
		}
	}
	
	// 从环境变量覆盖设置
	if zhipuKey := os.Getenv("ZHIPU_API_KEY"); zhipuKey != "" {
		if config.AI.Models[ai.ProviderZhipu].APIKey == "" {
			modelConfig := config.AI.Models[ai.ProviderZhipu]
			modelConfig.APIKey = zhipuKey
			config.AI.Models[ai.ProviderZhipu] = modelConfig
		}
	}
	
	if deepseekKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekKey != "" {
		if config.AI.Models[ai.ProviderDeepseek].APIKey == "" {
			modelConfig := config.AI.Models[ai.ProviderDeepseek]
			modelConfig.APIKey = deepseekKey
			config.AI.Models[ai.ProviderDeepseek] = modelConfig
		}
	}
	
	// 兼容旧的环境变量
	if apiKey := os.Getenv("AI_API_KEY"); apiKey != "" {
		provider := config.AI.Provider
		if modelConfig, exists := config.AI.Models[provider]; exists && modelConfig.APIKey == "" {
			modelConfig.APIKey = apiKey
			config.AI.Models[provider] = modelConfig
		}
	}
	
	if provider := os.Getenv("AI_PROVIDER"); provider != "" {
		config.AI.Provider = ai.Provider(provider)
	}
	
	return &config, nil
}

func createDefaultConfig(configDir, configFile string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	
	defaultConfig := Config{
		AI: ai.Config{
			Provider: ai.ProviderZhipu,
			Models: map[ai.Provider]ai.ModelConfig{
				ai.ProviderZhipu: {
					APIKey:  "",
					BaseURL: "https://open.bigmodel.cn/api/paas/v4",
					Model:   "glm-4",
				},
				ai.ProviderDeepseek: {
					APIKey:  "",
					BaseURL: "https://api.deepseek.com",
					Model:   "deepseek-chat",
				},
			},
			MaxTokens:   2048,
			Temperature: 0.7,
		},
		UI: UIConfig{
			Theme:      "dark",
			ShowTokens: false,
			AutoSave:   true,
			MaxHistory: 100,
		},
		Features: Features{
			EnableFileWatch: true,
			AllowedCommands: []string{"ls", "cat", "grep", "find", "git"},
			SafeMode:        true,
		},
	}
	
	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}
	
	return os.WriteFile(configFile, data, 0644)
}

func (c *Config) Save() error {
	_, configFile, err := getConfigPaths()
	if err != nil {
		return fmt.Errorf("failed to get config paths: %w", err)
	}
	
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	return os.WriteFile(configFile, data, 0644)
}

// getConfigPaths 获取跨平台的配置文件路径
func getConfigPaths() (configDir, configFile string, err error) {
	var homeDir string
	
	// 优先从环境变量获取配置目录
	if configDir = os.Getenv("AI_ASSISTANT_CONFIG_DIR"); configDir != "" {
		configFile = filepath.Join(configDir, "config.yaml")
		return configDir, configFile, nil
	}
	
	// 获取用户主目录
	homeDir, err = os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// 根据操作系统设置配置目录
	switch runtime.GOOS {
	case "windows":
		// Windows: %APPDATA%\AI-Assistant
		appData := os.Getenv("APPDATA")
		if appData != "" {
			configDir = filepath.Join(appData, "AI-Assistant")
		} else {
			configDir = filepath.Join(homeDir, "AI-Assistant")
		}
	case "darwin":
		// macOS: ~/Library/Application Support/AI-Assistant
		configDir = filepath.Join(homeDir, "Library", "Application Support", "AI-Assistant")
	default:
		// Linux/Unix: ~/.ai-assistant
		configDir = filepath.Join(homeDir, ".ai-assistant")
	}
	
	configFile = filepath.Join(configDir, "config.yaml")
	return configDir, configFile, nil
}

// GetConfigDir 获取配置目录路径（用于外部调用）
func GetConfigDir() (string, error) {
	configDir, _, err := getConfigPaths()
	return configDir, err
}