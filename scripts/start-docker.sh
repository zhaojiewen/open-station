#!/bin/bash

# Open Station 一键启动脚本 (Docker)
# 自动检测并安装 Docker

set -e

echo "=========================================="
echo "   Open Station - 一键启动 (Docker)"
echo "=========================================="

# 检测操作系统
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        return "macos"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get &> /dev/null; then
            return "debian"
        elif command -v yum &> /dev/null || command -v dnf &> /dev/null; then
            return "redhat"
        elif command -v pacman &> /dev/null; then
            return "arch"
        fi
        return "linux"
    else
        return "unknown"
    fi
}

# 检查并安装 Docker
install_docker_macos() {
    echo ""
    echo "检测到 macOS 系统"
    echo ""

    # 检查 Homebrew
    if ! command -v brew &> /dev/null; then
        echo "Homebrew 未安装，正在安装..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

        # 配置 Homebrew PATH
        if [[ -d "/opt/homebrew" ]]; then
            eval "$(/opt/homebrew/bin/brew shellenv)"
        elif [[ -d "/usr/local/Homebrew" ]]; then
            eval "$(/usr/local/Homebrew/bin/brew shellenv)"
        fi
    fi

    echo "安装 Docker..."

    # 选择安装方式
    echo ""
    echo "Docker 安装选项:"
    echo "  1. Docker Desktop (推荐，图形界面)"
    echo "  2. Colima (轻量级，命令行)"
    read -p "选择 [1]: " DOCKER_CHOICE
    DOCKER_CHOICE=${DOCKER_CHOICE:-1}

    case $DOCKER_CHOICE in
        1)
            echo "安装 Docker Desktop..."
            brew install --cask docker

            echo ""
            echo "启动 Docker Desktop..."
            open /Applications/Docker.app

            echo "等待 Docker Desktop 启动..."
            sleep 30

            # 等待 Docker 就绪
            until docker info &> /dev/null; do
                echo "等待 Docker 启动..."
                sleep 5
            done
            ;;
        2)
            echo "安装 Colima..."
            brew install colima docker docker-compose

            echo ""
            echo "启动 Colima..."
            colima start

            echo "等待 Colima 启动..."
            sleep 15
            ;;
        *)
            echo "无效选择，安装 Docker Desktop"
            brew install --cask docker
            open /Applications/Docker.app
            sleep 30
            ;;
    esac

    echo "✅ Docker 已安装并启动"
}

install_docker_debian() {
    echo ""
    echo "检测到 Debian/Ubuntu 系统"
    echo ""

    echo "安装 Docker..."

    # 更新包列表
    sudo apt-get update

    # 安装依赖
    sudo apt-get install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg \
        lsb-release

    # 添加 Docker GPG key
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg

    # 添加 Docker 仓库
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    # 安装 Docker
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

    # 启动 Docker
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加当前用户到 docker 组
    sudo usermod -aG docker $USER

    echo "✅ Docker 已安装"
    echo "⚠️  请运行 'newgrp docker' 或重新登录以使用 docker 命令"
}

install_docker_redhat() {
    echo ""
    echo "检测到 RedHat/CentOS/Fedora 系统"
    echo ""

    echo "安装 Docker..."

    # 安装依赖
    if command -v dnf &> /dev/null; then
        sudo dnf install -y dnf-plugins-core
        sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
        sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    else
        sudo yum install -y yum-utils
        sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    fi

    # 启动 Docker
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加当前用户到 docker 组
    sudo usermod -aG docker $USER

    echo "✅ Docker 已安装"
    echo "⚠️  请运行 'newgrp docker' 或重新登录以使用 docker 命令"
}

install_docker_arch() {
    echo ""
    echo "检测到 Arch Linux 系统"
    echo ""

    echo "安装 Docker..."

    sudo pacman -Syu --noconfirm docker docker-compose

    # 启动 Docker
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加当前用户到 docker 组
    sudo usermod -aG docker $USER

    echo "✅ Docker 已安装"
}

