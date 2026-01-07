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
	if statusStr := c.Query("status"); statusStr != "" {
		statusCode, err := strconv.Atoi(statusStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 参数无效，应为整数"})
			return
		}
		params.StatusCode = &statusCode
	}
	if isStreaming := c.Query("isStreaming"); isStreaming != "" {
		val := isStreaming == "true" || isStreaming == "1"
		params.IsStreaming = &val
	}
	if from := c.Query("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from 时间格式错误，应为 RFC3339 格式"})
			return
		}
		params.From = &t
	}
	if to := c.Query("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to 时间格式错误，应为 RFC3339 格式"})
			return
		}
		params.To = &t
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
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from 时间格式错误，应为 RFC3339 格式"})
			return
		}
		from = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to 时间格式错误，应为 RFC3339 格式"})
			return
		}
		to = &t
	}

	groupBy := c.DefaultQuery("groupBy", "day")
	allowedGroupBy := map[string]bool{"day": true, "model": true, "apiKey": true}
	if !allowedGroupBy[groupBy] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupBy 参数无效，允许值: day, model, apiKey"})
		return
	}

	result, err := h.logService.GetUsageSummary(userID, from, to, groupBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// AdminListRequestLogs 管理员获取所有请求日志列表
func (h *RequestLogHandler) AdminListRequestLogs(c *gin.Context) {
	params := service.ListRequestLogsParams{
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
	// 可选：按用户过滤
	if userID := c.Query("userId"); userID != "" {
		params.UserID = userID
	}
	if apiKeyID := c.Query("apiKeyId"); apiKeyID != "" {
		params.APIKeyID = apiKeyID
	}
	if model := c.Query("model"); model != "" {
		params.Model = model
	}
	if statusStr := c.Query("status"); statusStr != "" {
		statusCode, err := strconv.Atoi(statusStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "status 参数无效，应为整数"})
			return
		}
		params.StatusCode = &statusCode
	}
	if isStreaming := c.Query("isStreaming"); isStreaming != "" {
		val := isStreaming == "true" || isStreaming == "1"
		params.IsStreaming = &val
	}
	if from := c.Query("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from 时间格式错误，应为 RFC3339 格式"})
			return
		}
		params.From = &t
	}
	if to := c.Query("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to 时间格式错误，应为 RFC3339 格式"})
			return
		}
		params.To = &t
	}

	result, err := h.logService.ListAdmin(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取日志失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// AdminGetDistinctModels 管理员获取使用过的模型列表
func (h *RequestLogHandler) AdminGetDistinctModels(c *gin.Context) {
	models, err := h.logService.GetDistinctModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取模型列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

// AdminGetUsageSummary 管理员获取全局用量统计
func (h *RequestLogHandler) AdminGetUsageSummary(c *gin.Context) {
	var from, to *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "from 时间格式错误，应为 RFC3339 格式"})
			return
		}
		from = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "to 时间格式错误，应为 RFC3339 格式"})
			return
		}
		to = &t
	}

	groupBy := c.DefaultQuery("groupBy", "day")
	allowedGroupBy := map[string]bool{"day": true, "model": true, "apiKey": true, "user": true}
	if !allowedGroupBy[groupBy] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "groupBy 参数无效，允许值: day, model, apiKey, user"})
		return
	}

	// 可选：按用户过滤
	var userID *string
	if uid := c.Query("userId"); uid != "" {
		userID = &uid
	}

	result, err := h.logService.GetUsageSummaryAdmin(userID, from, to, groupBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}
