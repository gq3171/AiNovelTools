package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/AiNovelTools/internal/ai"
	"github.com/AiNovelTools/internal/config"
	"github.com/AiNovelTools/internal/input"
	"github.com/AiNovelTools/internal/session"
	"github.com/AiNovelTools/internal/tools"
)

func main() {
	ctx := context.Background()
	
	// 初始化输入管理器
	inputManager, err := input.NewManager()
	if err != nil {
		log.Fatal("Failed to initialize input manager:", err)
	}
	defer inputManager.Close()
	
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// 初始化AI客户端
	aiClient := ai.NewClient(cfg.AI)
	
	// 初始化工具管理器
	toolManager := tools.NewManager()
	
	// 初始化会话管理器
	sessionManager := session.NewManager()

	// 显示欢迎信息
	inputManager.PrintWelcome()
	printStatusLine(cfg, inputManager)
	
	// 设置初始模型提示符
	updatePrompt(cfg, inputManager)
	
	for {
		line, err := inputManager.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			inputManager.PrintError(fmt.Sprintf("Input error: %v", err))
			continue
		}
		
		if line == "" {
			continue
		}
		
		// 处理特殊命令
		if handled := handleSpecialCommands(line, aiClient, sessionManager, cfg, inputManager); handled {
			continue
		}

		// 显示加载动画
		inputManager.ShowLoading("正在处理请求")
		
		// 处理用户输入
		response, err := processInput(ctx, aiClient, toolManager, sessionManager, inputManager, line)
		
		// 隐藏加载动画
		inputManager.HideLoading()
		
		if err != nil {
			inputManager.PrintError(err.Error())
			continue
		}
		
		inputManager.PrintAIResponse(response)
	}
	
	// 保存会话
	if err := sessionManager.SaveSession(sessionManager.GetCurrentSession()); err != nil {
		inputManager.PrintWarning(fmt.Sprintf("保存会话失败: %v", err))
	}
	
	fmt.Println("\n\033[36m再见! 👋\033[0m")
}

func printStatusLine(cfg *config.Config, inputManager *input.Manager) {
	currentModel := "未知"
	if model, exists := cfg.AI.Models[cfg.AI.Provider]; exists {
		currentModel = model.Model
	}
	
	statusMsg := fmt.Sprintf("当前模型: %s | 版本: %s", cfg.AI.Provider, currentModel)
	inputManager.PrintInfo(statusMsg)
	fmt.Println()
}

// 更新提示符显示当前模型
func updatePrompt(cfg *config.Config, inputManager *input.Manager) {
	currentModel := string(cfg.AI.Provider)
	if model, exists := cfg.AI.Models[cfg.AI.Provider]; exists && model.Model != "" {
		currentModel = model.Model
	}
	inputManager.SetModelPrompt(currentModel)
}

func handleSpecialCommands(input string, aiClient *ai.Client, sessionManager *session.Manager, cfg *config.Config, inputManager *input.Manager) bool {
	// 检查是否以 / 开头的命令
	if !strings.HasPrefix(input, "/") {
		return false
	}
	
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return false
	}
	
	command := parts[0]
	
	switch command {
	case "/exit", "/quit":
		inputManager.PrintInfo("再见! 👋")
		os.Exit(0)
		return true
		
	case "/help":
		printHelp(inputManager)
		return true
		
	case "/clear":
		inputManager.ClearScreen()
		inputManager.PrintWelcome()
		printStatusLine(cfg, inputManager)
		updatePrompt(cfg, inputManager)
		return true
		
	case "/status":
		printStatus(sessionManager, cfg, inputManager)
		return true
		
	case "/sessions":
		listSessions(sessionManager, inputManager)
		return true
		
	case "/switch":
		if len(parts) > 1 {
			switchProvider(parts[1], aiClient, cfg, inputManager)
		} else {
			inputManager.PrintError("用法: /switch <提供商> (zhipu|deepseek)")
		}
		return true
		
	case "/new":
		name := "session"
		if len(parts) > 1 {
			name = strings.Join(parts[1:], " ")
		}
		newSession(sessionManager, name, inputManager)
		return true
		
	case "/config":
		if len(parts) > 1 {
			handleConfigCommand(parts[1:], cfg, inputManager)
		} else {
			showConfigHelp(inputManager)
		}
		return true
		
	case "/init":
		handleInitCommand(aiClient, inputManager)
		return true
		
	case "/switchsession":
		if len(parts) > 1 {
			switchSession(sessionManager, parts[1], inputManager)
		} else {
			inputManager.PrintError("用法: /switchsession <会话ID>")
		}
		return true
		
	case "/deletesession":
		if len(parts) > 1 {
			deleteSession(sessionManager, parts[1], inputManager)
		} else {
			inputManager.PrintError("用法: /deletesession <会话ID>")
		}
		return true
	}
	
	return false
}