install_docker() {
    OS=$(detect_os)

    case $OS in
        macos)
            install_docker_macos
            ;;
        debian)
            install_docker_debian
            ;;
        redhat)
            install_docker_redhat
            ;;
        arch)
            install_docker_arch
            ;;
        *)
            echo "❌ 无法识别操作系统: $OSTYPE"
            echo ""
            echo "请手动安装 Docker:"
            echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
            echo "  Ubuntu: https://docs.docker.com/engine/install/ubuntu/"
            echo "  CentOS: https://docs.docker.com/engine/install/centos/"
            echo "  其他: https://docs.docker.com/engine/install/"
            exit 1
            ;;
    esac
}

# 检查 Docker 和 Docker Compose
check_docker() {
    # 检查 Docker
    if ! command -v docker &> /dev/null; then
        echo ""
        echo "⚠️  Docker 未安装"
        echo ""
        read -p "是否自动安装 Docker? [Y/n]: " INSTALL_DOCKER

        if [[ "$INSTALL_DOCKER" =~ ^[Yy]?$ ]]; then
            install_docker
        else
            echo ""
            echo "请先安装 Docker:"
            echo "  macOS: brew install --cask docker"
            echo "  Ubuntu: sudo apt-get install docker-ce"
            echo "  或访问: https://docs.docker.com/get-docker/"
            exit 1
        fi
    fi

    # 检查 Docker 是否运行
    if ! docker info &> /dev/null; then
        echo ""
        echo "⚠️  Docker 未运行"
        echo ""

        if [[ "$OSTYPE" == "darwin"* ]]; then
            echo "启动 Docker Desktop..."
            open /Applications/Docker.app 2>/dev/null || colima start 2>/dev/null || {
                echo "请手动启动 Docker Desktop 或 Colima"
                exit 1
            }

            # 等待启动
            echo "等待 Docker 启动..."
            until docker info &> /dev/null; do
                sleep 5
            done

            echo "✅ Docker 已启动"
        else
            echo "启动 Docker 服务..."
            sudo systemctl start docker || sudo service docker start
            sleep 10
        fi
    fi

    # 检查 Docker Compose
    if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
        echo ""
        echo "⚠️  Docker Compose 未安装"
        echo ""

        if [[ "$OSTYPE" == "darwin"* ]]; then
            echo "安装 Docker Compose..."
            brew install docker-compose
        else
            echo "Docker Compose 通常随 Docker 安装"
            echo "如未安装，请运行:"
            echo "  sudo apt-get install docker-compose-plugin"
            exit 1
        fi
    fi

    echo "✅ Docker 环境已就绪"
}

# 检查端口占用
check_ports() {
    echo ""
    echo "检查端口..."

    PORTS=(8080 5432 6379)

    for PORT in "${PORTS[@]}"; do
        if lsof -Pi :$PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
            echo "⚠️  端口 $PORT 已被占用"
            read -p "是否停止占用进程? [Y/n]: " STOP_PROCESS

            if [[ "$STOP_PROCESS" =~ ^[Yy]?$ ]]; then
                lsof -ti :$PORT | xargs kill -9 2>/dev/null || true
                echo "✅ 已停止端口 $PORT 的进程"
            else
                echo "请先释放端口 $PORT"
                exit 1
            fi
        fi
    done

    echo "✅ 端口检查完成"
}

# 创建配置文件
create_config() {
    if [ ! -f ".env" ]; then
        echo ""
        echo "创建配置文件..."

        cat > .env << EOF
# Provider API Keys (可选，稍后可编辑此文件)
OPENAI_API_KEY=
ANTHROPIC_API_KEY=
GEMINI_API_KEY=
DEEPSEEK_API_KEY=
GLM_API_KEY=

# 管理员配置
ADMIN_USER=admin
ADMIN_EMAIL=admin@localhost
ADMIN_PASS=admin123
EOF

        echo "✅ 配置文件已创建: .env"
        echo ""
        echo "请编辑 .env 文件添加 Provider API Keys（可选）"
    fi
}

