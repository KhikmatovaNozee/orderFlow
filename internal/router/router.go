package router

import (
	"log/slog"

	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"
	healthhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/health"
	orderhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/order"
	producthandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/product"
	statshandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/stats"
	"github.com/KhikmatovaNozee/orderFlow/internal/middleware"
	"github.com/KhikmatovaNozee/orderFlow/internal/service/auth"

	"github.com/gin-gonic/gin"
)

func New(
	log *slog.Logger,
	authHandler *authhandler.Handler,
	jwtService *auth.JWTService,
	healthHandler *healthhandler.Handler,
	productHandler *producthandler.Handler,
	orderHandler *orderhandler.Handler,
	statsHandler *statshandler.Handler,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logging(log))
	r.Static("/uploads", "./uploads")

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

		authed.GET("/products", productHandler.List)
		authed.GET("/products/:id", productHandler.Get)

		authed.POST("/orders", orderHandler.Place)
		authed.GET("/orders", orderHandler.List)
		authed.GET("/orders/:id", orderHandler.Get)
		authed.PUT("/orders/:id/pay", orderHandler.Pay)
		authed.PUT("/orders/:id/cancel", orderHandler.Cancel)
	}

	manage := authed.Group("/manage")
	manage.Use(middleware.RequireRole("seller"))
	{
		manage.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "seller access granted",
			})
		})

		manage.POST("/products", productHandler.Create)
		manage.PUT("/products/:id", productHandler.Update)
		manage.DELETE("/products/:id", productHandler.Delete)
		manage.POST("/products/:id/photo", productHandler.UploadPhoto)
		manage.GET("/orders", orderHandler.ListSellerOrders)
		manage.GET("/orders/:id", orderHandler.GetSellerOrder)
		manage.GET("/stats", statsHandler.Get)
		manage.PUT("/orders/:id/ship", orderHandler.Ship)
	}

	return r
}