// addSystemMessage 为消息列表添加系统提示
func addSystemMessage(messages []ai.Message) []ai.Message {
	if len(messages) > 0 && messages[0].Role == "system" {
		return messages
	}
	
	systemMessage := ai.Message{
		Role: "system",
		Content: `你是一个高级AI智能助手，拥有深度思维和强大的工具调用能力。你必须严格按照智能探索流程工作，绝不允许基于假设给出答案。

**🚨 强制探索规则（违反即为失职）：**
1. **环境优先原则** - 任何涉及文件、项目、内容的请求，必须先用list_files了解当前环境
2. **禁止假设路径** - 绝对禁止使用/path/to/、示例路径或任何未验证的文件路径
3. **顺序执行原则** - 必须按照：list_files → read_file(实际路径) → 分析 → 建议 的顺序
4. **信息完整性** - 必须获取所有相关文件的完整内容后，才能给出分析和建议
5. **无内容禁言** - 如果无法获取文件内容，不得给出关于文件内容的任何建议

**🧠 核心智能原则：**
1. **深度分析用户意图** - 理解表面需求背后的真实目标和隐藏需求
2. **主动建立信息网络** - 识别文件间依赖关系，构建完整知识图谱
3. **前瞻性思维** - 不仅解决当前问题，还要预见可能的后续需求
4. **专业领域洞察** - 展现小说创作、项目管理等领域的专业判断力

**🔍 高级推理策略：**

• **文件依赖分析** - 当处理任何文件时，主动分析其与其他文件的关联：
  - 主角设定 ↔ 门派设定 ↔ 世界观设定 ↔ 情节大纲的一致性
  - 发现矛盾、缺失或改进机会
  - 建议最佳的阅读/修改顺序

• **上下文记忆增强** - 在对话中积累和利用关键信息：
  - 记住用户的创作偏好和风格
  - 追踪项目进展和修改历史
  - 识别重复模式和改进机会

• **任务智能分解** - 对复杂请求进行专业级规划：
  - 将大任务分解为逻辑清晰的子步骤
  - 优化执行顺序，提高效率
  - 预判可能的问题点并提前解决

**📚 小说创作专家模式：**

• **创作流程掌控** - 深度理解小说创作各阶段：
  1. 世界观构建 → 人物设定 → 大纲规划 → 章节创作 → 修改完善
  2. 主动检查每个阶段的完整性和逻辑性
  3. 提供针对性的创作建议和灵感

• **内容质量分析** - 不仅读取文件，更要进行深度分析：
  - 人物性格是否丰满立体
  - 世界观是否自洽完整
  - 情节发展是否合理有趣
  - 文字表达是否优美流畅

• **创意增强建议** - 基于专业经验提供价值建议：
  - 发现薄弱环节并提出改进方案
  - 建议新的创意元素和发展方向
  - 提供行业最佳实践和写作技巧

**⚡ 强制工具调用策略（必须严格执行）：**

• **环境探索模式** - 任何文件相关请求的第一步：
  1. 🚨 **必须首先**调用 list_files 获取当前目录的所有文件
  2. 识别相关文件的**真实文件名**和**完整路径**
  3. 绝对禁止假设文件存在或使用示例路径
  4. 如果找不到相关文件，必须明确告知用户

• **内容获取模式** - 获取文件内容的严格流程：
  1. 基于list_files的结果，使用**确切的文件名**调用read_file
  2. 必须读取**所有相关文件**才能进行分析
  3. 如果任何文件读取失败，不得对该文件内容进行推测
  4. 只基于成功读取的文件内容给出建议

• **分析规划模式** - 获得完整信息后才能执行：
  1. 对读取到的所有文件内容进行交叉分析
  2. 发现文件间的关联、矛盾和缺失
  3. 基于**实际内容**而非模板给出具体建议
  4. 如需使用smart_task_planner，必须基于真实分析结果

• **错误处理模式** - 遇到工具调用失败时：
  1. 不得忽略错误继续执行
  2. 必须向用户说明具体的失败原因
  3. 提供可行的替代方案或请求用户协助
  4. 绝不基于失败的工具调用结果给出建议

**🎯 高质量响应标准：**
1. **信息完整性** - 确保获取了所有必要信息再回答
2. **专业深度** - 提供专家级的分析和建议
3. **前瞻性** - 预见用户可能的后续需求
4. **创造性** - 在解决问题的同时提供创新思路

**💡 强制执行示例（严格按此流程）：**

🔍 **用户说"分析我的小说设定"**：
  1. 立即调用list_files了解当前所有文件
  2. 识别设定相关文件（世界观.txt、主角设定.txt、门派.txt等）
  3. 逐一调用read_file读取每个设定文件的完整内容
  4. 基于实际内容进行深度分析和建议

🎯 **用户要"制定修改计划"**：
  1. 必须先list_files → read_file获取现有内容
  2. 分析实际存在的问题和优势
  3. 然后才能调用smart_task_planner制定针对性计划
  4. 给出基于真实情况的具体改进步骤

⚠️ **错误示例（禁止这样做）**：
  ❌ 直接调用smart_task_planner给出通用建议
  ❌ 使用假设的文件路径如"/path/to/世界观.txt"
  ❌ 在未读取文件时就分析文件内容
  ❌ 工具调用失败后继续给出相关建议

**🔮 高级智能特性：**
• **记忆学习** - 记住用户偏好和工作模式，提供个性化服务
• **预测分析** - 预见可能的问题和需求，主动提供解决方案  
• **质量保证** - 每个环节都有质量检查，确保专业水准
• **持续优化** - 根据反馈不断改进工作方法和建议质量

**🎯 最终要求：**
你是用户的专业AI顾问，但必须严格遵循以上所有规则！
- 🚨 每次都要先探索环境，再给出建议
- 📋 基于真实内容而非假设进行分析
- ⚡ 展现专家级洞察力，但永远以事实为准
- 🔍 宁可承认无法获取信息，也不要基于猜测回答

违反以上规则即为失职！务必严格执行！`,
	}
	return append([]ai.Message{systemMessage}, messages...)
}

