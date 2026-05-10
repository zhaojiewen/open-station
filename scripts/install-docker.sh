#!/bin/bash

# Open Station Docker 安装脚本
# 支持自动安装 Docker 到不同操作系统

set -e

echo "=========================================="
echo "   Docker 自动安装"
echo "=========================================="

# 检测操作系统
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        if command -v apt-get &> /dev/null; then
            echo "debian"
        elif command -v yum &> /dev/null || command -v dnf &> /dev/null; then
            echo "redhat"
        elif command -v pacman &> /dev/null; then
            echo "arch"
        else
            echo "linux"
        fi
    else
        echo "unknown"
    fi
}

# macOS 安装
install_macos() {
    echo ""
    echo "安装 Docker 到 macOS..."
    echo ""

    # 检查 Homebrew
    if ! command -v brew &> /dev/null; then
        echo "安装 Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

        # 配置 PATH
        if [[ -d "/opt/homebrew" ]]; then
            eval "$(/opt/homebrew/bin/brew shellenv)"
        elif [[ -d "/usr/local/Homebrew" ]]; then
            eval "$(/usr/local/Homebrew/bin/brew shellenv)"
        fi

        echo "✅ Homebrew 已安装"
    else
        echo "✅ Homebrew 已存在"
    fi

    echo ""
    echo "选择 Docker 安装方式:"
    echo "  1. Docker Desktop (推荐，图形界面，约 500MB)"
    echo "  2. Colima (轻量级，命令行，约 50MB)"
    echo ""
    read -p "选择 [1]: " CHOICE
    CHOICE=${CHOICE:-1}

    case $CHOICE in
        1)
            echo ""
            echo "安装 Docker Desktop..."
            brew install --cask docker

            echo ""
            echo "启动 Docker Desktop..."
            open /Applications/Docker.app

            echo "等待 Docker Desktop 启动 (约 30 秒)..."
            sleep 30

            # 等待就绪
            until docker info &> /dev/null; do
                echo "等待 Docker 就绪..."
                sleep 5
            done

            echo "✅ Docker Desktop 已安装并启动"
            ;;
        2)
            echo ""
            echo "安装 Colima 和 Docker CLI..."
            brew install colima docker docker-compose

            echo ""
            echo "启动 Colima..."
            colima start

            echo "等待 Colima 启动..."
            sleep 15

            until docker info &> /dev/null; do
                echo "等待 Docker 就绪..."
                sleep 5
            done

            echo "✅ Colima 已安装并启动"
            ;;
        *)
            echo "无效选择"
            exit 1
            ;;
    esac

    echo ""
    echo "验证安装:"
    docker --version
    docker compose version || docker-compose --version
}

# Debian/Ubuntu 安装
install_debian() {
    echo ""
    echo "安装 Docker 到 Debian/Ubuntu..."
    echo ""

    # 更新包管理器
    echo "更新包列表..."
    sudo apt-get update

    # 安装依赖
    echo "安装依赖包..."
    sudo apt-get install -y \
        apt-transport-https \
        ca-certificates \
        curl \
        gnupg \
        lsb-release

    # 添加 Docker GPG key
    echo "添加 Docker GPG key..."
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
        sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg 2>/dev/null

    # 添加 Docker 仓库
    echo "添加 Docker 仓库..."
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] \
      https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | \
      sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    # 安装 Docker
    echo "安装 Docker..."
    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

    # 启动 Docker
    echo "启动 Docker 服务..."
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加用户到 docker 组
    echo "添加用户到 docker 组..."
    sudo usermod -aG docker $USER

    echo ""
    echo "✅ Docker 已安装"
    echo ""
    echo "验证安装:"
    sudo docker --version
    sudo docker compose version

    echo ""
    echo "⚠️  重要提示:"
    echo "  请运行 'newgrp docker' 或重新登录，以使用 docker 命令"
    echo "  或使用 'sudo docker' 命令"
}

