#!/bin/bash

# Open Station 快速安装脚本
# 用于企业快速部署 AI Gateway

set -e

echo "=========================================="
echo "   Open Station - 企业级AI网关快速安装"
echo "=========================================="

# 检查依赖
check_dependencies() {
    echo "检查依赖..."

    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        echo "❌ Docker 未安装"
        echo "请先安装 Docker: https://docs.docker.com/get-docker/"
        exit 1
    fi
    echo "✅ Docker 已安装"

    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo "❌ Docker Compose 未安装"
        echo "请先安装 Docker Compose"
        exit 1
    fi
    echo "✅ Docker Compose 已安装"

    # 检查 Go (可选，用于本地开发)
    if command -v go &> /dev/null; then
        echo "✅ Go 已安装 (版本: $(go version | awk '{print $3}'))"
    else
        echo "⚠️  Go 未安装 (可选，仅用于本地开发)"
    fi
}

# 创建配置文件
create_config() {
    echo ""
    echo "创建配置文件..."

    CONFIG_DIR="configs"
    mkdir -p $CONFIG_DIR

    # 提示用户配置
    echo ""
    echo "请配置管理员信息:"
    read -p "管理员用户名 [admin]: " ADMIN_USER
    ADMIN_USER=${ADMIN_USER:-admin}

    read -p "管理员邮箱 [admin@company.local]: " ADMIN_EMAIL
    ADMIN_EMAIL=${ADMIN_EMAIL:-admin@company.local}

    read -sp "管理员密码 [changeme123]: " ADMIN_PASS
    echo
    ADMIN_PASS=${ADMIN_PASS:-changeme123}

    # 生成配置文件
    cat > $CONFIG_DIR/config.yaml << EOF
server:
  port: 8080
  mode: release

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  dbname: ai_gateway
  sslmode: disable
  max_open_conns: 100
  max_idle_conns: 10
  conn_max_lifetime: 1h

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 100

providers:
  openai:
    base_url: https://api.openai.com/v1
    api_key: \${OPENAI_API_KEY}
    timeout: 120s
  claude:
    base_url: https://api.anthropic.com/v1
    api_key: \${ANTHROPIC_API_KEY}
    timeout: 120s
  gemini:
    base_url: https://generativelanguage.googleapis.com/v1beta
    api_key: \${GEMINI_API_KEY}
    timeout: 120s
  deepseek:
    base_url: https://api.deepseek.com/v1
    api_key: \${DEEPSEEK_API_KEY}
    timeout: 120s
  glm:
    base_url: https://open.bigmodel.cn/api/paas/v4
    api_key: \${GLM_API_KEY}
    timeout: 120s

billing:
  default_currency: USD
  min_balance_alert: 10.00

rate_limit:
  default_user_rps: 10
  default_user_burst: 20
  default_tenant_rps: 100
  default_tenant_burst: 200
  redis_key_prefix: "ratelimit:"
  window_size: 1s

logging:
  level: info
  format: json
  output: stdout

admin:
  default_tenant_slug: "admin-tenant"
  super_admin_email: "${ADMIN_EMAIL}"
  default_admin_user: "${ADMIN_USER}"
  default_admin_pass: "${ADMIN_PASS}"
  initial_api_key_name: "Initial Manager Key"
EOF

    echo "✅ 配置文件已创建: $CONFIG_DIR/config.yaml"
}

# 配置 Provider API Keys
configure_providers() {
    echo ""
    echo "配置 Provider API Keys (可选，按回车跳过):"

    ENV_FILE=".env"

    read -p "OpenAI API Key: " OPENAI_KEY
    if [ -n "$OPENAI_KEY" ]; then
        echo "OPENAI_API_KEY=$OPENAI_KEY" >> $ENV_FILE
    fi

    read -p "Anthropic API Key: " ANTHROPIC_KEY
    if [ -n "$ANTHROPIC_KEY" ]; then
        echo "ANTHROPIC_API_KEY=$ANTHROPIC_KEY" >> $ENV_FILE
    fi

    read -p "Gemini API Key: " GEMINI_KEY
    if [ -n "$GEMINI_KEY" ]; then
        echo "GEMINI_API_KEY=$GEMINI_KEY" >> $ENV_FILE
    fi

    read -p "DeepSeek API Key: " DEEPSEEK_KEY
    if [ -n "$DEEPSEEK_KEY" ]; then
        echo "DEEPSEEK_API_KEY=$DEEPSEEK_KEY" >> $ENV_FILE
    fi

    read -p "GLM API Key: " GLM_KEY
    if [ -n "$GLM_KEY" ]; then
        echo "GLM_API_KEY=$GLM_KEY" >> $ENV_FILE
    fi

    if [ -f "$ENV_FILE" ]; then
        echo "✅ Provider API Keys 已保存到 $ENV_FILE"
        export $(cat $ENV_FILE | xargs)
    fi
}

