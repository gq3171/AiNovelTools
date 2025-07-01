#!/bin/bash

# AI Assistant Setup Script for Linux/macOS
echo "===================================="
echo "    AI Assistant Setup Script"
echo "===================================="
echo

# 检查Go是否安装
if ! command -v go &> /dev/null; then
    echo "[错误] 未找到Go语言环境，请先安装Go 1.21+"
    echo "安装方法:"
    echo "  Ubuntu/Debian: sudo apt install golang-go"
    echo "  CentOS/RHEL: sudo yum install golang"
    echo "  macOS: brew install go"
    echo "  或访问: https://golang.org/dl/"
    exit 1
fi

# 显示Go版本
echo "[信息] 检测到Go环境:"
go version

# 编译程序
echo
echo "[步骤1] 编译AI Assistant..."
go build -o ai-assistant main.go
if [ $? -ne 0 ]; then
    echo "[错误] 编译失败"
    exit 1
fi
echo "[成功] 编译完成"

# 创建配置目录
echo
echo "[步骤2] 创建配置目录..."
if [[ "$OSTYPE" == "darwin"* ]]; then
    CONFIG_DIR="$HOME/Library/Application Support/AI-Assistant"
else
    CONFIG_DIR="$HOME/.ai-assistant"
fi

if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    echo "[成功] 配置目录已创建: $CONFIG_DIR"
else
    echo "[信息] 配置目录已存在: $CONFIG_DIR"
fi

# 配置向导
echo
echo "[步骤3] 配置向导"
echo
echo "请选择要配置的AI模型:"
echo "1. 智谱AI (GLM-4)"
echo "2. Deepseek"
echo "3. 两个都配置"
echo "4. 跳过配置"
echo
read -p "请输入选择 (1-4): " choice

config_zhipu() {
    echo
    echo "配置智谱AI:"
    echo "1. 访问 https://open.bigmodel.cn/"
    echo "2. 注册账号并获取API密钥"
    echo "3. 在下面输入API密钥"
    echo
    read -p "请输入智谱AI API密钥: " zhipu_key
    if [ ! -z "$zhipu_key" ]; then
        echo "export ZHIPU_API_KEY=\"$zhipu_key\"" >> ~/.bashrc
        export ZHIPU_API_KEY="$zhipu_key"
        echo "[成功] 智谱AI API密钥已保存到 ~/.bashrc"
    fi
}

config_deepseek() {
    echo
    echo "配置Deepseek:"
    echo "1. 访问 https://platform.deepseek.com/"
    echo "2. 注册账号并获取API密钥"
    echo "3. 在下面输入API密钥"
    echo
    read -p "请输入Deepseek API密钥: " deepseek_key
    if [ ! -z "$deepseek_key" ]; then
        echo "export DEEPSEEK_API_KEY=\"$deepseek_key\"" >> ~/.bashrc
        export DEEPSEEK_API_KEY="$deepseek_key"
        echo "[成功] Deepseek API密钥已保存到 ~/.bashrc"
    fi
}

case $choice in
    1)
        config_zhipu
        ;;
    2)
        config_deepseek
        ;;
    3)
        config_zhipu
        config_deepseek
        ;;
    4)
        echo "[信息] 跳过配置"
        ;;
    *)
        echo "[错误] 无效选择"
        ;;
esac

echo
echo "===================================="
echo "          安装完成！"
echo "===================================="
echo
echo "使用方法:"
echo "1. 运行程序: ./ai-assistant"
echo "2. 输入 'help' 查看帮助"
echo "3. 输入 'config' 管理配置"
echo "4. 输入 'status' 查看当前状态"
echo
echo "配置文件位置: $CONFIG_DIR/config.yaml"
echo
if [ $choice -eq 1 ] || [ $choice -eq 2 ] || [ $choice -eq 3 ]; then
    echo "注意: 请运行 'source ~/.bashrc' 或重新打开终端以加载环境变量"
fi
echo