# RedHat/CentOS/Fedora 安装
install_redhat() {
    echo ""
    echo "安装 Docker 到 RedHat/CentOS/Fedora..."
    echo ""

    # 安装依赖
    if command -v dnf &> /dev/null; then
        echo "使用 dnf 安装..."
        sudo dnf install -y dnf-plugins-core
        sudo dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
        sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    else
        echo "使用 yum 安装..."
        sudo yum install -y yum-utils
        sudo yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
        sudo yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
    fi

    # 启动 Docker
    echo "启动 Docker 服务..."
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加用户到 docker 组
    echo "添加用户到 docker 组..."
    sudo usermod -aG docker $USER

    echo ""
    echo "✅ Docker 已安装"
    echo ""
    echo "验证安装:"
    sudo docker --version
    sudo docker compose version

    echo ""
    echo "⚠️  重要提示:"
    echo "  请运行 'newgrp docker' 或重新登录，以使用 docker 命令"
}

# Arch Linux 安装
install_arch() {
    echo ""
    echo "安装 Docker 到 Arch Linux..."
    echo ""

    echo "安装 Docker..."
    sudo pacman -Syu --noconfirm docker docker-compose

    # 启动 Docker
    echo "启动 Docker 服务..."
    sudo systemctl enable docker
    sudo systemctl start docker

    # 添加用户到 docker 组
    echo "添加用户到 docker 组..."
    sudo usermod -aG docker $USER

    echo ""
    echo "✅ Docker 已安装"
    echo ""
    echo "验证安装:"
    sudo docker --version
    docker-compose --version

    echo ""
    echo "⚠️  重要提示:"
    echo "  请运行 'newgrp docker' 或重新登录，以使用 docker 命令"
}

# 主安装函数
install_docker() {
    OS=$(detect_os)

    case $OS in
        macos)
            install_macos
            ;;
        debian)
            install_debian
            ;;
        redhat)
            install_redhat
            ;;
        arch)
            install_arch
            ;;
        *)
            echo ""
            echo "❌ 无法识别操作系统: $OSTYPE"
            echo ""
            echo "请手动安装 Docker:"
            echo ""
            echo "官方文档:"
            echo "  macOS: https://docs.docker.com/desktop/install/mac-install/"
            echo "  Ubuntu: https://docs.docker.com/engine/install/ubuntu/"
            echo "  CentOS: https://docs.docker.com/engine/install/centos/"
            echo "  Fedora: https://docs.docker.com/engine/install/fedora/"
            echo "  Debian: https://docs.docker.com/engine/install/debian/"
            echo "  Arch: https://docs.docker.com/engine/install/archlinux/"
            echo ""
            echo "或使用 Docker 官方安装脚本:"
            echo "  curl -fsSL https://get.docker.com | sh"
            exit 1
            ;;
    esac
}

# 检查现有安装
check_existing() {
    if command -v docker &> /dev/null; then
        echo ""
        echo "Docker 已安装:"
        docker --version

        if docker info &> /dev/null; then
            echo "Docker 服务运行中"
        else
            echo "Docker 服务未运行"

            read -p "是否启动 Docker? [Y/n]: " START_DOCKER
            if [[ "$START_DOCKER" =~ ^[Yy]?$ ]]; then
                if [[ "$OSTYPE" == "darwin"* ]]; then
                    open /Applications/Docker.app 2>/dev/null || colima start 2>/dev/null
                else
                    sudo systemctl start docker || sudo service docker start
                fi

                echo "等待 Docker 启动..."
                sleep 15

                until docker info &> /dev/null; do
                    sleep 5
                done

                echo "✅ Docker 已启动"
            fi
        fi

        echo ""
        read -p "是否重新安装? [y/N]: " REINSTALL
        if [[ ! "$REINSTALL" =~ ^[Yy]$ ]]; then
            echo "保留现有安装"
            exit 0
        fi
    fi
}

# 主流程
main() {
    check_existing
    install_docker

    echo ""
    echo "=========================================="
    echo "   安装完成"
    echo "=========================================="
    echo ""
    echo "下一步:"
    echo "  1. 启动 Open Station: make start"
    echo "  2. 或使用脚本: ./scripts/start-docker.sh"
    echo ""
}

main