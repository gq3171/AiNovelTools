# AI Assistant

一个类似Claude Code的智能AI助手工具，支持智谱和Deepseek模型。

## 功能特性

- 🤖 支持多个AI模型提供商（智谱、Deepseek）
- 🛠️ 内置工具调用系统（文件操作、代码执行、搜索等）
- 💬 智能对话管理和上下文维护
- 📝 会话记忆和状态管理
- ⚙️ 灵活的配置管理
- 🎨 彩色命令行界面
- ⌨️ 高级输入支持（历史命令、自动补全、退格键）
- 🚀 跨平台支持（Windows、Linux、macOS）

## 安装

### 快速安装（推荐）

**Windows系统:**
1. 双击运行 `setup.bat`
2. 按照向导配置API密钥
3. 运行 `ai-assistant.exe`

**Linux/macOS系统:**
```bash
chmod +x setup.sh
./setup.sh
./ai-assistant
```

### 手动安装

1. 克隆项目
```bash
git clone <repository-url>
cd AiNovelTools
```

2. 安装依赖
```bash
go mod tidy
```

3. 编译程序
```bash
# Windows
go build -o ai-assistant.exe main.go

# Linux/macOS
go build -o ai-assistant main.go
```

4. 配置API密钥
```bash
# 方式1: 环境变量（推荐）
export ZHIPU_API_KEY="your-zhipu-api-key"
export DEEPSEEK_API_KEY="your-deepseek-api-key"

# 方式2: 程序内配置
./ai-assistant
> config set zhipu.api_key your-zhipu-key
> config set deepseek.api_key your-deepseek-key
```

## 配置

程序会根据操作系统自动选择配置文件位置：
- **Windows**: `%APPDATA%\AI-Assistant\config.yaml`
- **macOS**: `~/Library/Application Support/AI-Assistant/config.yaml`
- **Linux**: `~/.ai-assistant/config.yaml`

首次运行时会自动创建默认配置文件：

```yaml
ai:
  provider: zhipu
  models:
    zhipu:
      api_key: ""
      base_url: "https://open.bigmodel.cn/api/paas/v4"
      model: "glm-4"
    deepseek:
      api_key: ""
      base_url: "https://api.deepseek.com"
      model: "deepseek-chat"
  max_tokens: 2048
  temperature: 0.7
ui:
  theme: dark
  show_tokens: false
  auto_save: true
  max_history: 100
features:
  enable_file_watch: true
  allowed_commands:
    - ls
    - cat
    - grep
    - find
    - git
  safe_mode: true
```

## 使用方法

### 系统命令（需要 / 前缀）

- `/help` - 显示帮助信息
- `/status` - 显示当前状态
- `/sessions` - 列出所有会话
- `/new [名称]` - 创建新会话
- `/switch <提供商>` - 切换AI提供商
- `/config` - 配置管理
- `/clear` - 清屏  
- `/exit` `/quit` - 退出程序

### 配置管理

```bash
> /config show          # 显示当前配置
> /config path          # 显示配置文件路径
> /config set zhipu.api_key sk-xxx      # 设置智谱AI密钥
> /config set deepseek.api_key sk-xxx   # 设置Deepseek密钥
> /config set ai.provider zhipu         # 切换默认提供商
> /config edit          # 用默认编辑器打开配置文件
```

### AI对话

直接输入你的问题或请求（无需前缀），AI助手会帮助你：

```bash
> 读取文件 main.go
> 列出当前目录下的文件
> 在项目中搜索 TODO
> 解释这段代码
> 帮我调试这个函数
```

### 内置工具

AI助手可以调用以下工具：

- `read_file` - 读取文件内容
- `write_file` - 写入文件内容
- `list_files` - 列出目录内容
- `execute_command` - 执行系统命令
- `search` - 搜索文件内容

## 高级输入功能

程序提供了类似Claude Code的现代命令行体验：

