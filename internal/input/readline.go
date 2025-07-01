package input

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chzyer/readline"
	"github.com/AiNovelTools/internal/config"
)

type Manager struct {
	rl          *readline.Instance
	historyFile string
	commands    []string
}

// 预定义的命令列表（用于自动补全）
var builtinCommands = []string{
	"/help", "/clear", "/status", "/sessions", "/new", "/switch", "/config", "/exit", "/quit",
	"/config show", "/config path", "/config set", "/config edit",
	"/switch zhipu", "/switch deepseek",
}

func NewManager() (*Manager, error) {
	// 获取历史文件路径
	configDir, err := config.GetConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	historyFile := filepath.Join(configDir, "history")

	// 创建自动补全函数
	completer := readline.NewPrefixCompleter(
		readline.PcItem("/help"),
		readline.PcItem("/clear"),
		readline.PcItem("/status"),
		readline.PcItem("/sessions"),
		readline.PcItem("/new"),
		readline.PcItem("/switch",
			readline.PcItem("zhipu"),
			readline.PcItem("deepseek"),
		),
		readline.PcItem("/config",
			readline.PcItem("show"),
			readline.PcItem("path"),
			readline.PcItem("set",
				readline.PcItem("zhipu.api_key"),
				readline.PcItem("deepseek.api_key"),
				readline.PcItem("ai.provider"),
			),
			readline.PcItem("edit"),
		),
		readline.PcItem("/exit"),
		readline.PcItem("/quit"),
	)

	// 配置readline
	cfg := &readline.Config{
		Prompt:              "\033[36m[zhipu] ❯ \033[0m", // 默认提示符
		HistoryFile:         historyFile,
		AutoComplete:        completer,
		InterruptPrompt:     "^C",
		EOFPrompt:           "exit",
		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create readline: %w", err)
	}

	return &Manager{
		rl:          rl,
		historyFile: historyFile,
		commands:    builtinCommands,
	}, nil
}

// 过滤输入字符
func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ: // 禁用Ctrl+Z
		return r, false
	}
	return r, true
}

func (m *Manager) SetPrompt(prompt string) {
	m.rl.SetPrompt(prompt)
}

// 设置模型提示符
func (m *Manager) SetModelPrompt(modelName string) {
	prompt := fmt.Sprintf("\033[36m[%s] ❯ \033[0m", modelName)
	m.rl.SetPrompt(prompt)
}

func (m *Manager) ReadLine() (string, error) {
	line, err := m.rl.Readline()
	if err == readline.ErrInterrupt {
		if len(line) == 0 {
			return "", io.EOF
		} else {
			return "", nil // 清空当前行，继续
		}
	} else if err == io.EOF {
		return "", io.EOF
	}
	return strings.TrimSpace(line), err
}

func (m *Manager) Close() error {
	return m.rl.Close()
}

// 添加命令到历史（用于动态命令补全）
func (m *Manager) AddCommand(cmd string) {
	for _, existing := range m.commands {
		if existing == cmd {
			return
		}
	}
	m.commands = append(m.commands, cmd)
	sort.Strings(m.commands)
}

// 获取命令建议
func (m *Manager) GetSuggestions(prefix string) []string {
	var suggestions []string
	for _, cmd := range m.commands {
		if strings.HasPrefix(cmd, prefix) {
			suggestions = append(suggestions, cmd)
		}
	}
	return suggestions
}

// 设置多行模式提示符
func (m *Manager) SetMultiLinePrompt() {
	m.rl.SetPrompt("\033[36m│ \033[0m")
}

// 重置正常提示符
func (m *Manager) ResetPrompt(modelName string) {
	prompt := fmt.Sprintf("\033[36m[%s] ❯ \033[0m", modelName)
	m.rl.SetPrompt(prompt)
}

// 打印欢迎信息
func (m *Manager) PrintWelcome() {
	fmt.Println("\033[1;34m╔══════════════════════════════════════╗\033[0m")
	fmt.Println("\033[1;34m║           AI 智能助手 v1.0           ║\033[0m")
	fmt.Println("\033[1;34m║      支持智谱AI & Deepseek模型       ║\033[0m")
	fmt.Println("\033[1;34m╚══════════════════════════════════════╝\033[0m")
	fmt.Println()
	fmt.Println("\033[90m💡 使用提示:\033[0m")
	fmt.Println("\033[90m  • 使用 ↑↓ 方向键浏览历史命令\033[0m")
	fmt.Println("\033[90m  • 使用 Tab 键自动补全命令\033[0m")
	fmt.Println("\033[90m  • 输入 '/help' 查看所有命令\033[0m")
	fmt.Println("\033[90m  • 使用 Ctrl+C 中断，Ctrl+D 退出\033[0m")
	fmt.Println()
}

// 清屏
func (m *Manager) ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

// 显示加载动画
func (m *Manager) ShowLoading(message string) {
	fmt.Printf("\033[33m⏳ %s...\033[0m", message)
}

// 隐藏加载动画
func (m *Manager) HideLoading() {
	fmt.Print("\r\033[K") // 清除当前行
}

// 打印成功消息
func (m *Manager) PrintSuccess(message string) {
	fmt.Printf("\033[32m✅ %s\033[0m\n", message)
}

// 打印错误消息
func (m *Manager) PrintError(message string) {
	fmt.Printf("\033[31m❌ Error: %s\033[0m\n", message)
}

// 打印警告消息
func (m *Manager) PrintWarning(message string) {
	fmt.Printf("\033[33m⚠️  Warning: %s\033[0m\n", message)
}

// 打印信息消息
func (m *Manager) PrintInfo(message string) {
	fmt.Printf("\033[34mℹ️  %s\033[0m\n", message)
}

// 打印AI响应
func (m *Manager) PrintAIResponse(response string) {
	fmt.Printf("\033[32m🤖 %s\033[0m\n", response)
}