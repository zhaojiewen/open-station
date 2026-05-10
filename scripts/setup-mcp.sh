#!/bin/bash

# Open Station MCP 配置脚本
# 快速配置 Claude Code CLI 或其他 MCP 客户端

set -e

echo "=========================================="
echo "   Open Station MCP 配置"
echo "=========================================="

# 默认配置
DEFAULT_URL="http://localhost:8080/mcp"
DEFAULT_API_KEY=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --url)
            MCP_URL="$2"
            shift 2
            ;;
        --api-key)
            API_KEY="$2"
            shift 2
            ;;
        --claude)
            CLIENT="claude"
            shift
            ;;
        --cursor)
            CLIENT="cursor"
            shift
            ;;
        --vscode)
            CLIENT="vscode"
            shift
            ;;
        --list)
            LIST_MODE=true
            shift
            ;;
        --help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --url URL         MCP 服务地址 (默认: $DEFAULT_URL)"
            echo "  --api-key KEY     API Key (如未提供会提示输入)"
            echo "  --claude          配置 Claude Code CLI"
            echo "  --cursor          配置 Cursor IDE"
            echo "  --vscode          配置 VS Code"
            echo "  --list            列出已配置的 MCP 服务器"
            echo "  --help            显示帮助"
            echo ""
            echo "示例:"
            echo "  $0 --claude --api-key sk-xxx"
            echo "  $0 --cursor"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            exit 1
            ;;
    esac
done

# 设置默认值
MCP_URL=${MCP_URL:-$DEFAULT_URL}

# 列表模式
if [ "$LIST_MODE" = true ]; then
    echo "已配置的 MCP 服务器:"
    echo ""

    # Claude Code CLI
    if [ -f "$HOME/.claude/settings.json" ]; then
        echo "Claude Code CLI (~/.claude/settings.json):"
        cat "$HOME/.claude/settings.json" | jq '.mcpServers' 2>/dev/null || cat "$HOME/.claude/settings.json"
        echo ""
    fi

    # Cursor
    if [ -f "$HOME/.cursor/mcp.json" ]; then
        echo "Cursor (~/.cursor/mcp.json):"
        cat "$HOME/.cursor/mcp.json" | jq '.mcpServers' 2>/dev/null || cat "$HOME/.cursor/mcp.json"
        echo ""
    fi

    exit 0
fi

# 获取 API Key
if [ -z "$API_KEY" ]; then
    echo ""
    read -p "请输入 API Key: " API_KEY
    if [ -z "$API_KEY" ]; then
        echo "❌ API Key 不能为空"
        exit 1
    fi
fi

echo ""
echo "MCP 服务地址: $MCP_URL"
echo "API Key: ${API_KEY:0:20}..."
echo ""

# 配置 Claude Code CLI
configure_claude() {
    echo "配置 Claude Code CLI..."

    mkdir -p "$HOME/.claude"

    # 备份现有配置
    if [ -f "$HOME/.claude/settings.json" ]; then
        cp "$HOME/.claude/settings.json" "$HOME/.claude/settings.json.backup"
        echo "已备份现有配置到 ~/.claude/settings.json.backup"
    fi

    # 合并配置
    if [ -f "$HOME/.claude/settings.json" ]; then
        # 使用 jq 合并
        jq --arg url "$MCP_URL" --arg key "$API_KEY" \
            '.mcpServers.open-station = {"url": $url, "headers": {"Authorization": "Bearer " + $key}}' \
            "$HOME/.claude/settings.json" > "$HOME/.claude/settings.json.tmp"
        mv "$HOME/.claude/settings.json.tmp" "$HOME/.claude/settings.json"
    else
        # 创建新配置
        cat > "$HOME/.claude/settings.json" << EOF
{
  "mcpServers": {
    "open-station": {
      "url": "$MCP_URL",
      "headers": {
        "Authorization": "Bearer $API_KEY"
      }
    }
  }
}
EOF
    fi

    echo "✅ Claude Code CLI 已配置"
    echo "   配置文件: ~/.claude/settings.json"
}

# 配置 Cursor IDE
configure_cursor() {
    echo "配置 Cursor IDE..."

    mkdir -p "$HOME/.cursor"

    cat > "$HOME/.cursor/mcp.json" << EOF
{
  "mcpServers": {
    "open-station": {
      "url": "$MCP_URL",
      "headers": {
        "Authorization": "Bearer $API_KEY"
      }
    }
  }
}
EOF

    echo "✅ Cursor IDE 已配置"
    echo "   配置文件: ~/.cursor/mcp.json"
}

# 配置 VS Code
configure_vscode() {
    echo "配置 VS Code..."

    mkdir -p "$HOME/.vscode"

    cat > "$HOME/.vscode/mcp.json" << EOF
{
  "mcpServers": {
    "open-station": {
      "url": "$MCP_URL",
      "headers": {
        "Authorization": "Bearer $API_KEY"
      }
    }
  }
}
EOF

    echo "✅ VS Code 已配置"
    echo "   配置文件: ~/.vscode/mcp.json"
}

# 根据客户端配置
case ${CLIENT:-"claude"} in
    claude)
        configure_claude
        ;;
    cursor)
        configure_cursor
        ;;
    vscode)
        configure_vscode
        ;;
    *)
        echo "未知客户端: $CLIENT"
        exit 1
        ;;
esac

# 测试连接
echo ""
echo "测试 MCP 连接..."

RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$MCP_URL" \
    -H "Authorization: Bearer $API_KEY" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}')

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | head -1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ MCP 连接成功"
    echo ""
    echo "可用工具:"
    echo "$BODY" | jq '.result.serverInfo' 2>/dev/null || echo "$BODY"
else
    echo "⚠️  MCP 连接测试失败 (HTTP $HTTP_CODE)"
fi

# 显示使用指南
echo ""
echo "=========================================="
echo "   使用指南"
echo "=========================================="
echo ""
echo "启动 Claude Code CLI:"
echo "  claude"
echo ""
echo "在 Claude 中使用 MCP 工具:"
echo "  > \"What's my balance?\""
echo "  > \"Create API key for user john@example.com\""
echo "  > \"List all API keys\""
echo ""
echo "管理命令:"
echo "  - 查看配置: $0 --list"
echo "  - 重新配置: $0 --claude --api-key <new-key>"
echo ""
echo "API Key 管理:"
echo "  - 创建新 Key: curl -X POST $MCP_URL ..."
echo "  - 查看余额: curl -X POST $MCP_URL -d '{\"method\":\"tools/call\",\"params\":{\"name\":\"check_balance\"}}'"
echo ""