### 📚 历史命令
- 使用 **↑↓** 方向键浏览命令历史
- 历史记录持久保存，跨会话可用
- 支持智能历史搜索

### 🎯 自动补全
- 使用 **Tab** 键触发智能补全
- 支持命令、参数和选项补全
- 示例：
  ```bash
  > /conf<Tab>     → /config
  > /config s<Tab> → /config show / /config set
  > /switch <Tab>  → /switch zhipu / /switch deepseek
  ```

### ⌨️ 快捷键支持
- **Ctrl+C**: 中断当前操作
- **Ctrl+D**: 退出程序  
- **Ctrl+A**: 移动到行首
- **Ctrl+E**: 移动到行尾
- **Ctrl+L**: 清屏
- **Backspace/Delete**: 正确处理字符删除

### 🎨 用户体验
- **智能提示符**：`[glm-4] ❯` 实时显示当前AI模型
- 彩色语法高亮
- 实时加载动画 ⏳
- 分类消息显示：
  - ✅ 成功消息（绿色）
  - ❌ 错误消息（红色）  
  - ⚠️ 警告消息（黄色）
  - ℹ️ 信息消息（蓝色）
- 美观的帮助界面和状态显示

## 架构设计

```
├── main.go              # 主程序入口
├── internal/
│   ├── ai/              # AI模型接口
│   │   ├── client.go    # AI客户端
│   │   ├── zhipu.go     # 智谱API实现
│   │   └── deepseek.go  # Deepseek API实现
│   ├── config/          # 配置管理
│   │   └── config.go
│   ├── input/           # 高级输入处理
│   │   └── readline.go  # readline封装
│   ├── tools/           # 工具调用系统
│   │   └── manager.go
│   └── session/         # 会话管理
│       └── manager.go
├── setup.bat            # Windows安装脚本
└── setup.sh             # Linux/macOS安装脚本
```

## API密钥配置

### 智谱AI
1. 访问 [智谱AI开放平台](https://open.bigmodel.cn/)
2. 注册账号并获取API密钥
3. 设置环境变量：`export AI_API_KEY="your-zhipu-api-key"`

### Deepseek
1. 访问 [Deepseek平台](https://platform.deepseek.com/)
2. 注册账号并获取API密钥
3. 设置环境变量：`export DEEPSEEK_API_KEY="your-deepseek-api-key"`

### 同时配置多个模型
```bash
# 设置所有模型的密钥，可以自由切换
export ZHIPU_API_KEY="your-zhipu-key"
export DEEPSEEK_API_KEY="your-deepseek-key"

# 或者直接编辑配置文件 ~/.ai-assistant/config.yaml
```

## 开发

### 添加新的AI提供商

1. 在 `internal/ai/` 目录下创建新的提供商实现
2. 实现 `AIProvider` 接口
3. 在 `client.go` 中注册新提供商

### 添加新工具

1. 在 `internal/tools/manager.go` 中实现 `Tool` 接口
2. 在 `NewManager()` 中注册新工具

### 🚀 **使用体验**
现在你可以像使用Claude Code一样：
1. **智能提示符**：`[glm-4] ❯` 实时显示当前模型
2. **上下键**浏览历史命令
3. **Tab键**自动补全
4. **退格键**正常删除字符
5. **命令前缀**：系统命令用 `/`，AI对话直接输入
6. **Ctrl+C**中断操作，**Ctrl+D**优雅退出

### 📺 **提示符演示**
```bash
[glm-4] ❯ /switch deepseek
✅ 已切换到 deepseek 提供商
[deepseek-chat] ❯ 你好，现在用的什么模型？
🤖 你好！我现在使用的是 Deepseek 模型...
[deepseek-chat] ❯ /switch zhipu  
✅ 已切换到 zhipu 提供商
[glm-4] ❯ 
```

程序提供了现代化的命令行体验，让AI助手使用起来更加流畅和高效！🎊

## 许可证

MIT License