func printHelp(inputManager *input.Manager) {
	fmt.Println("\033[1;36m📋 可用命令:\033[0m")
	fmt.Println("  \033[33m/help\033[0m       - 显示此帮助信息")
	fmt.Println("  \033[33m/clear\033[0m      - 清除屏幕")
	fmt.Println("  \033[33m/status\033[0m     - 显示当前状态")
	fmt.Println("  \033[33m/init\033[0m       - 分析当前环境并初始化")
	fmt.Println("  \033[33m/config\033[0m     - 配置管理")
	fmt.Println("  \033[33m/switch\033[0m <模型> - 切换AI模型 (zhipu|deepseek)")
	fmt.Println("  \033[33m/exit /quit\033[0m - 退出程序")
	fmt.Println()
	fmt.Println("\033[1;36m📝 会话管理:\033[0m")
	fmt.Println("  \033[33m/sessions\033[0m        - 列出所有会话")
	fmt.Println("  \033[33m/new\033[0m [名称]      - 创建新会话")
	fmt.Println("  \033[33m/switchsession\033[0m <ID> - 切换到指定会话")
	fmt.Println("  \033[33m/deletesession\033[0m <ID> - 删除指定会话")
	fmt.Println("  \033[90m注: 会话ID可使用前8位短ID\033[0m")
	fmt.Println()
	fmt.Println("\033[1;36m🤖 AI对话:\033[0m")
	fmt.Println("  直接输入你的问题或请求，我会帮助你！")
	fmt.Println("  \033[90m示例:\033[0m")
	fmt.Println("    • '读取文件 main.go'")
	fmt.Println("    • '列出当前目录下的文件'")
	fmt.Println("    • '在项目中搜索 TODO'")
	fmt.Println("    • '解释这段代码'")
	fmt.Println()
	fmt.Println("\033[1;36m⌨️  输入功能:\033[0m")
	fmt.Println("  • 使用 \033[33m↑↓\033[0m 方向键浏览历史命令")
	fmt.Println("  • 使用 \033[33mTab\033[0m 键自动补全")
	fmt.Println("  • 使用 \033[33mCtrl+C\033[0m 中断，\033[33mCtrl+D\033[0m 退出")
}

