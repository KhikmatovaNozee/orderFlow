package auth

import (
	"errors"
	"net/http"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *auth.Service
}

func NewHandler(service *auth.Service) *Handler {
	return &Handler{service: service}
}

type registerRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (h *Handler) Register(c *gin.Context) {
	var req registerRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	user, err := h.service.Register(
		c.Request.Context(),
		req.Login,
		req.Password,
		req.Role)

	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalid):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid login, password or role"})
		case errors.Is(err, model.ErrConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "login already exists"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         user.ID,
		"login":      user.Login,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}
