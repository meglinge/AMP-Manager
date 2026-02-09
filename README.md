# AMP Manager

Amp CLI 反向代理管理系统 —— 一个功能完整的 Web 管理平台，用于自托管 [Amp](https://ampcode.com) 代理服务，支持多渠道路由、用户分组、余额计费和全面的使用量监控。

## 功能特性

### 代理核心
- 🔄 **智能渠道路由** - 多渠道负载均衡，支持权重、优先级和分组路由策略
- 🧠 **多 Provider 支持** - OpenAI、Anthropic Claude、Google Gemini，兼容 `/v1`、`/v1beta` 标准接口
- 🔀 **模型映射** - 灵活的模型名称映射和转换规则
- 📡 **流式/非流式代理** - 完整支持 SSE 流式响应和普通请求
- 🔁 **自动重试** - 可配置的失败重试机制和错误分类
- 🌐 **本地网页搜索** - 拦截 `webSearch2` 请求，通过 DuckDuckGo 本地执行
- 📄 **网页内容提取** - 本地处理 `extractWebPageContent`，直接获取网页内容
- 🚫 **广告拦截** - 自动拦截 Amp 内置广告请求
- 🧬 **协议转换** - Claude ↔ OpenAI 格式自动转换（translator 模块）

### 管理平台
- 🔐 **用户系统** - JWT 认证，支持管理员/普通用户角色
- 👥 **分组管理** - 用户和渠道分组，实现精细化权限控制
- 💰 **余额计费** - 用户余额管理、费率倍率、自动扣费
- 🔑 **API Key 管理** - 创建和管理 Amp CLI 的 API 密钥
- 📊 **仪表盘** - 实时费用统计、热门模型、每日趋势、缓存命中率
- 📈 **使用量监控** - 请求日志、Token 用量、成本分析
- 💲 **价格管理** - 各模型定价查看和同步
- ⚙️ **系统设置** - 重试策略、超时配置、缓存 TTL、数据库备份恢复
- 🔒 **数据加密** - AES-256 加密存储上游 API Key

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go 1.24 + Gin + SQLite (modernc) |
| 前端 | React + Vite + TypeScript + Tailwind CSS + shadcn/ui |
| 部署 | Docker (多架构) + Docker Compose |
| CI/CD | GitHub Actions |

## 快速开始

### Docker Compose 部署 (推荐)

```bash
# 克隆仓库
git clone https://github.com/meglinge/AMP-Manager.git
cd AMP-Manager

# 复制并修改环境变量
cp .env.example .env
# 编辑 .env，修改 JWT_SECRET 和 ADMIN_PASSWORD

# 启动服务
docker-compose up -d
```

访问 `http://localhost:16823` 即可使用。

### 一键管理脚本

```bash
chmod +x manage.sh
./manage.sh
```

提供以下功能：启动/停止服务、更新并重启（拉取代码 + 重新构建）、查看日志和状态。

## 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `ADMIN_USERNAME` | 管理员用户名 | `admin` |
| `ADMIN_PASSWORD` | 管理员密码 (**必须修改**) | `admin123` |
| `SERVER_PORT` | 服务端口 | `16823` |
| `JWT_SECRET` | JWT 签名密钥 (**必须修改**, ≥32字符) | - |
| `JWT_ISSUER` | JWT 签发者 | `ampmanager` |
| `JWT_AUDIENCE` | JWT 受众 | `ampmanager-users` |
| `DATA_ENCRYPTION_KEY` | 数据加密密钥 (正好32字符, AES-256) | 空 (明文存储) |
| `CORS_ALLOWED_ORIGINS` | CORS 允许来源 (逗号分隔) | `*` |
| `RATE_LIMIT_AUTH_RPS` | 认证端点每秒请求限制 | `5` |
| `RATE_LIMIT_PROXY_RPS` | 代理端点每秒请求限制 | `100` |

## 客户端配置

创建 API Key 后，配置 Amp CLI 环境变量：

```bash
# Linux / macOS
export AMP_URL="http://your-server:16823"
export AMP_API_KEY="your-api-key"
```

```powershell
# Windows PowerShell (永久)
[Environment]::SetEnvironmentVariable("AMP_URL", "http://your-server:16823", "User")
[Environment]::SetEnvironmentVariable("AMP_API_KEY", "your-api-key", "User")
```

## 本地开发

### 后端

```bash
go run ./cmd/server
```

> 需设置 `ALLOW_INSECURE_DEFAULTS=true` 跳过安全检查。服务启动在 `http://localhost:16823`

### 前端

```bash
cd web
pnpm install
pnpm dev
```

> 开发服务器启动在 `http://localhost:5173`

### 构建

```bash
# 构建前端
cd web && pnpm run build && cd ..

# 复制到嵌入目录
# Windows:
xcopy /E /I /Y "web\dist" "internal\web\dist"
# Linux/macOS:
cp -r web/dist internal/web/dist

# 构建后端 (前端文件会嵌入到二进制中)
go build -o ampmanager ./cmd/server
```

也可使用项目提供的构建脚本：`build.ps1` (Windows) 或 `build.sh` (Linux/macOS)。

## 目录结构

```
AMPManager/
├── cmd/server/              # 程序入口
├── internal/
│   ├── amp/                 # 代理核心 (路由、流式处理、重试、搜索、协议转换)
│   ├── billing/             # 计费模块 (价格计算、价格存储)
│   ├── config/              # 配置管理
│   ├── crypto/              # 加密工具 (AES-256)
│   ├── database/            # SQLite 数据库 (建表、迁移)
│   ├── handler/             # HTTP 处理器
│   ├── middleware/          # 认证、限流中间件
│   ├── model/               # 数据模型
│   ├── repository/          # 数据访问层
│   ├── response/            # 响应格式化
│   ├── router/              # 路由注册
│   ├── service/             # 业务逻辑层
│   ├── translator/          # 协议转换 (Claude ↔ OpenAI)
│   ├── util/                # 工具函数
│   └── web/                 # 嵌入的前端静态文件
├── web/                     # 前端源码 (React + TypeScript)
│   └── src/
│       ├── api/             # API 客户端
│       ├── components/      # 通用组件
│       ├── lib/             # 工具库
│       └── pages/           # 页面 (仪表盘、渠道、用户、分组、日志等)
├── CLIProxyAPI/             # 参考实现 (只读)
├── data/                    # 数据库文件 (运行时自动创建)
├── .github/workflows/       # CI/CD
├── docker-compose.yml       # Docker 部署
├── Dockerfile               # 多阶段构建 (支持多架构)
└── manage.sh                # 一键管理脚本
```

## API 概览

### 管理接口 (`/api/manage/auth/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/manage/auth/register` | 用户注册 |
| POST | `/api/manage/auth/login` | 用户登录 |

### 用户接口 (`/api/me/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/me/balance` | 获取余额 |
| GET | `/api/me/dashboard` | 仪表盘数据 |
| GET | `/api/me/amp/settings` | Amp 代理设置 |
| PUT | `/api/me/amp/settings` | 更新代理设置 |
| GET | `/api/me/amp/api-keys` | API Key 列表 |
| POST | `/api/me/amp/api-keys` | 创建 API Key |
| GET | `/api/me/amp/request-logs` | 请求日志 |
| GET | `/api/me/amp/usage/summary` | 用量统计 |

### 管理员接口 (`/api/admin/*`)

| 方法 | 路径 | 说明 |
|------|------|------|
| CRUD | `/api/admin/channels` | 渠道管理 |
| CRUD | `/api/admin/groups` | 分组管理 |
| CRUD | `/api/admin/model-metadata` | 模型元数据 |
| GET | `/api/admin/users` | 用户列表 |
| POST | `/api/admin/users/:id/topup` | 用户充值 |
| PATCH | `/api/admin/users/:id/group` | 设置用户分组 |
| GET | `/api/admin/prices` | 价格列表 |
| POST | `/api/admin/prices/refresh` | 同步价格 |
| * | `/api/admin/system/*` | 系统设置 (数据库、重试、超时、缓存) |

### 代理接口

| 路径 | 说明 |
|------|------|
| `/api/provider/:provider/*` | 多 Provider 代理 (Amp CLI 使用) |
| `/v1/chat/completions` | OpenAI 兼容接口 |
| `/v1/messages` | Anthropic 兼容接口 |
| `/v1/responses` | OpenAI Responses API |
| `/v1beta/models/*` | Gemini 兼容接口 |
| `/threads/:threadID` | 线程跳转到 ampcode.com |

## 默认账户

首次启动自动创建管理员：
- 用户名：`admin`
- 密码：`admin123`（可通过 `ADMIN_PASSWORD` 环境变量修改）

> 生产环境 **必须** 修改默认密码和 JWT_SECRET，否则服务将拒绝启动。

## 致谢

本项目的代理核心实现参考了 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI)，感谢原作者的开源贡献。

## 许可证

[MIT License](LICENSE) © 2026 乔浩
