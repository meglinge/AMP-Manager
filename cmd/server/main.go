package main

import (
	"log"
	"os"

	"ampmanager/internal/amp"
	"ampmanager/internal/config"
	"ampmanager/internal/database"
	"ampmanager/internal/router"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	gin.SetMode(gin.ReleaseMode)

	cfg := config.Load()

	if err := database.Init("./data/data.db"); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer database.Close()

	// 初始化日志写入器
	amp.InitLogWriter(database.GetDB())
	defer amp.StopLogWriter()

	// 初始化 pending 请求清理器
	amp.InitPendingCleaner(database.GetDB())
	defer amp.StopPendingCleaner()

	userService := service.NewUserService()
	if err := userService.EnsureAdmin(); err != nil {
		log.Printf("警告: 管理员账户创建失败: %v", err)
	}

	r := router.Setup()

	// 加载重试配置
	sysConfigService := service.NewSystemConfigService()
	if configJSON, err := sysConfigService.GetRetryConfigJSON(); err == nil && configJSON != "" {
		amp.InitRetryTransportConfig(configJSON)
	}

	port := cfg.ServerPort
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	log.Printf("服务器启动在 http://0.0.0.0:%s", port)
	if err := r.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
