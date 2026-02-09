package handler

import (
	"errors"
	"net/http"

	"ampmanager/internal/model"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	groupService *service.GroupService
}

func NewGroupHandler() *GroupHandler {
	return &GroupHandler{
		groupService: service.NewGroupService(),
	}
}

func (h *GroupHandler) List(c *gin.Context) {
	groups, err := h.groupService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分组列表失败"})
		return
	}
	if groups == nil {
		groups = []*model.GroupResponse{}
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (h *GroupHandler) Get(c *gin.Context) {
	id := c.Param("id")
	group, err := h.groupService.GetByID(id)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取分组失败"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) Create(c *gin.Context) {
	var req model.GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "details": err.Error()})
		return
	}

	group, err := h.groupService.Create(&req)
	if err != nil {
		if errors.Is(err, service.ErrGroupNameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建分组失败"})
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req model.GroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误", "details": err.Error()})
		return
	}

	group, err := h.groupService.Update(id, &req)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, service.ErrGroupNameExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新分组失败"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.groupService.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrGroupNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除分组失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "分组已删除"})
}
