#!/bin/bash

# Open Station 快速测试脚本

set -e

echo "=========================================="
echo "   Open Station 功能测试"
echo "=========================================="

API_URL="${API_URL:-http://localhost:8080}"
API_KEY="${API_KEY:-}"

# 获取 API Key
if [ -z "$API_KEY" ]; then
    if [ -f "server.log" ]; then
        API_KEY=$(grep -o 'sk-[a-f0-9]*' server.log | tail -1)
    fi
    if [ -z "$API_KEY" ]; then
        echo "请提供 API Key:"
        echo "  API_KEY=sk-xxx ./scripts/test.sh"
        exit 1
    fi
fi

echo "API URL: $API_URL"
echo "API Key: ${API_KEY:0:20}..."
echo ""

# 测试健康检查
test_health() {
    echo "1. 健康检查..."
    HEALTH=$(curl -s "$API_URL/health")
    if [ "$HEALTH" = '{"status":"ok"}' ] || [ "$HEALTH" = 'ok' ]; then
        echo "   ✅ API Gateway 运行正常"
    else
        echo "   ❌ API Gateway 异常: $HEALTH"
        return 1
    fi
}

# 测试模型列表
test_models() {
    echo "2. 模型列表..."
    MODELS=$(curl -s "$API_URL/v1/models" -H "Authorization: Bearer $API_KEY")
    COUNT=$(echo "$MODELS" | jq '.data | length' 2>/dev/null || echo "0")
    if [ "$COUNT" -gt 0 ]; then
        echo "   ✅ 可用模型: $COUNT 个"
        echo "$MODELS" | jq '.data[].id' -r | head -5 | while read m; do echo "      - $m"; done
    else
        echo "   ❌ 模型列表获取失败"
        return 1
    fi
}

# 测试 MCP 初始化
test_mcp() {
    echo "3. MCP 初始化..."
    MCP_RESP=$(curl -s -X POST "$API_URL/mcp" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}')

    SERVER_NAME=$(echo "$MCP_RESP" | jq -r '.result.serverInfo.name' 2>/dev/null)
    if [ "$SERVER_NAME" = "open-station" ]; then
        SESSION_ID=$(curl -s -X POST "$API_URL/mcp" \
            -H "Authorization: Bearer $API_KEY" \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' \
            -i 2>/dev/null | grep -i "MCP-Session-Id" | cut -d' ' -f2 | tr -d '\r\n')
        echo "   ✅ MCP 服务正常 (Session: $SESSION_ID)"

        # 测试工具列表
        TOOLS=$(curl -s -X POST "$API_URL/mcp" \
            -H "Authorization: Bearer $API_KEY" \
            -H "MCP-Session-Id: $SESSION_ID" \
            -H "Content-Type: application/json" \
            -d '{"jsonrpc":"2.0","id":2,"method":"tools/list"}')
        TOOL_COUNT=$(echo "$TOOLS" | jq '.result.tools | length' 2>/dev/null || echo "0")
        echo "   ✅ MCP 工具: $TOOL_COUNT 个"
    else
        echo "   ❌ MCP 初始化失败"
        return 1
    fi
}

# 测试余额查询
test_balance() {
    echo "4. 余额查询..."
    SESSION_ID=$(curl -s -X POST "$API_URL/mcp" \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' \
        -i 2>/dev/null | grep -i "MCP-Session-Id" | cut -d' ' -f2 | tr -d '\r\n')

    BALANCE=$(curl -s -X POST "$API_URL/mcp" \
        -H "Authorization: Bearer $API_KEY" \
        -H "MCP-Session-Id: $SESSION_ID" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"check_balance","arguments":{}}}')

    BALANCE_TEXT=$(echo "$BALANCE" | jq -r '.result.content[0].text' 2>/dev/null)
    if echo "$BALANCE_TEXT" | grep -q "Balance"; then
        echo "   ✅ $BALANCE_TEXT"
    else
        echo "   ⚠️  余额查询: $BALANCE_TEXT"
    fi
}

# 运行测试
test_health
test_models
test_mcp
test_balance

echo ""
echo "=========================================="
echo "   测试完成!"
echo "=========================================="
echo ""
echo "下一步:"
echo "  1. 配置 MCP: make mcp-config"
echo "  2. 创建用户: 在 Claude Code 中使用 create_api_key 工具"
echo "  3. 查看文档: docs/mcp-integration.md"
echo ""