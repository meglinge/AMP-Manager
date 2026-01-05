# AMP Manager

Amp 反代管理系统

## 技术栈

- **后端**: Go + Gin + SQLite
- **前端**: React + Vite + Tailwind CSS

## Docker 部署 (推荐)

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

访问 http://localhost:8080 即可使用

### 环境变量

在 `.env` 文件中配置：

```env
JWT_SECRET=your-secret-key-change-in-production
ADMIN_PASSWORD=admin123
PROXY_BASE_URL=http://your-domain:8080
```

## 本地开发

### 后端

```bash
go run ./cmd/server
```

服务将在 http://localhost:8080 启动

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
xcopy /E /I /Y "web\dist" "internal\web\dist"  # Windows
# cp -r web/dist internal/web/dist  # Linux/macOS

# 构建后端
go build -o ampmanager ./cmd/server
```

## API 接口

### 用户注册

```
POST /api/auth/register
Content-Type: application/json

{
  "username": "用户名",
  "email": "邮箱",
  "password": "密码"
}
```
