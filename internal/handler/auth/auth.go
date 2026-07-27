package auth

import (
	"errors"
	"net/http"

	"github.com/KhikmatovaNozee/orderFlow/internal/model"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service    *auth.Service
	jwtService *auth.JWTService
}
type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh"`
}

func NewHandler(service *auth.Service, jwtService *auth.JWTService) *Handler {
	return &Handler{
		service:    service,
		jwtService: jwtService,
	}
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

	user, err := h.service.Register(c.Request.Context(), req.Login, req.Password, req.Role)

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

func (h *Handler) Login(c *gin.Context) {
	var req loginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	accessToken, refreshToken, err := h.service.Login(
		c.Request.Context(),
		req.Login,
		req.Password,
	)

	if err != nil {
		if errors.Is(err, model.ErrInvalid) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid login or password",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	accessToken, refreshToken, err := h.service.Refresh(
		c.Request.Context(),
		req.RefreshToken,
	)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if err := h.service.Logout(
		c.Request.Context(),
		req.RefreshToken,
	); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "invalid refresh token",
		})
		return
	}

	c.Status(http.StatusNoContent)
}
