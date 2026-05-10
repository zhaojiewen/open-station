#!/bin/bash

# Open Station 启动脚本 - 带 Provider 配置
# 支持启动时配置 Provider，也可跳过稍后配置

set -e

echo "=========================================="
echo "   Open Station - 启动配置"
echo "=========================================="

# 获取 Manager API Key
MANAGER_KEY="${MANAGER_KEY:-}"
if [ -z "$MANAGER_KEY" ]; then
    if [ -f "server.log" ]; then
        MANAGER_KEY=$(grep -o 'sk-[a-f0-9]*' server.log | tail -1)
    fi
fi

# Provider 配置
configure_providers() {
    echo ""
    echo "=========================================="
    echo "   Provider API 配置"
    echo "=========================================="
    echo ""
    echo "支持配置多个 Provider，同一 Provider 可配置多个账户用于故障切换"
    echo ""
    echo "可配置的 Provider:"
    echo "  1. openai     - GPT-4o, GPT-4o-mini, O1"
    echo "  2. anthropic  - Claude Opus, Sonnet, Haiku"
    echo "  3. gemini     - Gemini 2.5, 3.1"
    echo "  4. deepseek   - V4 Pro, V4 Flash"
    echo "  5. glm        - GLM-4, GLM-5"
    echo ""
    echo "⚠️  提示:"
    echo "  - 可以跳过配置，稍后通过 MCP/API 配置"
    echo "  - 每个 Provider 可配置多个账户，用于故障切换"
    echo "  - 首个账户自动设为默认，后续账户按优先级排序"
    echo ""

    read -p "是否现在配置 Provider? [y/N]: " CONFIGURE
    if [[ ! "$CONFIGURE" =~ ^[Yy]$ ]]; then
        echo ""
        echo "跳过 Provider 配置"
        echo ""
        echo "稍后可通过以下方式配置:"
        echo "  - Claude Code CLI: 'Create provider account for openai'"
        echo "  - MCP 工具: ./scripts/setup-provider.sh"
        echo "  - API 接口: POST /admin/providers"
        echo ""
        return 0
    fi

    echo ""
    configure_provider "openai" "OpenAI"
    configure_provider "anthropic" "Anthropic (Claude)"
    configure_provider "gemini" "Google Gemini"
    configure_provider "deepseek" "DeepSeek"
    configure_provider "glm" "GLM (智谱)"
}

configure_provider() {
    PROVIDER=$1
    DISPLAY_NAME=$2

    echo "----------------------------------------"
    echo "配置 $DISPLAY_NAME"
    echo "----------------------------------------"

    # 获取主账户 API Key
    read -p "$DISPLAY_NAME API Key (可跳过): " API_KEY

    if [ -z "$API_KEY" ]; then
        echo "跳过 $DISPLAY_NAME 配置"
        echo ""
        return 0
    fi

    # 获取账户名称
    DEFAULT_NAME="$PROVIDER-primary"
    read -p "账户名称 [$DEFAULT_NAME]: " ACCOUNT_NAME
    ACCOUNT_NAME=${ACCOUNT_NAME:-$DEFAULT_NAME}

    # 配置主账户
    echo "创建 $DISPLAY_NAME 主账户..."
    create_provider_account "$PROVIDER" "$ACCOUNT_NAME" "$API_KEY" 0

    # 检查是否配置备用账户
    echo ""
    read -p "是否配置备用账户? [y/N]: " ADD_BACKUP
    if [[ "$ADD_BACKUP" =~ ^[Yy]$ ]]; then
        configure_backup_accounts "$PROVIDER" "$DISPLAY_NAME"
    fi

    echo ""
}

configure_backup_accounts() {
    PROVIDER=$1
    DISPLAY_NAME=$2
    PRIORITY=1

    while true; do
        read -p "备用账户 API Key (可结束): " BACKUP_KEY

        if [ -z "$BACKUP_KEY" ]; then
            break
        fi

        read -p "备用账户名称 [$PROVIDER-backup-$PRIORITY]: " BACKUP_NAME
        BACKUP_NAME=${BACKUP_NAME:-"$PROVIDER-backup-$PRIORITY"}

        # 月度限额（可选）
        read -p "月度限额 ($)(可选): " MONTHLY_LIMIT

        echo "创建备用账户 (优先级: $PRIORITY)..."
        create_provider_account "$PROVIDER" "$BACKUP_NAME" "$BACKUP_KEY" $PRIORITY "$MONTHLY_LIMIT"

        PRIORITY=$((PRIORITY + 1))
        echo ""
    done
}

create_provider_account() {
    PROVIDER=$1
    NAME=$2
    API_KEY=$3
    PRIORITY=$4
    MONTHLY_LIMIT=$5

    # 构建 JSON
    if [ -n "$MONTHLY_LIMIT" ]; then
        PARAMS="{\"provider\":\"$PROVIDER\",\"name\":\"$NAME\",\"api_key\":\"$API_KEY\",\"priority\":$PRIORITY,\"monthly_limit\":$MONTHLY_LIMIT}"
    else
        PARAMS="{\"provider\":\"$PROVIDER\",\"name\":\"$NAME\",\"api_key\":\"$API_KEY\",\"priority\":$PRIORITY}"
    fi

    # 调用 MCP 创建账户
    RESULT=$(curl -s -X POST http://localhost:8080/mcp \
        -H "Authorization: Bearer $MANAGER_KEY" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"create_provider_account\",\"arguments\":$PARAMS}}")

    # 检查结果
    if echo "$RESULT" | grep -q "created successfully"; then
        ACCOUNT_ID=$(echo "$RESULT" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
        echo "✅ 账户创建成功: $NAME (ID: $ACCOUNT_ID)"

        if [ "$PRIORITY" -eq 0 ]; then
            echo "   已设为默认账户"
        else
            echo "   优先级: $PRIORITY (故障切换顺序)"
        fi
    else
        echo "⚠️  创建失败"
        echo "$RESULT" | jq '.' 2>/dev/null || echo "$RESULT"
    fi
}

# 显示配置结果
show_provider_status() {
    echo ""
    echo "=========================================="
    echo "   Provider 配置状态"
    echo "=========================================="
    echo ""

    # 获取所有 Provider 状态
    curl -s -X POST http://localhost:8080/mcp \
        -H "Authorization: Bearer $MANAGER_KEY" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_provider_status","arguments":{}}}' \
        | jq '.result.content[0].text' -r | jq '.' 2>/dev/null

    echo ""
    echo "配置管理:"
    echo "  - 查看账户: Claude Code 中输入 'Show provider accounts'"
    echo "  - 添加账户: Claude Code 中输入 'Create provider account for xxx'"
    echo "  - 设置默认: Claude Code 中输入 'Set xxx as default for openai'"
    echo "  - 禁用账户: Claude Code 中输入 'Disable provider account xxx'"
    echo ""
    echo "故障切换:"
    echo "  - 当账户遇到 rate limit 或余额不足时自动切换到备用账户"
    echo "  - 连续失败 5 次自动标记为 limited"
    echo "  - 每月 1 日自动重置用量统计"
    echo ""
}

# 主流程
main() {
    # 检查服务是否运行
    if ! curl -s http://localhost:8080/health > /dev/null 2>&1; then
        echo "⚠️  Open Station 服务未运行"
        echo "请先启动服务: make start"
        exit 1
    fi

    # 检查 Manager Key
    if [ -z "$MANAGER_KEY" ]; then
        echo "⚠️  未找到 Manager API Key"
        read -p "请输入 Manager API Key: " MANAGER_KEY
    fi

    # 配置 Providers
    configure_providers

    # 显示状态
    show_provider_status
}

main