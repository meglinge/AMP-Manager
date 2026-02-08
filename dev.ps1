# AMP Manager 开发启动脚本
# 使用方法: .\dev.ps1

$Host.UI.RawUI.WindowTitle = "AMP Manager Dev"

Write-Host ""
Write-Host "==============================" -ForegroundColor Cyan
Write-Host "   AMP Manager 开发启动脚本" -ForegroundColor Cyan
Write-Host "==============================" -ForegroundColor Cyan
Write-Host ""

# 设置开发环境变量（跳过安全检查）
$env:ALLOW_INSECURE_DEFAULTS = "true"

Write-Host "[1/4] 安装前端依赖..." -ForegroundColor Green
Push-Location web
& pnpm install 2>&1 | Write-Host

Write-Host ""
Write-Host "[2/4] 编译前端..." -ForegroundColor Green
& pnpm run build 2>&1 | Write-Host
Pop-Location

Write-Host ""
Write-Host "[3/4] 复制前端文件到嵌入目录..." -ForegroundColor Green
New-Item -ItemType Directory -Path "internal\web\dist" -Force | Out-Null
Copy-Item -Path "web\dist\*" -Destination "internal\web\dist\" -Recurse -Force

Write-Host ""
Write-Host "[4/4] 启动后端..." -ForegroundColor Green
Write-Host ""
Write-Host "==============================" -ForegroundColor Cyan
Write-Host "   访问: http://localhost:8080" -ForegroundColor Yellow
Write-Host "   按 Ctrl+C 停止服务" -ForegroundColor Yellow
Write-Host "==============================" -ForegroundColor Cyan
Write-Host ""

& go run ./cmd/server
