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

AI助手可以调用以下工具，提供像Claude Code一样的完整文件操作能力：

#### 📖 **文件读写**
- `read_file` - 读取文件内容
- `write_file` - 写入文件内容
- `edit_file` - 编辑文件（支持行范围替换和模式替换）

#### 📁 **目录操作**
- `list_files` - 列出目录内容
- `create_directory` - 创建目录（包括父目录）

#### 🔧 **文件管理**
- `delete_file` - 删除文件或目录
- `rename_file` - 重命名文件或目录
- `copy_file` - 复制文件到指定位置
- `move_file` - 移动文件到指定位置
- `file_info` - 获取文件详细信息（大小、权限、修改时间等）

#### 🔍 **搜索和替换**
- `search` - 在文件中搜索文本内容
- `replace_text` - 批量文本替换（支持正则表达式）

#### ⚡ **系统命令**
- `execute_command` - 执行系统命令

#### 🧠 **智能环境感知**
- `get_current_directory` - 获取当前工作目录和目录信息
- `get_system_info` - 获取系统信息（OS、架构、Go版本等）
- `get_project_info` - 智能分析项目类型、结构和依赖
- `get_working_context` - 获取完整工作上下文
- `get_smart_context` - 智能上下文分析（包含历史、偏好、建议）

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

## 🧠 智能环境感知

AI助手现在具备了Claude Code级别的环境感知能力！

### 📍 **自动环境识别**
- **当前目录感知** - 自动获取和分析工作目录
- **系统信息检测** - OS、架构、Go版本、环境变量
- **项目类型识别** - 智能识别Go、Node.js、Java等项目类型
- **目录结构分析** - 理解项目结构和文件用途

### 🎯 **智能上下文管理**
- **工作历史跟踪** - 记录最近的文件操作和项目活动
- **个人偏好学习** - 保存用户习惯和自定义设置
- **会话连续性** - 跨会话保持工作上下文
- **智能建议系统** - 基于项目类型和历史提供个性化建议

### 💡 **增强的AI体验**
- **上下文感知对话** - AI完全理解当前工作环境
- **项目特定建议** - 根据项目类型提供针对性帮助
- **智能工作流程** - 自动化常见开发任务
- **个性化助手** - 学习用户工作模式并适应

### 🔧 **环境感知工具清单**
```bash
> 获取当前目录信息
get_current_directory

> 获取系统环境信息  
get_system_info

> 分析项目结构和类型
get_project_info

> 获取完整工作上下文
get_working_context

> 智能上下文分析（包含历史、偏好、建议）
get_smart_context
```

现在，您的AI助手不仅能操作文件，还能**理解环境、记住历史、学习偏好、提供智能建议** - 真正实现了与Claude Code相同的智能体验！🚀

## 📚 专业小说写作功能

AI助手专门为小说写作优化，解决了**历史聊天记录管理**这一核心问题！

### 🎯 **小说写作核心问题解决**

#### 📖 **章节连贯性管理**
- **完整历史记录** - 永久保存所有创作对话，不丢失任何细节
- **结构化存储** - 按角色、情节、设定分类管理所有内容
- **智能检索** - 秒级查找相关历史讨论和设定信息
- **上下文感知** - 自动提供当前章节的相关背景和要点

#### 🎭 **角色一致性保证**
- **角色档案管理** - 详细记录每个角色的性格、背景、关系
- **行为模式追踪** - 确保角色行为符合既定性格特征
- **对话风格维持** - 保持每个角色独特的说话方式
- **发展轨迹记录** - 追踪角色在故事中的成长变化

#### 📈 **情节线管理**
- **多线程情节** - 同时管理主线、支线、感情线、悬疑线
- **伏笔呼应** - 记录埋下的伏笔，确保后续呼应
- **时间线管理** - 避免情节时间逻辑错误
- **冲突解决** - 追踪各种冲突的设置和解决

#### 🌍 **世界观统一**
- **设定规则** - 维护一致的世界观设定和规则
- **背景元素** - 管理地理、历史、文化等背景信息
- **概念词典** - 专有名词和概念的统一定义
- **规则检查** - 自动检查是否违反已设定的世界观规则

### 🛠️ **小说写作专用工具**

```bash
# 初始化小说项目
> init_novel_project title="我的小说" author="作者名" genre="奇幻"

# 获取完整小说上下文
> get_novel_context

# 添加角色设定
> add_character name="主角名" background="角色背景" personality="性格特点"

# 添加情节线
> add_plot_line name="主线" type="main" description="情节描述"

# 获取章节写作上下文
> get_chapter_context chapter=5

# 搜索历史创作记录
> search_novel_history query="角色名" max_results=10
```

### 💡 **智能写作助手特性**

#### 🔍 **智能内容检索**
- **多维搜索** - 按角色、情节、设定、关键词快速查找
- **关联推荐** - 自动推荐相关的历史内容和设定
- **语义理解** - 理解查询意图，返回最相关的信息

#### 🎯 **上下文感知创作**
- **章节背景** - 自动提供当前章节需要的所有背景信息
- **连贯性检查** - 智能提醒可能的前后矛盾
- **创作建议** - 基于历史内容提供个性化写作建议

#### 📊 **创作进度管理**
- **写作统计** - 字数、章节、角色出场统计
- **进度追踪** - 各情节线的进展状况
- **完成度分析** - 评估故事完整性和待完善部分

### 🚀 **使用场景示例**

#### 场景1：开始新章节
```bash
[glm-4] ❯ get_chapter_context chapter=8
🤖 第8章写作上下文：
   - 相关角色：李明、张晓、王教授
   - 活跃情节：主线-寻找真相，支线-感情发展
   - 前章要点：李明发现了关键线索
   - 注意事项：需要呼应第3章的伏笔

[glm-4] ❯ 继续写第8章，李明拿着证据去找王教授...
```

#### 场景2：检查角色一致性
```bash
[glm-4] ❯ search_novel_history query="李明的性格特点"
🤖 找到3条相关记录：
   - 第1章：李明性格内向，不善表达情感
   - 第4章：在压力下会变得冲动
   - 第6章：对朋友非常忠诚
   建议：保持这些性格特征的一致性

[glm-4] ❯ 李明在第8章应该如何反应？
```

### 🎊 **解决方案优势**

| 传统AI写作 | 我们的解决方案 | 提升效果 |
|-----------|---------------|----------|
| ❌ 记忆丢失 | ✅ 永久记录 | 100%历史保留 |
| ❌ 角色不一致 | ✅ 智能检查 | 角色连贯性保证 |
| ❌ 情节矛盾 | ✅ 情节线管理 | 逻辑一致性 |
| ❌ 设定混乱 | ✅ 世界观统一 | 设定标准化 |
| ❌ 查找困难 | ✅ 智能检索 | 秒级信息获取 |

现在，您可以放心地创作**长篇小说**，AI助手会确保每一个细节都与前文保持完美的连贯性！📚✨

## 许可证

MIT License
