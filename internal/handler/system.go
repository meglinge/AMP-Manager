package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
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