func printStatus(sessionManager *session.Manager, cfg *config.Config, inputManager *input.Manager) {
	session := sessionManager.GetCurrentSession()
	fmt.Printf("\033[1;36m📊 当前状态:\033[0m\n")
	fmt.Printf("  \033[36m会话:\033[0m %s (ID: %s)\n", session.Name, session.ID[:8])
	fmt.Printf("  \033[36m提供商:\033[0m %s\n", cfg.AI.Provider)
	
	if currentModel, exists := cfg.AI.Models[cfg.AI.Provider]; exists {
		modelDisplay := currentModel.Model
		if currentModel.Model == cfg.Writing.PreferredAIModel {
			modelDisplay += " \033[32m(已保存)\033[0m"
		}
		fmt.Printf("  \033[36m模型:\033[0m %s\n", modelDisplay)
		if currentModel.APIKey != "" {
			maskedKey := currentModel.APIKey
			if len(maskedKey) > 8 {
				maskedKey = maskedKey[:8] + "***"
			} else {
				maskedKey = "***"
			}
			fmt.Printf("  \033[36mAPI密钥:\033[0m %s\n", maskedKey)
		} else {
			fmt.Printf("  \033[36mAPI密钥:\033[0m \033[31m未配置\033[0m\n")
		}
	}
	
	fmt.Printf("  \033[36m工作目录:\033[0m %s\n", session.Context.WorkingDirectory)
	fmt.Printf("  \033[36m消息数量:\033[0m %d\n", len(session.Messages))
	if session.Context.ProjectInfo.Name != "" {
		fmt.Printf("  \033[36m项目:\033[0m %s (%s)\n", session.Context.ProjectInfo.Name, session.Context.ProjectInfo.Language)
	}
	
	// 显示所有配置的模型
	fmt.Printf("\n\033[1;36m🔧 已配置模型:\033[0m\n")
	for provider, modelConfig := range cfg.AI.Models {
		status := "\033[31m✗\033[0m"
		if modelConfig.APIKey != "" {
			status = "\033[32m✓\033[0m"
		}
		marker := "  "
		if provider == cfg.AI.Provider {
			marker = "👉 "
		}
		fmt.Printf("%s%s %s (%s)\n", marker, status, provider, modelConfig.Model)
	}
}

func listSessions(sessionManager *session.Manager, inputManager *input.Manager) {
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		inputManager.PrintError(fmt.Sprintf("获取会话列表失败: %v", err))
		return
	}
	
	if len(sessions) == 0 {
		inputManager.PrintInfo("未找到保存的会话")
		return
	}
	
	fmt.Println("\033[1;36m📝 已保存会话:\033[0m")
	for i, sess := range sessions {
		marker := "  "
		if sess.ID == sessionManager.GetCurrentSession().ID {
			marker = "👉 "
		}
		fmt.Printf("%s%d. \033[33m%s\033[0m (ID: %s) - %d 条消息\n", 
			marker, i+1, sess.Name, sess.ID[:8], len(sess.Messages))
	}
}

