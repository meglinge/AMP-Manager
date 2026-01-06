package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"ampmanager/internal/amp"
	"ampmanager/internal/model"
	"ampmanager/internal/repository"

	"github.com/gin-gonic/gin"
)

const retryConfigKey = "retry_config"

type SystemHandler struct {
	configRepo *repository.SystemConfigRepository
}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{
		configRepo: repository.NewSystemConfigRepository(),
	}
}

// GetRetryConfig 获取重试配置
func (h *SystemHandler) GetRetryConfig(c *gin.Context) {
	value, err := h.configRepo.Get(retryConfigKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置失败"})
		return
	}

	// 如果没有配置，返回默认值
	if value == "" {
		defaultCfg := amp.DefaultRetryConfig()
		c.JSON(http.StatusOK, model.RetryConfigResponse{
			Enabled:           defaultCfg.Enabled,
			MaxAttempts:       defaultCfg.MaxAttempts,
			GateTimeoutMs:     defaultCfg.GateTimeout.Milliseconds(),
			MaxBodyBytes:      defaultCfg.MaxBodyBytes,
			BackoffBaseMs:     defaultCfg.BackoffBase.Milliseconds(),
			BackoffMaxMs:      defaultCfg.BackoffMax.Milliseconds(),
			RetryOn429:        defaultCfg.RetryOn429,
			RetryOn5xx:        defaultCfg.RetryOn5xx,
			RespectRetryAfter: defaultCfg.RespectRetryAfter,
			RetryOnEmptyBody:  defaultCfg.RetryOnEmptyBody,
		})
		return
	}

	var resp model.RetryConfigResponse
	if err := json.Unmarshal([]byte(value), &resp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析配置失败"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateRetryConfig 更新重试配置
func (h *SystemHandler) UpdateRetryConfig(c *gin.Context) {
	var req model.RetryConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 验证参数
	if req.MaxAttempts < 1 || req.MaxAttempts > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maxAttempts 必须在 1-10 之间"})
		return
	}
	if req.GateTimeoutMs < 1000 || req.GateTimeoutMs > 60000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gateTimeoutMs 必须在 1000-60000 之间"})
		return
	}

	// 保存到数据库
	resp := model.RetryConfigResponse{
		Enabled:           req.Enabled,
		MaxAttempts:       req.MaxAttempts,
		GateTimeoutMs:     req.GateTimeoutMs,
		MaxBodyBytes:      req.MaxBodyBytes,
		BackoffBaseMs:     req.BackoffBaseMs,
		BackoffMaxMs:      req.BackoffMaxMs,
		RetryOn429:        req.RetryOn429,
		RetryOn5xx:        req.RetryOn5xx,
		RespectRetryAfter: req.RespectRetryAfter,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "序列化配置失败"})
		return
	}

	if err := h.configRepo.Set(retryConfigKey, string(data)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败"})
		return
	}

	// 更新运行时配置
	rt := amp.GetRetryTransport()
	if rt != nil {
		rt.UpdateConfig(&amp.RetryConfig{
			Enabled:           req.Enabled,
			MaxAttempts:       req.MaxAttempts,
			GateTimeout:       time.Duration(req.GateTimeoutMs) * time.Millisecond,
			MaxBodyBytes:      req.MaxBodyBytes,
			BackoffBase:       time.Duration(req.BackoffBaseMs) * time.Millisecond,
			BackoffMax:        time.Duration(req.BackoffMaxMs) * time.Millisecond,
			RetryOn429:        req.RetryOn429,
			RetryOn5xx:        req.RetryOn5xx,
			RespectRetryAfter: req.RespectRetryAfter,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已更新", "config": resp})
}

func (h *SystemHandler) UploadDatabase(c *gin.Context) {
	file, err := c.FormFile("database")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择数据库文件"})
		return
	}

	if filepath.Ext(file.Filename) != ".db" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只支持 .db 文件"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取上传文件"})
		return
	}
	defer src.Close()

	dbPath := "./data/data.db"
	backupPath := "./data/data.db.backup." + time.Now().Format("20060102150405")

	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Rename(dbPath, backupPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "备份现有数据库失败"})
			return
		}
	}

	dst, err := os.Create(dbPath)
	if err != nil {
		os.Rename(backupPath, dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据库文件失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dbPath)
		os.Rename(backupPath, dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存数据库文件失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "数据库上传成功，请重启服务使更改生效",
		"backupFile": filepath.Base(backupPath),
	})
}

func (h *SystemHandler) DownloadDatabase(c *gin.Context) {
	dbPath := "./data/data.db"

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据库文件不存在"})
		return
	}

	filename := "ampmanager_" + time.Now().Format("20060102150405") + ".db"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")
	c.File(dbPath)
}

func (h *SystemHandler) ListBackups(c *gin.Context) {
	files, err := filepath.Glob("./data/data.db.backup.*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取备份列表失败"})
		return
	}

	backups := make([]gin.H, 0)
	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		backups = append(backups, gin.H{
			"filename": filepath.Base(f),
			"size":     info.Size(),
			"modTime":  info.ModTime(),
		})
	}

	c.JSON(http.StatusOK, backups)
}

func (h *SystemHandler) RestoreBackup(c *gin.Context) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	backupPath := filepath.Join("./data", req.Filename)
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	dbPath := "./data/data.db"
	currentBackup := "./data/data.db.backup." + time.Now().Format("20060102150405")

	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Rename(dbPath, currentBackup); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "备份当前数据库失败"})
			return
		}
	}

	src, err := os.Open(backupPath)
	if err != nil {
		os.Rename(currentBackup, dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取备份文件失败"})
		return
	}
	defer src.Close()

	dst, err := os.Create(dbPath)
	if err != nil {
		os.Rename(currentBackup, dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建数据库文件失败"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(dbPath)
		os.Rename(currentBackup, dbPath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复数据库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "数据库恢复成功，请重启服务使更改生效"})
}

func (h *SystemHandler) DeleteBackup(c *gin.Context) {
	filename := c.Param("filename")
	backupPath := filepath.Join("./data", filename)

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份文件不存在"})
		return
	}

	if err := os.Remove(backupPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除备份失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "备份已删除"})
}