# 启动服务
start_services() {
    echo ""
    echo "启动 Open Station 服务..."

    # 加载环境变量
    if [ -f ".env" ]; then
        export $(cat .env | grep -v '^#' | xargs)
    fi

    # 启动 Docker 服务
    if docker compose version &> /dev/null; then
        docker compose -f deployments/docker/docker-compose.yml up -d
    else
        docker-compose -f deployments/docker/docker-compose.yml up -d
    fi

    echo "等待服务就绪..."
    sleep 15

    # 检查服务
    MAX_WAIT=60
    WAIT_COUNT=0

    until curl -s http://localhost:8080/health > /dev/null 2>&1; do
        WAIT_COUNT=$((WAIT_COUNT + 5))

        if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
            echo "⚠️  服务启动超时"
            echo ""
            echo "请检查日志:"
            echo "  docker compose -f deployments/docker/docker-compose.yml logs"
            exit 1
        fi

        echo "等待 API Gateway 启动... ($WAIT_COUNT/$MAX_WAIT 秒)"
        sleep 5
    done

    echo "✅ 服务已启动"
}

# 显示结果
show_result() {
    echo ""
    echo "=========================================="
    echo "   Open Station 已就绪"
    echo "=========================================="

    # 获取 API Key
    API_KEY=""

    if docker compose version &> /dev/null; then
        API_KEY=$(docker compose -f deployments/docker/docker-compose.yml logs app 2>&1 | grep -o 'sk-[a-f0-9]*' | tail -1)
    else
        API_KEY=$(docker-compose -f deployments/docker/docker-compose.yml logs app 2>&1 | grep -o 'sk-[a-f0-9]*' | tail -1)
    fi

    if [ -z "$API_KEY" ]; then
        # 如果日志中找不到，尝试直接调用 MCP 初始化获取
        sleep 5
        if docker compose version &> /dev/null; then
            API_KEY=$(docker compose -f deployments/docker/docker-compose.yml logs app 2>&1 | grep -o 'sk-[a-f0-9]*' | tail -1)
        else
            API_KEY=$(docker-compose -f deployments/docker/docker-compose.yml logs app 2>&1 | grep -o 'sk-[a-f0-9]*' | tail -1)
        fi
    fi

    echo ""
    echo "服务地址:"
    echo "  API Gateway:  http://localhost:8080"
    echo "  MCP Endpoint: http://localhost:8080/mcp"
    echo ""

    if [ -n "$API_KEY" ]; then
        echo "管理员 API Key:"
        echo "  $API_KEY"
        echo ""
        echo "快速测试:"
        echo "  curl http://localhost:8080/v1/models -H 'Authorization: Bearer $API_KEY'"
        echo ""

        # 自动配置 MCP
        if command -v claude &> /dev/null; then
            read -p "是否配置 Claude Code CLI MCP? [Y/n]: " CONFIGURE

            if [[ "$CONFIGURE" =~ ^[Yy]?$ ]]; then
                if [ -f "scripts/setup-mcp.sh" ]; then
                    ./scripts/setup-mcp.sh --claude --api-key "$API_KEY"
                else
                    echo ""
                    echo "手动配置 MCP:"
                    echo "  mkdir -p ~/.claude"
                    echo "  cat > ~/.claude/settings.json << 'EOF'"
                    echo '  {'
                    echo '    "mcpServers": {'
                    echo '      "open-station": {'
                    echo '        "url": "http://localhost:8080/mcp",'
                    echo '        "headers": { "Authorization": "Bearer $API_KEY" }'
                    echo '      }'
                    echo '    }'
                    echo '  }'
                    echo '  EOF'
                fi
            fi
        fi
    else
        echo "⚠️  未找到 API Key，请查看日志获取:"
        echo "  docker compose logs app | grep 'API Key'"
    fi

    echo ""
    echo "=========================================="
    echo "   管理命令"
    echo "=========================================="
    echo ""
    echo "查看状态:"
    echo "  make status"
    echo ""
    echo "查看日志:"
    echo "  make logs"
    echo ""
    echo "停止服务:"
    echo "  make stop"
    echo ""
    echo "创建用户:"
    echo "  ./scripts/user-admin.sh"
    echo ""
}

# 主流程
main() {
    check_docker
    check_ports
    create_config
    start_services
    show_result
}

# 运行
main