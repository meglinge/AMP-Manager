#!/bin/bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
WEB_DIR="$ROOT_DIR/web"
COMPOSE_FILE="$ROOT_DIR/docker-compose.dev.yml"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo -e "${RED}[错误] 未找到 $1，请先安装${NC}"
    exit 1
  fi
}

ensure_pnpm() {
  if ! command -v pnpm >/dev/null 2>&1; then
    echo -e "${YELLOW}[信息] 未找到 pnpm，正在安装...${NC}"
    npm install -g pnpm
  fi
}

ensure_air() {
  if command -v air >/dev/null 2>&1; then
    command -v air
    return
  fi

  echo -e "${YELLOW}[信息] 未找到 air，正在安装...${NC}"
  go install github.com/air-verse/air@latest
  local gobin
  gobin="$(go env GOPATH)/bin"
  if [ -x "$gobin/air" ]; then
    echo "$gobin/air"
    return
  fi

  echo -e "${RED}[错误] air 安装完成但未找到可执行文件，请确认 GOPATH/bin 已加入 PATH${NC}"
  exit 1
}

wait_for_postgres() {
  for _ in $(seq 1 60); do
    if (echo >/dev/tcp/127.0.0.1/5432) >/dev/null 2>&1; then
      return
    fi
    sleep 1
  done

  echo -e "${RED}[错误] 等待 PostgreSQL 就绪超时${NC}"
  exit 1
}

cleanup() {
  if [ -n "${FRONTEND_PID:-}" ]; then
    kill "$FRONTEND_PID" >/dev/null 2>&1 || true
  fi
  if [ -n "${BACKEND_PID:-}" ]; then
    kill "$BACKEND_PID" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT INT TERM

echo ""
echo "=============================="
echo "   AMP Manager 开发启动脚本"
echo "=============================="
echo ""

require_command node
ensure_pnpm
require_command go
require_command docker
AIR_BIN="$(ensure_air)"

echo -e "${GREEN}[1/4] 启动 PostgreSQL 容器...${NC}"
docker compose -f "$COMPOSE_FILE" up -d postgres
wait_for_postgres

echo ""
echo -e "${GREEN}[2/4] 安装前端依赖...${NC}"
cd "$WEB_DIR"
pnpm install
cd "$ROOT_DIR"

export ALLOW_INSECURE_DEFAULTS=true
export AMP_DEV_RUNTIME_DB_CONFIG=true
export CORS_ALLOWED_ORIGINS=http://localhost:5274
export SERVER_PORT=16823

echo ""
echo -e "${GREEN}[3/4] 启动前端热更 (Vite)...${NC}"
cd "$WEB_DIR"
pnpm run dev -- --host 0.0.0.0 &
FRONTEND_PID=$!
cd "$ROOT_DIR"

echo ""
echo -e "${GREEN}[4/4] 启动后端热更 (Air)...${NC}"
"$AIR_BIN" &
BACKEND_PID=$!

echo ""
echo "=============================="
echo "   前端: http://localhost:5274"
echo "   后端: http://localhost:16823"
echo "   PostgreSQL 容器: localhost:5432"
echo "   默认数据库模式: 读取 ./data/config.json；首次缺省为 PostgreSQL"
echo "=============================="
echo ""

wait "$FRONTEND_PID" "$BACKEND_PID"