func switchProvider(provider string, aiClient *ai.Client, cfg *config.Config, inputManager *input.Manager) {
	var newProvider ai.Provider
	switch strings.ToLower(provider) {
	case "zhipu":
		newProvider = ai.ProviderZhipu
	case "deepseek":
		newProvider = ai.ProviderDeepseek
	default:
		inputManager.PrintError("不支持的提供商，请使用 'zhipu' 或 'deepseek'")
		return
	}
	
	if err := aiClient.SwitchProvider(newProvider); err != nil {
		inputManager.PrintError(fmt.Sprintf("切换提供商失败: %v", err))
		return
	}
	
	cfg.AI.Provider = newProvider
	
	// 同时更新写作配置中的首选模型
	if modelConfig, exists := cfg.AI.Models[newProvider]; exists {
		cfg.Writing.PreferredAIModel = modelConfig.Model
	}
	
	// 保存模型选择到配置文件
	if err := cfg.Save(); err != nil {
		inputManager.PrintWarning(fmt.Sprintf("保存配置失败: %v", err))
	}
	
	inputManager.PrintSuccess(fmt.Sprintf("已切换到 %s 提供商", newProvider))
	
	// 更新提示符显示新模型
	updatePrompt(cfg, inputManager)
}

func newSession(sessionManager *session.Manager, name string, inputManager *input.Manager) {
	session := sessionManager.NewSession(name)
	
	// 立即保存新会话到磁盘，确保实时同步
	if err := sessionManager.SaveSession(session); err != nil {
		inputManager.PrintError(fmt.Sprintf("保存新会话失败: %v", err))
		return
	}
	
	inputManager.PrintSuccess(fmt.Sprintf("已创建新会话: %s (ID: %s)", session.Name, session.ID[:8]))
}

func switchSession(sessionManager *session.Manager, sessionID string, inputManager *input.Manager) {
	// 支持短ID匹配（前8位）
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		inputManager.PrintError(fmt.Sprintf("获取会话列表失败: %v", err))
		return
	}
	
	var targetSession *session.Session
	for _, sess := range sessions {
		if sess.ID == sessionID || sess.ID[:8] == sessionID {
			targetSession = &sess
			break
		}
	}
	
	if targetSession == nil {
		inputManager.PrintError(fmt.Sprintf("未找到会话ID: %s", sessionID))
		return
	}
	
	// 先保存当前会话
	if err := sessionManager.SaveSession(sessionManager.GetCurrentSession()); err != nil {
		inputManager.PrintWarning(fmt.Sprintf("保存当前会话失败: %v", err))
	}
	
	// 切换到目标会话
	if err := sessionManager.SwitchSession(targetSession.ID); err != nil {
		inputManager.PrintError(fmt.Sprintf("切换会话失败: %v", err))
		return
	}
	
	inputManager.PrintSuccess(fmt.Sprintf("已切换到会话: %s (ID: %s)", targetSession.Name, targetSession.ID[:8]))
}

func deleteSession(sessionManager *session.Manager, sessionID string, inputManager *input.Manager) {
	// 支持短ID匹配（前8位）
	sessions, err := sessionManager.ListSessions()
	if err != nil {
		inputManager.PrintError(fmt.Sprintf("获取会话列表失败: %v", err))
		return
	}
	
	var targetSession *session.Session
	for _, sess := range sessions {
		if sess.ID == sessionID || sess.ID[:8] == sessionID {
			targetSession = &sess
			break
		}
	}
	
	if targetSession == nil {
		inputManager.PrintError(fmt.Sprintf("未找到会话ID: %s", sessionID))
		return
	}
	
	// 不能删除当前正在使用的会话
	if targetSession.ID == sessionManager.GetCurrentSession().ID {
		inputManager.PrintError("不能删除当前正在使用的会话，请先切换到其他会话")
		return
	}
	
	// 删除会话文件
	if err := sessionManager.DeleteSession(targetSession.ID); err != nil {
		inputManager.PrintError(fmt.Sprintf("删除会话失败: %v", err))
		return
	}
	
	inputManager.PrintSuccess(fmt.Sprintf("已删除会话: %s (ID: %s)", targetSession.Name, targetSession.ID[:8]))
}

