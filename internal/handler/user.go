package handler

import (
	"errors"
	"net/http"

	"ampmanager/internal/model"
	"ampmanager/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: service.NewUserService(),
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.Register(&req)
	if err != nil {
		status := http.StatusInternalServerError
		msg := "注册失败"

		if errors.Is(err, service.ErrUsernameExists) {
			status = http.StatusConflict
			msg = err.Error()
		}

		c.JSON(status, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusCreated, model.AuthResponse{
		ID:       user.ID,
		Username: user.Username,
		Message:  "注册成功",
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"details": err.Error(),
		})
		return
	}

	user, token, err := h.userService.Login(&req)
	if err != nil {
		status := http.StatusInternalServerError
		msg := "登录失败"

		if errors.Is(err, service.ErrInvalidCredentials) {
			status = http.StatusUnauthorized
			msg = err.Error()
		}

		c.JSON(status, gin.H{"error": msg})
		return
	}

	c.JSON(http.StatusOK, model.AuthResponse{
		ID:       user.ID,
		Username: user.Username,
		Token:    token,
		IsAdmin:  user.IsAdmin,
		Message:  "登录成功",
	})
}
