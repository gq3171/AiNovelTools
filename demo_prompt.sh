#!/bin/bash

echo "演示带模型名称的提示符功能..."
echo ""

# 模拟用户输入来展示提示符变化
cat << 'EOF' | timeout 10 ./ai-assistant || true
/help
你好，请介绍一下你自己
/switch deepseek
/status  
现在使用的是什么模型？
/config set ai.provider zhipu
/exit
EOF

echo ""
echo "演示完成！"
echo ""
echo "提示符格式："
echo "  [模型名称] ❯ "
echo ""
echo "特性："
echo "  • 实时显示当前使用的AI模型"
echo "  • 切换模型时自动更新提示符"
echo "  • 配置更改时即时反映"
echo "  • 支持智谱(glm-4)和Deepseek(deepseek-chat)"