#!/bin/bash
# open-station Claude Code CLI 配置脚本
# 用法: ./scripts/setup-claude-code.sh [api-key]

set -e

GATEWAY_URL="${GATEWAY_URL:-http://localhost:8080/v1}"
API_KEY="${1:-}"

if [ -z "$API_KEY" ]; then
    echo "用法: $0 <api-key>"
    echo ""
    echo "示例: $0 sk-abc123def456"
    echo ""
    echo "获取API Key:"
    echo "  1. 通过管理API创建: curl -X POST http://localhost:8080/admin/api-keys ..."
    echo "  2. 或联系管理员获取"
    exit 1
fi

echo "=== open-station Claude Code CLI 配置 ==="
echo ""
echo "网关端点: $GATEWAY_URL"
echo "API Key:  $API_KEY"
echo ""

# 创建 Claude Code 配置目录
CLAUDE_DIR="$HOME/.claude"
mkdir -p "$CLAUDE_DIR"

# 写入 settings.json
SETTINGS_FILE="$CLAUDE_DIR/settings.json"
echo "写入配置到: $SETTINGS_FILE"

cat > "$SETTINGS_FILE" << EOF
{
  "env": {
    "ANTHROPIC_BASE_URL": "$GATEWAY_URL",
    "ANTHROPIC_API_KEY": "$API_KEY"
  }
}
EOF

echo ""
echo "=== 配置完成 ==="
echo ""
echo "验证方式:"
echo "  1. 启动 Claude Code: claude"
echo "  2. 检查状态: claude /status"
echo ""
echo "可用模型:"
echo "  - Claude: claude-opus-4-7, claude-sonnet-4-6, claude-haiku-4-5"
echo "  - OpenAI: openai-gpt-4o, openai-gpt-4o-mini"
echo "  - DeepSeek: deepseek-v4-flash, deepseek-v4-pro"
echo "  - GLM: glm-4.7, glm-4.5-air, glm-4-flash (免费)"
echo "  - Gemini: gemini-2.5-flash, gemini-3-flash-preview"
echo ""
echo "示例:"
echo "  claude --model claude-opus-4-7"
echo "  claude --model openai-gpt-4o"
echo "  claude --model deepseek-v4-flash"
echo ""
echo "测试网关连接:"
curl -s -X POST "$GATEWAY_URL/messages" \
  -H "Content-Type: application/json" \
  -H "X-Api-Key: $API_KEY" \
  -d '{"model":"claude-sonnet-4-6","max_tokens":100,"messages":[{"role":"user","content":"Say hello"}]}' \
  | head -100

echo ""
echo "=== 完成 ==="