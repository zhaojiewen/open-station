#!/bin/bash

# Open Station 用户管理脚本
# 简化 API Key 创建和管理

set -e

API_URL="${API_URL:-http://localhost:8080}"
MANAGER_KEY="${MANAGER_KEY:-}"

# 获取 Manager Key
if [ -z "$MANAGER_KEY" ]; then
    if [ -f "server.log" ]; then
        MANAGER_KEY=$(grep -o 'sk-[a-f0-9]*' server.log | tail -1)
    fi
    if [ -z "$MANAGER_KEY" ]; then
        echo "请提供 Manager API Key:"
        read -p "Manager Key: " MANAGER_KEY
    fi
fi

# MCP 请求函数
mcp_call() {
    local METHOD="$1"
    local PARAMS="$2"

    # 初始化 session
    SESSION_ID=$(curl -s -X POST "$API_URL/mcp" \
        -H "Authorization: Bearer $MANAGER_KEY" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"admin","version":"1.0"}}}' \
        -i 2>/dev/null | grep -i "MCP-Session-Id" | cut -d' ' -f2 | tr -d '\r\n')

    # 调用工具
    curl -s -X POST "$API_URL/mcp" \
        -H "Authorization: Bearer $MANAGER_KEY" \
        -H "MCP-Session-Id: $SESSION_ID" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$METHOD\",\"arguments\":$PARAMS}}"
}

# 创建用户和 API Key
create_user() {
    echo ""
    echo "创建新用户和 API Key"
    echo "===================="
    echo ""

    read -p "用户邮箱: " EMAIL
    if [ -z "$EMAIL" ]; then
        echo "邮箱不能为空"
        return 1
    fi

    read -p "用户名 [$EMAIL]: " NAME
    NAME=${NAME:-$EMAIL}

    read -p "API Key 名称 [default-key]: " KEY_NAME
    KEY_NAME=${KEY_NAME:-default-key}

    echo ""
    echo "权限选择:"
    echo "  1. chat (普通用户)"
    echo "  2. chat + embeddings"
    echo "  3. admin (管理员)"
    read -p "选择 [1]: " PERM_CHOICE
    PERM_CHOICE=${PERM_CHOICE:-1}

    case $PERM_CHOICE in
        1) PERMISSIONS='["chat"]' ;;
        2) PERMISSIONS='["chat","embeddings"]' ;;
        3) PERMISSIONS='["admin","manage","chat"]' ;;
        *) PERMISSIONS='["chat"]' ;;
    esac

    echo ""
    echo "创建中..."

    RESULT=$(mcp_call "create_api_key" "{\"user_email\":\"$EMAIL\",\"user_name\":\"$NAME\",\"name\":\"$KEY_NAME\",\"permissions\":$PERMISSIONS}")

    # 解析结果
    RAW_KEY=$(echo "$RESULT" | jq -r '.result.content[0].text' | jq -r '.raw_key' 2>/dev/null)
    USER_ID=$(echo "$RESULT" | jq -r '.result.content[0].text' | jq -r '.user_id' 2>/dev/null)
    IS_NEW=$(echo "$RESULT" | jq -r '.result.content[0].text' | jq -r '.is_new_user' 2>/dev/null)

    if [ -n "$RAW_KEY" ]; then
        echo ""
        echo "✅ 创建成功!"
        echo ""
        echo "用户信息:"
        echo "  - 邮箱: $EMAIL"
        echo "  - 名称: $NAME"
        echo "  - ID: $USER_ID"
        echo "  - 新用户: $IS_NEW"
        echo ""
        echo "API Key:"
        echo "  $RAW_KEY"
        echo ""
        echo "请保存 API Key，用户需要使用此 Key 访问服务"
    else
        echo "❌ 创建失败"
        echo "$RESULT" | jq '.' 2>/dev/null || echo "$RESULT"
    fi
}

# 查询余额
check_balance() {
    echo ""
    echo "查询余额"
    echo "========"

    RESULT=$(mcp_call "check_balance" "{}")
    BALANCE=$(echo "$RESULT" | jq -r '.result.content[0].text' 2>/dev/null)

    echo ""
    echo "$BALANCE"
}

# 列出所有用户
list_users() {
    echo ""
    echo "用户列表"
    echo "========"

    RESULT=$(mcp_call "list_users" "{}")

    echo ""
    echo "$RESULT" | jq -r '.result.content[0].text' | jq '.' 2>/dev/null || echo "$RESULT"
}

# 列出所有 API Key
list_keys() {
    echo ""
    echo "API Key 列表"
    echo "============"

    RESULT=$(mcp_call "list_all_api_keys" "{}")

    echo ""
    echo "$RESULT" | jq -r '.result.content[0].text' | jq '.keys[] | {id, key_prefix, name, status}' 2>/dev/null || echo "$RESULT"
}

# 撤销 API Key
revoke_key() {
    echo ""
    echo "撤销 API Key"
    echo "============"

    read -p "API Key ID: " KEY_ID
    if [ -z "$KEY_ID" ]; then
        echo "Key ID 不能为空"
        return 1
    fi

    RESULT=$(mcp_call "revoke_api_key" "{\"api_key_id\":\"$KEY_ID\"}")

    echo ""
    echo "$RESULT" | jq -r '.result.content[0].text' 2>/dev/null || echo "$RESULT"
}

# 主菜单
main_menu() {
    echo ""
    echo "=========================================="
    echo "   Open Station 用户管理"
    echo "=========================================="
    echo ""
    echo "操作:"
    echo "  1. 创建用户和 API Key"
    echo "  2. 查询余额"
    echo "  3. 列出用户"
    echo "  4. 列出 API Key"
    echo "  5. 撤销 API Key"
    echo "  0. 退出"
    echo ""

    read -p "选择操作: " CHOICE

    case $CHOICE in
        1) create_user ;;
        2) check_balance ;;
        3) list_users ;;
        4) list_keys ;;
        5) revoke_key ;;
        0) exit 0 ;;
        *) echo "无效选择" ;;
    esac

    main_menu
}

main_menu