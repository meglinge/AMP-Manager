package handler

import (
	"net/http"
	"strconv"
	"time"

	"ampmanager/internal/middleware"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
)

type RequestLogHandler struct {
	logService *service.RequestLogService
}

func NewRequestLogHandler() *RequestLogHandler {
	return &RequestLogHandler{
		logService: service.NewRequestLogService(),
	}
}

// ListRequestLogs 获取请求日志列表
func (h *RequestLogHandler) ListRequestLogs(c *gin.Context) {
	userID := middleware.GetUserID(c)

	params := service.ListRequestLogsParams{
		UserID:   userID,
		Page:     1,
		PageSize: 20,
	}

	// 解析查询参数
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		params.Page = page
	}
	if pageSize, err := strconv.Atoi(c.Query("pageSize")); err == nil && pageSize > 0 {
		params.PageSize = pageSize
	}
	if apiKeyID := c.Query("apiKeyId"); apiKeyID != "" {
		params.APIKeyID = apiKeyID
	}
	if model := c.Query("model"); model != "" {
		params.Model = model
	}
	if statusCode, err := strconv.Atoi(c.Query("status")); err == nil {
		params.StatusCode = &statusCode
	}
	if isStreaming := c.Query("isStreaming"); isStreaming != "" {
		val := isStreaming == "true" || isStreaming == "1"
		params.IsStreaming = &val
	}
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			params.From = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			params.To = &t
		}
	}

	result, err := h.logService.List(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetRequestLog 获取单条日志详情
func (h *RequestLogHandler) GetRequestLog(c *gin.Context) {
	userID := middleware.GetUserID(c)
	logID := c.Param("id")

	log, err := h.logService.GetByID(logID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	if log == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "日志不存在"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// GetUsageSummary 获取用量统计
func (h *RequestLogHandler) GetUsageSummary(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var from, to *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &t
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &t
		}
	}

	groupBy := c.DefaultQuery("groupBy", "day")

	result, err := h.logService.GetUsageSummary(userID, from, to, groupBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}