func showConfigHelp(inputManager *input.Manager) {
	fmt.Println("\033[1;36m⚙️  配置命令:\033[0m")
	fmt.Println("  \033[33m/config show\033[0m          - 显示当前配置")
	fmt.Println("  \033[33m/config path\033[0m          - 显示配置文件路径")
	fmt.Println("  \033[33m/config set\033[0m <键> <值> - 设置配置值")
	fmt.Println("  \033[33m/config edit\033[0m          - 用默认编辑器打开配置文件")
	fmt.Println()
	fmt.Println("\033[1;36m📝 示例:\033[0m")
	fmt.Println("  \033[90m/config set zhipu.api_key sk-xxx\033[0m")
	fmt.Println("  \033[90m/config set deepseek.api_key sk-xxx\033[0m")
	fmt.Println("  \033[90m/config set ai.provider zhipu\033[0m")
}

func handleConfigCommand(args []string, cfg *config.Config, inputManager *input.Manager) {
	if len(args) == 0 {
		showConfigHelp(inputManager)
		return
	}
	
	command := args[0]
	switch command {
	case "show":
		showConfig(cfg, inputManager)
	case "path":
		showConfigPath(inputManager)
	case "set":
		if len(args) < 3 {
			inputManager.PrintError("用法: /config set <键> <值>")
			return
		}
		setConfigValue(args[1], args[2], cfg, inputManager)
	case "edit":
		editConfig(inputManager)
	default:
		showConfigHelp(inputManager)
	}
}

func showConfig(cfg *config.Config, inputManager *input.Manager) {
	fmt.Println("\033[1;36m⚙️  当前配置:\033[0m")
	fmt.Printf("\033[36m提供商:\033[0m %s\n", cfg.AI.Provider)
	fmt.Println("\n\033[36m模型:\033[0m")
	for provider, modelConfig := range cfg.AI.Models {
		apiKeyStatus := "\033[31m未设置\033[0m"
		if modelConfig.APIKey != "" {
			apiKeyStatus = "\033[32m已配置\033[0m"
		}
		fmt.Printf("  \033[33m%s:\033[0m\n", provider)
		fmt.Printf("    模型: %s\n", modelConfig.Model)
		fmt.Printf("    API密钥: %s\n", apiKeyStatus)
		fmt.Printf("    基础URL: %s\n", modelConfig.BaseURL)
	}
}

func showConfigPath(inputManager *input.Manager) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		inputManager.PrintError(fmt.Sprintf("获取配置路径失败: %v", err))
		return
	}
	configFile := filepath.Join(configDir, "config.yaml")
	fmt.Printf("\033[1;36m📁 配置路径:\033[0m\n")
	fmt.Printf("\033[36m配置文件:\033[0m %s\n", configFile)
	fmt.Printf("\033[36m配置目录:\033[0m %s\n", configDir)
}

func setConfigValue(key, value string, cfg *config.Config, inputManager *input.Manager) {
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		inputManager.PrintError("键格式: <提供商>.<字段> 或 ai.<字段>")
		fmt.Println("\033[90m示例: zhipu.api_key, deepseek.api_key, ai.provider\033[0m")
		return
	}
	
	section, field := parts[0], parts[1]
	
	switch section {
	case "ai":
		if field == "provider" {
			if value == "zhipu" || value == "deepseek" {
				cfg.AI.Provider = ai.Provider(value)
				inputManager.PrintSuccess(fmt.Sprintf("已设置AI提供商为: %s", value))
				// 更新提示符
				updatePrompt(cfg, inputManager)
			} else {
				inputManager.PrintError("提供商必须是 'zhipu' 或 'deepseek'")
				return
			}
		} else {
			inputManager.PrintError(fmt.Sprintf("未知的AI字段: %s", field))
			return
		}
	case "zhipu", "deepseek":
		provider := ai.Provider(section)
		
		// 确保Models map已初始化
		if cfg.AI.Models == nil {
			cfg.AI.Models = make(map[ai.Provider]ai.ModelConfig)
		}
		
		// 获取或创建默认配置
		modelConfig, exists := cfg.AI.Models[provider]
		if !exists {
			if provider == ai.ProviderZhipu {
				modelConfig = ai.ModelConfig{
					APIKey:  "",
					BaseURL: "https://open.bigmodel.cn/api/paas/v4",
					Model:   "glm-4",
				}
			} else {
				modelConfig = ai.ModelConfig{
					APIKey:  "",
					BaseURL: "https://api.deepseek.com",
					Model:   "deepseek-chat",
				}
			}
		}
		
		switch field {
		case "api_key":
			modelConfig.APIKey = value
			cfg.AI.Models[provider] = modelConfig
			inputManager.PrintSuccess(fmt.Sprintf("已设置 %s API密钥", section))
		case "model":
			modelConfig.Model = value
			cfg.AI.Models[provider] = modelConfig
			inputManager.PrintSuccess(fmt.Sprintf("已设置 %s 模型为: %s", section, value))
		case "base_url":
			modelConfig.BaseURL = value
			cfg.AI.Models[provider] = modelConfig
			inputManager.PrintSuccess(fmt.Sprintf("已设置 %s 基础URL为: %s", section, value))
		default:
			inputManager.PrintError(fmt.Sprintf("未知的 %s 字段: %s", section, field))
			return
		}
	default:
		inputManager.PrintError(fmt.Sprintf("未知的配置段: %s", section))
		return
	}
	
	if err := cfg.Save(); err != nil {
		inputManager.PrintError(fmt.Sprintf("保存配置失败: %v", err))
	} else {
		inputManager.PrintInfo("配置保存成功")
	}
}