# 启动基础设施
start_infrastructure() {
    echo ""
    echo "启动基础设施..."

    if [ -f "deployments/docker/docker-compose.yml" ]; then
        docker-compose -f deployments/docker/docker-compose.yml up -d postgres redis
        echo "等待数据库启动..."
        sleep 10

        # 检查服务状态
        if docker-compose -f deployments/docker/docker-compose.yml ps | grep -q "healthy"; then
            echo "✅ PostgreSQL 和 Redis 已启动"
        else
            echo "⚠️  服务状态检查中..."
            sleep 5
        fi
    else
        echo "❌ docker-compose.yml 未找到"
        exit 1
    fi
}

# 构建并启动服务
start_server() {
    echo ""
    echo "启动 Open Station 服务..."

    # 加载环境变量
    if [ -f ".env" ]; then
        export $(cat .env | grep -v '^#' | xargs)
    fi

    # 编译并运行
    if command -v go &> /dev/null; then
        go mod tidy
        go build -o open-station ./cmd/server

        # 后台运行
        nohup ./open-station -config configs/config.yaml > server.log 2>&1 &
        SERVER_PID=$!
        echo $SERVER_PID > server.pid

        echo "等待服务启动..."
        sleep 5

        # 检查服务
        if curl -s http://localhost:8080/health > /dev/null; then
            echo "✅ Open Station 服务已启动 (PID: $SERVER_PID)"
        else
            echo "⚠️  服务启动中，请稍后检查..."
        fi
    else
        echo "使用 Docker 运行..."
        docker-compose -f deployments/docker/docker-compose.yml up -d app
        echo "✅ Open Station 服务已在 Docker 中启动"
    fi
}

# 显示安装结果
show_result() {
    echo ""
    echo "=========================================="
    echo "   安装完成!"
    echo "=========================================="

    echo ""
    echo "服务地址:"
    echo "  - API Gateway: http://localhost:8080"
    echo "  - MCP Endpoint: http://localhost:8080/mcp"
    echo ""

    # 获取管理员 API Key
    echo "管理员信息:"
    echo "  - 用户名: $ADMIN_USER"
    echo "  - 邮箱: $ADMIN_EMAIL"
    echo ""

    # 提取 API Key
    if [ -f "server.log" ]; then
        API_KEY=$(grep -o 'sk-[a-f0-9]*' server.log | tail -1)
        if [ -n "$API_KEY" ]; then
            echo "  - Manager API Key: $API_KEY"
            echo ""

            # 创建 MCP 配置脚本
            cat > scripts/setup-mcp.sh << MCPSCRIPT
#!/bin/bash
# 配置 Claude Code CLI MCP

API_KEY="$API_KEY"
MCP_URL="http://localhost:8080/mcp"

echo "配置 Claude Code CLI MCP..."

mkdir -p ~/.claude

cat > ~/.claude/settings.json << 'EOF'
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

echo "✅ MCP 已配置到 ~/.claude/settings.json"
echo ""
echo "测试 MCP 连接:"
claude
MCPSCRIPT
            chmod +x scripts/setup-mcp.sh
            echo "运行 'scripts/setup-mcp.sh' 配置 Claude Code CLI"
        fi
    fi

    echo ""
    echo "快速测试:"
    echo "  curl http://localhost:8080/v1/models -H 'Authorization: Bearer \$API_KEY'"
    echo ""
    echo "停止服务:"
    echo "  ./scripts/stop.sh"
    echo ""
    echo "查看日志:"
    echo "  tail -f server.log"
}

# 创建停止脚本
create_stop_script() {
    cat > scripts/stop.sh << 'EOF'
#!/bin/bash
echo "停止 Open Station..."

# 停止 Go 服务
if [ -f "server.pid" ]; then
    PID=$(cat server.pid)
    kill $PID 2>/dev/null || true
    rm server.pid
fi

# 停止 Docker 服务
docker-compose -f deployments/docker/docker-compose.yml down

echo "✅ 服务已停止"
EOF
    chmod +x scripts/stop.sh
}

# 主流程
main() {
    check_dependencies
    create_config
    configure_providers
    start_infrastructure
    create_stop_script
    start_server
    show_result
}

main