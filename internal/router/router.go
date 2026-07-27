package router

import (
	"log/slog"

	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"
	healthhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/health"
	"github.com/KhikmatovaNozee/orderFlow/internal/middleware"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"

	"github.com/gin-gonic/gin"
)

func New(
	log *slog.Logger,
	authHandler *authhandler.Handler,
	jwtService *auth.JWTService,
	healthHandler *healthhandler.Handler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logging(log))

	r.GET("/health", healthHandler.Check)

	api := r.Group("/api/v1")
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", authHandler.Logout)
	}

	authed := api.Group("")
	authed.Use(middleware.Auth(jwtService))
	{
		authed.GET("/protected", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"user_id": c.MustGet("user_id"),
				"role":    c.MustGet("role"),
			})
		})
	}

	manage := authed.Group("/manage")
	manage.Use(middleware.RequireRole("seller"))
	{
		manage.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "seller access granted",
			})
		})
	}

	return r
}