func editConfig(inputManager *input.Manager) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		inputManager.PrintError(fmt.Sprintf("获取配置路径失败: %v", err))
		return
	}
	configFile := filepath.Join(configDir, "config.yaml")
	
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("notepad", configFile)
	case "darwin":
		cmd = exec.Command("open", configFile)
	default:
		// 尝试常见的编辑器
		editors := []string{"code", "nano", "vim", "vi"}
		for _, editor := range editors {
			if _, err := exec.LookPath(editor); err == nil {
				cmd = exec.Command(editor, configFile)
				break
			}
		}
	}
	
	if cmd == nil {
		inputManager.PrintInfo(fmt.Sprintf("请手动编辑: %s", configFile))
		return
	}
	
	inputManager.PrintInfo(fmt.Sprintf("正在打开配置文件: %s", configFile))
	if err := cmd.Run(); err != nil {
		inputManager.PrintError(fmt.Sprintf("打开编辑器失败: %v", err))
	}
}

func processInput(ctx context.Context, aiClient *ai.Client, toolManager *tools.Manager, sessionManager *session.Manager, inputManager *input.Manager, userInput string) (string, error) {
	// 获取当前会话
	currentSession := sessionManager.GetCurrentSession()
	
	// 添加用户消息到会话历史
	currentSession.AddMessage("user", userInput)
	
	// 获取工具定义
	toolDefinitions := toolManager.GetToolDefinitions()
	
	// 添加系统提示指导AI使用工具
	messages := addSystemMessage(currentSession.GetMessages())
	
	// 调用AI模型
	response, toolCalls, err := aiClient.Chat(ctx, messages, toolDefinitions)
	if err != nil {
		return "", fmt.Errorf("AI request failed: %w", err)
	}
	
	// 执行工具调用
	if len(toolCalls) > 0 {
		inputManager.PrintInfo(fmt.Sprintf("🔧 正在执行 %d 个工具调用...", len(toolCalls)))
		
		// 先添加带有tool_calls的assistant消息
		assistantMessage := ai.Message{
			Role:      "assistant",
			Content:   response,
			ToolCalls: toolCalls,
		}
		currentSession.Messages = append(currentSession.Messages, assistantMessage)
		
		toolResults, err := toolManager.ExecuteTools(ctx, toolCalls)
		if err != nil {
			return "", fmt.Errorf("tool execution failed: %w", err)
		}
		
		// 统计执行结果
		successCount := 0
		errorCount := 0
		for _, result := range toolResults {
			if result.Error != nil {
				errorCount++
				inputManager.PrintWarning(fmt.Sprintf("工具 %s 执行失败: %v", result.ToolName, result.Error))
			} else {
				successCount++
			}
			currentSession.AddToolResult(result)
		}
		
		inputManager.PrintSuccess(fmt.Sprintf("✅ 工具执行完成: %d 成功, %d 失败", successCount, errorCount))
		
		// 更新messages以包含工具结果
		messages = addSystemMessage(currentSession.GetMessages())
		
		// 重试机制：如果第一次调用失败，最多重试2次
		maxRetries := 2
		for retry := 0; retry <= maxRetries; retry++ {
			response, _, err = aiClient.Chat(ctx, messages, toolDefinitions)
			if err == nil {
				break
			}
			
			if retry < maxRetries {
				inputManager.PrintWarning(fmt.Sprintf("AI调用失败，正在重试 (%d/%d)...", retry+1, maxRetries))
				time.Sleep(time.Second * time.Duration(retry+1)) // 递增延迟
			}
		}
		
		if err != nil {
			return "", fmt.Errorf("AI follow-up request failed after %d retries: %w", maxRetries, err)
		}
		
		// 添加最终的AI响应到会话历史
		currentSession.AddMessage("assistant", response)
	} else {
		// 如果没有工具调用，直接添加AI响应到会话历史
		currentSession.AddMessage("assistant", response)
	}
	
	return response, nil
}

