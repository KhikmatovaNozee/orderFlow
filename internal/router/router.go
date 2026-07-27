package router

import (
	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"

	"github.com/gin-gonic/gin"
)

func New(authHandler *authhandler.Handler) *gin.Engine {
	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	api := r.Group("/api/v1")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})

		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
		}
	}

	return r
}
