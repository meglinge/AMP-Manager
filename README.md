# AMP Manager

Amp 反向代理管理系统 - 一个用于管理 Amp CLI 代理服务的 Web 管理平台。

## 功能特性

- 🔐 **用户认证** - JWT 身份验证，支持管理员和普通用户
- 🔑 **API Key 管理** - 创建和管理 Amp CLI 的 API 密钥
- 📡 **渠道管理** - 支持多渠道配置 (OpenAI、Claude、Gemini 等)
- 🔄 **模型映射** - 自定义模型名称映射
- 📊 **模型元数据** - 管理模型上下文长度等信息
- 🌐 **线程跳转** - 访问 `/threads/T-xxx` 自动跳转到官方 Amp 线程

## 技术栈

| 组件 | 技术 |
|------|------|
| 后端 | Go + Gin + SQLite |
| 前端 | React + Vite + Tailwind CSS |
| 部署 | Docker + Docker Compose |

## 快速开始

### Docker 部署 (推荐)

```bash
# 克隆仓库
git clone <仓库地址>
cd AMPManager

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

访问 http://localhost:16823 即可使用

### 使用管理脚本

```bash
chmod +x manage.sh
./manage.sh
```

脚本提供以下功能：
1. 启动服务
2. 停止服务
3. 更新并重启 (拉取代码 + 重新构建)
4. 查看日志
5. 查看状态

## 环境变量配置

在项目根目录创建 `.env` 文件：

```env
# JWT 密钥 (生产环境请修改)
JWT_SECRET=your-secret-key-change-in-production

# 管理员初始密码
ADMIN_PASSWORD=admin123

# 对外访问地址 (用于生成配置示例)
PROXY_BASE_URL=http://your-domain:16823
```

## 客户端配置

创建 API Key 后，在终端配置环境变量：

### Linux / macOS

```bash
export AMP_URL="http://your-server:16823"
export AMP_API_KEY="your-api-key"
```

### Windows PowerShell (永久)

```powershell
[Environment]::SetEnvironmentVariable("AMP_URL", "http://your-server:16823", "User")
[Environment]::SetEnvironmentVariable("AMP_API_KEY", "your-api-key", "User")
```

## 本地开发

### 后端

```bash
go run ./cmd/server
```

服务将在 http://localhost:16823 启动

### 前端

```bash
cd web
pnpm install
pnpm dev
```

前端开发服务器将在 http://localhost:5173 启动

### 构建

```bash
# 构建前端
cd web && pnpm run build && cd ..

# 复制到嵌入目录
# Windows:
xcopy /E /I /Y "web\dist" "internal\web\dist"
# Linux/macOS:
cp -r web/dist internal/web/dist

# 构建后端
go build -o ampmanager ./cmd/server
```

## 目录结构

```
AMPManager/
├── cmd/server/          # 程序入口
├── internal/
│   ├── amp/             # Amp 代理核心逻辑
│   ├── config/          # 配置管理
│   ├── database/        # 数据库连接
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   ├── router/          # 路由配置
│   ├── service/         # 业务逻辑层
│   └── web/             # 嵌入的前端文件
├── web/                 # 前端源码
├── data/                # 数据库文件 (自动创建)
├── docker-compose.yml
├── Dockerfile
└── manage.sh            # 管理脚本
```

## API 接口

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/register` | 用户注册 |
| POST | `/api/auth/login` | 用户登录 |

### Amp 设置 (需要认证)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/me/amp/settings` | 获取设置 |
| PUT | `/api/me/amp/settings` | 更新设置 |
| GET | `/api/me/amp/api-keys` | 获取 API Key 列表 |
| POST | `/api/me/amp/api-keys` | 创建 API Key |

### 管理员接口 (需要管理员权限)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/admin/channels` | 获取渠道列表 |
| POST | `/api/admin/channels` | 创建渠道 |
| PUT | `/api/admin/channels/:id` | 更新渠道 |
| DELETE | `/api/admin/channels/:id` | 删除渠道 |

## 默认账户

首次启动时会自动创建管理员账户：

- 用户名: `admin`
- 密码: `admin123` (可通过 `ADMIN_PASSWORD` 环境变量修改)

## 许可证

MIT License
