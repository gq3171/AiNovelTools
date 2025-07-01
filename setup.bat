@echo off
chcp 65001 >nul
echo ====================================
echo    AI Assistant Setup for Windows
echo ====================================
echo.

:: 检查Go是否安装
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [错误] 未找到Go语言环境，请先安装Go 1.21+
    echo 下载地址: https://golang.org/dl/
    pause
    exit /b 1
)

:: 显示Go版本
echo [信息] 检测到Go环境:
go version

:: 编译程序
echo.
echo [步骤1] 编译AI Assistant...
go build -o ai-assistant.exe main.go
if %errorlevel% neq 0 (
    echo [错误] 编译失败
    pause
    exit /b 1
)
echo [成功] 编译完成

:: 创建配置目录
echo.
echo [步骤2] 创建配置目录...
if not exist "%APPDATA%\AI-Assistant" (
    mkdir "%APPDATA%\AI-Assistant"
    echo [成功] 配置目录已创建: %APPDATA%\AI-Assistant
) else (
    echo [信息] 配置目录已存在: %APPDATA%\AI-Assistant
)

:: 配置向导
echo.
echo [步骤3] 配置向导
echo.
echo 请选择要配置的AI模型:
echo 1. 智谱AI (GLM-4)
echo 2. Deepseek
echo 3. 两个都配置
echo 4. 跳过配置
echo.
set /p choice="请输入选择 (1-4): "

if "%choice%"=="1" goto config_zhipu
if "%choice%"=="2" goto config_deepseek
if "%choice%"=="3" goto config_both
if "%choice%"=="4" goto finish
goto invalid_choice

:config_zhipu
echo.
echo 配置智谱AI:
echo 1. 访问 https://open.bigmodel.cn/
echo 2. 注册账号并获取API密钥
echo 3. 在下面输入API密钥
echo.
set /p zhipu_key="请输入智谱AI API密钥: "
if not "%zhipu_key%"=="" (
    setx ZHIPU_API_KEY "%zhipu_key%" >nul
    echo [成功] 智谱AI API密钥已保存到环境变量
)
goto finish

:config_deepseek
echo.
echo 配置Deepseek:
echo 1. 访问 https://platform.deepseek.com/
echo 2. 注册账号并获取API密钥
echo 3. 在下面输入API密钥
echo.
set /p deepseek_key="请输入Deepseek API密钥: "
if not "%deepseek_key%"=="" (
    setx DEEPSEEK_API_KEY "%deepseek_key%" >nul
    echo [成功] Deepseek API密钥已保存到环境变量
)
goto finish

:config_both
echo.
echo 配置智谱AI:
echo 1. 访问 https://open.bigmodel.cn/
echo 2. 注册账号并获取API密钥
echo.
set /p zhipu_key="请输入智谱AI API密钥: "
if not "%zhipu_key%"=="" (
    setx ZHIPU_API_KEY "%zhipu_key%" >nul
    echo [成功] 智谱AI API密钥已保存
)

echo.
echo 配置Deepseek:
echo 1. 访问 https://platform.deepseek.com/
echo 2. 注册账号并获取API密钥
echo.
set /p deepseek_key="请输入Deepseek API密钥: "
if not "%deepseek_key%"=="" (
    setx DEEPSEEK_API_KEY "%deepseek_key%" >nul
    echo [成功] Deepseek API密钥已保存
)
goto finish

:invalid_choice
echo [错误] 无效选择
goto finish

:finish
echo.
echo ====================================
echo          安装完成！
echo ====================================
echo.
echo 使用方法:
echo 1. 运行程序: ai-assistant.exe
echo 2. 输入 'help' 查看帮助
echo 3. 输入 'config' 管理配置
echo 4. 输入 'status' 查看当前状态
echo.
echo 配置文件位置: %APPDATA%\AI-Assistant\config.yaml
echo.
echo 注意: 如果刚设置了环境变量，请重新打开命令行窗口
echo.
pause