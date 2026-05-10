package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zhaojiewen/open-station/internal/domain/entity"
)

// UserHandler handles user-related requests
type UserHandler struct{}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	user := c.MustGet("user").(*entity.User)
	c.JSON(http.StatusOK, gin.H{
		"id":     user.ID,
		"email":  user.Email,
		"name":   user.Name,
		"role":   user.Role,
		"status": user.Status,
	})
}