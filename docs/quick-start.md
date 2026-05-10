# Open Station 快速入门指南

## 一键安装（企业推荐）

### 自动安装 Docker（首次使用）

如果系统未安装 Docker，会自动提示安装：

```bash
# 方式一：一键启动（自动检测并安装 Docker）
make start

# 方式二：单独安装 Docker
make install-docker

# 方式三：手动运行脚本
./scripts/install-docker.sh
```

支持系统：
- macOS (Docker Desktop / Colima)
- Ubuntu/Debian
- CentOS/Fedora/RHEL
- Arch Linux

### Docker 方式（最简单）

```bash
# 1. 克隆项目
git clone https://github.com/zhaojiewen/open-station.git
cd open-station

# 2. 一键启动（自动检测 Docker，未安装会提示安装）
make start

# 或使用脚本
./scripts/start-docker.sh
```

启动后会自动：
- 检测并安装 Docker（如果未安装）
- 检查端口占用
- 启动 PostgreSQL + Redis + API Gateway
- 创建管理员账号和 API Key
- 提示配置 MCP

### 本地开发方式

```bash
# 交互式安装
make install

# 或
./scripts/quick-install.sh
```

## MCP 配置

### Claude Code CLI

```bash
# 自动配置（使用启动时生成的 API Key）
make mcp-config

# 或手动指定
./scripts/setup-mcp.sh --claude --api-key sk-your-manager-key
```

### 其他 IDE

```bash
# Cursor IDE
make mcp-config-cursor

# VS Code
make mcp-config-vscode
```

## MCP 工具使用

在 Claude Code CLI 中：

```bash
claude
```

### 用户工具（6个）

```
> "What's my balance?"                    # 查询余额
> "Show my usage for this month"          # 用量统计
> "Get my billing info"                   # 计费信息
> "Show my API keys"                      # 查看 Key
```

### 管理工具（9个）

```
> "Create API key for john@example.com"   # 创建用户+Key
> "List all API keys"                     # 查看所有 Key
> "List all users"                        # 用户列表
> "Add $100 to tenant xxx"                # 充值
> "Revoke API key sk-xxx"                 # 撤销 Key
```

## 非技术用户管理

使用交互式脚本：

```bash
./scripts/user-admin.sh
```

菜单操作：
- 创建用户和 API Key
- 查询余额
- 列出用户/Key
- 撤销 Key

## API 代理使用

### Claude Code CLI 代理

```bash
# 配置环境变量
export ANTHROPIC_BASE_URL="http://localhost:8080/v1"
export ANTHROPIC_API_KEY="sk-user-key"

claude --model claude-opus-4-7
claude --model openai-gpt-4o
claude --model deepseek-v4-flash
```

### 直接 API 调用

```bash
# 查看模型列表
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-user-key"

# Chat 请求
curl -X POST http://localhost:8080/v1/messages \
  -H "Authorization: Bearer sk-user-key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

## 常用命令

```bash
make help          # 查看所有命令
make start         # 启动服务
make stop          # 停止服务
make status        # 查看状态
make logs          # 查看日志
make test          # 功能测试
make mcp-config    # 配置 MCP
```

## 环境变量

创建 `.env` 文件：

```bash
# Provider API Keys
OPENAI_API_KEY=sk-xxx
ANTHROPIC_API_KEY=sk-xxx
GEMINI_API_KEY=xxx
DEEPSEEK_API_KEY=sk-xxx
GLM_API_KEY=xxx

# 管理员配置
ADMIN_USER=admin
ADMIN_EMAIL=admin@company.com
ADMIN_PASS=your-password
```

## 配置文件

编辑 `configs/config.yaml`：

```yaml
server:
  port: 8080

admin:
  default_admin_user: admin
  default_admin_pass: changeme123
  initial_api_key_name: "Manager Key"
```

## 测试服务

```bash
# 快速测试
make test

# 或
API_KEY=sk-xxx ./scripts/test.sh
```

测试内容：
- 健康检查
- 模型列表
- MCP 初始化
- 余额查询

## 故障排查

### 服务未启动

```bash
make status        # 检查状态
make logs          # 查看日志
docker ps          # 检查容器
```

### MCP 连接失败

```bash
# 检查 API Key
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-your-key"

# 检查 MCP
curl -X POST http://localhost:8080/mcp \
  -H "Authorization: Bearer sk-your-key" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'
```

### 重置服务

```bash
make clean         # 清理所有数据
make start         # 重新启动
```

## 更多文档

- [完整 README](../README.md)
- [MCP 集成指南](./mcp-integration.md)
- [Claude Code 集成](./claude-code-integration.md)