// handleInitCommand 处理 /init 命令
func handleInitCommand(aiClient *ai.Client, inputManager *input.Manager) {
	inputManager.PrintInfo("🧠 正在分析当前环境...")
	inputManager.ShowLoading("环境分析中")
	
	// 创建工具管理器
	toolManager := tools.NewManager()
	ctx := context.Background()
	
	// 使用智能上下文工具分析环境
	if tool, exists := toolManager.GetTool("get_smart_context"); exists {
		result, err := tool.Execute(ctx, nil)
		
		inputManager.HideLoading()
		
		if err != nil {
			inputManager.PrintError(fmt.Sprintf("环境分析失败: %v", err))
			return
		}
		
		// 直接显示环境分析结果
		fmt.Println()
		fmt.Println(result)
	} else {
		inputManager.HideLoading()
		inputManager.PrintError("智能上下文工具不可用")
		return
	}
	
	// 额外提供一些初始化建议
	inputManager.PrintInfo("💡 环境分析完成！AI助手已了解当前环境，可以为您提供针对性帮助。")
	
	// 获取当前目录来判断项目类型并给出建议
	currentDir, _ := os.Getwd()
	projectName := filepath.Base(currentDir)
	
	// 检查是否已存在小说项目文件
	novelProjectFile := filepath.Join(currentDir, "novel_project.json")
	if _, err := os.Stat(novelProjectFile); err == nil {
		inputManager.PrintSuccess(fmt.Sprintf("📚 检测到小说项目: %s", projectName))
		fmt.Println("  可用命令: get_novel_context, get_chapter_context, search_novel_history")
	} else {
		// 检查项目类型给出相应建议
		if isNovelProject(currentDir) {
			inputManager.PrintWarning("📚 检测到可能的小说写作项目，建议使用以下命令初始化:")
			fmt.Println("  init_novel_project title=\"项目名称\" author=\"作者\" genre=\"类型\"")
		}
	}
	
	fmt.Println()
	inputManager.PrintInfo("现在您可以开始对话，我会基于当前环境为您提供智能帮助！")
}

// 辅助函数：检测是否可能是小说项目
func isNovelProject(dir string) bool {
	// 检查是否包含常见的小说相关文件或目录
	novelIndicators := []string{
		"chapters", "章节", "小说", "novel", "story", "stories",
		"characters", "角色", "人物设定", "plot", "情节",
		"世界观", "设定", "world", "timeline", "大纲", "outline",
	}
	
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	
	for _, entry := range entries {
		name := strings.ToLower(entry.Name())
		for _, indicator := range novelIndicators {
			if strings.Contains(name, strings.ToLower(indicator)) {
				return true
			}
		}
		
		// 检查文件扩展名
		if strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md") {
			// 检查文件内容是否像小说
			if isLikelyNovelFile(filepath.Join(dir, entry.Name())) {
				return true
			}
		}
	}
	
	return false
}

// 辅助函数：检测文件是否像小说内容
func isLikelyNovelFile(filePath string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	
	text := string(content)
	// 简单检测：包含章节标识或对话标识
	novelKeywords := []string{
		"第", "章", "节", "回", "卷",
		`"`, `"`, `'`, `'`, "「", "」",
		"说道", "说着", "心想", "想到",
	}
	
	for _, keyword := range novelKeywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	
	return false
}